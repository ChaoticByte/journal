package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/awnumar/memguard"
)


var EntryIdAlreadyExists = errors.New("There already exists an entry at this timestamp!")
var EntryNotFound = errors.New("No entry exists at this timestamp!")
var UnsupportedJournalVersion = errors.New("Unsupported journal version!")
var FilepathIsDirectory = errors.New("The given filepath points to a directory!")
var JournalClosed = errors.New("Journal already closed, can't access data.")
var UnknownFileReadErr = errors.New("Unknown file read error")
var FileModifiedExternally = errors.New("The file was modified by another process since last read/write!")


// Journal Version -> App Version
// 0 ->     < 1.0.0
// 1 -> since 1.0.0
const JournalVersion = uint8(0)

const JournalFileMode = 0o644

const JournalPos_Version = 0
const JournalPos_Entries = 1

type JournalFile struct {
	Version uint8
	Filepath string
	entries map[uint64]EncryptedEntry
	needWrite bool
	closed bool
	statLastModTime time.Time
}

func (j *JournalFile) GetEntries() []uint64 {
	if j.closed { return []uint64{} }
	es := []uint64{}
	for ts := range j.entries {
		if ts != 0 { // filter out reserved entry 0
			es = append(es, ts)
		}
	}
	return es
}

func (j *JournalFile) GetLatestEntry() uint64 {
	// returns timestamp, or 0 if nonexistent
	es := j.GetEntries()
	if len(es) == 0 { return 0 }
	slices.Sort(es)
	return es[len(es)-1]
}

func (j *JournalFile) GetPreviousEntry(current uint64) uint64 {
	// returns timestamp, or 0 if not found
	es := j.GetEntries()
	if len(es) == 0 { return 0 }
	slices.Sort(es)
	last := uint64(0)
	for _, ts := range es {
		if current == ts {
			return last
		}
		last = ts
	}
	return 0
}

func (j *JournalFile) GetNextEntry(current uint64) uint64 {
	// returns timestamp, or 0 if not found
	es := j.GetEntries()
	if len(es) == 0 { return 0 }
	slices.Sort(es)
	lastWasCurrent := false
	for _, ts := range es {
		if lastWasCurrent {
			return ts
		}
		if current == ts {
			lastWasCurrent = true
		}
	}
	return 0
}

func (j *JournalFile) GetEntry(ts uint64) *EncryptedEntry {
	if j.closed { return nil }
	e, found := j.entries[ts]
	if !found { return nil }
	return &e
}

func (j *JournalFile) AddEntry(e *EncryptedEntry) error {
	if j.closed { return JournalClosed }
	if _, exists := j.entries[e.Timestamp]; exists {
		return EntryIdAlreadyExists
	}
	j.entries[e.Timestamp] = *e
	j.needWrite = true
	return nil
}

func (j *JournalFile) DeleteEntry(ts uint64) error {
	if j.closed { return JournalClosed }
	delete(j.entries, ts)
	j.needWrite = true
	return nil
}

func (j *JournalFile) Write() error {
	if j.closed { return JournalClosed }
	// check if the file was modified since the last check
	mod, err := j.CheckIfExternallyModified()
	if err != nil { 
		if !os.IsNotExist(err) {
			return err
		}
	}
	if mod {
		return FileModifiedExternally
	}
	// write to file, if j.need_write
	if j.needWrite {
		// write to temporary file first, to prevent corrupted files
		tmp := fmt.Sprintf("%s.tmp_%v", j.Filepath, time.Now().UnixMicro())
		fTmp, err := os.OpenFile(tmp, os.O_WRONLY | os.O_CREATE, JournalFileMode)
		if err != nil { return err }
		_, err = fTmp.Write([]byte{j.Version})
		if err != nil { return err }
		es := []*EncryptedEntry{}
		for _, v := range j.entries {
			es = append(es, &v)
		}
		entryData := SerializeEntries(es)
		_, err = fTmp.Write(entryData)
		if err != nil { return err }
		// move temporary file to real file
		err = os.Rename(tmp, j.Filepath)
		j.needWrite = false
	}
	err = j.updateLastModifiedTime()
	return err
}

func (j *JournalFile) Close() {
	j.Write()
	j.closed = true
}

func (j *JournalFile) CheckIfExternallyModified() (modified bool, err error) {
	f, err := os.Stat(j.Filepath)
	if err != nil { return false, err }
	t := f.ModTime()
	return !j.statLastModTime.Equal(t), err
}

func (j *JournalFile) updateLastModifiedTime() error {
	f, err := os.Stat(j.Filepath)
	if err != nil { return err }
	j.statLastModTime = f.ModTime()
	return nil
}

func (j *JournalFile) read() error {
	if j.closed { return JournalClosed }
	// read from file (only at start or manually)
	f, err := os.OpenFile(j.Filepath, os.O_RDONLY, JournalFileMode)
	if err != nil { return err }
	data, err := io.ReadAll(f)
	j.Version = data[0]
	// Check if version is supported
	if j.Version != JournalVersion {
		return UnsupportedJournalVersion
	}
	// read entries
	j.entries = map[uint64]EncryptedEntry{}
	es := DeserializeEntries(data[JournalPos_Entries:])
	for _, e := range es {
		j.entries[e.Timestamp] = *e
	}
	err = j.updateLastModifiedTime()
	return err
}


func OpenJournalFile(file string, password *memguard.Enclave) (*JournalFile, error) {
	j := JournalFile{}
	j.Filepath = file
	// check file
	fileinfo, err := os.Stat(j.Filepath)
	if os.IsNotExist(err) {
		// create reserved entry 0
		e := &EncryptedEntry{Timestamp: 0}
		cipherText, salt, noncePfx, err := EncryptText(password, rand.Text(), e.Timestamp)
		if err != nil { return nil, err }
		e.EncryptedText = cipherText
		e.Salt = salt
		e.NoncePfx = noncePfx
		// init journal
		j.Version = JournalVersion
		j.entries = map[uint64]EncryptedEntry{}
		err = j.AddEntry(e); if err != nil { return &j, err }
		j.needWrite = true
		err = j.Write(); if err != nil { return &j, err }
	} else {
		if err != nil { return &j, err }
		if fileinfo == nil {
			return &j, UnknownFileReadErr
		} else if fileinfo.IsDir() {
			return &j, FilepathIsDirectory
		}
	}
	err = j.read(); if err != nil { return &j, err }
	// check password by decrypting reserved entry 0
	_, err = j.GetEntry(0).Decrypt(password)
	return &j, err
}


type EncryptedEntry struct {
	Timestamp uint64  // Unix time in microseconds, works until year 294246
	Salt [12]byte
	NoncePfx [16]byte // Nonce = random 16 bytes prefix + 8 byte timestamp
	EncryptedText []byte
}

func (e *EncryptedEntry) Decrypt(password *memguard.Enclave) (string, error) {
	txt, err := DecryptText(password, e.EncryptedText, e.Salt, e.NoncePfx, e.Timestamp)
	return txt, err
}

func (e *EncryptedEntry) EtLength() uint32 {
	return uint32(len(e.EncryptedText))
}

func NewEncryptedEntry(text string, password *memguard.Enclave) (*EncryptedEntry, error) {
	e := EncryptedEntry{}
	e.Timestamp = uint64(time.Now().UnixMicro())
	ct, s, n, err := EncryptText(password, text, e.Timestamp)
	if err != nil {
		return &e, err
	}
	e.EncryptedText = ct
	e.Salt = s
	e.NoncePfx = n
	return &e, err
}

func SerializeEntries(es []*EncryptedEntry) []byte {
	ees := []*encodedEntry{}
	for _, e := range es {
		ee := encodeEntry(e)
		ees = append(ees, ee)
	}
	return serializeEncodedEntries(ees)
}

func DeserializeEntries(data []byte) []*EncryptedEntry {
	ees := deserializeEncodedEntries(data)
	es := []*EncryptedEntry{}
	for _, ee := range ees {
		e := decodeEntry(ee)
		es = append(es, e)
	}
	return es
}

// very internal

type encodedEntry struct {
	// all integers are ordered big-endian
	Timestamp [8]byte   //  0- 7   uint64
	Salt [12]byte       //  8-19
	NoncePfx [16]byte   // 20-35
	CtLength [4]byte    // 36-39
	CipherText []byte   // 40-...  utf-8-encoded, encrypted
}

const payloadStart = 40

func encodeEntry(e *EncryptedEntry) *encodedEntry {
	ee := encodedEntry{}
	// timestamp
	binary.BigEndian.PutUint64(ee.Timestamp[:], e.Timestamp)
	// encrypt
	ee.CipherText = e.EncryptedText
	ee.Salt = e.Salt
	ee.NoncePfx = e.NoncePfx
	// length
	binary.BigEndian.PutUint32(ee.CtLength[:], e.EtLength())
	// done
	return &ee
}

func decodeEntry(ee *encodedEntry) *EncryptedEntry {
	e := EncryptedEntry{}
	e.Timestamp = binary.BigEndian.Uint64(ee.Timestamp[:])
	e.Salt = ee.Salt
	e.NoncePfx = ee.NoncePfx
	e.EncryptedText = ee.CipherText
	return &e
}

func serializeEncodedEntries(ees []*encodedEntry) []byte {
	b := []byte{}
	for _, ee := range ees {
		b = append(b, ee.Timestamp[:]...)
		b = append(b, ee.Salt[:]...)
		b = append(b, ee.NoncePfx[:]...)
		b = append(b, ee.CtLength[:]...)
		b = append(b, ee.CipherText...)
	}
	return b
}

func deserializeEncodedEntries(data []byte) []*encodedEntry {
	ees := []*encodedEntry{}
	lenD := len(data)
	o := 0 // offset
	for {
		if lenD < o + payloadStart { break } // no more valid data.
		ee := encodedEntry{}
		ee.Timestamp = [8]byte(data[o+0:o+8])
		ee.Salt = [12]byte(data[o+8:o+20])
		ee.NoncePfx = [16]byte(data[o+20:o+36])
		ee.CtLength = [4]byte(data[o+36:o+payloadStart])
		ctLen := int(binary.BigEndian.Uint32(ee.CtLength[:]))
		if lenD < o + payloadStart + ctLen { break } // no more valid data.
		ee.CipherText = data[o+payloadStart:o+payloadStart+ctLen]
		ees = append(ees, &ee)
		o += payloadStart + int(ctLen)
	}
	return ees
}

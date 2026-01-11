package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

const EntryIdAlreadyExistsMsg = "There already exists an entry at this timestamp!"
const EntryIdDoesNotExistMsg = "No entry exists at this timestamp!"
const UnsupportedJournalVersionMsg = "Unsupported journal version!"
const FilepathIsDirectoryMsg = "The given filepath points to a directory!"
const JournalClosedMsg = "Journal already closed, can't access data."
const UnknownFileReadErrMsg = "Unknown file read error"

// App Version -> Journal Version
// <1.0 -> 0
const JournalVersion = uint8(0)

const JournalFileMode = 0o644

const JournalPos_Version = 0
const JournalPos_Lock = 1
const JournalPos_Entries = 2

type JournalFile struct {
	Version uint8
	Readonly bool
	Filepath string
	entries map[uint64]EncryptedEntry
	need_read bool
	need_write bool
	closed bool
}

func (j *JournalFile) GetEntries() []*EncryptedEntry {
	if j.closed { return []*EncryptedEntry{} }
	es := []*EncryptedEntry{}
	for _, e := range j.entries {
		es = append(es, &e)
	}
	return es
}

func (j *JournalFile) GetEntry(ts uint64) *EncryptedEntry {
	if j.closed { return nil }
	j.Read()
	e := j.entries[ts]
	return &e
}

func (j *JournalFile) AddEntry(e *EncryptedEntry) error {
	if j.closed { return errors.New(JournalClosedMsg) }
	if _, exists := j.entries[e.Timestamp]; exists {
		return errors.New(EntryIdAlreadyExistsMsg)
	}
	j.entries[e.Timestamp] = *e
	j.need_write = true
	return nil
}

func (j *JournalFile) HideEntry(ts uint64) error {
	if j.closed { return errors.New(JournalClosedMsg) }
	if _, exists := j.entries[ts]; !exists {
		return errors.New(EntryIdAlreadyExistsMsg)
	}
	j.need_write = true
	return nil
}

func (j *JournalFile) DeleteEntry(ts uint64) error {
	if j.closed { return errors.New(JournalClosedMsg) }
	delete(j.entries, ts)
	j.need_write = true
	return nil
}

func (j *JournalFile) Read() error {
	if j.closed { return errors.New(JournalClosedMsg) }
	// read from file, if j.need_read (only at start)
	if j.need_read {
		f, err := os.OpenFile(j.Filepath, os.O_RDONLY, JournalFileMode)
		if err != nil { return err }
		data, err := io.ReadAll(f)
		j.Version = data[0]
		// Check if version is supported
		if j.Version != JournalVersion {
			return errors.New(UnsupportedJournalVersionMsg)
		}
		// read entries
		j.entries = map[uint64]EncryptedEntry{}
		es := DeserializeEntries(data[JournalPos_Entries:])
		for _, e := range es {
			j.entries[e.Timestamp] = *e
		}
		j.need_read = false
	}
	return nil
}

func (j *JournalFile) Write() error {
	if j.closed { return errors.New(JournalClosedMsg) }
	// write to file, if j.need_write
	if j.need_write && !j.Readonly {
		// write to temporary file first, to prevent corrupted files
		tmp := fmt.Sprintf("%s.tmp_%v", j.Filepath, time.Now().UnixMicro())
		fTmp, err := os.OpenFile(tmp, os.O_WRONLY | os.O_CREATE, JournalFileMode)
		if err != nil { return err }
		_, err = fTmp.Write([]byte{
			j.Version,
			1, // journal should be locked
		})
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
		j.need_write = false
	}
	return nil
}

func (j *JournalFile) Lock() error {
	if j.closed { return errors.New(JournalClosedMsg) }
	// Lock file so another instance knows that it is opened here.
	// Only call once or after Unlock()!
	f, err := os.OpenFile(j.Filepath, os.O_RDWR, JournalFileMode)
	if err != nil { return err }
	data := make([]byte, 1)
	_, err = f.ReadAt(data, JournalPos_Lock)
	if err != nil { return err }
	j.Readonly = data[0] > 0
	if !j.Readonly {
		data[0] = 1
		_, err = f.WriteAt(data, JournalPos_Lock)
	}
	return err
}

func (j *JournalFile) Unlock() error {
	if j.closed { return errors.New(JournalClosedMsg) }
	f, err := os.OpenFile(j.Filepath, os.O_WRONLY, JournalFileMode)
	if err != nil { return err }
	data := []byte{0}
	_, err = f.WriteAt(data, JournalPos_Lock)
	return err
}

func (j *JournalFile) Close() error {
	err := j.Unlock()
	j.closed = true
	return err
}


func OpenJournalFile(file string) (*JournalFile, error) {
	j := JournalFile{}
	j.Filepath = file
	// check file
	fileinfo, err := os.Stat(j.Filepath)
	doesNotExist := os.IsNotExist(err)
	if !doesNotExist {
		if err != nil { return &j, err }
		if fileinfo == nil {
			return &j, errors.New(UnknownFileReadErrMsg)
		} else if fileinfo.IsDir() {
			return &j, errors.New(FilepathIsDirectoryMsg)
		}
		// read
		j.need_read = true
		err = j.Lock(); if err != nil { return &j, err }
		err = j.Read(); if err != nil { return &j, err }
	} else {
		// init
		j.Version = JournalVersion
		j.entries = map[uint64]EncryptedEntry{}
		j.need_read = false
		j.need_write = true
		err = j.Write()
		if err != nil { return &j, err }
		err = j.Lock()
		if err != nil { return &j, err }
		j.Readonly = false
	}
	return &j, err
}


type EncryptedEntry struct {
	Timestamp uint64  // Unix time in microseconds, works until year 294246
	Hidden bool
	Salt [12]byte
	NoncePfx [16]byte // Nonce = random 16 bytes prefix + 8 byte timestamp
	EncryptedText []byte
}

func (e *EncryptedEntry) Decrypt(password string) (string, error) {
	txt, err := DecryptText(password, e.EncryptedText, e.Salt, e.NoncePfx, e.Timestamp)
	return txt, err
}

func (e *EncryptedEntry) EtLength() uint32 {
	return uint32(len(e.EncryptedText))
}

func NewEncryptedEntry(text string, password string) (*EncryptedEntry, error) {
	e := EncryptedEntry{}
	e.Timestamp = uint64(time.Now().UnixMicro())
	e.Hidden = false
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
	Hidden byte         //  8      0 = false, > 0 = true
	Salt [12]byte       //  9-20
	NoncePfx [16]byte   // 21-36
	CtLength [4]byte    // 37-40
	CipherText []byte   // 41-...  utf-8-encoded, encrypted
}

const payloadStart = 41

func encodeEntry(e *EncryptedEntry) *encodedEntry {
	ee := encodedEntry{}
	// timestamp
	binary.BigEndian.PutUint64(ee.Timestamp[:], e.Timestamp)
	// deleted flag
	if e.Hidden { ee.Hidden = 1 } else { ee.Hidden = 0 }
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
	e.Hidden = ee.Hidden > 0
	e.Salt = ee.Salt
	e.NoncePfx = ee.NoncePfx
	e.EncryptedText = ee.CipherText
	return &e
}

func serializeEncodedEntries(ees []*encodedEntry) []byte {
	b := []byte{}
	for _, ee := range ees {
		b = append(b, ee.Timestamp[:]...)
		b = append(b, ee.Hidden)
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
		ee.Hidden = data[o+8]
		ee.Salt = [12]byte(data[o+9:o+21])
		ee.NoncePfx = [16]byte(data[o+21:o+37])
		ee.CtLength = [4]byte(data[o+37:o+payloadStart])
		ctLen := int(binary.BigEndian.Uint32(ee.CtLength[:]))
		if lenD < o + payloadStart + ctLen { break } // no more valid data.
		ee.CipherText = data[o+payloadStart:o+payloadStart+ctLen]
		ees = append(ees, &ee)
		o += payloadStart + int(ctLen)
	}
	return ees
}

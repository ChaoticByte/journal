package main

import (
	"encoding/binary"
	"time"
)

type Entry struct {
	Timestamp uint64 // Unix time in s
	Deleted bool
	Text string
}

func NewEntry(text string) *Entry {
	e := Entry{}
	e.Timestamp = uint64(time.Now().Unix())
	e.Deleted = false
	e.Text = text
	return &e
}

func SerializeAndEncryptEntries(es []*Entry, password string) ([]byte, error) {
	ees := []*encodedEntry{}
	for _, e := range es {
		ee, err := encodeEncryptEntry(e, password)
		if err != nil {
			return []byte{}, err
		}
		ees = append(ees, ee)
	}
	return serializeEncodedEntries(ees), nil
}

func DeserializeAndDecryptEntries(data []byte, password string) ([]*Entry, error) {
	ees := deserializeEncodedEntries(data)
	es := []*Entry{}
	for _, ee := range ees {
		e, err := decodeDecryptEntry(ee, password)
		if err != nil {
			return es, err
		}
		es = append(es, e)
	}
	return es, nil
}

// very internal

type encodedEntry struct {
	// all integers are ordered big-endian
	Timestamp [8]byte	//  0- 7   uint64
	Deleted byte		//  8      0 = false, > 0 = true
	Salt [12]byte		//  9-20
	Nonce [24]byte		// 21-44
	CtLength [4]byte	// 45-48
	CipherText []byte   // 49-...  utf-8-encoded, encrypted
}

const payloadStart = 49

func encodeEncryptEntry(e *Entry, password string) (*encodedEntry, error) {
	ee := encodedEntry{}
	// timestamp
	binary.BigEndian.PutUint64(ee.Timestamp[:], e.Timestamp)
	// deleted flag
	if e.Deleted { ee.Deleted = 1 } else { ee.Deleted = 0 }
	// encrypt
	var err error = nil
	ct, s, n, err := EncryptText(password, e.Text)
	if err != nil {
		return &ee, err
	}
	ee.CipherText = ct
	ee.Salt = s
	ee.Nonce = n
	// length
	binary.BigEndian.PutUint32(ee.CtLength[:], uint32(len(ee.CipherText)))
	// done
	return &ee, err
}

func decodeDecryptEntry(ee *encodedEntry, password string) (*Entry, error) {
	e := Entry{}
	e.Timestamp = binary.BigEndian.Uint64(ee.Timestamp[:])
	e.Deleted = ee.Deleted > 0
	// TODO: decrypt
	txt, err := DecryptText(password, ee.CipherText, ee.Salt, ee.Nonce)
	if err != nil {
		return &e, err
	}
	e.Text = txt
	return &e, err
}

func serializeEncodedEntries(ees []*encodedEntry) []byte {
	b := []byte{}
	for _, ee := range ees {
		b = append(b, ee.Timestamp[:]...)
		b = append(b, ee.Deleted)
		b = append(b, ee.Salt[:]...)
		b = append(b, ee.Nonce[:]...)
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
		ee.Deleted = data[o+8]
		ee.Salt = [12]byte(data[o+9:o+21])
		ee.Nonce = [24]byte(data[o+21:o+45])
		ee.CtLength = [4]byte(data[o+45:o+payloadStart])
		ctLen := int(binary.BigEndian.Uint32(ee.CtLength[:]))
		if lenD < o + payloadStart + ctLen { break } // no more valid data.
		ee.CipherText = data[o+payloadStart:o+payloadStart+ctLen]
		ees = append(ees, &ee)
		o += payloadStart + int(ctLen)
	}
	return ees
}

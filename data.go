package main

import (
	"encoding/binary"
	"time"
)

type EncryptedEntry struct {
	Timestamp uint64 // Unix time in s
	Deleted bool
	Salt [12]byte
	Nonce [24]byte
	EncryptedText []byte
}

func (e *EncryptedEntry) Decrypt(password string) (string, error) {
	txt, err := DecryptText(password, e.EncryptedText, e.Salt, e.Nonce)
	if err != nil {
		return "", err
	}
	return txt, err
}

func (e *EncryptedEntry) EtLength() uint32 {
	return uint32(len(e.EncryptedText))
}

func NewEncryptedEntry(text string, password string) (*EncryptedEntry, error) {
	e := EncryptedEntry{}
	e.Timestamp = uint64(time.Now().Unix())
	e.Deleted = false
	ct, s, n, err := EncryptText(password, text)
	if err != nil {
		return &e, err
	}
	e.EncryptedText = ct
	e.Salt = s
	e.Nonce = n
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
	Timestamp [8]byte	//  0- 7   uint64
	Deleted byte		//  8      0 = false, > 0 = true
	Salt [12]byte		//  9-20
	Nonce [24]byte		// 21-44
	CtLength [4]byte	// 45-48
	CipherText []byte   // 49-...  utf-8-encoded, encrypted
}

const payloadStart = 49

func encodeEntry(e *EncryptedEntry) *encodedEntry {
	ee := encodedEntry{}
	// timestamp
	binary.BigEndian.PutUint64(ee.Timestamp[:], e.Timestamp)
	// deleted flag
	if e.Deleted { ee.Deleted = 1 } else { ee.Deleted = 0 }
	// encrypt
	ee.CipherText = e.EncryptedText
	ee.Salt = e.Salt
	ee.Nonce = e.Nonce
	// length
	binary.BigEndian.PutUint32(ee.CtLength[:], e.EtLength())
	// done
	return &ee
}

func decodeEntry(ee *encodedEntry) *EncryptedEntry {
	e := EncryptedEntry{}
	e.Timestamp = binary.BigEndian.Uint64(ee.Timestamp[:])
	e.Deleted = ee.Deleted > 0
	e.Salt = ee.Salt
	e.Nonce = ee.Nonce
	e.EncryptedText = ee.CipherText
	return &e
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

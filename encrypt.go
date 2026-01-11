package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

func EncryptText(password string, cleartext string, time uint64) ([]byte, [12]byte, [16]byte, error) {
	salt := [12]byte{}
	_, err := rand.Read(salt[:])
	if err != nil { return []byte{}, salt, [16]byte{}, err }
	key := derive_key(password, salt)
	noncePfx := [16]byte{}
	_, err = rand.Read(noncePfx[:])
	if err != nil { return []byte{}, salt, noncePfx, err }
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil { return []byte{}, salt, noncePfx, err }
	nonce := []byte{}
	nonce = append(nonce, noncePfx[:]...)
	nonce = binary.BigEndian.AppendUint64(nonce, time)
	src := []byte(cleartext)
	dst := aead.Seal(nil, nonce, src, nil)
	return dst, salt, noncePfx, err
}

func DecryptText(password string, ciphertext []byte, salt [12]byte, noncePfx [16]byte, time uint64) (string, error) {
	nonce := []byte{}
	nonce = append(nonce, noncePfx[:]...)
	nonce = binary.BigEndian.AppendUint64(nonce, time)
	if len(nonce) != 24 { return "", errors.New(ErrMsgInvalidNonceLen) }
	key := derive_key(password, salt)
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil { return "", err }
	dst, err := aead.Open(nil, nonce[:], ciphertext, nil)
	if err != nil { return "", err }
	result := string(dst)
	return result, err
}

// internal

const a2_time = 10
const a2_mem = 128*1024
const a2_thr = 2

func derive_key(password string, salt [12]byte) [32]byte {
	return [32]byte(
		argon2.IDKey([]byte(password), salt[:], a2_time, a2_mem, a2_thr, 32))
}

package main

import (
	"crypto/rand"
	"encoding/binary"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20"
)

func EncryptText(password string, cleartext string, time uint64) ([]byte, [12]byte, [16]byte, error) {
	salt := [12]byte{}
	_, err := rand.Read(salt[:])
	if err != nil {
		return []byte{}, salt, [16]byte{}, err
	}
	key := derive_key(password, salt)
	noncePfx := [16]byte{}
	_, err = rand.Read(noncePfx[:])
	if err != nil {
		return []byte{}, salt, noncePfx, err
	}
	nonce := []byte{}
	nonce = append(nonce, noncePfx[:]...)
	nonce = binary.BigEndian.AppendUint64(nonce, time)
	c, err := chacha20.NewUnauthenticatedCipher(key[:], nonce)
	if err != nil {
		return []byte{}, salt, noncePfx, err
	}
	src := []byte(cleartext)
	dst := make([]byte, len(cleartext))
	c.XORKeyStream(dst, src)
	return dst, salt, noncePfx, err
}

func DecryptText(password string, ciphertext []byte, salt [12]byte, nonce [24]byte) (string, error) {
	key := derive_key(password, salt)
	c, err := chacha20.NewUnauthenticatedCipher(key[:], nonce[:])
	if err != nil {
		return "", err
	}
	dst := make([]byte, len(ciphertext))
	c.XORKeyStream(dst, ciphertext)
	result := string(dst)
	return result, err
}

// TODO: maybe use XChaCha20-Poly1305?

// internal

const a2_time = 10
const a2_mem = 128*1024
const a2_thr = 2

func derive_key(password string, salt [12]byte) [32]byte {
	return [32]byte(
		argon2.IDKey([]byte(password), salt[:], a2_time, a2_mem, a2_thr, 32))
}

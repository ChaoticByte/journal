package main

import (
	"crypto/rand"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20"
)

func EncryptText(password string, cleartext string) ([]byte, [12]byte, [24]byte, error) {
	salt := [12]byte{}
	_, err := rand.Read(salt[:])
	if err != nil {
		return []byte{}, salt, [24]byte{}, err
	}
	key := derive_key(password, salt)
	nonce := [24]byte{}
	_, err = rand.Read(nonce[:])
	if err != nil {
		return []byte{}, salt, nonce, err
	}
	c, err := chacha20.NewUnauthenticatedCipher(key[:], nonce[:])
	if err != nil {
		return []byte{}, salt, nonce, err
	}
	src := []byte(cleartext)
	dst := make([]byte, len(cleartext))
	c.XORKeyStream(dst, src)
	return dst, salt, nonce, err
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

const a2_time = 1
const a2_mem = 64*1024
const a2_thr = 2

func derive_key(password string, salt [12]byte) [32]byte {
	return [32]byte(
		argon2.IDKey([]byte(password), salt[:], a2_time, a2_mem, a2_thr, 32))
}

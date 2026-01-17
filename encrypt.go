package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"

	"github.com/awnumar/memguard"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

/*

The cipher used for encryption is XChaCha20-Poly1305, which is an AEAD algorithm.

XChaCha20 is a ChaCha20 streaming cipher with a 24 byte nonce length.
Poly1305 is the message authentication part (checks if the decrypted data is correct).

No 'associated data' is written or read.

The 32 byte key for encryption is derived using Argon2ID. A 12-byte random salt is used.

*/

const ErrMsgInvalidNonceLen = "Assembled nonce has an invalid length!"

func EncryptText(password *memguard.Enclave, cleartext string, time uint64) ([]byte, [12]byte, [16]byte, error) {
	// create random salt
	salt := [12]byte{}
	_, err := rand.Read(salt[:])
	if err != nil { return []byte{}, salt, [16]byte{}, err }
	// derive key
	lb, err := password.Open()
	defer lb.Destroy()
	if err != nil { return []byte{}, salt, [16]byte{}, err }
	key := derive_key(lb.Bytes(), salt)
	lb.Destroy()
	// assemble nonce
	noncePfx := [16]byte{}
	_, err = rand.Read(noncePfx[:])
	if err != nil { return []byte{}, salt, noncePfx, err }
	nonce := []byte{}
	nonce = append(nonce, noncePfx[:]...)
	nonce = binary.BigEndian.AppendUint64(nonce, time)
	// create aead cipher
	aead, err := chacha20poly1305.NewX(key[:])
	key = [32]byte{} // remove key from memory
	if err != nil { return []byte{}, salt, noncePfx, err }
	// encrypt
	src := []byte(cleartext)
	dst := aead.Seal(nil, nonce, src, nil)
	return dst, salt, noncePfx, err
}

func DecryptText(password *memguard.Enclave, ciphertext []byte, salt [12]byte, noncePfx [16]byte, time uint64) (string, error) {
	// derive key
	lb, err := password.Open()
	defer lb.Destroy()
	if err != nil { return "", err }
	key := derive_key(lb.Bytes(), salt)
	lb.Destroy()
	// assemble nonce
	nonce := []byte{}
	nonce = append(nonce, noncePfx[:]...)
	nonce = binary.BigEndian.AppendUint64(nonce, time)
	if len(nonce) != 24 { return "", errors.New(ErrMsgInvalidNonceLen) }
	// create aead cipher
	aead, err := chacha20poly1305.NewX(key[:])
	key = [32]byte{} // remove key from memory
	if err != nil { return "", err }
	// decrypt
	dst, err := aead.Open(nil, nonce[:], ciphertext, nil)
	if err != nil { return "", err }
	result := string(dst)
	return result, err
}

// key derivation

const a2_time = 10
const a2_mem = 128*1024
const a2_thr = 2

func derive_key(password []byte, salt [12]byte) [32]byte {
	return [32]byte(
		argon2.IDKey(password, salt[:], a2_time, a2_mem, a2_thr, 32))
}

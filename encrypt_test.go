// Copyright (c) 2026, Julian Müller (ChaoticByte)

package main

import (
	crand "crypto/rand"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/awnumar/memguard"
)

func TestKeyDerivation(t *testing.T) {
	password1 := []byte("test")
	password2 := make([]byte, 20)
	crand.Read(password2)
	salt1 := [12]byte{}
	crand.Read(salt1[:])
	salt2 := [12]byte{}
	crand.Read(salt2[:])
	//
	key1 := derive_key(password1, salt1)
	key2 := derive_key(password2, salt2)
	//
	if key1 == key2 { t.Error("derived key1 == key2!") }
	//
	key1_salt2 := derive_key(password1, salt2)
	key2_salt1 := derive_key(password2, salt1)
	if key1 == key1_salt2 { t.Error("derived key1 == (key1 with wrong salt)!") }
	if key2 == key2_salt1 { t.Error("derived key2 == (key2 with wrong salt)!") }
	//
	rekey1 := derive_key(password1, salt1)
	rekey2 := derive_key(password2, salt2)
	if rekey1 != key1 { t.Error("kdf is non-deterministic! derived key1 != re-key1!") }
	if rekey2 != key2 { t.Error("kdf is non-deterministic! derived key2 != re-key2!") }
}

func trialWithTamperedInput(t *testing.T, what string, password *memguard.Enclave, ciphertext []byte, salt [12]byte, noncePfx [16]byte, time uint64, original string) {
	clt_t, err := DecryptText(password, ciphertext, salt, noncePfx, time)
	if err == nil || err.Error() != "chacha20poly1305: message authentication failed" {
		t.Errorf("Could decrypt with tampered %v; message authentication not functioning properly!", what)
	} else if clt_t == original {
		t.Errorf("Could decrypt with tampered %v and no error given by aead.Open()! message authentication not functioning properly!", what)
	}
}

func TestCrypto(t *testing.T) {
	password1 := memguard.NewEnclave([]byte("test"))
	password2_bytes := make([]byte, 20)
	crand.Read(password2_bytes)
	password2 := memguard.NewEnclave(password2_bytes)
	t1 := uint64(time.Now().UnixMicro())
	time.Sleep(time.Duration(1.0 + rand.Float64()) * time.Second)
	t2 := uint64(time.Now().UnixMicro())
	cleartext := "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet."
	//
	cit1, salt1, noncePfx1, err1 := EncryptText(password1, cleartext, t1)
	if err1 != nil { t.Fatalf("Could not encrypt with password1, err: %v", err1) }
	cit2, salt2, noncePfx2, err2 := EncryptText(password2, cleartext, t2)
	if err2 != nil { t.Fatalf("Could not encrypt with password2, err: %v", err2) }
	//
	if salt1 == salt2 {
		t.Error("salt1 and salt2 are the same!")
	}
	if noncePfx1 == noncePfx2 {
		t.Error("random part of nonce 1 and 2 are the same")
	}
	if slices.Equal(cit1, []byte(cleartext)) {
		t.Error("ciphertext1 == cleartext!")
	}
	if slices.Equal(cit2, []byte(cleartext)) {
		t.Error("ciphertext2 == cleartext!")
	}
	//
	clt_decrypted1, err := DecryptText(password1, cit1, salt1, noncePfx1, t1)
	if err != nil {
		t.Error("Could not decrypt ciphertext1 using password1!")
	}
	if clt_decrypted1 != cleartext {
		t.Error("Decrypted ciphertext1 does not equal original ciphertext!")
	}
	clt_decrypted2, err := DecryptText(password2, cit2, salt2, noncePfx2, t2)
	if err != nil {
		t.Error("Could not decrypt ciphertext2 using password1!")
	}
	if clt_decrypted2 != cleartext {
		t.Error("Decrypted ciphertext2 does not equal original ciphertext!")
	}
	//
	trialWithTamperedInput(t, "wrong password", password2, cit1, salt1, noncePfx1, t1, cleartext)
	trialWithTamperedInput(t, "wrong password", password1, cit2, salt2, noncePfx2, t2, cleartext)
	trialWithTamperedInput(t, "tampered nonce prefix", password1, cit1, salt1, noncePfx2, t1, cleartext)
	trialWithTamperedInput(t, "tampered nonce prefix", password2, cit2, salt2, noncePfx1, t2, cleartext)
	//
	cit1_tampered := make([]byte, len(cit1))
	copy(cit1_tampered, cit1)
	if cit1_tampered[3] < 255 {
		cit1_tampered[3] += 1
	} else {
		cit1_tampered[3] -= 1
	}
	trialWithTamperedInput(t, "tampered ciphertext", password1, cit1_tampered, salt1, noncePfx1, t1, cleartext)
}

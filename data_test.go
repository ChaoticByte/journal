// Copyright (c) 2026, Julian Müller (ChaoticByte)

package main

import (
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/awnumar/memguard"
)

const JournalTestFile = "/tmp/journal_test"

func TestDataformat(t *testing.T) {
	passwd := memguard.NewEnclave([]byte("secureTestP4ssw0rd!"))
	defer memguard.Purge()
	defer os.Remove(JournalTestFile)
	var j *JournalFile
	var err error
	t.Run("CreateJournalFile", func(t *testing.T) {
		// create test journal
		os.Remove(JournalTestFile) // possibly remove old file
		j, err = OpenJournalFile(JournalTestFile, passwd)
		if err != nil {
			t.Error("Could not create test journal; ", err)
		}
	})
	// define entries
	entryTexts := []string {
		"Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat.",
		"Sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat.",
		"Sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet.",
		"Consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet.",
		"This is just a test",
		"Another test",
		"test",
		"aaaa",
	}
	tb := strings.Builder{}
	for i := range 1024*1024 { // big entry
		fmt.Fprintf(&tb, "%x", i)
		if err != nil {
			t.Error("Unknown error when writing to string builder")
		}
	}
	entryTexts = append(entryTexts, tb.String())
	t.Run("CreateAndAddEntries", func(t *testing.T) {
		for i, txt := range entryTexts {
			e, err := NewEncryptedEntry(txt, passwd)
			if err != nil {
				t.Errorf("Could not create entry %v! %v", i, err)
			}
			err = j.AddEntry(e)
			if err != nil {
				t.Errorf("Could not add entry %v to journal! %v", i, err)
			}
			j.Write()
			time.Sleep(time.Duration(rand.Float64() + 1.0) * time.Second)
		}
	})
	t.Run("ReopenJournalFile", func(t *testing.T) {
		// re-open
		j.Close()
		j, err = OpenJournalFile(JournalTestFile, passwd)
		if err != nil {
			t.Error("Could not open test journal; ", err)
		}
	})
	t.Run("ReadEntries", func(t *testing.T) {
		es := j.GetEntries()
		len_es := len(es)
		lenInput := len(entryTexts)
		if len_es != lenInput {
			t.Errorf("Invalid number of entries! Expected %v, but got %v", lenInput, len_es)
		}
		slices.Sort(es) // this is important to get the right order!
		for i, ts := range es {
			e := j.GetEntry(ts)
			if e == nil {
				t.Errorf("Could not get entry %v!", ts)
			}
			txt, err := e.Decrypt(passwd)
			if err != nil {
				t.Errorf("Could not decrypt entry %v! %v", ts, err)
			}
			if txt != entryTexts[i] {
				t.Errorf("Decrypted text of entry %v does not match input text!", ts)
			}
		}
	})
	var removed_ts uint64
	t.Run("RemoveEntries", func(t *testing.T) {
		removed_ts = j.GetEntries()[0]
		err := j.DeleteEntry(removed_ts)
		if err != nil {
			t.Errorf("Could not delete entry %v from journal!", removed_ts)
		}
	})
	t.Run("ReopenJournalFile2", func(t *testing.T) {
		// re-open
		j.Close()
		j, err = OpenJournalFile(JournalTestFile, passwd)
		if err != nil {
			t.Error("Could not open test journal; ", err)
		}
	})
	t.Run("CheckRemovedEntries", func(t *testing.T) {
		for _, ts := range j.GetEntries() {
			if ts == removed_ts {
				t.Error("Found deleted entry!")
			}
		}
	})
}

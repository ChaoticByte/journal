package main

import (
	"fmt"
	"os"
	"time"
)

// import "os"

func HandleErrorExit(err error) {
	if err != nil {
		fmt.Println("Error", err);
		os.Exit(1)
	}
}

func main() {
	// Cli(os.Args)
	pass := "test password"
	// add entries
	j, err := OpenJournalFile("/home/julian/Dokumente/journal_test"); HandleErrorExit(err)
	e, err := NewEncryptedEntry("aaaaaaa", pass); HandleErrorExit(err);
	j.AddEntry(e)
	time.Sleep(10 * time.Millisecond)
	e2, err := NewEncryptedEntry("bcdefghij", pass); HandleErrorExit(err);
	j.AddEntry(e2)
	for _, e := range j.GetEntries() {
		t, err := e.Decrypt(pass); HandleErrorExit(err);
		fmt.Println(t)
	}
	// write and close
	err = j.Write(); HandleErrorExit(err);
	j.Close()
	// open second journal with same file
	j2, err := OpenJournalFile("/home/julian/Dokumente/journal_test"); HandleErrorExit(err)
	defer j2.Close()
	for _, e := range j2.GetEntries() {
		t, err := e.Decrypt(pass); HandleErrorExit(err);
		fmt.Println(t)
	}
}

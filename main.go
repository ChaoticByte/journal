package main

import (
	"fmt"
	"os"
)

// import "os"

func HandleErrorExit(err error) {
	if err != nil {
		fmt.Println(err);
		os.Exit(1)
	}
}

func main() {
	// Cli(os.Args)
	pass := "test password"
	entries := []*EncryptedEntry{}
	e, err := NewEncryptedEntry("aaaaaaa", pass); HandleErrorExit(err);
	entries = append(entries, e)
	e2, err := NewEncryptedEntry("bcdefghij", pass); HandleErrorExit(err);
	entries = append(entries, e2)
	for _, e := range entries {
		t, err := e.Decrypt(pass); HandleErrorExit(err);
		fmt.Println(t)
	}
	data := SerializeEntries(entries)
	fmt.Println(data)
	entries_rese := DeserializeEntries(data)
	for _, e := range entries_rese {
		t, err := e.Decrypt(pass); HandleErrorExit(err);
		fmt.Println(t)
	}
}

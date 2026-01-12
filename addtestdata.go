// +build addtestdata

package main

import "os"

func main() {
	if len(os.Args) < 2 {
		panic("must pass test string as cmdline arg!")
	}
	pass := []byte("test")
	j, err := OpenJournalFile("./testjournal")
	defer j.Close()
	if err != nil {
		panic(err)
	}
	j.Readonly = false // enforce
	e, _ := NewEncryptedEntry(os.Args[1], pass)
	j.AddEntry(e)
}

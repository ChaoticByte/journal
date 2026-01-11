package main

import (
	"fmt"
	"os"
)

// import "os"

func main() {
	// Cli(os.Args)
	entries := []*Entry{}
	entries = append(entries, NewEntry("aaaaaaa"))
	entries = append(entries, NewEntry("bbbbb"))
	for _, e := range entries { fmt.Println(*e) }
	data, err := SerializeAndEncryptEntries(entries, "test password")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(data)
	entries_rese, err := DeserializeAndDecryptEntries(data, "test password")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, e := range entries_rese { fmt.Println(*e) }
}

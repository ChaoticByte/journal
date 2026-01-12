// +build !addtestdata

package main

import "os"

var j *JournalFile

func main() {
	CliEntrypoint(os.Args)
}

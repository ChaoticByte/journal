package main

import (
	"io"
	"os"
	"syscall"
)

var NanoEditorCmd = []string{"/usr/bin/nano", "--restricted", "--saveonexit", "--ignorercfiles", "--unix"}
var Editors map[string][]string = map[string][]string{
	"nano": NanoEditorCmd,
}

func GetTextFromEditor(editorCmd []string, existingText string) (string, error) {
	tmpfile, err := os.CreateTemp(existingText, ".jrnl") // default mode is 0600
	if err != nil { return existingText, err }
	defer os.Remove(tmpfile.Name()) // this may fail due to redundant call of os.Remove
	_, err = tmpfile.Write([]byte(existingText))
	if err != nil { return existingText, err }
	// call editor command
	ecmdln := make([]string, len(editorCmd))
	copy(ecmdln, editorCmd)
	ecmdln = append(ecmdln, tmpfile.Name())
	proc, err := os.StartProcess(
		editorCmd[0],
		ecmdln,
		&os.ProcAttr{
			Dir: "/",
			Env: os.Environ(),
			Files: []*os.File{
				os.Stdin, os.Stdout, os.Stderr,
			},
			Sys: &syscall.SysProcAttr{},
		},
	)
	if err != nil { return existingText, err }
	_, err = proc.Wait()
	if err != nil { return existingText, err }
	// read back text from file
	result, err := io.ReadAll(tmpfile)
	return string(result), err
}

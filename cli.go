package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/term"
)

// ANSI ESCAPE CODES

const (
	// suffixes
	A_SFX_MODE = "m"
	A_SFX_COLOR = A_SFX_MODE

	// modes
	A_SET_BOLD = "1"
	A_RES_BOLD = "22"
	A_SET_DIM = "2"
	A_RES_DIM = "22"
	A_SET_ITALIC = "3"
	A_RES_ITALIC = "23"
	A_SET_UNDERLINE = "4"
	A_RES_UNDERLINE = "24"
	A_SET_BLINK = "5"
	A_RES_BLINK = "25"
	A_SET_INVERTED = "7"
	A_RES_INVERTED = "27"
	A_SET_HIDDEN = "8"
	A_RES_HIDDEN = "28"
	A_SET_STRIKETHROUGH = "9"
	A_RES_STRIKETHROUGH = "29"

	// colors
	A_COL_BLACK_FG = "30"
	A_COL_BLACK_BG = "40"
	A_COL_RED_FG = "31"
	A_COL_RED_BG = "41"
	A_COL_GREEN_FG = "32"
	A_COL_GREEN_BG = "42"
	A_COL_YELLOW_FG = "33"
	A_COL_YELLOW_BG = "43"
	A_COL_BLUE_FG = "34"
	A_COL_BLUE_BG = "44"
	A_COL_MAGENTA_FG = "35"
	A_COL_MAGENTA_BG = "45"
	A_COL_CYAN_FG = "36"
	A_COL_CYAN_BG = "46"
	A_COL_WHITE_FG = "37"
	A_COL_WHITE_BG = "47"

	// reset colors
	A_COL_RES_FG = "39"
	A_COL_RES_BG = "49"

	// bright colors
	A_COL_BRIGHT_BLACK_FG = "90"
	A_COL_BRIGHT_BLACK_BG = "100"
	A_COL_BRIGHT_RED_FG = "91"
	A_COL_BRIGHT_RED_BG = "101"
	A_COL_BRIGHT_GREEN_FG = "92"
	A_COL_BRIGHT_GREEN_BG = "102"
	A_COL_BRIGHT_YELLOW_FG = "93"
	A_COL_BRIGHT_YELLOW_BG = "103"
	A_COL_BRIGHT_BLUE_FG = "94"
	A_COL_BRIGHT_BLUE_BG = "104"
	A_COL_BRIGHT_MAGENTA_FG = "95"
	A_COL_BRIGHT_MAGENTA_BG = "105"
	A_COL_BRIGHT_CYAN_FG = "96"
	A_COL_BRIGHT_CYAN_BG = "106"
	A_COL_BRIGHT_WHITE_FG = "97"
	A_COL_BRIGHT_WHITE_BG = "107"
)

func AE(suffix string, codes ...string) string {
	seq := "\u001b["
	seq += strings.Join(codes, ";")
	seq += suffix
	return seq
}

// Screen & erase

const A_ERASE_SCREEN = "\u001b[2J"
const A_ERASE_REST_OF_LINE = "\u001b[0K"
const A_ERASE_LINE = "\u001b[2K"

// const A_SAVE_SCREEN = "\u001b[?47h"
// const A_RESTORE_SCREEN = "\u001b[?47l"

// Cursor

const A_CUR_HOME = "\u001b[H"
const A_SAVE_CUR_POS = "\u001b7"
const A_RESTORE_CUR_POS = "\u001b8"

func ACurUp(lines int) string {
	return fmt.Sprintf("\u001b[%vA", lines)
}

func ACurDown(lines int) string {
	return fmt.Sprintf("\u001b[%vB", lines)
}

func ACurRight(cols int) string {
	return fmt.Sprintf("\u001b[%vC", cols)
}

func ACurLeft(cols int) string {
	return fmt.Sprintf("\u001b[%vD", cols)
}


// //

// some settings

const TextLeftMargin = 2

// terminal helpers

func Out(stuff ...any) {
	for _, s := range stuff {
		fmt.Print(s)
	}
}

func Nl() {
	Out("\n")
}

func Nnl(n int) {
	for range n {
		Out("\n")
	}
}

func MultipleChoice(choices [][2]string) int {
	// accepts a list of choices, each is a short keyword and the actual choice
	// returns the index.
	Nl()
	defer Nl()
	// returns the key
	for _, c := range choices {
		fmt.Printf("  %s) %s\n", c[0], c[1])
	}
	Nl()
	Out("> ", A_SAVE_CUR_POS)
	for {
		Out(A_RESTORE_CUR_POS, A_ERASE_REST_OF_LINE)
		var a string
		fmt.Scanln(&a)
		for i, c := range choices {
			if c[0] == a {
				return i
			}
		}
	}
}

func ReadPass() ([]byte, error) {
	Nl()
	Out("[ ] ")
	Out(A_SAVE_CUR_POS)
	// term.ReadPassword is very clever by not handling
	// sigints correctly (at least in go version 1.25.5),
	// so here we are with a very pleasant (not) workarout
	fd := int(os.Stdout.Fd())
	s, err := term.GetState(fd); if err != nil { return []byte{}, err }
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		term.Restore(fd, s)
		Nl()
		os.Exit(0)
	}()
	defer term.Restore(fd, s) // doppelt hält besser
	// my god.
	for {
		Out(A_RESTORE_CUR_POS, A_ERASE_REST_OF_LINE)
		pw, err := term.ReadPassword(fd); Nl()
		if err != nil || len(pw) > 0 {
			Nl()
			return pw, err
		}
	}
}

//

func mainloop(j *JournalFile) {
	Out(A_ERASE_SCREEN, A_CUR_HOME)
	defer Out(A_ERASE_SCREEN, A_CUR_HOME)
	// just for fun
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		Out(A_ERASE_SCREEN, A_CUR_HOME)
		os.Exit(0)
	}()
	// :)
	// ok.
	fmt.Scanln()
}

func ShowUsageAndExit(a0 string, code int) {
	a0Parts := strings.Split(a0, "/")
	binName := a0Parts[len(a0Parts)-1]
	Out("Usage: ", binName, " <path>\n\nPositional arguments\n\n\t<path>  Path to the journal file\n\n")
	os.Exit(code)
}

func CliEntrypoint(args []string) {
	// parse cli args
	if len(args) < 2 {
		ShowUsageAndExit(args[0], 1)
	}
	a1 := args[1]
	if a1 == "-h" || a1 == "--help" {
		ShowUsageAndExit(args[0], 0)
	}

	// clear screen and go home
	Out(A_ERASE_SCREEN, A_CUR_HOME); Nl()

	// read password
	Out("Please enter your encryption key."); Nl()
	_, err := ReadPass()
	if err != nil {
		Out("Couldn't get password from commandline safely."); Nl()
		Out(err); Nl()
		os.Exit(1)
	}

	// try to open journal file
	Out("Opening journal file at ", AE(A_SFX_MODE, A_SET_DIM), a1, AE(A_SFX_MODE, A_RES_DIM), " ...")
	Nnl(2);
	j, err := OpenJournalFile(a1)
	if err != nil { 
		Out(AE(A_SFX_COLOR, A_COL_RED_FG), "Couldn't open journal file!", AE(A_SFX_COLOR, A_COL_RES_FG)); Nl()
		Out(err); Nl()
	}
	if j.Readonly {
		Out(AE(A_SFX_COLOR, A_COL_RED_FG), "This journal is locked by another process!", AE(A_SFX_COLOR, A_COL_RES_FG)); Nnl(2)
		Out("Do you want to open it in Readonly-Mode or Read-Write-Mode (potentially dangerous)?"); Nl()
		if c := MultipleChoice([][2]string{
			{"ro", "readonly"},     // 0
			{"rw", "read & write"},	// 1
		}); c == 1 {
			j.Readonly = false
			Out("Program is in read-write mode. Be careful!")
			time.Sleep(4 * time.Second)
			Nl()
		} else {
			Out("Program is in readonly mode.")
			time.Sleep(2 * time.Second)
			Nl()
		}
	}
	if !j.Readonly { defer j.Close() }

	// mainloop
	mainloop(j)

}

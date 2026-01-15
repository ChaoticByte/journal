package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strconv"
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

const A_RESET = "\u001b[3J\u001bc" // at least one of them should work
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

func Readline() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], err
}

func MultiPrompt(choices []string, hiddenChoices[]string) int {
	// returns the index.
	// hidden choices return (-1)-i

	// i dont like this
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if j != nil { j.Close() }
		Out(A_RESET, A_CUR_HOME)
		os.Exit(0)
	}()

	Nl()
	defer Nl()
	for _, c := range choices { Out("  ", c); Nl() }; Nl()
	Out("> ", A_SAVE_CUR_POS)
	for {
		Out(A_RESTORE_CUR_POS, A_ERASE_REST_OF_LINE)
		a, _ := Readline()
		for i, c := range choices {
			if c == a {
				return i
			}
		}
		for i, c := range hiddenChoices {
			if c == a {
				return -1 - i
			}
		}
	}
}

func AdvancedMultiPrompt(choices [][2]string, hiddenChoices[]string) int {
	// accepts a list of choices, each is a short keyword and the actual choice
	// returns the index.

	// i dont like this
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if j != nil { j.Close() }
		Out(A_RESET, A_CUR_HOME)
		os.Exit(0)
	}()

	//
	Nl()
	defer Nl()
	for _, c := range choices {
		fmt.Printf("  %s) %s\n", c[0], c[1])
	}
	Nl()
	Out("> ", A_SAVE_CUR_POS)
	for {
		Out(A_RESTORE_CUR_POS, A_ERASE_REST_OF_LINE)
		a, _ := Readline()
		for i, c := range choices {
			if c[0] == a {
				return i
			}
		}
		for i, c := range hiddenChoices {
			if c == a {
				return -1 - i
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
	// so here we are with a another very pleasant (not) workarout
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

const (
	UiMainloopCtxListYears = iota
	UiMainloopCtxListMonths
	UiMainloopCtxListEntries
	UiMainloopCtxShowEntry
	UiMainloopCtxEditEntry
)

const EntryTimeFormat = "Monday, 02. January 2006 15:04:05 MST"

func mainloop(passwd []byte) int {
	Out(A_ERASE_SCREEN, A_CUR_HOME)
	defer Out(A_RESET, A_CUR_HOME)

	// just for fun ;)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		j.Close()
		Out(A_RESET, A_CUR_HOME)
		os.Exit(0)
	}()
	// :)

	// ok.
	// let's start.
	// pls.
	lastMode := -1
	mode := -1
	selYear := -1
	selMonth := ""
	var selEntry uint64 = 0
	//
	for {
		Out(A_RESET, A_CUR_HOME); Nl()
		switch mode {
		case UiMainloopCtxListYears:
			years := []int{}
			choices := []string{}
			es := j.GetEntries()
			slices.Sort(es)
			for _, ts := range es {
				year := time.UnixMicro(int64(ts)).Local().Year()
				if !slices.Contains(years, year) {
					years = append(years, year)
					choices = append(choices, strconv.Itoa(year))
				}
			}
			lastMode = mode
			Out("Please select a year."); Nl()
			sel := MultiPrompt(choices, []string{"new", "exit", "quit", "q"})
			if sel == -1 {
				mode = UiMainloopCtxEditEntry
				continue
			} else if sel < -1 {
				return 0 // exit
			}
			selYear = years[sel]
			mode = UiMainloopCtxListMonths
		case UiMainloopCtxListMonths:
			months := []string{}
			choices := [][2]string{}
			i := 0
			es := j.GetEntries()
			slices.Sort(es)
			for _, ts := range es {
				year := time.UnixMicro(int64(ts)).Local().Year()
				if year == selYear {
					month := time.UnixMicro(int64(ts)).Local().Month().String()
					if !slices.Contains(months, month) {
						months = append(months, month)
						choices = append(choices, [2]string{strconv.Itoa(i+1), month})
						i += 1
					}
				}
			}
			lastMode = mode
			Out("Please select a month."); Nl()
			sel := AdvancedMultiPrompt(choices, []string{"", "new", "exit", "quit", "q"})
			if sel == -1 {
				mode = UiMainloopCtxListYears
			} else if sel == -2 {
				mode = UiMainloopCtxEditEntry
			} else if sel < -2 {
				return 0 // exit
			} else {
				selMonth = months[sel]
				mode = UiMainloopCtxListEntries
			}
		case UiMainloopCtxListEntries:
			entries := []uint64{}
			choices := [][2]string{}
			es := j.GetEntries()
			slices.Sort(es)
			i := 0
			for _, ts := range es {
				year := time.UnixMicro(int64(ts)).Local().Year()
				month := time.UnixMicro(int64(ts)).Local().Month().String()
				if year == selYear && month == selMonth {
					if !slices.Contains(entries, ts) {
						entries = append(entries, ts)
						choices = append(choices, [2]string{strconv.Itoa(i+1), time.UnixMicro(int64(ts)).Format(EntryTimeFormat)})
						i += 1
					}
				}
			}
			Out("Please select an entry."); Nl()
			sel := AdvancedMultiPrompt(choices, []string{"", "new", "exit", "quit", "q"})
			lastMode = mode
			if sel == -1 {
				mode = UiMainloopCtxListMonths
			} else if sel == -2 {
				mode = UiMainloopCtxEditEntry
			} else if sel < -2 {
				return 0 // exit
			} else {
				selEntry = entries[sel]
				mode = UiMainloopCtxShowEntry
			}
		case UiMainloopCtxShowEntry:
			e := j.GetEntry(selEntry)
			if e != nil {
				Out("[Decrypting ...] ")
				txt, err := e.Decrypt(passwd)
				Out("\r", A_ERASE_LINE)
				if err != nil {
					Out("Entry could not be decrypted!")
					Out("Either the password is wrong or the entry is corrupted.")
				} else {
					Out(time.UnixMicro(int64(e.Timestamp)).Format(EntryTimeFormat)); Nnl(2)
					Out(txt); Nnl(2)
				}
			} else {
				Out("Entry not found!"); Nnl(2)
				Out("[Press ENTER to go back]"); Readline()
				mode = lastMode
				continue
			}
			Out("> "); Readline()
			mode = lastMode
		case UiMainloopCtxEditEntry:
			handleErr := func(err error, out ...any) {
				Out(out...); Nl()
				Out(err.Error()); Nnl(2)
				Out("[Press ENTER to go back]"); Readline()
				mode = lastMode
			}
			Out("Write your new entry. Will listen until END."); Nnl(2)
			lines := []string{}
			for {
				Out("| ")
				line, err := Readline()
				if err == io.EOF { break }
				if line == "END" { break }
				lines = append(lines, line)
			}
			e, err := NewEncryptedEntry(strings.Join(lines, "\n"), passwd)
			if err != nil {
				handleErr(err, "Error creating new entry")
				continue
			}
			err = j.AddEntry(e)
			if err != nil {
				handleErr(err, "Error adding new entry to journal")
				continue
			}
			selEntry = e.Timestamp
			err = j.Write()
			if err == FileModifiedExternally {
				Out("The file was modified by another program since the last read/write."); Nnl(2)
				Out("[Press enter when you are ready to overwrite the journal file]"); Readline()
				err = j.updateLastModifiedTime()
				if err != nil {
					Out("Couldn't overwrite file. aborting."); Nl()
					Out(err); Nnl(2)
					Out("[Press Enter to exit program]"); Readline()
					return 1
				}
				err = j.Write()
				if err != nil {
					Out("Couldn't overwrite file. aborting."); Nl()
					Out(err); Nnl(2)
					Out("[Press Enter to exit program]"); Readline()
					return 1
				}
			} else if err != nil {
				Out("Couldn't write journal file. aborting."); Nl()
				Out(err); Nnl(2)
				Out("[Press Enter to exit program]"); Readline()
			}
			mode = UiMainloopCtxShowEntry
		default:
			mode = UiMainloopCtxListYears
		}
	}
}

func ShowUsageAndExit(a0 string, code int) {
	a0Parts := strings.Split(a0, "/")
	binName := a0Parts[len(a0Parts)-1]
	Out("Usage: ", binName, " <path>\n\nPositional arguments\n\n\t<path>  Path to the journal file\n\n")
	os.Exit(code)
}

func CliEntrypoint() {
	args := os.Args
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
	passwd, err := ReadPass()
	if err != nil {
		Out("Couldn't get password from commandline safely."); Nl()
		Out(err); Nl()
		os.Exit(1)
	}

	// try to open journal file
	Out("Opening journal file at ", AE(A_SFX_MODE, A_SET_DIM), a1, AE(A_SFX_MODE, A_RES_DIM), " ...")
	Nnl(2);
	j, err = OpenJournalFile(a1)
	if err != nil { 
		Out(AE(A_SFX_COLOR, A_COL_RED_FG), "Couldn't open journal file!", AE(A_SFX_COLOR, A_COL_RES_FG)); Nl()
		Out(err); Nnl(2)
		Out("[Press Enter to exit]"); Readline()
		os.Exit(1)
	}
	defer j.Close()

	// mainloop
	os.Exit(mainloop(passwd))

}

package main

// Copyright (c) 2026, Julian Müller (ChaoticByte)

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

	"github.com/awnumar/memguard"
	"golang.org/x/term"
)

/*

This file includes stuff for the terminal user interface.
Entrypoint() -> mainloop() -> ...

*/

// terminal helpers

func Out(stuff ...any) {
	// write stuff, without spaces between stuff1, stuff2, etc.
	for _, s := range stuff {
		fmt.Print(s)
	}
}

func Nl() { Out("\n") }

func Nnl(n int) {
	// write n lines
	for range n {
		Out("\n")
	}
}

func Readline() (string, error) {
	// read a single line from stdin
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], err
}

func MultiChoiceOrCommand(choices [][2]string, commands []string, prompt string, helpLine string) int {
	// Get a multiple-choice answer or a command from the user.
	// returns the index, (-1 - index) for commands

	// Handle SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if j != nil { j.Close() }
		Out(AS_RESET, AS_CUR_HOME)
		memguard.SafeExit(0)
	}()

	defer Nl()

	// output prompt, if any
	if prompt != "" { Out(prompt); Nnl(2) }

	// print choices, if any
	if len(choices) > 0 {
		Nl()
		for _, c := range choices {
			Out(" ", Am(AC_SET_BOLD), c[0], Am(AC_RESET_BOLD), "  ", c[1]); Nl()
		}
		Nl()
	}

	// print help line
	if helpLine != "" {
		Nl()
		Out(Am(AC_SET_UNDERLINE, AC_SET_DIM), "commands:", Am(AC_RESET_UNDERLINE, AC_RESET_DIM))
		Nnl(2)
		Out(helpLine)
		Nnl(2)
	}

	Nl()
	defer Out(Am(AC_COL_RESET_FG))
	for {
		// read lines until a valid choice or command is entered
		Out(Am(AC_SET_BOLD, AC_COL_BRIGHT_YELLOW_FG), "> ", Am(AC_RESET_BOLD, AC_COL_RESET_FG))
		a, err := Readline()
		if err == io.EOF { Nl(); continue }
		for i, c := range choices {
			if c[0] == a {
				return i
			}
		}
		for i, c := range commands {
			if c == a {
				return -1 - i
			}
		}
	}
}

func ReadPass() (*memguard.Enclave, error) {
	// Read a password using term.ReadPassword()
	// with additional fancy workarounds and shit

	Nl()
	Out("[ ] ")
	Out(AS_SAVE_CUR_POS)

	// Handle SIGINT (+ manual cleanup of term.Readpassword)
	fd := int(os.Stdout.Fd())
	s, err := term.GetState(fd); if err != nil { return nil, err }
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		term.Restore(fd, s)
		Nl()
		memguard.SafeExit(0)
	}()
	defer term.Restore(fd, s) // doppelt hält besser

	for {
		Out(AS_RESTORE_CUR_POS, AS_ERASE_REST_OF_LINE)
		pw, err := term.ReadPassword(fd); Nl()
		if err != nil || len(pw) > 0 {
			encl := memguard.NewEnclave(pw)
			pw = nil // empty the plaintext password
			Nl()
			return encl, err
		}
	}
}

//

const (
	UiListYears = iota
	UiListMonths
	UiListEntries
	UiShowEntry
	UiNewEntry
)

const EntryTimeFormat = "Monday, 02. January 2006 15:04:05 MST"

func mainloop(passwd *memguard.Enclave) int {

	// erase screen and reset screen on exit.
	Out(AS_ERASE_SCREEN, AS_CUR_HOME)
	defer Out(AS_RESET, AS_CUR_HOME)

	// Handle SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		j.Close()
		Out(AS_RESET, AS_CUR_HOME)
		memguard.SafeExit(0)
	}()
	// :)

	// ui mode
	lastMode := -1
	mode := -1
	// selections
	selYear := -1
	selMonth := ""
	selEntry := uint64(1) // entry 0 is reserved, so use as default.

	getHelp := func () string {
		// returns the help line for the current mode
		cmds := []string{}
		addCmd := func(cmd string, expl string) {
			cmds = append(
				cmds,
				cmd + " " +
				Am(AC_SET_DIM) + expl + Am(AC_RESET_DIM))
		}
		if mode != UiListYears {
			addCmd("Enter", "back")
		}
		if mode == UiShowEntry {
			addCmd("delete", "Delete this entry")
		}
		if mode == UiShowEntry {
			addCmd("a", "Previous")
			addCmd("d", "Next")
		}
		if mode == UiListYears || mode == UiListMonths || mode == UiListEntries || mode == UiShowEntry {
			addCmd("l", "Latest entry")
			addCmd("n", "New Entry")
			addCmd("q", "Exit the program")
		}
		return strings.Join(cmds, "\n")
	}

	writeJournalFile := func () int {
		// returns a code to exit or -1 if no error

		handleErr2 := func (err error, msg string) int {
			Out(msg); Nl()
			Out(err); Nnl(2)
			Out(Am(AC_SET_DIM), "[Press Enter to exit program]", Am(AC_RESET_DIM))
			Readline()
			return 1
		}

		err := j.Write()
		if err == FileModifiedExternally {
			Out("The file was modified by another program since the last read/write.")
			Nnl(2)
			Out("[Press Enter when you are ready to overwrite the journal file]")
			Readline(); Nl()
			err = j.updateLastModifiedTime()
			if err != nil {
				return handleErr2(err, "Couldn't overwrite file. aborting.")
			}
			err = j.Write()
			if err != nil {
				return handleErr2(err, "Couldn't overwrite file. aborting.")
			}
		} else if err != nil {
			return handleErr2(err, "Couldn't write journal file. aborting.")
		}

		return -1
	}

	for { // the actual main loop

		// reset screen and put cursor in the top left
		Out(AS_RESET, AS_CUR_HOME);

		if mode == UiListYears || mode == UiListMonths || mode == UiListEntries {

			// choices
			choices := [][2]string{}

			// used later
			years := []int{}
			months := []string{}
			entries := []uint64{}

			// collect list of choices based on filtered entries
			es := j.GetEntries()
			if len(es) > 0 {
				slices.Sort(es)
				i := 0
				for _, ts := range es {
					year := time.UnixMicro(int64(ts)).Local().Year()
					month := time.UnixMicro(int64(ts)).Local().Month().String()
					switch mode {
					case UiListYears:
						if !slices.Contains(years, year) {
							years = append(years, year)
							choices = append(choices, [2]string{strconv.Itoa(year), ""})
						}
					case UiListMonths:
						if year == selYear {
							if !slices.Contains(months, month) {
								months = append(months, month)
								choices = append(choices, [2]string{strconv.Itoa(i+1), month})
								i += 1
							}
						}
					case UiListEntries:
						if year == selYear && month == selMonth {
							if !slices.Contains(entries, ts) {
								entries = append(entries, ts)
								choices = append(
									choices,
									[2]string{
										strconv.Itoa(i+1),
										time.UnixMicro(int64(ts)).Format(EntryTimeFormat)},
								)
								i += 1
							}
						}
					}
				}
			}

			// commands
			commands := []string{}
			if mode == UiListYears {
				commands = []string{"l", "n", "q"}
			} else {
				commands = []string{"", "l", "n", "q"}
			}

			// prompt
			prompt := ""
			if len(es) > 0 {
				switch mode {
				case UiListYears:
					prompt = "Please select a " + Am(AC_SET_UNDERLINE) + "year"+ Am(AC_RESET_UNDERLINE)
				case UiListMonths:
					prompt = "Please select a " + Am(AC_SET_UNDERLINE) + "month"+ Am(AC_RESET_UNDERLINE)
				case UiListEntries:
					prompt = "Please select an " + Am(AC_SET_UNDERLINE) + "entry"+ Am(AC_RESET_UNDERLINE)
				}
			} else {
				switch mode {
				case UiListYears:
					prompt = "Years (There are no entries yet)"
				case UiListMonths:
					prompt = "Months (There are no entries yet)"
				case UiListEntries:
					prompt = "Entries (There are no entries yet)"
				}
			}

			sel := MultiChoiceOrCommand(
				choices,
				commands,
				Am(AC_COL_BRIGHT_GREEN_FG) + prompt + Am(AC_COL_RESET_FG),
				getHelp())

			// prepare next iteration (or exit)
			// based on user input

			lastMode = mode

			if mode == UiListYears {
				if sel == -1 {
					latest := j.GetLatestEntry()
					if latest > 0 {
						selEntry = latest
						mode = UiShowEntry
					}
				} else if sel == -2 {
					mode = UiNewEntry
				} else if sel < -2 {
					return 0 // exit
				} else {
					selYear = years[sel]
					mode = UiListMonths
				}
			} else if mode == UiListMonths || mode == UiListEntries {
				if sel == -1 {
					if mode == UiListMonths {
						mode = UiListYears
					} else {
						mode = UiListMonths
					}
				} else if sel == -2 {
					latest := j.GetLatestEntry()
					if latest > 0 {
						selEntry = latest
						mode = UiShowEntry
					}
				} else if sel == -3 {
					mode = UiNewEntry
					continue
				} else if sel < -3 {
					return 0 // exit
				} else {
					if mode == UiListMonths {
						selMonth = months[sel]
						mode = UiListEntries
					} else {
					selEntry = entries[sel]
					mode = UiShowEntry
					}
				}
			}

		} else if mode == UiShowEntry {

			// show a selected entry

			e := j.GetEntry(selEntry)
			if e != nil {
				Out("[Decrypting ...] ")
				txt, err := e.Decrypt(passwd)
				Out("\r", AS_ERASE_LINE)
				if err != nil {
					Out("Entry could not be decrypted!"); Nl()
					Out("Either the password is wrong or the entry is corrupted."); Nnl(2)
				} else {
					// Output Entry
					Out(Am(AC_SET_UNDERLINE),
						time.UnixMicro(int64(e.Timestamp)).Format(EntryTimeFormat),
						Am(AC_RESET_UNDERLINE))
					Nnl(4); Out(txt); Nnl(3)
					txt = "" // don't keep the plaintext in memory
				}
			} else {
				// this will likely never get called
				// but catched a nil pointer deref
				Out("Entry not found!"); Nnl(2)
				Out(Am(AC_SET_DIM), "[Press Enter to go back]", Am(AC_RESET_DIM))
				Readline()
				mode = lastMode
				continue
			}

			sel := MultiChoiceOrCommand(
				[][2]string{},
				[]string{"", "a", "d", "l", "q", "n", "delete"},
				"", getHelp())

			switch sel {
			case -1:
				mode = lastMode
			case -2:
				prev := j.GetPreviousEntry(selEntry)
				if prev > 0 {
					selEntry = prev
					mode = UiShowEntry
				}
			case -3:
				next := j.GetNextEntry(selEntry)
				if next > 0 {
					selEntry = next
					mode = UiShowEntry
				}
			case -4:
				latest := j.GetLatestEntry()
				if latest > 0 {
					selEntry = latest
					mode = UiShowEntry
				}
			case -5:
				return 0 // exit
			case -6:
				mode = UiNewEntry
			case -7:
				Nl(); Out(AS_ERASE_REST_OF_SCREEN)
				answer := MultiChoiceOrCommand(
					[][2]string{{"yes", ""}, {"no", ""}},
					[]string{},
					"Do you really want to delete this entry?", "")
				if answer == 0 {
					mode = lastMode
					j.DeleteEntry(selEntry)
					statusCode := writeJournalFile()
					if statusCode >= 0 {
						return statusCode
					}
				}
			}

		} else if mode == UiNewEntry {

			// Create a new entry

			handleErr := func(err error, out ...any) {
				Out(out...); Nl()
				Out(err.Error()); Nnl(2)
				Out(Am(AC_SET_DIM), "[Press Enter to go back]", Am(AC_RESET_DIM))
				Readline()
				mode = lastMode
			}

			header := func () {
				Out(Am(AC_COL_GREEN_FG),
					"Write a new entry; ",
					Am(AC_COL_RESET_FG, AC_SET_DIM),
					"Save it by hitting ", Am(AC_RESET_DIM), "Ctrl+D",
					Am(AC_SET_DIM), " in an empty line.\n",
					"You can delete the previous line with ",
					Am(AC_RESET_DIM), "dd", Am(AC_SET_DIM),
					" and ", Am(AC_RESET_DIM), "Enter", Am(AC_RESET_DIM), ".")
				Nnl(2)
			}

			header()

			// read text from stdin (rune by rune)
			lines := []string{}
			for {
				line, err := Readline()
				if err == io.EOF {
					break
				} else if err != nil {
					handleErr(err, "Couldn't read terminal input")
				}
				if line == "dd" {
					ll := len(lines)
					if ll < 1 {
						lines = []string{}
					} else {
						lines = lines[:ll-1]
					}
					Out(AS_RESET, AS_CUR_HOME)
					header()
					for _, l := range lines {
						Out(l); Nl()
					}
				} else {
					lines = append(lines, line)
				}
			}

			// Try to create new EncryptedEntry from the input text

			e, err := NewEncryptedEntry(strings.Trim(strings.Join(lines, "\n"), " \n"), passwd)
			if err != nil {
				handleErr(err, "Error creating new entry")
				continue
			}

			// empty input
			lines = nil

			err = j.AddEntry(e)
			if err != nil {
				handleErr(err, "Error adding new entry to journal")
				continue
			}
			selEntry = e.Timestamp

			// Update journal file
			statusCode := writeJournalFile()
			if statusCode >= 0 {
				return statusCode
			}

			mode = UiShowEntry

		} else {

			mode = UiListYears

		}
	}
}

func PrintVersion() {
	Out(Am(AC_SET_BOLD), "Journal " + Am(AC_RESET_BOLD, AC_COL_CYAN_FG) + Version + Am(AC_COL_RESET_FG)); Nnl(2)
}

func ShowUsageAndExit(a0 string, code int) {
	PrintVersion()
	a0Parts := strings.Split(a0, "/")
	binName := a0Parts[len(a0Parts)-1]
	Out("Usage: ",
		binName,
		" <path>\n\nPositional arguments\n\n\t<path>  Path to the journal file\n\n")
	os.Exit(code)
}

func Entrypoint() {
	memguard.CatchInterrupt()
	defer memguard.Purge()

	// parse cli args
	args := os.Args
	if len(args) < 2 {
		ShowUsageAndExit(args[0], 1)
	}
	a1 := args[1]
	if a1 == "-h" || a1 == "--help" {
		ShowUsageAndExit(args[0], 0)
	}

	// clear screen and go to top left corner
	Out(AS_ERASE_SCREEN, AS_CUR_HOME);

	PrintVersion()

	Out("Please enter your encryption key."); Nl()
	passwd, err := ReadPass()
	if err != nil || passwd == nil {
		Out("Couldn't get password from commandline safely."); Nl()
		Out(err); Nl()
		memguard.SafeExit(1)
	}

	Out("Opening journal file at ", Am(AC_SET_DIM), a1, Am(AC_RESET_DIM), " ...")
	Nnl(2);
	j, err = OpenJournalFile(a1, passwd)
	if err != nil { 
		Out(Am(AC_COL_RED_FG), "Couldn't open journal file!", Am(AC_COL_RESET_FG))
		Nl()
		Out(err); Nnl(2)
		Out("[Press Enter to exit]"); Readline()
		memguard.SafeExit(1)
	}
	defer j.Close()

	memguard.SafeExit(mainloop(passwd))
}

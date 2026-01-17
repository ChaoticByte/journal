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
		Out(A_RESET, A_CUR_HOME)
		os.Exit(0)
	}()

	defer Nl()

	// output prompt, if any
	if prompt != "" { Out(Am(A_SET_DIM), prompt, Am(A_RESET_DIM)); Nnl(2) }

	// print choices, if any
	if len(choices) > 0 {
		Nl()
		for _, c := range choices {
			Out(c[0], ") ", c[1]); Nl()
		}
		Nl()
	}

	// output > and save cursor position
	Nl(); Out("> ", A_SAVE_CUR_POS)
	if helpLine != "" {
		Nnl(3)
		Out(Am(A_SET_UNDERLINE, A_SET_DIM), "commands:", Am(A_RESET_UNDERLINE, A_RESET_DIM))
		Nnl(2)
		Out(helpLine)
	}

	for {
		// read lines until a valid choice or command is entered
		Out(A_RESTORE_CUR_POS, A_ERASE_REST_OF_LINE)
		a, _ := Readline()
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

func ReadPass() ([]byte, error) {
	// Read a password using term.ReadPassword()
	// with additional fancy workarounds and shit

	Nl()
	Out("[ ] ")
	Out(A_SAVE_CUR_POS)

	// Handle SIGINT (+ manual cleanup of term.Readpassword)
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
	UiListYears = iota
	UiListMonths
	UiListEntries
	UiShowEntry
	UiNewEntry
)

const EntryTimeFormat = "Monday, 02. January 2006 15:04:05 MST"

func mainloop(passwd []byte) int {

	// erase screen and reset screen on exit.
	Out(A_ERASE_SCREEN, A_CUR_HOME)
	defer Out(A_RESET, A_CUR_HOME)

	// Handle SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		j.Close()
		Out(A_RESET, A_CUR_HOME)
		os.Exit(0)
	}()
	// :)

	// ui mode
	lastMode := -1
	mode := -1
	// selections
	selYear := -1
	selMonth := ""
	selEntry := uint64(1) // entry 0 is reserved, so use as default.

	getHelpLine := func() string {
		// returns the help line for the current mode
		cmds := []string{}
		addCmd := func(cmd string, expl string) {
			cmds = append(
				cmds,
				Am(A_SET_BOLD) + cmd + Am(A_RESET_BOLD) + " " +
				Am(A_SET_DIM) + expl + Am(A_RESET_DIM))
		}
		if mode != UiListYears {
			addCmd("Enter", "back")
		}
		if mode == UiShowEntry {
			addCmd("delete", "Delete this entry")
		}
		if mode == UiListYears || mode == UiListMonths || mode == UiListEntries || mode == UiShowEntry {
			addCmd("new", "New Entry")
			addCmd("q", "Exit the program")
		}
		return strings.Join(cmds, "  ")
	}

	writeJournalFile := func () int {
		handleErr2 := func (err error, msg string) int {
			Out(msg); Nl()
			Out(err); Nnl(2)
			Out(Am(A_SET_DIM), "[Press Enter to exit program]", Am(A_RESET_DIM))
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
		Out(A_RESET, A_CUR_HOME); Nl()

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
				commands = []string{"new", "q"}
			} else {
				commands = []string{"", "new", "q"}
			}

			// prompt
			prompt := ""
			if len(es) > 0 {
				switch mode {
				case UiListYears:
					prompt = "Please select a year:"
				case UiListMonths:
					prompt = "Please select a month:"
				case UiListEntries:
					prompt = "Please select an entry:"
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
				prompt,
				getHelpLine())

			// prepare next iteration (or exit)
			// based on user input

			lastMode = mode

			switch mode {
			case UiListYears:
				if sel == -1 {
					mode = UiNewEntry
					continue
				} else if sel < -1 {
					return 0 // exit
				}
				selYear = years[sel]
				mode = UiListMonths
			case UiListMonths:
				if sel == -1 {
					mode = UiListYears
				} else if sel == -2 {
					mode = UiNewEntry
				} else if sel < -2 {
					return 0 // exit
				} else {
					selMonth = months[sel]
					mode = UiListEntries
				}
			case UiListEntries:
				if sel == -1 {
					mode = UiListMonths
				} else if sel == -2 {
					mode = UiNewEntry
				} else if sel < -2 {
					return 0 // exit
				} else {
					selEntry = entries[sel]
					mode = UiShowEntry
				}
			}

		} else if mode == UiShowEntry {

			// show a selected entry

			e := j.GetEntry(selEntry)
			if e != nil {
				Out("[Decrypting ...] ")
				txt, err := e.Decrypt(passwd)
				Out("\r", A_ERASE_LINE)
				if err != nil {
					Out("Entry could not be decrypted!"); Nl()
					Out("Either the password is wrong or the entry is corrupted."); Nnl(2)
				} else {
					// Output Entry
					Out(Am(A_SET_UNDERLINE),
						time.UnixMicro(int64(e.Timestamp)).Format(EntryTimeFormat),
						Am(A_RESET_UNDERLINE))
					Nnl(4); Out(txt); Nnl(3)
				}
			} else {
				// this will likely never get called
				// but catched a nil pointer deref
				Out("Entry not found!"); Nnl(2)
				Out(Am(A_SET_DIM), "[Press Enter to go back]", Am(A_RESET_DIM))
				Readline()
				mode = lastMode
				continue
			}

			sel := MultiChoiceOrCommand(
				[][2]string{},
				[]string{"", "q", "new", "delete"},
				"", getHelpLine())

			switch sel {
			case -1:
				mode = lastMode
			case -2:
				return 0 // exit
			case -3:
				mode = UiNewEntry
			case -4:
				Nl(); Out(A_ERASE_REST_OF_SCREEN)
				answer := MultiChoiceOrCommand(
					[][2]string{{"yes", ""}, {"no", ""}},
					[]string{},
					"Are you sure that you want to delete this entry?", "")
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
				Out(Am(A_SET_DIM), "[Press Enter to go back]", Am(A_RESET_DIM))
				Readline()
				mode = lastMode
			}

			Out(Am(A_SET_DIM),
				"Write your new entry. Save it by hitting Ctrl+D in an empty line.",
				Am(A_RESET_DIM))
			Nnl(2)

			// read text from stdin (rune by rune)
			builder := strings.Builder{}
			reader := bufio.NewReader(os.Stdin)
			for {
				r, _, err := reader.ReadRune()
				if err == io.EOF { break }
				builder.WriteRune(r)
				if err != nil {
					handleErr(err, "Couldn't read terminal input")
				}
			}

			// Try to create new EncryptedEntry from the input text

			e, err := NewEncryptedEntry(strings.Trim(builder.String(), " \n"), passwd)
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

func ShowUsageAndExit(a0 string, code int) {
	a0Parts := strings.Split(a0, "/")
	binName := a0Parts[len(a0Parts)-1]
	Out("Usage: ",
		binName,
		" <path>\n\nPositional arguments\n\n\t<path>  Path to the journal file\n\n")
	os.Exit(code)
}

func Entrypoint() {
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
	Out(A_ERASE_SCREEN, A_CUR_HOME); Nl()

	Out("Please enter your encryption key."); Nl()
	passwd, err := ReadPass()
	if err != nil {
		Out("Couldn't get password from commandline safely."); Nl()
		Out(err); Nl()
		os.Exit(1)
	}

	Out("Opening journal file at ", Am(A_SET_DIM), a1, Am(A_RESET_DIM), " ...")
	Nnl(2);
	j, err = OpenJournalFile(a1, passwd)
	if err != nil { 
		Out(Am(A_COL_RED_FG), "Couldn't open journal file!", Am(A_COL_RESET_FG))
		Nl()
		Out(err); Nnl(2)
		Out("[Press Enter to exit]"); Readline()
		os.Exit(1)
	}
	defer j.Close()

	os.Exit(mainloop(passwd))
}

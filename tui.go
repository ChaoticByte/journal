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

// terminal helpers

func Out(stuff ...any) {
	for _, s := range stuff {
		fmt.Print(s)
	}
}

func Nl() { Out("\n") }

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

func MultiChoiceOrCommand(choices [][2]string, commands []string, prompt string, helpLine string) int {
	// returns the index, (-1 - index) for commands

	// i dont like this
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if j != nil { j.Close() }
		Out(A_RESET, A_CUR_HOME)
		os.Exit(0)
	}()

	defer Nl()

	if prompt != "" { Out(AColMode(A_SET_DIM), prompt, AColMode(A_RESET_DIM)); Nnl(2) }

	if len(choices) > 0 {
		Nl()
		for _, c := range choices {
			Out(c[0], ") ", c[1]); Nl()
		}
		Nnl(2)
	}

	Out("> ", A_SAVE_CUR_POS)
	if helpLine != "" {
		Nnl(3)
		Out(AColMode(A_SET_UNDERLINE, A_SET_DIM), "commands:", AColMode(A_RESET_UNDERLINE, A_RESET_DIM))
		Nnl(2)
		Out(helpLine)
	}

	for {
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

	lastMode := -1
	mode := -1
	selYear := -1
	selMonth := ""
	var selEntry uint64 = 0

	getHelpLine := func() string {
		cmds := []string{}
		addCmd := func(cmd string, expl string) {
			cmds = append(
				cmds,
				AColMode(A_SET_BOLD) + cmd + AColMode(A_RESET_BOLD) + " " +
				AColMode(A_SET_DIM) + expl + AColMode(A_RESET_DIM))
		}
		if mode != UiMainloopCtxListYears {
			addCmd("Enter", "back")
		}
		if mode == UiMainloopCtxListYears || mode == UiMainloopCtxListMonths || mode == UiMainloopCtxListEntries {
			addCmd("new", "New Entry")
			addCmd("q", "Exit the program")
		}
		return strings.Join(cmds, "  ")
	}

	//
	for {
		Out(A_RESET, A_CUR_HOME); Nl()
		switch mode {

		case UiMainloopCtxListYears:
			years := []int{}
			choices := [][2]string{}
			es := j.GetEntries()
			slices.Sort(es)
			for _, ts := range es {
				year := time.UnixMicro(int64(ts)).Local().Year()
				if !slices.Contains(years, year) {
					years = append(years, year)
					choices = append(choices, [2]string{strconv.Itoa(year), ""})
				}
			}
			lastMode = mode
			sel := MultiChoiceOrCommand(
				choices,
				[]string{"new", "q"},
				"Please select a year.",
				getHelpLine())
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
			sel := MultiChoiceOrCommand(
				choices,
				[]string{"", "new", "q"},
				"Please select a month.",
				getHelpLine())
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
			sel := MultiChoiceOrCommand(
				choices,
				[]string{"", "new", "q"},
				"Please select an entry.",
				getHelpLine())
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
				Out(AColMode(A_SET_DIM),
					"[Press ENTER to go back]",
					AColMode(A_RESET_DIM))
				Readline()
				mode = lastMode
				continue
			}
			Out(AColMode(A_SET_DIM), "[Press Enter to go back]", AColMode(A_RESET_DIM)); Readline()
			mode = lastMode

		case UiMainloopCtxEditEntry:
			handleErr := func(err error, out ...any) {
				Out(out...); Nl()
				Out(err.Error()); Nnl(2)
				Out(AColMode(A_SET_DIM),
					"[Press ENTER to go back]",
					AColMode(A_RESET_DIM))
				Readline()
				mode = lastMode
			}
			Out(AColMode(A_SET_DIM),
				"Write your new entry. Save it by hitting Ctrl+D in an empty line.",
				AColMode(A_RESET_DIM))
			Nnl(2)

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

			e, err := NewEncryptedEntry(builder.String(), passwd)
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
				Out("The file was modified by another program since the last read/write.")
				Nnl(2)
				Out("[Press Enter when you are ready to overwrite the journal file]")
				Readline(); Nl()
				err = j.updateLastModifiedTime()
				if err != nil {
					Out("Couldn't overwrite file. aborting."); Nl()
					Out(err); Nnl(2)
					Out(AColMode(A_SET_DIM), "[Press Enter to exit program]", AColMode(A_RESET_DIM))
					Readline()
					return 1
				}
				err = j.Write()
				if err != nil {
					Out("Couldn't overwrite file. aborting."); Nl()
					Out(err); Nnl(2)
					Out(AColMode(A_SET_DIM), "[Press Enter to exit program]", AColMode(A_RESET_DIM))
					Readline()
					return 1
				}
			} else if err != nil {
				Out("Couldn't write journal file. aborting."); Nl()
				Out(err); Nnl(2)
				Out(AColMode(A_SET_DIM),"[Press Enter to exit program]", AColMode(A_RESET_DIM))
				Readline()
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
	Out("Usage: ",
		binName,
		" <path>\n\nPositional arguments\n\n\t<path>  Path to the journal file\n\n")
	os.Exit(code)
}

func Entrypoint() {
	args := os.Args
	// parse cli args
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

	Out("Opening journal file at ", AColMode(A_SET_DIM), a1, AColMode(A_RESET_DIM), " ...")
	Nnl(2);
	j, err = OpenJournalFile(a1)
	if err != nil { 
		Out(AColMode(A_COL_RED_FG), "Couldn't open journal file!", AColMode(A_COL_RESET_FG))
		Nl()
		Out(err); Nnl(2)
		Out("[Press Enter to exit]"); Readline()
		os.Exit(1)
	}
	defer j.Close()

	os.Exit(mainloop(passwd))
}

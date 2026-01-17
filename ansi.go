package main

import (
	"fmt"
	"strings"
)

/*

This file includes ANSI Escape Codes, Sequences, functions, etc.
Some of those may not be actively used right now, but stay for
future usage.

*/

const (
	// modes
	A_SET_BOLD = "1"
	A_RESET_BOLD = "22"
	A_SET_DIM = "2"
	A_RESET_DIM = "22"
	A_SET_ITALIC = "3"
	A_RESET_ITALIC = "23"
	A_SET_UNDERLINE = "4"
	A_RESET_UNDERLINE = "24"
	A_SET_BLINK = "5"
	A_RESET_BLINK = "25"
	A_SET_INVERTED = "7"
	A_RESET_INVERTED = "27"
	A_SET_HIDDEN = "8"
	A_RESET_HIDDEN = "28"
	A_SET_STRIKETHROUGH = "9"
	A_RESET_STRIKETHROUGH = "29"

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
	A_COL_RESET_FG = "39"
	A_COL_RESET_BG = "49"

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

func Am(codes ...string) string {
	seq := "\u001b["
	seq += strings.Join(codes, ";")
	seq += "m"
	return seq
}

// Screen & erase

const A_ERASE_REST_OF_SCREEN = "\u001b[0J"
const A_ERASE_SCREEN = "\u001b[2J"
const A_RESET = "\u001b[3J\u001bc" // at least one of them should work
const A_ERASE_REST_OF_LINE = "\u001b[0K"
const A_ERASE_LINE = "\u001b[2K"

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

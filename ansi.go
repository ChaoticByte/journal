package main

// Copyright (c) 2026, Julian Müller (ChaoticByte)

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
	AC_SET_BOLD = "1"
	AC_RESET_BOLD = "22"
	AC_SET_DIM = "2"
	AC_RESET_DIM = "22"
	AC_SET_ITALIC = "3"
	AC_RESET_ITALIC = "23"
	AC_SET_UNDERLINE = "4"
	AC_RESET_UNDERLINE = "24"
	AC_SET_BLINK = "5"
	AC_RESET_BLINK = "25"
	AC_SET_INVERTED = "7"
	AC_RESET_INVERTED = "27"
	AC_SET_HIDDEN = "8"
	AC_RESET_HIDDEN = "28"
	AC_SET_STRIKETHROUGH = "9"
	AC_RESET_STRIKETHROUGH = "29"

	// colors
	AC_COL_BLACK_FG = "30"
	AC_COL_BLACK_BG = "40"
	AC_COL_RED_FG = "31"
	AC_COL_RED_BG = "41"
	AC_COL_GREEN_FG = "32"
	AC_COL_GREEN_BG = "42"
	AC_COL_YELLOW_FG = "33"
	AC_COL_YELLOW_BG = "43"
	AC_COL_BLUE_FG = "34"
	AC_COL_BLUE_BG = "44"
	AC_COL_MAGENTA_FG = "35"
	AC_COL_MAGENTA_BG = "45"
	AC_COL_CYAN_FG = "36"
	AC_COL_CYAN_BG = "46"
	AC_COL_WHITE_FG = "37"
	AC_COL_WHITE_BG = "47"

	// reset colors
	AC_COL_RESET_FG = "39"
	AC_COL_RESET_BG = "49"

	// bright colors
	AC_COL_BRIGHT_BLACK_FG = "90"
	AC_COL_BRIGHT_BLACK_BG = "100"
	AC_COL_BRIGHT_RED_FG = "91"
	AC_COL_BRIGHT_RED_BG = "101"
	AC_COL_BRIGHT_GREEN_FG = "92"
	AC_COL_BRIGHT_GREEN_BG = "102"
	AC_COL_BRIGHT_YELLOW_FG = "93"
	AC_COL_BRIGHT_YELLOW_BG = "103"
	AC_COL_BRIGHT_BLUE_FG = "94"
	AC_COL_BRIGHT_BLUE_BG = "104"
	AC_COL_BRIGHT_MAGENTA_FG = "95"
	AC_COL_BRIGHT_MAGENTA_BG = "105"
	AC_COL_BRIGHT_CYAN_FG = "96"
	AC_COL_BRIGHT_CYAN_BG = "106"
	AC_COL_BRIGHT_WHITE_FG = "97"
	AC_COL_BRIGHT_WHITE_BG = "107"
)

func Am(codes ...string) string {
	seq := "\u001b["
	seq += strings.Join(codes, ";")
	seq += "m"
	return seq
}

// Screen & erase

const AS_ERASE_REST_OF_SCREEN = "\u001b[0J"
const AS_ERASE_SCREEN = "\u001b[2J"
const AS_RESET = "\u001b[3J\u001bc" // at least one of them should work
const AS_ERASE_REST_OF_LINE = "\u001b[0K"
const AS_ERASE_LINE = "\u001b[2K"

// Cursor

const AS_CUR_HOME = "\u001b[H"
const AS_SAVE_CUR_POS = "\u001b7"
const AS_RESTORE_CUR_POS = "\u001b8"

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

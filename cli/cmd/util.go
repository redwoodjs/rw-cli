package cmd

import (
	"os"

	"golang.org/x/term"
)

// Uses defaults if the terminal size cannot be determined
func getTerminalSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

// Returns the terminal width if it's less than 80. Else returns 80
func getClampedTerminalWidth() int {
	termWidth, _ := getTerminalSize()
	return min(80, termWidth)
}

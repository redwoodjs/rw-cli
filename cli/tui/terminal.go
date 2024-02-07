package tui

import (
	"os"

	"golang.org/x/term"
)

// Uses defaults if the terminal size cannot be determined
func GetTerminalSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

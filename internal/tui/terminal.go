package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type fdReader interface {
	Fd() uintptr
}

type fdWriter interface {
	Fd() uintptr
}

var isTerminalFunc = term.IsTerminal

// SetIsTerminalFuncForTesting overrides the terminal detection function and returns a restore function.
// This is intended for cross-package tests that need deterministic TTY detection.
func SetIsTerminalFuncForTesting(fn func(int) bool) func() {
	previous := isTerminalFunc
	isTerminalFunc = fn
	return func() {
		isTerminalFunc = previous
	}
}

// ShouldUseTUI decides if interactive TUI should be launched.
func ShouldUseTUI(quiet bool, in io.Reader, out io.Writer) bool {
	if quiet {
		return false
	}
	return IsTerminalReader(in) && IsTerminalWriter(out)
}

// IsTerminalReader reports whether the reader wraps a file descriptor bound to a terminal.
func IsTerminalReader(reader io.Reader) bool {
	if r, ok := reader.(fdReader); ok {
		return isTerminalFunc(int(r.Fd()))
	}
	return false
}

// IsTerminalWriter reports whether the writer wraps a file descriptor bound to a terminal.
func IsTerminalWriter(writer io.Writer) bool {
	if w, ok := writer.(fdWriter); ok {
		return isTerminalFunc(int(w.Fd()))
	}
	return false
}

// ProgramOptions builds Bubble Tea program options with the provided I/O, disabling the renderer when no terminal is present.
func ProgramOptions(in io.Reader, out io.Writer) []tea.ProgramOption {
	options := []tea.ProgramOption{
		tea.WithInput(in),
		tea.WithOutput(out),
	}

	if !IsTerminalReader(in) || !IsTerminalWriter(out) {
		options = append(options, tea.WithoutRenderer())
	}

	return options
}

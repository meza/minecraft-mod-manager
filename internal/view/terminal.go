package view

import (
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

type fdReader interface {
	Fd() uintptr
}

type fdWriter interface {
	Fd() uintptr
}

var isTerminalFunc = term.IsTerminal
var isTerminalFuncOverridden bool
var unicodeSupportFunc = defaultUnicodeSupport
var colorProfileFunc = termenv.EnvColorProfile

// SetIsTerminalFuncForTesting overrides the terminal detection function and returns a restore function.
// This is intended for cross-package tests that need deterministic TTY detection.
func SetIsTerminalFuncForTesting(fn func(int) bool) func() {
	previous := isTerminalFunc
	previousOverride := isTerminalFuncOverridden
	isTerminalFunc = fn
	isTerminalFuncOverridden = true
	return func() {
		isTerminalFunc = previous
		isTerminalFuncOverridden = previousOverride
	}
}

// SupportsUnicode reports whether Unicode output should be used for the current environment.
func SupportsUnicode() bool {
	return unicodeSupportFunc()
}

// SupportsColor reports whether color output should be used for the provided writer.
func SupportsColor(writer io.Writer) bool {
	if !IsTerminalWriter(writer) {
		return false
	}
	return colorProfileFunc() != termenv.Ascii
}

// SetUnicodeSupportFuncForTesting overrides the Unicode detection function and returns a restore function.
// This is intended for cross-package tests that need deterministic Unicode detection.
func SetUnicodeSupportFuncForTesting(fn func() bool) func() {
	previous := unicodeSupportFunc
	unicodeSupportFunc = fn
	return func() {
		unicodeSupportFunc = previous
	}
}

// SetColorProfileFuncForTesting overrides the color profile function and returns a restore function.
// This is intended for cross-package tests that need deterministic color detection.
func SetColorProfileFuncForTesting(fn func() termenv.Profile) func() {
	previous := colorProfileFunc
	colorProfileFunc = fn
	return func() {
		colorProfileFunc = previous
	}
}

// SupportsPrompting reports whether the provided I/O supports interactive prompting.
func SupportsPrompting(in io.Reader, out io.Writer) bool {
	return IsTerminalReader(in) && IsTerminalWriter(out)
}

// SupportsControlSequences reports whether output supports dynamic rendering.
func SupportsControlSequences(out io.Writer) bool {
	return IsTerminalWriter(out)
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

// ProgramOptions builds Bubble Tea program options with the provided I/O, disabling the renderer when output lacks control sequences.
func ProgramOptions(in io.Reader, out io.Writer) []tea.ProgramOption {
	options := []tea.ProgramOption{
		tea.WithInput(in),
		tea.WithOutput(out),
	}

	if !SupportsControlSequences(out) {
		options = append(options, tea.WithoutRenderer())
	}

	return options
}

func defaultUnicodeSupport() bool {
	if isTerminalFuncOverridden {
		return isTerminalFunc(int(os.Stdout.Fd()))
	}
	if !IsTerminalWriter(os.Stdout) {
		return false
	}
	return termenv.EnvColorProfile() != termenv.Ascii
}

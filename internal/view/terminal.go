package view

import (
	"io"
	"os"
	"strings"

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
var terminalSizeFunc = term.GetSize
var unicodeSupportFunc = defaultUnicodeSupport
var colorProfileFunc = termenv.EnvColorProfile

// SetIsTerminalFuncForTesting overrides the terminal detection function and returns a restore function.
// This is intended for cross-package tests that need deterministic TTY detection.
func SetIsTerminalFuncForTesting(fn func(int) bool) func() {
	previous := isTerminalFunc
	isTerminalFunc = fn
	return func() {
		isTerminalFunc = previous
	}
}

// SetTerminalSizeFuncForTesting overrides the terminal size lookup and returns a restore function.
// This is intended for cross-package tests that need deterministic terminal sizing.
func SetTerminalSizeFuncForTesting(fn func(int) (int, int, error)) func() {
	previous := terminalSizeFunc
	terminalSizeFunc = fn
	return func() {
		terminalSizeFunc = previous
	}
}

// TerminalSize returns the terminal size for the provided writer or zeros when unavailable.
func TerminalSize(writer io.Writer) (width int, height int) {
	writerDescriptor, ok := writer.(fdWriter)
	if !ok {
		return 0, 0
	}
	fileDescriptor := int(writerDescriptor.Fd())
	if !isTerminalFunc(fileDescriptor) {
		return 0, 0
	}
	width, height, err := terminalSizeFunc(fileDescriptor)
	if err != nil {
		return 0, 0
	}
	return width, height
}

// SupportsUnicode reports whether Unicode output should be used for the current environment.
func SupportsUnicode() bool {
	return unicodeSupportFunc()
}

// SupportsColor reports whether color output should be used for the current environment.
func SupportsColor(writer io.Writer) bool {
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
	input := in
	if !SupportsPrompting(in, out) {
		input = nil
	}

	options := []tea.ProgramOption{
		tea.WithInput(input),
		tea.WithOutput(out),
	}

	if supportsDynamicRendering(in, out) {
		options = append(options, tea.WithMouseCellMotion())
	} else {
		options = append(options, tea.WithoutRenderer())
	}

	return options
}

// UnboundedWindowHeightFilter clears the WindowSizeMsg height to avoid renderer cropping.
func UnboundedWindowHeightFilter(_ tea.Model, msg tea.Msg) tea.Msg {
	windowSize, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return msg
	}
	windowSize.Height = 0
	return windowSize
}

func supportsDynamicRendering(in io.Reader, out io.Writer) bool {
	return SupportsPrompting(in, out) && SupportsControlSequences(out)
}

func defaultUnicodeSupport() bool {
	locale := os.Getenv("LC_ALL")
	if locale == "" {
		locale = os.Getenv("LC_CTYPE")
	}
	if locale == "" {
		locale = os.Getenv("LANG")
	}
	if locale == "" {
		return true
	}

	normalized := strings.ToUpper(locale)
	if strings.Contains(normalized, "UTF-8") || strings.Contains(normalized, "UTF8") {
		return true
	}
	if normalized == "POSIX" || normalized == "C" || strings.HasPrefix(normalized, "C.") {
		return false
	}
	return true
}

package terminal

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	KeyEnter      = "\r"
	KeyEscape     = "\x1b"
	KeyCtrlC      = "\x03"
	KeyBackspace  = "\x7f"
	KeyArrowUp    = "\x1b[A"
	KeyArrowDown  = "\x1b[B"
	KeyArrowRight = "\x1b[C"
	KeyArrowLeft  = "\x1b[D"
)

// MouseEvent describes an SGR mouse event for PTY input.
type MouseEvent struct {
	Button tea.MouseButton
	Action tea.MouseAction
	X      int
	Y      int
	Alt    bool
	Ctrl   bool
	Shift  bool
}

// EncodeSGRMouseSequence returns an SGR mouse escape sequence for the event.
func EncodeSGRMouseSequence(event MouseEvent) string {
	buttonCode, isWheel := encodeMouseButton(event.Button)
	modifierBits := 0
	if event.Shift {
		modifierBits |= 0b0000_0100
	}
	if event.Alt {
		modifierBits |= 0b0000_1000
	}
	if event.Ctrl {
		modifierBits |= 0b0001_0000
	}
	if event.Action == tea.MouseActionMotion {
		modifierBits |= 0b0010_0000
	}

	controlCode := buttonCode | modifierBits
	final := "M"
	if event.Action == tea.MouseActionRelease && !isWheel {
		final = "m"
	}

	column := event.X + 1
	row := event.Y + 1
	return fmt.Sprintf("\x1b[<%d;%d;%d%s", controlCode, column, row, final)
}

func encodeMouseButton(button tea.MouseButton) (int, bool) {
	const (
		bitWheel = 0b0100_0000
		bitAdd   = 0b1000_0000
	)

	switch button { //nolint:exhaustive // Default case handles unknown button values.
	case tea.MouseButtonLeft:
		return 0, false
	case tea.MouseButtonMiddle:
		return 1, false
	case tea.MouseButtonRight:
		return 2, false
	case tea.MouseButtonNone:
		return 3, false
	case tea.MouseButtonWheelUp:
		return bitWheel, true
	case tea.MouseButtonWheelDown:
		return bitWheel | 1, true
	case tea.MouseButtonWheelLeft:
		return bitWheel | 2, true
	case tea.MouseButtonWheelRight:
		return bitWheel | 3, true
	case tea.MouseButtonBackward:
		return bitAdd, false
	case tea.MouseButtonForward:
		return bitAdd | 1, false
	default:
		return 0, false
	}
}

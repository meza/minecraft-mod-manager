package terminal

import "fmt"

// Capabilities describe whether terminal input/output should be treated as TTYs.
type Capabilities struct {
	InputIsTerminal  bool
	OutputIsTerminal bool
}

// TTYCapabilities returns capabilities for an interactive TTY session.
func TTYCapabilities() Capabilities {
	return Capabilities{InputIsTerminal: true, OutputIsTerminal: true}
}

// NonTTYCapabilities returns capabilities for non-interactive terminal output.
func NonTTYCapabilities() Capabilities {
	return Capabilities{InputIsTerminal: false, OutputIsTerminal: false}
}

// Size captures terminal dimensions.
type Size struct {
	Columns int
	Rows    int
}

func (size Size) String() string {
	return fmt.Sprintf("%dx%d", size.Columns, size.Rows)
}

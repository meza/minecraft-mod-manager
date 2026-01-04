package interaction

import (
	"io"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

// ExecutionMode describes the terminal interaction mode for a command run.
type ExecutionMode int

const (
	ExecutionModeInteractive ExecutionMode = iota
	ExecutionModeUnattended
	ExecutionModeNonTTY
)

// ExecutionModeInput defines the inputs needed to resolve an execution mode.
type ExecutionModeInput struct {
	Unattended bool
	In         io.Reader
	Out        io.Writer
}

// ResolveExecutionMode determines the interaction mode based on the provided input.
func ResolveExecutionMode(input ExecutionModeInput) ExecutionMode {
	if input.Unattended {
		return ExecutionModeUnattended
	}
	if view.SupportsPrompting(input.In, input.Out) {
		return ExecutionModeInteractive
	}
	return ExecutionModeNonTTY
}

// IsInteractive reports whether the mode supports interactive prompting.
func (mode ExecutionMode) IsInteractive() bool {
	return mode == ExecutionModeInteractive
}

func (mode ExecutionMode) String() string {
	switch mode {
	case ExecutionModeInteractive:
		return "interactive"
	case ExecutionModeUnattended:
		return "unattended"
	case ExecutionModeNonTTY:
		return "non_tty"
	default:
		return "unknown"
	}
}

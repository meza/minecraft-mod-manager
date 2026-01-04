package interaction

import (
	"io"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

// ConfigInitGate defines how to decide whether a missing-config prompt can be shown.
type ConfigInitGate struct {
	Unattended      bool
	In              io.Reader
	Out             io.Writer
	UnattendedError func(config.Metadata) error
	NoTTYError      func(config.Metadata) error
}

// CheckConfigInitGate returns an error when prompts are not allowed for the given configuration.
func CheckConfigInitGate(meta config.Metadata, gate ConfigInitGate) error {
	if gate.Unattended {
		return gate.UnattendedError(meta)
	}
	if view.SupportsPrompting(gate.In, gate.Out) {
		return nil
	}
	return gate.NoTTYError(meta)
}

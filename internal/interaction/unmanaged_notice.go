package interaction

import (
	"errors"

	"github.com/meza/minecraft-mod-manager/internal/output"
)

// UnmanagedNoticeOptions controls how unmanaged-file notices are emitted.
type UnmanagedNoticeOptions struct {
	Output     *output.Output
	Message    string
	UseError   bool
	Visibility output.LogVisibility
}

// LogUnmanagedNotice writes the provided unmanaged-file message using the configured output mode.
func LogUnmanagedNotice(options UnmanagedNoticeOptions) error {
	if options.Output == nil {
		return errors.New("missing output")
	}
	if options.UseError {
		return options.Output.Error(options.Message)
	}
	visibility := options.Visibility
	if visibility == 0 {
		visibility = output.LogForce
	}
	return options.Output.Log(options.Message, visibility)
}

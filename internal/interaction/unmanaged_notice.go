package interaction

import (
	"errors"
	"strings"

	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

// ErrUnmanagedFiles reports that unmanaged files were detected after emitting the notice.
var ErrUnmanagedFiles = errors.New("unmanaged files in mods folder")

// UnmanagedGateInput describes the unmanaged-file gate inputs.
type UnmanagedGateInput struct {
	Fs                     afero.Fs
	Meta                   config.Metadata
	Config                 models.ModsJSON
	Lock                   []models.ModInstall
	ColorMode              view.ColorMode
	Write                  func([]string) error
	AllowMissingModsFolder bool
}

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

// EmitUnmanagedNotice writes the notice message when non-empty using the provided writer.
func EmitUnmanagedNotice(message string, write func([]string) error) error {
	if strings.TrimSpace(message) == "" {
		return nil
	}
	if write == nil {
		return errors.New("missing output writer")
	}
	return write([]string{message})
}

// RequireNoUnmanagedFiles emits the unmanaged notice and returns ErrUnmanagedFiles when found.
func RequireNoUnmanagedFiles(input UnmanagedGateInput) (UnmanagedNotice, error) {
	var notice UnmanagedNotice
	var err error
	if input.AllowMissingModsFolder {
		notice, err = BuildUnmanagedNoticeIfModsFolderExists(input.Fs, input.Meta, input.Config, input.Lock, input.ColorMode)
	} else {
		notice, err = BuildUnmanagedNotice(input.Fs, input.Meta, input.Config, input.Lock, input.ColorMode)
	}
	if err != nil {
		return UnmanagedNotice{}, err
	}
	if err := EmitUnmanagedNotice(notice.Message, input.Write); err != nil {
		return UnmanagedNotice{}, err
	}
	if len(notice.Files) > 0 {
		return notice, ErrUnmanagedFiles
	}
	return notice, nil
}

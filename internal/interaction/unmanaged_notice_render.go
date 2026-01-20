package interaction

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfiles"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
)

// UnmanagedNotice captures unmanaged files plus the rendered notice message.
type UnmanagedNotice struct {
	Files   []string
	Message string
}

// BuildUnmanagedNotice scans for unmanaged jar files and formats the standard notice message.
func BuildUnmanagedNotice(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, colorMode view.ColorMode) (UnmanagedNotice, error) {
	files, err := modfiles.ListUnmanagedFiles(fs, meta, cfg, lock)
	if err != nil {
		return UnmanagedNotice{}, err
	}
	if len(files) == 0 {
		return UnmanagedNotice{}, nil
	}
	return UnmanagedNotice{
		Files:   files,
		Message: RenderUnmanagedNotice(files, colorMode),
	}, nil
}

// BuildUnmanagedNoticeIfModsFolderExists returns an empty notice when the mods folder does not exist.
func BuildUnmanagedNoticeIfModsFolderExists(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, colorMode view.ColorMode) (UnmanagedNotice, error) {
	notice, err := BuildUnmanagedNotice(fs, meta, cfg, lock, colorMode)
	if err == nil {
		return notice, nil
	}

	var readErr *modfiles.ReadError
	if errors.As(err, &readErr) && readErr.Path == meta.ModsFolderPath(cfg) && errors.Is(readErr.Err, os.ErrNotExist) {
		return UnmanagedNotice{}, nil
	}
	return UnmanagedNotice{}, err
}

// RenderUnmanagedNotice renders the standardized unmanaged files notice.
func RenderUnmanagedNotice(files []string, colorMode view.ColorMode) string {
	if len(files) == 0 {
		return ""
	}

	var builder strings.Builder

	header := i18n.T("cmd.list.unmanaged.header", nil)
	header = view.RenderIfColorEnabled(colorMode, view.TitleStyle, header)
	if err := view.WriteString(&builder, header); err != nil {
		return ""
	}
	if err := view.WriteString(&builder, "\n"); err != nil {
		return ""
	}

	icon := view.ErrorIcon(colorMode)
	for index, file := range files {
		if index > 0 {
			if err := view.WriteString(&builder, "\n"); err != nil {
				return ""
			}
		}
		entry := fmt.Sprintf("%s %s", icon, filepath.Base(file))
		if err := view.WriteString(&builder, entry); err != nil {
			return ""
		}
	}

	if err := view.WriteString(&builder, "\n\n"); err != nil {
		return ""
	}

	description := i18n.T("cmd.list.unmanaged.description", nil)
	if err := view.WriteString(&builder, description); err != nil {
		return ""
	}
	if err := view.WriteString(&builder, "\n"); err != nil {
		return ""
	}

	cta := i18n.T("cmd.list.unmanaged.cta", nil)
	if colorMode.Enabled() {
		cta = view.CtaStyle.Render(cta)
	}
	if err := view.WriteString(&builder, cta); err != nil {
		return ""
	}

	return builder.String()
}

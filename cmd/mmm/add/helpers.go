package add

import (
	"errors"
	"io"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

func colorModeForOutput(out io.Writer) tui.ColorMode {
	if tui.IsTerminalWriter(out) {
		return tui.ColorEnabled
	}
	return tui.ColorDisabled
}

func modNameForConfig(cfg models.ModsJSON, platformValue models.Platform, projectID string) string {
	for i := range cfg.Mods {
		if cfg.Mods[i].Type == platformValue && cfg.Mods[i].ID == projectID {
			return cfg.Mods[i].Name
		}
	}
	return projectID
}

func normalizePlatform(value string) models.Platform {
	switch strings.ToLower(value) {
	case string(models.CURSEFORGE):
		return models.CURSEFORGE
	case string(models.MODRINTH):
		return models.MODRINTH
	default:
		return models.Platform(strings.ToLower(value))
	}
}

func normalizedAddIdentifiers(opts addOptions) (models.Platform, string) {
	return normalizePlatform(opts.Platform), opts.ProjectID
}

func alternatePlatform(platformValue models.Platform) models.Platform {
	if platformValue == models.CURSEFORGE {
		return models.MODRINTH
	}
	return models.CURSEFORGE
}

func integrityErrorMessage(err error, modName string) (string, bool) {
	var missingHash modinstall.MissingHashError
	if errors.As(err, &missingHash) {
		return i18n.T("cmd.add.error.missing_hash_remote", &i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var hashMismatch modinstall.HashMismatchError
	if errors.As(err, &hashMismatch) {
		return i18n.T("cmd.add.error.hash_mismatch", &i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var outsideRoot modpath.OutsideRootError
	if errors.As(err, &outsideRoot) {
		return i18n.T("cmd.add.error.symlink_outside_mods", &i18n.Tvars{
			Data: &i18n.TData{
				"name": modName,
				"path": outsideRoot.ResolvedPath,
				"root": outsideRoot.Root,
			},
		}), true
	}

	return "", false
}

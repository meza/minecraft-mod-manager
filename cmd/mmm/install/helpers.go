package install

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func modVersionLabel(mod models.Mod) string {
	if mod.Version != nil && strings.TrimSpace(*mod.Version) != "" {
		return strings.TrimSpace(*mod.Version)
	}
	return "latest"
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

func colorModeForOutput(output io.Writer) view.ColorMode {
	if view.SupportsColor(output) && view.SupportsControlSequences(output) {
		return view.ColorEnabled
	}
	return view.ColorDisabled
}

func uniqueUint32s(values []uint32) []uint32 {
	seen := make(map[uint32]struct{}, len(values))
	result := make([]uint32, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func findConfiguredModIndex(cfg models.ModsJSON, hits []scanHit) int {
	for _, hit := range hits {
		for i := range cfg.Mods {
			mod := cfg.Mods[i]
			if mod.Type == hit.Platform && mod.ID == hit.Project {
				return i
			}
		}
	}
	return -1
}

func fileIsManaged(filePath string, installations []models.ModInstall) bool {
	filename := filepath.Base(filePath)
	for _, install := range installations {
		if install.FileName == filename {
			return true
		}
	}
	return false
}

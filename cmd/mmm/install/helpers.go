package install

import (
	"fmt"
	"io"
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

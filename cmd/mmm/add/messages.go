package add

import (
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func renderAddSuccessLine(colorMode view.ColorMode, name string, id string, platformValue models.Platform) string {
	idValue := view.RenderIfColorEnabled(colorMode, view.ParenStyle, id)
	platformText := view.RenderIfColorEnabled(colorMode, view.ParenStyle, string(platformValue))
	message := i18n.T("cmd.mod.display", &i18n.Tvars{
		Data: &i18n.TData{
			"name":     name,
			"id":       idValue,
			"platform": platformText,
		},
	})
	return fmt.Sprintf("%s %s", view.SuccessIcon(colorMode), message)
}

func renderAddSuccessLines(colorMode view.ColorMode, name string, id string, platformValue models.Platform) []string {
	header := i18n.T("cmd.add.header.success", nil)
	return []string{header, renderAddSuccessLine(colorMode, name, id, platformValue)}
}

func renderFinalErrorLine(colorMode view.ColorMode, message string) string {
	line := fmt.Sprintf("%s %s", view.FinalErrorIcon(colorMode), message)
	if colorMode.Enabled() {
		return view.ErrorStyle.Render(line)
	}
	return line
}

func projectNotFoundSummary(platformValue models.Platform, projectID string) string {
	return i18n.T("cmd.add.error.not_found", &i18n.Tvars{
		Data: &i18n.TData{
			"id":       projectID,
			"platform": string(platformValue),
		},
	})
}

func projectNotFoundUnattendedSummary(platformValue models.Platform, projectID string) string {
	return i18n.T("cmd.add.error.not_found_unattended", &i18n.Tvars{
		Data: &i18n.TData{
			"id":       projectID,
			"platform": string(platformValue),
		},
	})
}

func projectNotFoundHint(platformValue models.Platform, projectID string) string {
	return i18n.T("cmd.add.error.not_found_hint", &i18n.Tvars{
		Data: &i18n.TData{
			"id":       projectID,
			"platform": string(platformValue),
		},
	})
}

func noCompatibleSummary(loader string, gameVersion string) string {
	return i18n.T("cmd.add.error.no_compatible", &i18n.Tvars{
		Data: &i18n.TData{
			"gameVersion": gameVersion,
			"loader":      loader,
		},
	})
}

func downloadFailedSummary(platformValue models.Platform, projectID string) string {
	return i18n.T("cmd.add.error.download_failed", &i18n.Tvars{
		Data: &i18n.TData{
			"id":       projectID,
			"platform": string(platformValue),
		},
	})
}

func downloadFailedDetails(platformValue models.Platform, retryCount int) string {
	lines := []string{
		i18n.T("cmd.add.error.download_failed_retries", &i18n.Tvars{
			Data: &i18n.TData{"count": fmt.Sprintf("%d", retryCount)},
		}),
		i18n.T("cmd.add.error.download_failed_platform", &i18n.Tvars{
			Data: &i18n.TData{"platform": string(platformValue)},
		}),
		"",
		i18n.T("cmd.add.error.download_failed_network", &i18n.Tvars{
			Data: &i18n.TData{"platform": string(platformValue)},
		}),
	}
	return strings.Join(lines, "\n")
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

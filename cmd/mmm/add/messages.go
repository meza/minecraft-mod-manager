package add

import (
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func errorMessageForUnknownPlatform(platform string) string {
	return i18n.T("cmd.add.error.unknown_platform", i18n.Tvars{
		Data: &i18n.TData{
			"platform": platform,
		},
	})
}

func errorMessageForModNotFound(projectID string, platform models.Platform) string {
	return i18n.T("cmd.add.error.mod_not_found", i18n.Tvars{
		Data: &i18n.TData{
			"id":       projectID,
			"platform": platform,
		},
	})
}

func errorMessageForNoFile(projectID string, platform models.Platform) string {
	return i18n.T("cmd.add.error.no_file", i18n.Tvars{
		Data: &i18n.TData{
			"id":       projectID,
			"platform": platform,
		},
	})
}

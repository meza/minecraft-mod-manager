package init

import (
	"github.com/charmbracelet/huh"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"strings"
)

func (m Model) selectedReleaseTypes() []models.ReleaseType {
	if len(m.options.releaseTypes) > 0 {
		return m.options.releaseTypes
	}

	return m.form.Get("releaseTypes").([]models.ReleaseType)
}

func releaseTypesInput() *huh.MultiSelect[models.ReleaseType] {
	releaseTypeOptions := getReleaseTypeOptions()

	return huh.NewMultiSelect[models.ReleaseType]().
		Key("releaseTypes").
		Title(i18n.T("cmd.init.tui.release-types.question")).
		Options(releaseTypeOptions...)
}

func getReleaseTypeOptions() []huh.Option[models.ReleaseType] {
	releaseTypeOptions := make([]huh.Option[models.ReleaseType], 0)

	for _, releaseType := range models.AllReleaseTypes() {
		releaseTypeOptions = append(releaseTypeOptions, huh.NewOption(string(releaseType), releaseType))
	}
	return releaseTypeOptions
}

func parseReleaseTypes(releaseTypes string) []models.ReleaseType {
	var parsed []models.ReleaseType
	for _, rt := range strings.Split(releaseTypes, ",") {
		parsed = append(parsed, models.ReleaseType(strings.TrimSpace(rt)))
	}
	return parsed
}

func isValidReleaseTypes(releaseTypes string) bool {
	if releaseTypes == "" {
		return false
	}

	for _, rt := range parseReleaseTypes(releaseTypes) {
		if !isValidReleaseType(rt) {
			return false
		}
	}
	return true
}

func isValidReleaseType(releaseType models.ReleaseType) bool {
	for _, rt := range models.AllReleaseTypes() {
		if rt == releaseType {
			return true
		}
	}
	return false
}

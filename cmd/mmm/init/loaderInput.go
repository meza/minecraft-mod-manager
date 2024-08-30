package init

import (
	"github.com/charmbracelet/huh"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func (m Model) selectedLoader() models.Loader {
	if m.options.loader != "" && isValidLoader(m.options.loader) {
		return m.options.loader
	}

	return m.form.Get("loader").(models.Loader)
}

func loaderInput() *huh.Select[models.Loader] {

	loaderOptions := getLoaderOptions()

	return huh.NewSelect[models.Loader]().
		Key("loader").
		Title(i18n.T("cmd.init.tui.loader.question")).
		Options(loaderOptions...)
}

func isValidLoader(loader models.Loader) bool {
	for _, l := range models.AllLoaders() {
		if l == loader {
			return true
		}
	}
	return false
}

func getLoaderOptions() []huh.Option[models.Loader] {
	loaderOptions := make([]huh.Option[models.Loader], 0)

	for _, loader := range models.AllLoaders() {
		loaderOptions = append(loaderOptions, huh.NewOption(string(loader), loader))
	}
	return loaderOptions
}

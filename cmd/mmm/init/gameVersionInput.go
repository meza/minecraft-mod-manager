package init

import (
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"net/http"
)

func (m Model) selectedGameVersion() string {
	if m.options.gameVersion != "" && isValidGameVersion(m.options.gameVersion) {
		return m.options.gameVersion
	}

	return m.form.GetString("gameVersion")
}

func gameVersionInput() *huh.Input {
	return huh.NewInput().
		Suggestions(minecraft.GetAllMineCraftVersions(http.DefaultClient)).
		Key("gameVersion").
		Validate(func(value string) error {
			if value == "" {
				return fmt.Errorf(i18n.T("cmd.init.tui.game-version.error"))
			}

			if !minecraft.IsValidVersion(value, http.DefaultClient) {
				return fmt.Errorf(i18n.T("cmd.init.tui.game-version.invalid"))
			}
			return nil
		}).
		Title(i18n.T("cmd.init.tui.game-version.question"))
}

func isValidGameVersion(version string) bool {
	if version == "" {
		return false
	}

	return minecraft.IsValidVersion(version, http.DefaultClient)
}

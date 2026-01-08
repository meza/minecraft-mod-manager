package test

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestTestViewFinalSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	reason := i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(models.CURSEFORGE)},
	})

	snapshot := renderTestFinalView(testViewInput{
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "mod-a", Name: "Alpha Mod", Type: models.MODRINTH}, Status: testItemStatusSupported},
			{Mod: models.Mod{ID: "mod-b", Name: "Beta Mod", Type: models.CURSEFORGE}, Status: testItemStatusUnsupported},
			{Mod: models.Mod{ID: "mod-c", Name: "Gamma Mod", Type: models.CURSEFORGE}, Status: testItemStatusInconclusive, Reason: reason},
		},
	})

	snaps.MatchSnapshot(t, snapshot)
}

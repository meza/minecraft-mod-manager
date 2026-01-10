package update

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestUpdateViewSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	items := []updateItem{
		{
			ConfigIndex: 0,
			Mod:         models.Mod{ID: "mod-a", Name: "Alpha Mod", Type: models.MODRINTH},
			DisplayName: "Alpha Mod",
			Status:      updateItemStatusUpdating,
		},
		{
			ConfigIndex: 1,
			Mod:         models.Mod{ID: "mod-b", Name: "Beta Mod", Type: models.MODRINTH},
			DisplayName: "Beta Mod",
			Status:      updateItemStatusDownloading,
			Progress: &updateProgress{
				ratio:      0.2,
				downloaded: 200,
				total:      1000,
			},
		},
		{
			ConfigIndex: 2,
			Mod:         models.Mod{ID: "mod-c", Name: "Gamma Mod", Type: models.CURSEFORGE},
			DisplayName: "Gamma Mod",
			Status:      updateItemStatusUpdated,
		},
		{
			ConfigIndex: 3,
			Mod:         models.Mod{ID: "mod-d", Name: "Pinned Mod", Type: models.MODRINTH},
			DisplayName: "Pinned Mod",
			Status:      updateItemStatusSkipped,
		},
		{
			ConfigIndex: 4,
			Mod:         models.Mod{ID: "mod-e", Name: "Failed Mod", Type: models.MODRINTH},
			DisplayName: "Failed Mod",
			Status:      updateItemStatusFailed,
			FailReason:  "cmd.update.error.platform",
		},
		{
			ConfigIndex: 5,
			Mod:         models.Mod{ID: "mod-f", Name: "Zeta Mod", Type: models.MODRINTH},
			DisplayName: "Zeta Mod",
			Status:      updateItemStatusUpToDate,
		},
	}

	t.Run("running", func(t *testing.T) {
		output := renderUpdateRunningView(updateRunningViewInput{
			items:        items,
			colorMode:    view.ColorDisabled,
			spinnerFrame: "\u280B",
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("results_success", func(t *testing.T) {
		output := renderUpdateResultsView(updateResultsViewInput{
			items: []updateItem{
				items[5],
				items[2],
				items[3],
			},
			colorMode: view.ColorDisabled,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("results_partial", func(t *testing.T) {
		output := renderUpdateResultsView(updateResultsViewInput{
			items:     items,
			colorMode: view.ColorDisabled,
		})
		snaps.MatchSnapshot(t, output)
	})
}

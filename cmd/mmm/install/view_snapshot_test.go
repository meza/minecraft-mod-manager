package install

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestInstallViewSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []installItem{
		{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha", Status: installItemPending},
		{
			Mod:         models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
			DisplayName: "Beta",
			Status:      installItemDownloading,
			Progress: &installProgress{
				ratio:      0.5,
				downloaded: 512 * 1024,
				total:      1024 * 1024,
			},
		},
		{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, DisplayName: "Gamma", Status: installItemSuccess},
	}

	t.Run("running", func(t *testing.T) {
		output := renderInstallRunningView(view.ColorDisabled, items, "")
		snaps.MatchSnapshot(t, output)
	})

	t.Run("success", func(t *testing.T) {
		complete := []installItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha", Status: installItemSuccess},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE}, DisplayName: "Beta", Status: installItemSuccess},
		}
		output := renderInstallSuccessView(view.ColorDisabled, complete)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("download_failed", func(t *testing.T) {
		failed := []installItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha", Status: installItemSuccess},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE}, DisplayName: "Beta", Status: installItemFailed, FailureReason: "network"},
		}
		output := renderInstallDownloadFailedViewWithHint(view.ColorDisabled, failed)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("write_lock_failed", func(t *testing.T) {
		output := renderInstallWriteLockFailedView(view.ColorDisabled, items, "/cfg/modlist-lock.json")
		snaps.MatchSnapshot(t, output)
	})
}

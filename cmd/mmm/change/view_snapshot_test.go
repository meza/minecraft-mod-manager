package change

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestChangeViewSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []changeItem{
		{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha"},
		{
			Mod:            models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
			DisplayName:    "Beta",
			CompatStatus:   changeCompatSupported,
			DownloadStatus: changeDownloadInProgress,
			Download: changeDownloadProgress{
				ratio:      0.5,
				downloaded: 512 * 1024,
				total:      1024 * 1024,
			},
		},
		{
			Mod:            models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH},
			DisplayName:    "Gamma",
			CompatStatus:   changeCompatSupported,
			DownloadStatus: changeDownloadSucceeded,
			SwitchStatus:   changeSwitchPending,
		},
	}

	t.Run("running", func(t *testing.T) {
		output := view.RenderViewSections(buildChangeSections(changeViewInput{
			stage:        changeStageRunning,
			target:       "1.21.1",
			items:        items,
			colorMode:    view.ColorDisabled,
			spinnerFrame: ".",
		}), view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("compatibility_failed", func(t *testing.T) {
		failedItems := []changeItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha", CompatStatus: changeCompatUnsupported},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE}, DisplayName: "Beta", CompatStatus: changeCompatSupported},
		}
		output := view.RenderViewSections(buildChangeSections(changeViewInput{
			stage:        changeStageCompatibilityFailed,
			target:       "1.21.1",
			items:        failedItems,
			colorMode:    view.ColorDisabled,
			spinnerFrame: ".",
		}), view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("compatibility_failure_running", func(t *testing.T) {
		items := []changeItem{
			{Mod: models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}, DisplayName: "Alpha", CompatStatus: changeCompatUnsupported},
			{Mod: models.Mod{ID: "beta", Name: "Beta", Type: models.CURSEFORGE}, DisplayName: "Beta", CompatStatus: changeCompatChecking},
			{Mod: models.Mod{ID: "gamma", Name: "Gamma", Type: models.MODRINTH}, DisplayName: "Gamma", CompatStatus: changeCompatSupported},
		}
		output := view.RenderViewSections(buildChangeSections(changeViewInput{
			stage:        changeStageCompatibilityFailureDetected,
			target:       "1.21.1",
			items:        items,
			colorMode:    view.ColorDisabled,
			spinnerFrame: ".",
		}), view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("downloading", func(t *testing.T) {
		downloadItems := []changeItem{
			{
				Mod:            models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
				DisplayName:    "Alpha",
				CompatStatus:   changeCompatSupported,
				DownloadStatus: changeDownloadInProgress,
				Download: changeDownloadProgress{
					ratio:      0.5,
					downloaded: 512 * 1024,
					total:      1024 * 1024,
				},
			},
		}
		output := view.RenderViewSections(buildChangeSections(changeViewInput{
			stage:        changeStageRunning,
			target:       "1.21.1",
			items:        downloadItems,
			colorMode:    view.ColorDisabled,
			spinnerFrame: ".",
		}), view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("switching", func(t *testing.T) {
		switchItems := []changeItem{
			{
				Mod:            models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
				DisplayName:    "Alpha",
				CompatStatus:   changeCompatSupported,
				DownloadStatus: changeDownloadSucceeded,
				SwitchStatus:   changeSwitchInProgress,
			},
		}
		output := view.RenderViewSections(buildChangeSections(changeViewInput{
			stage:        changeStageSwitching,
			target:       "1.21.1",
			items:        switchItems,
			colorMode:    view.ColorDisabled,
			spinnerFrame: ".",
		}), view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("success", func(t *testing.T) {
		successItems := []changeItem{
			{
				Mod:             models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
				DisplayName:     "Alpha",
				CompatStatus:    changeCompatSupported,
				DownloadStatus:  changeDownloadSucceeded,
				SwitchStatus:    changeSwitchSucceeded,
				ResolvedVersion: "1.21.1",
			},
		}
		output := view.RenderViewSections(buildChangeSections(changeViewInput{
			stage:        changeStageSuccess,
			target:       "1.21.1",
			items:        successItems,
			colorMode:    view.ColorDisabled,
			spinnerFrame: ".",
		}), view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})
}

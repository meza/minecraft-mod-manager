package scan

import (
	"errors"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestScanViewSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	spinnerFrame := "\u280B"

	t.Run("running", func(t *testing.T) {
		items := []scanItem{
			{FileName: "appleskin.jar", Status: scanItemStatusScanning},
			{FileName: "bc.jar", Status: scanItemStatusScanning},
			{FileName: "delta.jar", Status: scanItemStatusScanning},
			{FileName: "flaky.jar", Status: scanItemStatusScanning},
			{FileName: "inventorysorter.jar", Status: scanItemStatusScanning},
			{FileName: "lithium-123.23.jar", Status: scanItemStatusScanning},
			{FileName: "shulkerbox.jar", Status: scanItemStatusScanning},
			{FileName: "sodium.jar", Status: scanItemStatusScanning},
			{FileName: "soundsbegone.jar", Status: scanItemStatusScanning},
			{FileName: "unmanaged.jar", Status: scanItemStatusScanning},
			{FileName: "zeta.jar", Status: scanItemStatusScanning},
		}

		output := renderScanRunningView(scanRunningViewInput{
			items:        items,
			colorMode:    view.ColorDisabled,
			spinnerFrame: spinnerFrame,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("running_with_sections", func(t *testing.T) {
		items := []scanItem{
			{FileName: "lithium-123.23.jar", Status: scanItemStatusScanning},
			{FileName: "sodium.jar", Status: scanItemStatusScanning},
			{
				FileName: "appleskin.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "bc.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "inventorysorter.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "shulkerbox.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
			},
			{
				FileName: "soundsbegone.jar",
				Status:   scanItemStatusRecognized,
				Match: scanMatch{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			{FileName: "unmanaged-a.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-b.jar", Status: scanItemStatusUnknown},
			{FileName: "unmanaged-c.jar", Status: scanItemStatusUnknown},
			{FileName: "flaky-a.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-b.jar", Status: scanItemStatusUnsure},
			{FileName: "flaky-c.jar", Status: scanItemStatusUnsure},
		}

		output := renderScanRunningView(scanRunningViewInput{
			items:        items,
			colorMode:    view.ColorDisabled,
			spinnerFrame: spinnerFrame,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("results", func(t *testing.T) {
		output := renderScanResultsView(scanResultsViewInput{
			matches: []scanMatch{
				{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "shulkerbox.jar",
					Name:      "Shulker Box Tooltip",
					ProjectID: "shulker-tooltip",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "sodium.jar",
					Name:      "Sodium",
					ProjectID: "sodium",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			unknown: []string{
				"unmanaged-a.jar",
				"unmanaged-b.jar",
				"unmanaged-c.jar",
				"unmanaged-d.jar",
			},
			unsure: []scanUnsure{
				{Path: "flaky-a.jar", Error: errors.New("timeout")},
				{Path: "flaky-b.jar", Error: errors.New("timeout")},
				{Path: "flaky-c.jar", Error: errors.New("timeout")},
			},
			colorMode: view.ColorDisabled,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("adoption_prompt", func(t *testing.T) {
		sections := buildScanResultSections(scanResultsViewInput{
			matches: []scanMatch{
				{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "bc.jar",
					Name:      "Better Clouds",
					ProjectID: "better-clouds",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "soundsbegone.jar",
					Name:      "Sounds Be Gone!",
					ProjectID: "sounds-be-gone",
					Platform:  models.MODRINTH,
				},
			},
			unknown: []string{
				"unmanaged-a.jar",
				"unmanaged-b.jar",
				"unmanaged-c.jar",
			},
			unsure: []scanUnsure{
				{Path: "flaky-a.jar", Error: errors.New("timeout")},
				{Path: "flaky-b.jar", Error: errors.New("timeout")},
				{Path: "flaky-c.jar", Error: errors.New("timeout")},
			},
			colorMode: view.ColorDisabled,
		})

		prompt := newScanConfirmPromptModel(i18n.T("cmd.scan.prompt.add", nil))
		sections = append(sections, prompt.View())
		output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("adoption_written", func(t *testing.T) {
		sections := buildScanResultSections(scanResultsViewInput{
			unknown: []string{
				"unmanaged-A.jar",
				"unmanaged-B.jar",
				"unmanaged-C.jar",
			},
			unsure: []scanUnsure{
				{Path: "flaky-a.jar", Error: errors.New("timeout")},
				{Path: "flaky-b.jar", Error: errors.New("timeout")},
				{Path: "flaky-c.jar", Error: errors.New("timeout")},
			},
			colorMode: view.ColorDisabled,
		})

		prompt := newScanConfirmPromptModel(i18n.T("cmd.scan.prompt.add", nil))
		prompt.Value = prompt.yesOption.short
		sections = append(sections, prompt.View())
		sections = append(sections, renderScanAddedSection(scanAddedViewInput{
			added: []scanMatch{
				{Name: "AppleSkin", ProjectID: "appleskin", Platform: models.MODRINTH},
				{Name: "Better Clouds", ProjectID: "better-clouds", Platform: models.MODRINTH},
				{Name: "Inventory Sorting", ProjectID: "inventory-sorting", Platform: models.MODRINTH},
				{Name: "Lithium", ProjectID: "lithium", Platform: models.MODRINTH},
				{Name: "Sounds Be Gone!", ProjectID: "sounds-be-gone", Platform: models.MODRINTH},
				{Name: "Sodium", ProjectID: "sodium", Platform: models.MODRINTH},
			},
			colorMode: view.ColorDisabled,
		}))

		output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("failure", func(t *testing.T) {
		output := renderScanFailureLine(view.ColorDisabled, errors.New("boom"))
		snaps.MatchSnapshot(t, output)
	})

	t.Run("unattended_results", func(t *testing.T) {
		output := renderScanResultsView(scanResultsViewInput{
			matches: []scanMatch{
				{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			colorMode: view.ColorDisabled,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("unattended_add", func(t *testing.T) {
		sections := buildScanResultSections(scanResultsViewInput{
			unknown: []string{
				"unmanaged-A.jar",
				"unmanaged-B.jar",
				"unmanaged-C.jar",
			},
			unsure: []scanUnsure{
				{Path: "flaky-a.jar", Error: errors.New("timeout")},
				{Path: "flaky-b.jar", Error: errors.New("timeout")},
			},
			colorMode: view.ColorDisabled,
		})
		sections = append(sections, renderScanAddedSection(scanAddedViewInput{
			added: []scanMatch{
				{Name: "AppleSkin", ProjectID: "appleskin", Platform: models.MODRINTH},
				{Name: "Inventory Sorting", ProjectID: "inventory-sorting", Platform: models.MODRINTH},
				{Name: "Lithium", ProjectID: "lithium", Platform: models.MODRINTH},
			},
			colorMode: view.ColorDisabled,
		}))
		output := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
		snaps.MatchSnapshot(t, output)
	})

	t.Run("non_tty_results", func(t *testing.T) {
		output := renderScanResultsView(scanResultsViewInput{
			matches: []scanMatch{
				{
					FileName:  "appleskin.jar",
					Name:      "AppleSkin",
					ProjectID: "appleskin",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "inventorysorter.jar",
					Name:      "Inventory Sorting",
					ProjectID: "inventory-sorting",
					Platform:  models.MODRINTH,
				},
				{
					FileName:  "lithium-123.23.jar",
					Name:      "Lithium",
					ProjectID: "lithium",
					Platform:  models.MODRINTH,
				},
			},
			colorMode: view.ColorDisabled,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("quiet_unknown_unsure", func(t *testing.T) {
		output := renderScanQuietView(scanResultsViewInput{
			unknown: []string{
				"unmanaged-A.jar",
				"unmanaged-B.jar",
				"unmanaged-C.jar",
			},
			unsure: []scanUnsure{
				{Path: "flaky-a.jar", Error: errors.New("timeout")},
				{Path: "flaky-b.jar", Error: errors.New("timeout")},
			},
			colorMode: view.ColorDisabled,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("quiet_add_no_output", func(t *testing.T) {
		snaps.MatchSnapshot(t, "")
	})
}

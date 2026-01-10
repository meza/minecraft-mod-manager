package update

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestUpdateTranscriptNonTTYSnapshots(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	t.Run("no_updates", func(t *testing.T) {
		items := []updateItem{
			{
				ConfigIndex: 0,
				Mod:         models.Mod{ID: "fabric-api", Name: "Fabric API", Type: models.MODRINTH},
				DisplayName: "Fabric API",
				Status:      updateItemStatusUpdating,
			},
			{
				ConfigIndex: 1,
				Mod:         models.Mod{ID: "pinned-mod", Name: "Pinned Mod", Type: models.CURSEFORGE},
				DisplayName: "Pinned Mod",
				Status:      updateItemStatusUpdating,
			},
		}
		indexByKey := map[int]int{0: 0, 1: 1}
		updates := []updateItemStatusMsg{
			{index: 0, status: updateItemStatusUpToDate},
			{index: 1, status: updateItemStatusSkipped},
		}
		finalItems := []updateItem{
			{
				ConfigIndex: 0,
				Mod:         models.Mod{ID: "fabric-api", Name: "Fabric API", Type: models.MODRINTH},
				DisplayName: "Fabric API",
				Status:      updateItemStatusUpToDate,
			},
			{
				ConfigIndex: 1,
				Mod:         models.Mod{ID: "pinned-mod", Name: "Pinned Mod", Type: models.CURSEFORGE},
				DisplayName: "Pinned Mod",
				Status:      updateItemStatusSkipped,
			},
		}
		output := runUpdateTranscriptSnapshot(items, indexByKey, updates, updateExecutionOutcome{
			items:   finalItems,
			errType: updateExecutionErrorNone,
		})
		snaps.MatchSnapshot(t, output)
	})

	t.Run("partial_success", func(t *testing.T) {
		items := []updateItem{
			{
				ConfigIndex: 0,
				Mod:         models.Mod{ID: "fabric-api", Name: "Fabric API", Type: models.MODRINTH},
				DisplayName: "Fabric API",
				Status:      updateItemStatusUpdating,
			},
			{
				ConfigIndex: 1,
				Mod:         models.Mod{ID: "inventory-sorting", Name: "Inventory Sorting", Type: models.MODRINTH},
				DisplayName: "Inventory Sorting",
				Status:      updateItemStatusUpdating,
			},
			{
				ConfigIndex: 2,
				Mod:         models.Mod{ID: "pinned-mod", Name: "Pinned Mod", Type: models.CURSEFORGE},
				DisplayName: "Pinned Mod",
				Status:      updateItemStatusUpdating,
			},
			{
				ConfigIndex: 3,
				Mod:         models.Mod{ID: "modmenu", Name: "Mod Menu", Type: models.MODRINTH},
				DisplayName: "Mod Menu",
				Status:      updateItemStatusUpdating,
			},
		}
		indexByKey := map[int]int{0: 0, 1: 1, 2: 2, 3: 3}
		updates := []updateItemStatusMsg{
			{index: 0, status: updateItemStatusUpToDate},
			{index: 1, status: updateItemStatusUpdated},
			{index: 2, status: updateItemStatusSkipped},
			{index: 3, status: updateItemStatusFailed, failReason: "cmd.update.error.platform"},
		}
		finalItems := []updateItem{
			{
				ConfigIndex: 0,
				Mod:         models.Mod{ID: "fabric-api", Name: "Fabric API", Type: models.MODRINTH},
				DisplayName: "Fabric API",
				Status:      updateItemStatusUpToDate,
			},
			{
				ConfigIndex: 1,
				Mod:         models.Mod{ID: "inventory-sorting", Name: "Inventory Sorting", Type: models.MODRINTH},
				DisplayName: "Inventory Sorting",
				Status:      updateItemStatusUpdated,
			},
			{
				ConfigIndex: 2,
				Mod:         models.Mod{ID: "pinned-mod", Name: "Pinned Mod", Type: models.CURSEFORGE},
				DisplayName: "Pinned Mod",
				Status:      updateItemStatusSkipped,
			},
			{
				ConfigIndex: 3,
				Mod:         models.Mod{ID: "modmenu", Name: "Mod Menu", Type: models.MODRINTH},
				DisplayName: "Mod Menu",
				Status:      updateItemStatusFailed,
				FailReason:  "cmd.update.error.platform",
			},
		}
		output := runUpdateTranscriptSnapshot(items, indexByKey, updates, updateExecutionOutcome{
			items:   finalItems,
			errType: updateExecutionErrorNone,
		})
		snaps.MatchSnapshot(t, output)
	})
}

func TestUpdateQuietPartialSuccessSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	items := []updateItem{
		{
			ConfigIndex: 0,
			Mod:         models.Mod{ID: "modmenu", Name: "Mod Menu", Type: models.MODRINTH},
			DisplayName: "Mod Menu",
			Status:      updateItemStatusFailed,
			FailReason:  "cmd.update.error.platform",
		},
	}
	lines := renderUpdateQuietFailure(updateResultsViewInput{
		items:     items,
		colorMode: view.ColorDisabled,
	})
	output := strings.TrimRight(view.RenderViewSections(lines, view.SectionSeparatorParagraph), "\n")
	snaps.MatchSnapshot(t, output)
}

func runUpdateTranscriptSnapshot(
	items []updateItem,
	indexByKey map[int]int,
	updates []updateItemStatusMsg,
	outcome updateExecutionOutcome,
) string {
	buffer := &bytes.Buffer{}
	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, buffer, nil)

	for _, update := range updates {
		_, cmd := model.Update(update)
		runTeaCmd(cmd)
	}
	_, cmd := model.Update(updateExecutionFinishedMsg{outcome: outcome})
	runTeaCmd(cmd)

	return strings.TrimRight(buffer.String(), "\n")
}

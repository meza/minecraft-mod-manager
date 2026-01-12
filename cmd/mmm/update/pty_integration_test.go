package update

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
	terminalpty "github.com/meza/minecraft-mod-manager/testutil/terminal/pty"
)

func TestUpdateCommandInteractivePTYRunningSnapshotShortHeight(t *testing.T) {
	runUpdatePTYRunningSnapshot(t, 25)
}

func TestUpdateCommandInteractivePTYRunningSnapshotMediumHeight(t *testing.T) {
	runUpdatePTYRunningSnapshot(t, 12)
}

func TestUpdateCommandInteractivePTYRunningSnapshotTallHeight(t *testing.T) {
	runUpdatePTYRunningSnapshot(t, tallSnapshotRows())
}

func TestUpdateCommandInteractivePTYFinalSnapshotShortHeight(t *testing.T) {
	runUpdatePTYFinalSnapshot(t, 25)
}

func TestUpdateCommandInteractivePTYFinalSnapshotMediumHeight(t *testing.T) {
	runUpdatePTYFinalSnapshot(t, 12)
}

func TestUpdateCommandInteractivePTYFinalSnapshotTallHeight(t *testing.T) {
	runUpdatePTYFinalSnapshot(t, tallSnapshotRows())
}

func TestUpdateCommandInteractivePTYFinalSnapshotLongListShortHeight(t *testing.T) {
	runUpdatePTYFinalSnapshotWithItems(t, 25, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYFinalSnapshotLongListMediumHeight(t *testing.T) {
	runUpdatePTYFinalSnapshotWithItems(t, 12, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYFinalSnapshotLongListTallHeight(t *testing.T) {
	runUpdatePTYFinalSnapshotWithItems(t, tallSnapshotRows(), sampleUpdateItemsLongList())
}

func TestExtractPTYTranscriptSnapshotReportsMissingTranscript(t *testing.T) {
	_, err := extractPTYTranscriptSnapshot("hello", "expected transcript")
	require.ErrorContains(t, err, "expected transcript not found")
}

func runUpdatePTYRunningSnapshot(t *testing.T, rows uint16) {
	runUpdatePTYRunningSnapshotWithItems(t, rows, sampleUpdateItems())
}

func TestUpdateCommandInteractivePTYRunningSnapshotLongListShortHeight(t *testing.T) {
	runUpdatePTYRunningSnapshotWithItems(t, 25, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYRunningSnapshotLongListMediumHeight(t *testing.T) {
	runUpdatePTYRunningSnapshotWithItems(t, 12, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYRunningSnapshotLongListTallHeight(t *testing.T) {
	runUpdatePTYRunningSnapshotWithItems(t, tallSnapshotRows(), sampleUpdateItemsLongList())
}

func tallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}

func runUpdatePTYRunningSnapshotWithItems(t *testing.T, rows uint16, items []updateItem) {
	terminal.ApplyFixtures(t)

	originalRunUpdateProgram := runUpdateProgram
	runUpdateProgram = defaultRunUpdateProgram
	t.Cleanup(func() { runUpdateProgram = originalRunUpdateProgram })

	columns := 120
	if rows == tallSnapshotRows() {
		columns = 80
	}
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: columns, Rows: int(rows)}))
	require.NotNil(t, session)

	release := make(chan struct{})
	updatesSent := make(chan struct{})

	cmd := commandWithUpdateRunner(func(ctx context.Context, cmd *cobra.Command) error {
		indexByKey := updateIndexByConfig(items)

		execRunner := func(ctx context.Context, sender updateExecSender) updateExecutionOutcome {
			close(updatesSent)
			<-release
			finalItems := finalizeUpdateItems(items)
			return updateExecutionOutcome{items: finalItems, errType: updateExecutionErrorNone}
		}

		model := newUpdateModel(updateModelInput{
			ctx:           ctx,
			colorMode:     colorModeForOutput(cmd.OutOrStdout()),
			items:         items,
			indexByKey:    indexByKey,
			suppressFinal: true,
			execRunner:    execRunner,
		})

		options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
		options = append(options, tea.WithAltScreen())
		result, runErr := runUpdateProgram(model, options...)
		if runErr != nil {
			return runErr
		}
		_, outcomeErr := updateOutcomeFromModel(result)
		return outcomeErr
	})

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "(mod-")
	}, terminalpty.WithWaitDuration(2*time.Second))

	snaps.MatchSnapshot(t, terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
		RowLimit:               int(rows),
		PadRows:                true,
	}))

	close(release)
	select {
	case <-execErr:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update command to finish")
	}
	require.NoError(t, session.Close())
}

func runUpdatePTYFinalSnapshot(t *testing.T, rows uint16) {
	runUpdatePTYFinalSnapshotWithItems(t, rows, sampleUpdateItems())
}

func runUpdatePTYFinalSnapshotWithItems(t *testing.T, rows uint16, items []updateItem) {
	terminal.ApplyFixtures(t)

	originalRunUpdateProgram := runUpdateProgram
	runUpdateProgram = defaultRunUpdateProgram
	t.Cleanup(func() { runUpdateProgram = originalRunUpdateProgram })

	columns := 120
	if rows == tallSnapshotRows() {
		columns = 80
	}
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: columns, Rows: int(rows)}))
	require.NotNil(t, session)

	cmd := commandWithUpdateRunner(func(ctx context.Context, cmd *cobra.Command) error {
		indexByKey := updateIndexByConfig(items)

		execRunner := func(ctx context.Context, sender updateExecSender) updateExecutionOutcome {
			finalItems := finalizeUpdateItems(items)
			return updateExecutionOutcome{items: finalItems, errType: updateExecutionErrorNone}
		}

		model := newUpdateModel(updateModelInput{
			ctx:           ctx,
			colorMode:     colorModeForOutput(cmd.OutOrStdout()),
			items:         items,
			indexByKey:    indexByKey,
			suppressFinal: true,
			execRunner:    execRunner,
		})

		options := view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())
		options = append(options, tea.WithAltScreen())
		result, runErr := runUpdateProgram(model, options...)
		if runErr != nil {
			return runErr
		}
		typed, ok := result.(*updateModel)
		if !ok {
			return errors.New("unexpected update model")
		}
		if outputErr := writeInteractiveUpdateTranscript(cmd, typed); outputErr != nil {
			return outputErr
		}
		return typed.outcome.err
	})

	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-execErr:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update command to finish")
	}

	expectedTranscript := renderUpdateResultsView(updateResultsViewInput{
		items:     finalizeUpdateItems(items),
		colorMode: view.ColorDisabled,
	})

	require.NoError(t, session.Close())
	transcript, err := extractPTYTranscriptSnapshot(session.OutputString(), expectedTranscript)
	require.NoError(t, err)
	snaps.MatchSnapshot(t, transcript)
}

func commandWithUpdateRunner(runner func(context.Context, *cobra.Command) error) *cobra.Command {
	return &cobra.Command{
		Use: "update",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runner(cmd.Context(), cmd)
		},
	}
}

func sampleUpdateItems() []updateItem {
	mods := []models.Mod{
		{ID: "mod-a", Name: "Alpha Mod", Type: models.MODRINTH},
		{ID: "mod-b", Name: "Beta Mod", Type: models.MODRINTH},
		{ID: "mod-c", Name: "Gamma Mod", Type: models.MODRINTH},
		{ID: "mod-d", Name: "Delta Mod", Type: models.MODRINTH},
		{ID: "mod-e", Name: "Epsilon Mod", Type: models.MODRINTH},
		{ID: "mod-f", Name: "Zeta Mod", Type: models.MODRINTH},
		{ID: "mod-g", Name: "Eta Mod", Type: models.MODRINTH},
		{ID: "mod-h", Name: "Theta Mod", Type: models.MODRINTH},
		{ID: "mod-i", Name: "Iota Mod", Type: models.MODRINTH},
		{ID: "mod-j", Name: "Kappa Mod", Type: models.MODRINTH},
		{ID: "mod-k", Name: "Pinned Mod", Type: models.MODRINTH},
		{ID: "mod-l", Name: "Failed Mod", Type: models.MODRINTH},
	}

	items := make([]updateItem, 0, len(mods))
	for index, mod := range mods {
		status := updateItemStatusUpdating
		if mod.Name == "Pinned Mod" {
			status = updateItemStatusSkipped
		}
		if mod.Name == "Failed Mod" {
			status = updateItemStatusFailed
		}
		item := updateItem{
			ConfigIndex: index,
			Mod:         mod,
			DisplayName: mod.Name,
			Status:      status,
		}
		if mod.Name == "Gamma Mod" {
			item.Status = updateItemStatusDownloading
			item.Progress = &updateProgress{
				ratio:      0.2,
				downloaded: 200,
				total:      1000,
			}
		}
		if status == updateItemStatusFailed {
			item.FailReason = "cmd.update.error.platform"
		}
		items = append(items, item)
	}
	return items
}

func sampleUpdateItemsLongList() []updateItem {
	mods := []models.Mod{
		{ID: "mod-a", Name: "Alpha Mod", Type: models.MODRINTH},
		{ID: "mod-b", Name: "Beta Mod", Type: models.MODRINTH},
		{ID: "mod-c", Name: "Gamma Mod", Type: models.MODRINTH},
		{ID: "mod-d", Name: "Delta Mod", Type: models.MODRINTH},
		{ID: "mod-e", Name: "Epsilon Mod", Type: models.MODRINTH},
		{ID: "mod-f", Name: "Zeta Mod", Type: models.MODRINTH},
		{ID: "mod-g", Name: "Eta Mod", Type: models.MODRINTH},
		{ID: "mod-h", Name: "Theta Mod", Type: models.MODRINTH},
		{ID: "mod-i", Name: "Iota Mod", Type: models.MODRINTH},
		{ID: "mod-j", Name: "Kappa Mod", Type: models.MODRINTH},
		{ID: "mod-k", Name: "Lambda Mod", Type: models.MODRINTH},
		{ID: "mod-l", Name: "Mu Mod", Type: models.MODRINTH},
		{ID: "mod-m", Name: "Nu Mod", Type: models.MODRINTH},
		{ID: "mod-n", Name: "Xi Mod", Type: models.MODRINTH},
		{ID: "mod-o", Name: "Omicron Mod", Type: models.MODRINTH},
		{ID: "mod-p", Name: "Pi Mod", Type: models.MODRINTH},
		{ID: "mod-q", Name: "Rho Mod", Type: models.MODRINTH},
		{ID: "mod-r", Name: "Sigma Mod", Type: models.MODRINTH},
		{ID: "mod-s", Name: "Tau Mod", Type: models.MODRINTH},
		{ID: "mod-t", Name: "Upsilon Mod", Type: models.MODRINTH},
		{ID: "mod-u", Name: "Phi Mod", Type: models.MODRINTH},
		{ID: "mod-v", Name: "Chi Mod", Type: models.MODRINTH},
		{ID: "mod-w", Name: "Psi Mod", Type: models.MODRINTH},
		{ID: "mod-x", Name: "Omega Mod", Type: models.MODRINTH},
		{ID: "mod-y", Name: "Pinned Mod", Type: models.MODRINTH},
		{ID: "mod-z", Name: "Failed Mod", Type: models.MODRINTH},
	}

	items := make([]updateItem, 0, len(mods))
	for index, mod := range mods {
		status := updateItemStatusUpdating
		if mod.Name == "Pinned Mod" {
			status = updateItemStatusSkipped
		}
		if mod.Name == "Failed Mod" {
			status = updateItemStatusFailed
		}
		item := updateItem{
			ConfigIndex: index,
			Mod:         mod,
			DisplayName: mod.Name,
			Status:      status,
		}
		if mod.Name == "Gamma Mod" {
			item.Status = updateItemStatusDownloading
			item.Progress = &updateProgress{
				ratio:      0.2,
				downloaded: 200,
				total:      1000,
			}
		}
		if status == updateItemStatusFailed {
			item.FailReason = "cmd.update.error.platform"
		}
		items = append(items, item)
	}
	return items
}

func finalizeUpdateItems(items []updateItem) []updateItem {
	finalItems := cloneUpdateItems(items)
	for index, item := range finalItems {
		if item.Status == updateItemStatusUpdating || item.Status == updateItemStatusDownloading {
			if index%2 == 0 {
				item.Status = updateItemStatusUpdated
			} else {
				item.Status = updateItemStatusUpToDate
			}
			item.Progress = nil
			finalItems[index] = item
		}
	}
	return finalItems
}

func updateIndexByConfig(items []updateItem) map[int]int {
	index := make(map[int]int, len(items))
	for position, item := range items {
		index[item.ConfigIndex] = position
	}
	return index
}

func extractPTYTranscriptSnapshot(output string, expectedTranscript string) (string, error) {
	normalized := terminal.NormalizeOutput(output, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	expected := terminal.NormalizeOutput(expectedTranscript, terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	if expected == "" {
		return "", errors.New("expected transcript is empty")
	}
	summaryKey := summaryLineKey(expected)
	if summaryKey != "" {
		count := countSummaryLines(normalized, summaryKey)
		if count != 1 {
			return "", fmt.Errorf("summary line occurrence count is %d", count)
		}
	}
	start := strings.LastIndex(normalized, expected)
	if start < 0 {
		return "", errors.New("expected transcript not found")
	}
	return strings.TrimSpace(normalized[start:]), nil
}

func summaryLineKey(expected string) string {
	if strings.Contains(expected, "cmd.update.summary.incomplete") {
		return "cmd.update.summary.incomplete"
	}
	if strings.Contains(expected, "cmd.update.summary.success") {
		return "cmd.update.summary.success"
	}
	return ""
}

func countSummaryLines(normalized string, summaryKey string) int {
	count := 0
	for _, line := range strings.Split(normalized, "\n") {
		if !strings.Contains(line, summaryKey) {
			continue
		}
		if summaryKey == "cmd.update.summary.incomplete" && strings.Contains(line, "cmd.update.summary.incomplete_hint") {
			continue
		}
		count++
	}
	return count
}

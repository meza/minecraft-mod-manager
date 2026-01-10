//go:build !windows

package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestUpdateCommandInteractivePTYRunningSnapshotShortHeight(t *testing.T) {
	runUpdatePTYRunningSnapshot(t, 6)
}

func TestUpdateCommandInteractivePTYRunningSnapshotMediumHeight(t *testing.T) {
	runUpdatePTYRunningSnapshot(t, 12)
}

func TestUpdateCommandInteractivePTYRunningSnapshotTallHeight(t *testing.T) {
	runUpdatePTYRunningSnapshot(t, 20)
}

func TestUpdateCommandInteractivePTYFinalSnapshotShortHeight(t *testing.T) {
	runUpdatePTYFinalSnapshot(t, 6)
}

func TestUpdateCommandInteractivePTYFinalSnapshotMediumHeight(t *testing.T) {
	runUpdatePTYFinalSnapshot(t, 12)
}

func TestUpdateCommandInteractivePTYFinalSnapshotTallHeight(t *testing.T) {
	runUpdatePTYFinalSnapshot(t, 20)
}

func TestUpdateCommandInteractivePTYFinalSnapshotLongListShortHeight(t *testing.T) {
	runUpdatePTYFinalSnapshotWithItems(t, 6, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYFinalSnapshotLongListMediumHeight(t *testing.T) {
	runUpdatePTYFinalSnapshotWithItems(t, 12, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYFinalSnapshotLongListTallHeight(t *testing.T) {
	runUpdatePTYFinalSnapshotWithItems(t, 20, sampleUpdateItemsLongList())
}

func TestExtractPTYTranscriptSnapshotReportsMissingTranscript(t *testing.T) {
	_, err := extractPTYTranscriptSnapshot("hello", "expected transcript")
	require.ErrorContains(t, err, "expected transcript not found")
}

func runUpdatePTYRunningSnapshot(t *testing.T, rows uint16) {
	runUpdatePTYRunningSnapshotWithItems(t, rows, sampleUpdateItems())
}

func TestUpdateCommandInteractivePTYRunningSnapshotLongListShortHeight(t *testing.T) {
	runUpdatePTYRunningSnapshotWithItems(t, 6, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYRunningSnapshotLongListMediumHeight(t *testing.T) {
	runUpdatePTYRunningSnapshotWithItems(t, 12, sampleUpdateItemsLongList())
}

func TestUpdateCommandInteractivePTYRunningSnapshotLongListTallHeight(t *testing.T) {
	runUpdatePTYRunningSnapshotWithItems(t, 20, sampleUpdateItemsLongList())
}

func runUpdatePTYRunningSnapshotWithItems(t *testing.T, rows uint16, items []updateItem) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	originalRunUpdateProgram := runUpdateProgram
	runUpdateProgram = defaultRunUpdateProgram
	t.Cleanup(func() { runUpdateProgram = originalRunUpdateProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: rows}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-updatesSent:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for running updates")
	}

	require.Eventually(t, func() bool {
		contents := output.String()
		return strings.Contains(contents, "(mod-")
	}, 2*time.Second, 10*time.Millisecond)

	snaps.MatchSnapshot(t, normalizePTYSnapshot(output.String(), rows))

	close(release)
	select {
	case <-execErr:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update command to finish")
	}
	closePTY(t, slave)
	<-readDone
	require.NoError(t, normalizePTYReadError(<-readErr))
	closePTY(t, master)
}

func runUpdatePTYFinalSnapshot(t *testing.T, rows uint16) {
	runUpdatePTYFinalSnapshotWithItems(t, rows, sampleUpdateItems())
}

func runUpdatePTYFinalSnapshotWithItems(t *testing.T, rows uint16, items []updateItem) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	originalRunUpdateProgram := runUpdateProgram
	runUpdateProgram = defaultRunUpdateProgram
	t.Cleanup(func() { runUpdateProgram = originalRunUpdateProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: rows}))

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

	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)

	output := &lockedBuffer{}
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	select {
	case <-execErr:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update command to finish")
	}

	closePTY(t, slave)
	<-readDone
	require.NoError(t, normalizePTYReadError(<-readErr))
	closePTY(t, master)

	expectedTranscript := renderUpdateResultsView(updateResultsViewInput{
		items:     finalizeUpdateItems(items),
		colorMode: view.ColorDisabled,
	})
	transcript, err := extractPTYTranscriptSnapshot(output.String(), expectedTranscript)
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

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (buffer *lockedBuffer) Write(p []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.Write(p)
}

func (buffer *lockedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.String()
}

func stripControlSequences(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	value = stripOSCSequences(value)
	value = stripCSISequences(value)

	return value
}

func normalizePTYSnapshot(value string, rows uint16) string {
	normalized := stripControlSequences(value)
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	lines = trimTrailingEmptyLines(lines)
	if rows > 0 {
		target := int(rows)
		if len(lines) > target {
			lines = lines[len(lines)-target:]
		} else if len(lines) < target {
			padding := make([]string, target-len(lines))
			lines = append(lines, padding...)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizePTYTranscriptSnapshot(value string) string {
	normalized := stripControlSequences(value)
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	lines = trimTrailingEmptyLines(lines)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func extractPTYTranscriptSnapshot(output string, expectedTranscript string) (string, error) {
	normalized := normalizePTYTranscriptSnapshot(output)
	expected := normalizePTYTranscriptSnapshot(expectedTranscript)
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

func trimTrailingEmptyLines(lines []string) []string {
	for len(lines) > 0 {
		if strings.TrimSpace(lines[len(lines)-1]) != "" {
			return lines
		}
		lines = lines[:len(lines)-1]
	}
	return lines
}

func closePTY(t *testing.T, file *os.File) {
	if file == nil {
		return
	}
	if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		require.NoError(t, err)
	}
}

func normalizePTYReadError(err error) error {
	if err == nil || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EIO) {
		return nil
	}
	return err
}

var oscSequence = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

func stripOSCSequences(value string) string {
	return oscSequence.ReplaceAllString(value, "")
}

var csiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripCSISequences(value string) string {
	return csiSequence.ReplaceAllString(value, "")
}

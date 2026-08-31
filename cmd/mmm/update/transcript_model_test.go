package update

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestUpdateTranscriptModelOutputsTerminalLinesOnce(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []updateItem{
		{
			ConfigIndex: 0,
			Mod:         models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			DisplayName: "Alpha",
			Status:      updateItemStatusSkipped,
		},
		{
			ConfigIndex: 1,
			Mod:         models.Mod{ID: "beta", Name: "Beta", Type: models.MODRINTH},
			DisplayName: "Beta",
			Status:      updateItemStatusUpdating,
		},
	}
	indexByKey := map[int]int{0: 0, 1: 1}

	buffer := &bytes.Buffer{}
	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, buffer, nil)

	_, cmd := model.Update(updateItemStatusMsg{index: 0, status: updateItemStatusSkipped})
	runTeaCmd(cmd)
	assert.Contains(t, buffer.String(), "cmd.update.header")
	assert.Contains(t, buffer.String(), "cmd.update.item.skipped")

	_, cmd = model.Update(updateItemStatusMsg{index: 0, status: updateItemStatusSkipped})
	assert.Nil(t, cmd)
	assert.Equal(t, 1, strings.Count(buffer.String(), "cmd.update.item.skipped"))

	_, cmd = model.Update(updateItemProgressMsg{index: 1, progress: updateProgress{ratio: 0.2, downloaded: 20, total: 100}})
	assert.Nil(t, cmd)

	_, cmd = model.Update(updateItemStatusMsg{index: 1, status: updateItemStatusFailed, failReason: "boom"})
	runTeaCmd(cmd)
	assert.Contains(t, buffer.String(), "cmd.update.item.failed")
	assert.Equal(t, 1, strings.Count(buffer.String(), "cmd.update.header"))

	_, cmd = model.Update(updateExecutionFinishedMsg{outcome: updateExecutionOutcome{errType: updateExecutionErrorNone}})
	runTeaCmd(cmd)
	assert.Contains(t, buffer.String(), "cmd.update.summary.success")
}

func TestUpdateTranscriptModelAddsHeaderBeforeSummaryWhenNoOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	buffer := &bytes.Buffer{}
	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, nil, map[int]int{}, buffer, nil)

	_, cmd := model.Update(updateExecutionFinishedMsg{outcome: updateExecutionOutcome{errType: updateExecutionErrorNone}})
	runTeaCmd(cmd)
	assert.Contains(t, buffer.String(), "cmd.update.header")
	assert.Contains(t, buffer.String(), "cmd.update.summary.success")
}

func TestUpdateTranscriptModelReturnsOutputErrors(t *testing.T) {
	writeErr := errors.New("write failed")
	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, nil, map[int]int{}, &bytes.Buffer{}, nil)

	updated, cmd := model.Update(outputLineErrorMsg{Err: writeErr})
	assert.NotNil(t, cmd)
	assert.ErrorIs(t, updated.(*updateTranscriptModel).outcome.err, writeErr)
	assert.Equal(t, updateExecutionErrorUnknown, updated.(*updateTranscriptModel).outcome.errType)
}

func TestUpdateTranscriptModelSkipsUnknownIndex(t *testing.T) {
	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, nil, map[int]int{}, &bytes.Buffer{}, nil)
	_, cmd := model.Update(updateItemStatusMsg{index: 42, status: updateItemStatusUpdated})
	assert.Nil(t, cmd)
}

func TestUpdateTranscriptModelIgnoresUnknownMessage(t *testing.T) {
	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, nil, map[int]int{}, &bytes.Buffer{}, nil)
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.IsType(t, &updateTranscriptModel{}, updated)
}

func TestUpdateTranscriptModelInitQuitsWhenMissingSender(t *testing.T) {
	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, nil, map[int]int{}, &bytes.Buffer{}, nil)
	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func runTeaCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	runTeaMsg(cmd())
}

func runTeaMsg(msg tea.Msg) {
	if msg == nil {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, cmd := range batch {
			runTeaCmd(cmd)
		}
		return
	}

	value := reflect.ValueOf(msg)
	if !value.IsValid() || value.Kind() != reflect.Slice {
		return
	}

	for index := 0; index < value.Len(); index++ {
		cmd, ok := value.Index(index).Interface().(tea.Cmd)
		if !ok {
			continue
		}
		runTeaCmd(cmd)
	}
}

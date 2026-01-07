package install

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestInstallTranscriptModelOutputsCompletionLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
		},
	}
	items, indexByKey := buildInstallItems(cfg)

	buffer := &bytes.Buffer{}
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, buffer, func(context.Context, httpclient.Sender) installExecutionOutcome {
		return installExecutionOutcome{errType: installExecutionErrorNone}
	})

	_, cmd := model.Update(installItemSuccessMsg{key: installModKey(cfg.Mods[0])})
	if cmd != nil {
		_ = cmd()
	}
	assert.Contains(t, buffer.String(), "alpha")

	_, cmd = model.Update(installItemSuccessMsg{key: installModKey(cfg.Mods[0])})
	if cmd != nil {
		_ = cmd()
	}
	assert.Contains(t, buffer.String(), "alpha")

	_, cmd = model.Update(installItemFailureMsg{key: installModKey(cfg.Mods[1]), reason: "boom"})
	if cmd != nil {
		_ = cmd()
	}
	assert.Contains(t, buffer.String(), "beta")
	assert.Contains(t, buffer.String(), "cmd.install.item.download_failed")

	_, cmd = model.Update(installExecutionFinishedMsg{outcome: installExecutionOutcome{errType: installExecutionErrorDownload}})
	runTeaCmd(cmd)
	assert.Contains(t, buffer.String(), "cmd.install.summary.download_failed")
}

func TestInstallTranscriptModelSummaryLinesForWriteFailures(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	items, indexByKey := buildInstallItems(cfg)
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &bytes.Buffer{}, nil)

	model.outcome = installExecutionOutcome{errType: installExecutionErrorWriteLock, lockPath: "/lock"}
	lines := model.summaryLines()
	assert.Contains(t, lines[1], "cmd.install.summary.write_failed_abort")

	model.outcome = installExecutionOutcome{errType: installExecutionErrorWriteConfig, configPath: "/config"}
	lines = model.summaryLines()
	assert.Contains(t, lines[1], "cmd.install.summary.write_failed_abort")

	model.outcome = installExecutionOutcome{errType: installExecutionErrorCanceled}
	lines = model.summaryLines()
	assert.Contains(t, lines[1], "cmd.install.summary.canceled")

	model.outcome = installExecutionOutcome{errType: installExecutionErrorUnknown, err: errors.New("boom")}
	lines = model.summaryLines()
	assert.Contains(t, lines[1], "cmd.install.error.failed")
}

func TestInstallTranscriptModelOutputsAbortedLine(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildInstallItems(cfg)

	buffer := &bytes.Buffer{}
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, buffer, nil)

	_, cmd := model.Update(installItemAbortedMsg{key: installModKey(cfg.Mods[0])})
	runTeaCmd(cmd)
	assert.Contains(t, buffer.String(), "cmd.install.item.aborted")
}

func TestInstallTranscriptModelSkipsDuplicateFailure(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildInstallItems(cfg)

	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &bytes.Buffer{}, nil)
	_, cmd := model.Update(installItemFailureMsg{key: installModKey(cfg.Mods[0]), reason: "boom"})
	runTeaCmd(cmd)

	_, cmd = model.Update(installItemFailureMsg{key: installModKey(cfg.Mods[0]), reason: "boom"})
	assert.Nil(t, cmd)
}

func TestInstallTranscriptModelSkipsDuplicateAborted(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildInstallItems(cfg)

	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &bytes.Buffer{}, nil)
	_, cmd := model.Update(installItemAbortedMsg{key: installModKey(cfg.Mods[0])})
	runTeaCmd(cmd)

	_, cmd = model.Update(installItemAbortedMsg{key: installModKey(cfg.Mods[0])})
	assert.Nil(t, cmd)
}

func TestInstallTranscriptModelReturnsOutputErrors(t *testing.T) {
	writeErr := errors.New("write failed")
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, nil, map[string]int{}, &bytes.Buffer{}, nil)

	updated, cmd := model.Update(outputLineErrorMsg{Err: writeErr})
	assert.NotNil(t, cmd)
	assert.ErrorIs(t, updated.(*installTranscriptModel).outcome.err, writeErr)
	assert.Equal(t, installExecutionErrorUnknown, updated.(*installTranscriptModel).outcome.errType)
}

func TestInstallTranscriptModelSkipsUnknownKey(t *testing.T) {
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, nil, map[string]int{}, &bytes.Buffer{}, nil)
	_, cmd := model.Update(installItemFailureMsg{key: "missing", reason: "boom"})
	assert.Nil(t, cmd)
}

func TestInstallTranscriptModelIgnoresUnknownMessage(t *testing.T) {
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, nil, map[string]int{}, &bytes.Buffer{}, nil)
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.IsType(t, &installTranscriptModel{}, updated)
}

func TestSummaryLinesCmdNilWhenEmpty(t *testing.T) {
	assert.Nil(t, summaryLinesCmd(nil, nil))
	assert.Nil(t, summaryLinesCmd(nil, []string{}))
}

func TestInstallTranscriptModelInitQuitsWhenMissingSender(t *testing.T) {
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, nil, map[string]int{}, &bytes.Buffer{}, nil)
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

	for i := 0; i < value.Len(); i++ {
		cmd, ok := value.Index(i).Interface().(tea.Cmd)
		if !ok {
			continue
		}
		runTeaCmd(cmd)
	}
}

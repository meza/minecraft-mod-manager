package scan

import (
	"bytes"
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type scanErrorWriter struct{}

func (scanErrorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestScanTranscriptModelInitReturnsQuitWithoutSender(t *testing.T) {
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestScanTranscriptModelInitReturnsQuitWithoutRunner(t *testing.T) {
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		&bytes.Buffer{},
		false,
		false,
		nil,
	)
	model.bindSender(func(tea.Msg) {})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestScanTranscriptModelUpdateWritesLine(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		out,
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	updated, cmd := model.Update(scanItemUpdateMsg{
		key:    "alpha.jar",
		status: scanItemStatusRecognized,
		match:  scanMatch{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH},
	})
	assert.Nil(t, cmd)
	assert.Contains(t, out.String(), "alpha.jar")
	assert.False(t, updated.(*scanTranscriptModel).finished)
}

func TestScanTranscriptModelUpdateSkipsTerminalStatus(t *testing.T) {
	out := &bytes.Buffer{}
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusRecognized}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		out,
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	updated, cmd := model.Update(scanItemUpdateMsg{key: "alpha.jar", status: scanItemStatusRecognized})
	assert.Nil(t, cmd)
	assert.Equal(t, "", out.String())
	assert.False(t, updated.(*scanTranscriptModel).finished)
}

func TestScanTranscriptModelUpdateQuietAddSkipsOutput(t *testing.T) {
	out := &bytes.Buffer{}
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		out,
		true,
		true,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	updated, cmd := model.Update(scanItemUpdateMsg{key: "alpha.jar", status: scanItemStatusUnknown})
	assert.Nil(t, cmd)
	assert.Equal(t, "", out.String())
	assert.False(t, updated.(*scanTranscriptModel).finished)
}

func TestScanTranscriptModelUpdateHandlesOutputError(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		scanErrorWriter{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	updated, cmd := model.Update(scanItemUpdateMsg{key: "alpha.jar", status: scanItemStatusUnknown})
	assert.NotNil(t, cmd)
	assert.Error(t, updated.(*scanTranscriptModel).outcome.err)
}

func TestScanTranscriptModelUpdateFinishedWritesSummary(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		out,
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	outcome := scanExecutionOutcome{
		items:   []scanItem{{FileName: "alpha.jar", Status: scanItemStatusUnknown}},
		unknown: []string{"alpha.jar"},
	}
	updated, cmd := model.Update(scanExecutionFinishedMsg{outcome: outcome})
	assert.NotNil(t, cmd)
	assert.True(t, updated.(*scanTranscriptModel).finished)
}

func TestScanTranscriptModelUpdateFinishedQuietUsesQuietLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		out,
		true,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	outcome := scanExecutionOutcome{
		items:   []scanItem{{FileName: "alpha.jar", Status: scanItemStatusUnknown}},
		unknown: []string{"alpha.jar"},
	}
	updated, cmd := model.Update(scanExecutionFinishedMsg{outcome: outcome})
	assert.NotNil(t, cmd)
	assert.True(t, updated.(*scanTranscriptModel).finished)
}

func TestScanTranscriptModelUpdateOutputLineErrorMsg(t *testing.T) {
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	updated, cmd := model.Update(outputLineErrorMsg{Err: errors.New("boom")})
	assert.NotNil(t, cmd)
	assert.EqualError(t, updated.(*scanTranscriptModel).outcome.err, "boom")
}

func TestScanTranscriptModelUpdateUnknownMessage(t *testing.T) {
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Nil(t, cmd)
	assert.NotNil(t, updated.(*scanTranscriptModel))
}

func TestScanTranscriptModelUpdateSkipsWhenFinished(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)
	model.finished = true

	updated, cmd := model.Update(scanItemUpdateMsg{key: "alpha.jar", status: scanItemStatusUnknown})
	assert.Nil(t, cmd)
	assert.NotNil(t, updated.(*scanTranscriptModel))
}

func TestScanTranscriptModelFinalTranscriptLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)
	model.add = true
	model.outcome = scanExecutionOutcome{
		added: []scanMatch{{Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH}},
	}

	lines := model.finalTranscriptLines(nil)
	assert.NotEmpty(t, lines)

	model.quiet = true
	assert.Nil(t, model.finalTranscriptLines(nil))

	model.quiet = false
	model.outcome = scanExecutionOutcome{err: errors.New("boom")}
	lines = model.finalTranscriptLines(nil)
	assert.NotEmpty(t, lines)
}

func TestScanTranscriptModelFinalTranscriptLinesAddsBlankLineWhenPriorLines(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)
	model.add = true
	model.outcome = scanExecutionOutcome{
		added: []scanMatch{{Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH}},
	}

	lines := model.finalTranscriptLines([]string{"prior"})
	if assert.Len(t, lines, 2) {
		assert.Equal(t, "", lines[0])
	}
}

func TestScanTranscriptModelApplyTranscriptUpdateEmitsRecognizedDuringAdd(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		out,
		false,
		true,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	lines := model.applyTranscriptUpdate(scanItemUpdateMsg{
		key:    "alpha.jar",
		status: scanItemStatusRecognized,
		match:  scanMatch{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH},
	})
	if assert.Len(t, lines, 1) {
		assert.Contains(t, lines[0], "alpha.jar")
	}
}

func TestScanTranscriptModelApplyTranscriptUpdateEmitsRecognizedAndUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	items := []scanItem{
		{FileName: "alpha.jar", Status: scanItemStatusPending},
		{FileName: "beta.jar", Status: scanItemStatusPending},
	}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		out,
		false,
		true,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	lines := model.applyTranscriptUpdate(scanItemUpdateMsg{
		key:    "alpha.jar",
		status: scanItemStatusRecognized,
		match:  scanMatch{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH},
	})
	if assert.Len(t, lines, 1) {
		assert.Contains(t, lines[0], "alpha.jar")
	}

	lines = model.applyTranscriptUpdate(scanItemUpdateMsg{key: "beta.jar", status: scanItemStatusUnsure})
	if assert.Len(t, lines, 1) {
		assert.Contains(t, lines[0], "beta.jar")
	}
}

func TestScanTranscriptModelApplyTranscriptUpdateUnknownKey(t *testing.T) {
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}},
		map[string]int{},
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	lines := model.applyTranscriptUpdate(scanItemUpdateMsg{key: "missing", status: scanItemStatusUnknown})
	assert.Empty(t, lines)
}

func TestScanTranscriptModelApplyTranscriptUpdateSkipsTerminalPreviousStatus(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusRecognized}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	lines := model.applyTranscriptUpdate(scanItemUpdateMsg{key: "alpha.jar", status: scanItemStatusUnknown})
	assert.Empty(t, lines)
}

func TestScanTranscriptModelApplyTranscriptUpdateQuietSkipsRecognized(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		&bytes.Buffer{},
		true,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	lines := model.applyTranscriptUpdate(scanItemUpdateMsg{
		key:    "alpha.jar",
		status: scanItemStatusRecognized,
		match:  scanMatch{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH},
	})
	assert.Empty(t, lines)
}

func TestScanTranscriptModelUpdateItemUnknownKey(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		scanIndexByFile(items),
		&bytes.Buffer{},
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	_, ok := model.updateItem(scanItemUpdateMsg{key: "missing", status: scanItemStatusUnknown})
	assert.False(t, ok)
}

func TestScanTranscriptModelShouldOutputStatusQuiet(t *testing.T) {
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		&bytes.Buffer{},
		true,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)

	assert.False(t, model.shouldOutputStatus(scanItemStatusRecognized))
	assert.True(t, model.shouldOutputStatus(scanItemStatusUnsure))
}

func TestMissingScanTranscriptLinesQuiet(t *testing.T) {
	items := []scanItem{
		{FileName: "alpha.jar", Status: scanItemStatusPending},
		{FileName: "beta.jar", Status: scanItemStatusPending},
	}
	finalItems := []scanItem{
		{FileName: "alpha.jar", Status: scanItemStatusUnknown},
		{FileName: "beta.jar", Status: scanItemStatusUnsure},
	}
	lines := missingScanTranscriptLinesQuiet(view.ColorDisabled, items, finalItems)
	assert.Len(t, lines, 2)
}

func TestMissingScanTranscriptLinesWithFilterSkipsTerminalCurrent(t *testing.T) {
	current := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusRecognized}}
	finalItems := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusRecognized}}

	lines := missingScanTranscriptLinesWithFilter(view.ColorDisabled, current, finalItems, func(scanItem) bool { return true })
	assert.Empty(t, lines)
}

func TestMissingScanTranscriptLinesWithFilterSkipsEmptyLine(t *testing.T) {
	finalItems := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	lines := missingScanTranscriptLinesWithFilter(view.ColorDisabled, nil, finalItems, func(scanItem) bool { return true })
	assert.Empty(t, lines)
}

func TestMissingScanTranscriptLinesWithFilterSkipsExcluded(t *testing.T) {
	finalItems := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusUnknown}}
	lines := missingScanTranscriptLinesWithFilter(view.ColorDisabled, nil, finalItems, func(scanItem) bool { return false })
	assert.Empty(t, lines)
}

func TestMissingScanTranscriptLinesWithFilterEmptyFinal(t *testing.T) {
	lines := missingScanTranscriptLinesWithFilter(view.ColorDisabled, nil, nil, func(scanItem) bool { return true })
	assert.Nil(t, lines)
}

func TestMissingScanTranscriptLinesUsesDeterministicFileOrder(t *testing.T) {
	candidates := []scanCandidate{
		{FileName: "beta.jar"},
		{FileName: "alpha.jar"},
	}
	items, _ := buildScanItems(candidates)
	finalItems := cloneScanItems(items)
	finalItems[0].Status = scanItemStatusUnknown
	finalItems[1].Status = scanItemStatusUnknown

	lines := missingScanTranscriptLines(view.ColorDisabled, items, finalItems)
	if assert.Len(t, lines, 2) {
		assert.Contains(t, lines[0], "alpha.jar")
		assert.Contains(t, lines[1], "beta.jar")
	}
}

func TestSummaryLinesCmdNilOnEmpty(t *testing.T) {
	assert.Nil(t, summaryLinesCmd(&bytes.Buffer{}, nil))
}

func TestWriteOutputLineErrorsOnNilOutput(t *testing.T) {
	assert.Error(t, writeOutputLine(nil, "line"))
}

package test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestTestModelInitReturnsQuitWhenMissingSender(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestTestModelUpdateItem(t *testing.T) {
	items := []testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
	}
	indexByKey := map[string]int{testModKey(items[0].Mod): 0}
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         items,
		indexByKey:    indexByKey,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	updated, _ := model.Update(testItemUpdateMsg{
		key:    testModKey(items[0].Mod),
		status: testItemStatusSupported,
		reason: "ok",
	})

	result := updated.(*testModel)
	require.Len(t, result.items, 1)
	assert.Equal(t, testItemStatusSupported, result.items[0].Status)
	assert.Equal(t, "ok", result.items[0].Reason)
}

func TestTestModelUpdateItemIgnoresUnknownKey(t *testing.T) {
	items := []testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
	}
	indexByKey := map[string]int{testModKey(items[0].Mod): 0}
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         items,
		indexByKey:    indexByKey,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	model.updateItem("missing", func(item *testItem) {
		item.Status = testItemStatusSupported
	})

	require.Len(t, model.items, 1)
	assert.Equal(t, testItemStatusChecking, model.items[0].Status)
}

func TestTestModelUpdateSpinnerTick(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	updated, cmd := model.Update(spinner.TickMsg{Time: time.Now()})
	assert.NotNil(t, cmd)
	assert.NotNil(t, updated.(*testModel).spinner)
}

func TestTestModelUpdateStartMsgEnablesCompatibility(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:               context.Background(),
		targetVersion:     "1.20.1",
		colorMode:         view.ColorDisabled,
		items:             []testItem{},
		indexByKey:        map[string]int{},
		showCompatibility: false,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	updated, _ := model.Update(testStartMsg{})
	assert.True(t, updated.(*testModel).showCompatibility)
}

func TestTestModelUpdateFinalizeQuits(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	_, cmd := model.Update(testFinalizeMsg{})
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestTestModelUpdateIgnoresUnknownMessage(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.False(t, updated.(*testModel).done)
}

func TestTestModelUpdateWindowSizeSetsDimensions(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	result := updated.(*testModel)
	assert.Equal(t, 80, result.windowW)
	assert.Equal(t, 24, result.windowH)
}

func TestTestModelUpdateKeyMsgScrollsViewport(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.viewport.Height = 1
	model.viewport.Width = 10
	model.viewport.SetContent("one\ntwo\nthree")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, updated.(*testModel).viewport.YOffset)
}

func TestTestModelUpdateMouseMsgScrollsViewport(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.viewport.MouseWheelEnabled = true
	model.viewport.Height = 1
	model.viewport.Width = 10
	model.viewport.SetContent("one\ntwo\nthree")

	updated, _ := model.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	assert.Equal(t, 2, updated.(*testModel).viewport.YOffset)
}

func TestTestModelUpdateViewportUsesWindowHeight(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowW = 0
	model.viewport.Width = 12

	model.updateViewport("one\ntwo", 5)

	assert.Equal(t, 5, model.viewport.Height)
	assert.Equal(t, 12, model.viewport.Width)
}

func TestTestModelUpdateViewportClampsNegativeHeightToZero(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	model.updateViewport("one", -2)

	assert.Equal(t, 0, model.viewport.Height)
}

func TestTestModelViewShowsHeaderWhenWindowShort(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	mod := models.Mod{ID: "alpha", Type: models.MODRINTH}
	model := newTestModel(testModelInput{
		ctx:               context.Background(),
		targetVersion:     "1.20.1",
		colorMode:         view.ColorDisabled,
		items:             []testItem{{Mod: mod, Status: testItemStatusChecking}},
		indexByKey:        map[string]int{testModKey(mod): 0},
		showCompatibility: true,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowH = 2

	viewOutput := model.View()
	assert.Contains(t, viewOutput, i18n.T("cmd.test.header", &i18n.Tvars{
		Data: &i18n.TData{"version": "1.20.1"},
	}))
}

func TestTestModelViewUsesViewportWhenWindowSized(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "delta", Type: models.MODRINTH}, Status: testItemStatusChecking},
	}
	indexByKey := make(map[string]int, len(items))
	for i, item := range items {
		indexByKey[testModKey(item.Mod)] = i
	}
	model := newTestModel(testModelInput{
		ctx:               context.Background(),
		targetVersion:     "1.20.1",
		colorMode:         view.ColorDisabled,
		items:             items,
		indexByKey:        indexByKey,
		showCompatibility: true,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowW = 80
	model.windowH = 5

	viewOutput := model.View()
	expectedHeight := model.windowH

	assert.Contains(t, viewOutput, "cmd.test.header")
	assert.Contains(t, viewOutput, "cmd.compatibility.section")
	assert.Contains(t, viewOutput, "alpha")
	assert.Equal(t, expectedHeight, model.viewport.Height)
	assert.Equal(t, 80, model.viewport.Width)
}

func TestTestModelViewKeepsHeaderWithViewport(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "gamma", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "delta", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "epsilon", Type: models.MODRINTH}, Status: testItemStatusChecking},
	}
	indexByKey := make(map[string]int, len(items))
	for i, item := range items {
		indexByKey[testModKey(item.Mod)] = i
	}
	model := newTestModel(testModelInput{
		ctx:               context.Background(),
		targetVersion:     "1.21.11",
		colorMode:         view.ColorDisabled,
		items:             items,
		indexByKey:        indexByKey,
		showCompatibility: true,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowH = 4

	viewOutput := model.View()
	assert.Contains(t, viewOutput, "cmd.test.header")
	assert.Contains(t, viewOutput, "cmd.compatibility.section")
}

func TestTestModelRenderRunningViewUsesDefaults(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
		},
		indexByKey: map[string]int{testModKey(models.Mod{ID: "alpha", Type: models.MODRINTH}): 0},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	viewOutput := model.renderRunningView()
	assert.Contains(t, viewOutput, "Test Minecraft version 1.20.1")
	assert.Contains(t, viewOutput, "Compatibility:")
}

func TestTestModelRenderRunningViewShowsFailureSectionsWhenUnsupportedFound(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.21.11",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
			{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, Status: testItemStatusSupported},
			{Mod: models.Mod{ID: "gamma", Type: models.MODRINTH}, Status: testItemStatusUnsupported},
		},
		indexByKey: map[string]int{
			testModKey(models.Mod{ID: "alpha", Type: models.MODRINTH}): 0,
			testModKey(models.Mod{ID: "beta", Type: models.MODRINTH}):  1,
			testModKey(models.Mod{ID: "gamma", Type: models.MODRINTH}): 2,
		},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	viewOutput := model.renderRunningView()
	assert.Contains(t, viewOutput, "Test Minecraft version 1.21.11")
	assert.Contains(t, viewOutput, "Compatible mods:")
	assert.Contains(t, viewOutput, "Not compatible mods:")
	assert.NotContains(t, viewOutput, "Compatibility:")
}

func TestTestModelRenderRunningViewReturnsHeaderWhenListEmpty(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:               context.Background(),
		targetVersion:     "1.20.1",
		colorMode:         view.ColorDisabled,
		items:             []testItem{},
		indexByKey:        map[string]int{},
		showCompatibility: true,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowH = 6

	viewOutput := model.renderRunningView()
	assert.Contains(t, viewOutput, "Test Minecraft version 1.20.1")
	assert.Contains(t, viewOutput, "Compatibility:")
}

func TestTestModelRenderRunningViewReturnsEmptyWhenRenderFails(t *testing.T) {
	originalWriteString := view.WriteString
	view.WriteString = func(_ io.Writer, _ string) error {
		return errors.New("write failed")
	}
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})

	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
		},
		indexByKey: map[string]int{testModKey(models.Mod{ID: "alpha", Type: models.MODRINTH}): 0},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowH = 5

	assert.Equal(t, "", model.renderRunningView())
}

func TestTestModelRenderRunningViewKeepsAllItems(t *testing.T) {
	items := []testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusChecking},
		{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, Status: testItemStatusChecking},
	}
	indexByKey := make(map[string]int, len(items))
	for i, item := range items {
		indexByKey[testModKey(item.Mod)] = i
	}
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         items,
		indexByKey:    indexByKey,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowH = 10

	viewOutput := model.renderRunningView()
	assert.Contains(t, viewOutput, "alpha")
	assert.Contains(t, viewOutput, "beta")
}

func TestTestModelRenderFinalViewIgnoresWindowSize(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusSupported},
		{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, Status: testItemStatusSupported},
		{Mod: models.Mod{ID: "gamma", Type: models.MODRINTH}, Status: testItemStatusSupported},
		{Mod: models.Mod{ID: "delta", Type: models.MODRINTH}, Status: testItemStatusSupported},
		{Mod: models.Mod{ID: "epsilon", Type: models.MODRINTH}, Status: testItemStatusSupported},
	}
	indexByKey := make(map[string]int, len(items))
	for i, item := range items {
		indexByKey[testModKey(item.Mod)] = i
	}
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         items,
		indexByKey:    indexByKey,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowH = 4
	model.done = true

	viewOutput := model.View()
	assert.Contains(t, viewOutput, "cmd.test.header")
	assert.Contains(t, viewOutput, "cmd.test.section.compatible")
}

func TestTestModelRenderFinalViewUsesDefaults(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusSupported},
		},
		indexByKey: map[string]int{testModKey(models.Mod{ID: "alpha", Type: models.MODRINTH}): 0},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.done = true

	viewOutput := model.View()
	assert.Contains(t, viewOutput, "Test Minecraft version 1.20.1")
}

func TestTestModelRenderFinalViewFallsBackToHeader(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items: []testItem{
			{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusSupported},
		},
		indexByKey: map[string]int{testModKey(models.Mod{ID: "alpha", Type: models.MODRINTH}): 0},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})
	model.windowH = 1
	model.done = true

	viewOutput := model.View()
	assert.Contains(t, viewOutput, "Test Minecraft version 1.20.1")
}

func TestNewTestModelUsesLineSpinnerWhenUnicodeDisabled(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	assert.Equal(t, spinner.Line.Frames[0], model.spinner.StaticFrame())
}

func TestTestModelUpdateFinishes(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:           context.Background(),
		targetVersion: "1.20.1",
		colorMode:     view.ColorDisabled,
		items:         []testItem{},
		indexByKey:    map[string]int{},
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	updated, cmd := model.Update(testExecutionFinishedMsg{
		outcome: testExecutionOutcome{items: []testItem{{Status: testItemStatusSupported}}},
	})
	msg := cmd()
	_, ok := msg.(testFinalizeMsg)
	assert.True(t, ok)

	result := updated.(*testModel)
	assert.True(t, result.done)
	assert.Len(t, result.items, 1)
}

func TestTestModelViewHeaderOnlyWhenCompatibilityHidden(t *testing.T) {
	model := newTestModel(testModelInput{
		ctx:               context.Background(),
		targetVersion:     "1.20.1",
		colorMode:         view.ColorDisabled,
		items:             []testItem{},
		indexByKey:        map[string]int{},
		showCompatibility: false,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	viewOutput := model.View()
	assert.Contains(t, viewOutput, renderTestHeader("1.20.1"))
	assert.NotContains(t, viewOutput, i18n.T("cmd.compatibility.section", nil))
}

func TestDefaultRunTestProgramReturnsModel(t *testing.T) {
	originalRunner := runTestProgram
	t.Cleanup(func() { runTestProgram = originalRunner })
	runTestProgram = defaultRunTestProgram

	model := newTestModel(testModelInput{
		ctx:               context.Background(),
		targetVersion:     "1.20.1",
		colorMode:         view.ColorDisabled,
		items:             []testItem{},
		indexByKey:        map[string]int{},
		showCompatibility: true,
		execRunner: func(context.Context, testExecSender) testExecutionOutcome {
			return testExecutionOutcome{}
		},
	})

	result, err := runTestProgram(model, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	require.NoError(t, err)
	err = testOutcomeFromModel(result)
	assert.NoError(t, err)
}

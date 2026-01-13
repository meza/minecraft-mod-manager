package scan

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestScanModelInitReturnsQuitWithoutSender(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}},
		indexByKey: map[string]int{"alpha.jar": 0},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestScanModelInitReturnsQuitWithoutRunner(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}},
		indexByKey: map[string]int{"alpha.jar": 0},
		execRunner: nil,
	})
	model.bindSender(func(tea.Msg) {})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestScanModelInitReturnsBatch(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}},
		indexByKey: map[string]int{"alpha.jar": 0},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.bindSender(func(tea.Msg) {})

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestScanModelInitReturnsStartScanCmdInTestMode(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}},
		indexByKey: map[string]int{"alpha.jar": 0},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome {
			return scanExecutionOutcome{unknown: []string{"alpha.jar"}}
		},
	})
	model.bindSender(func(tea.Msg) {})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(scanExecutionFinishedMsg)
	assert.True(t, ok)
}

func TestScanModelStartScanCmdReturnsOutcome(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome {
			return scanExecutionOutcome{unknown: []string{"alpha.jar"}}
		},
	})
	model.bindSender(func(tea.Msg) {})

	cmd := model.startScanCmd()
	msg := cmd()
	result, ok := msg.(scanExecutionFinishedMsg)
	assert.True(t, ok)
	assert.Equal(t, []string{"alpha.jar"}, result.outcome.unknown)
}

func TestScanModelUpdateWindowSize(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	assert.Nil(t, cmd)
	typed := updated.(*scanModel)
	assert.Equal(t, 80, typed.windowW)
	assert.Equal(t, 12, typed.windowH)
}

func TestScanModelUpdateKeyAndMouse(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Nil(t, cmd)
	assert.NotNil(t, updated.(*scanModel))

	updated, cmd = model.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	assert.Nil(t, cmd)
	assert.NotNil(t, updated.(*scanModel))
}

func TestScanModelUpdateCancelKeyCancelsContext(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	assert.NoError(t, model.ctx.Err())

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.Nil(t, cmd)
	assert.ErrorIs(t, updated.(*scanModel).ctx.Err(), context.Canceled)
}

func TestScanModelCancelStopsExecution(t *testing.T) {
	started := make(chan struct{})
	execDone := make(chan tea.Msg, 1)

	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(ctx context.Context, _ scanExecSender) scanExecutionOutcome {
			close(started)
			<-ctx.Done()
			return scanExecutionOutcome{err: ctx.Err()}
		},
	})
	model.bindSender(func(tea.Msg) {})

	cmd := model.startScanCmd()
	go func() {
		execDone <- cmd()
	}()

	<-started
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	select {
	case msg := <-execDone:
		_, ok := msg.(scanExecutionFinishedMsg)
		assert.True(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for scan execution to stop")
	}
}

func TestScanModelUpdateSpinnerTick(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	updated, cmd := model.Update(spinner.TickMsg{})
	assert.NotNil(t, cmd)
	assert.NotNil(t, updated.(*scanModel))
}

func TestScanModelUpdateUnknownMessage(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.NotNil(t, updated.(*scanModel))
}

func TestScanModelUpdateItem(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: scanIndexByFile(items),
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	match := scanMatch{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH}
	updated, cmd := model.Update(scanItemUpdateMsg{key: "alpha.jar", status: scanItemStatusRecognized, match: match})
	assert.Nil(t, cmd)
	typed := updated.(*scanModel)
	assert.Equal(t, scanItemStatusRecognized, typed.items[0].Status)
	assert.Equal(t, match, typed.items[0].Match)
}

func TestScanModelUpdateItemUnknownKey(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: scanIndexByFile(items),
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	updated, cmd := model.Update(scanItemUpdateMsg{key: "missing", status: scanItemStatusUnknown})
	assert.Nil(t, cmd)
	typed := updated.(*scanModel)
	assert.Equal(t, scanItemStatusPending, typed.items[0].Status)
}

func TestScanModelUpdateExecutionFinishedAndFinalize(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: scanIndexByFile(items),
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	outcome := scanExecutionOutcome{items: items, unknown: []string{"alpha.jar"}}
	updated, cmd := model.Update(scanExecutionFinishedMsg{outcome: outcome})
	assert.NotNil(t, cmd)
	typed := updated.(*scanModel)
	assert.True(t, typed.finalRender)
	assert.Equal(t, outcome, typed.outcome)

	updated, cmd = model.Update(scanFinalizeMsg{})
	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
	assert.NotNil(t, updated.(*scanModel))
}

func TestScanModelViewFinalRenderUsesResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.finalRender = true
	model.outcome = scanExecutionOutcome{
		matches: []scanMatch{{FileName: "alpha.jar", Name: "Alpha", ProjectID: "alpha", Platform: models.MODRINTH}},
	}

	viewText := model.View()
	assert.Contains(t, viewText, "cmd.scan.header.results")
}

func TestScanModelViewUpdatesViewport(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusScanning}}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: scanIndexByFile(items),
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.windowW = 80
	model.windowH = 2

	_ = model.View()
	assert.Equal(t, 2, model.viewport.Height)
	assert.Equal(t, 80, model.viewport.Width)
}

func TestScanModelViewWithZeroHeightReturnsContent(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusScanning}}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: scanIndexByFile(items),
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.windowH = 0

	viewText := model.View()
	assert.Contains(t, viewText, "cmd.scan.header.running")
}

func TestScanModelUpdateViewportClamp(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.windowW = 100
	model.windowH = 1

	model.updateViewport("one\ntwo", model.windowH)
	assert.Equal(t, 1, model.viewport.Height)
	assert.Equal(t, 100, model.viewport.Width)
}

func TestScanModelUpdateViewportHandlesNegativeHeight(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	model.updateViewport("one", -3)
	assert.Equal(t, 0, model.viewport.Height)
}

func TestNewScanModelUsesLineSpinnerWhenUnicodeDisabled(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})

	assert.Equal(t, spinner.Line.Frames[0], model.spinner.StaticFrame())
}

func TestScanModelUpdateViewportUsesContentHeight(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.windowW = 100
	model.windowH = 10

	model.updateViewport("one", model.windowH)
	assert.Equal(t, 1, model.viewport.Height)
	assert.Equal(t, 100, model.viewport.Width)
}

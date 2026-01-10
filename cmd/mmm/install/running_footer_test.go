package install

import (
	"context"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestWithRunningFooterSkipsNilRender(t *testing.T) {
	ctx := WithRunningFooter(context.Background(), RunningFooter{})
	assert.Nil(t, runningFooterFromContext(ctx))
}

func TestRunningFooterFromContextReturnsFooter(t *testing.T) {
	ctx := WithRunningFooter(context.Background(), RunningFooter{
		Render: func(RunningFooterInput) string { return "waiting" },
	})
	footer := runningFooterFromContext(ctx)
	if assert.NotNil(t, footer) {
		assert.NotNil(t, footer.Render)
	}
}

func TestInstallModelRunningFooterRendersLine(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	model := newInstallModel(
		context.Background(),
		view.ColorDisabled,
		[]installItem{},
		map[string]int{},
		nil,
		func(context.Context, httpclient.Sender) installExecutionOutcome { return installExecutionOutcome{} },
		&RunningFooter{Render: func(RunningFooterInput) string { return "waiting" }},
	)

	output := model.View()
	assert.Contains(t, output, "waiting")
	assert.Equal(t, spinner.Line, model.spinner.Spinner)
}

func TestInstallModelSpinnerTickUpdatesFooter(t *testing.T) {
	model := newInstallModel(
		context.Background(),
		view.ColorDisabled,
		nil,
		nil,
		nil,
		func(context.Context, httpclient.Sender) installExecutionOutcome { return installExecutionOutcome{} },
		&RunningFooter{Render: func(RunningFooterInput) string { return "waiting" }},
	)

	_, cmd := model.Update(spinner.TickMsg{})
	assert.NotNil(t, cmd)
}

func TestInstallModelInitWithFooterReturnsBatch(t *testing.T) {
	model := newInstallModel(
		context.Background(),
		view.ColorDisabled,
		nil,
		nil,
		nil,
		func(context.Context, httpclient.Sender) installExecutionOutcome { return installExecutionOutcome{} },
		&RunningFooter{Render: func(RunningFooterInput) string { return "waiting" }},
	)
	model.bindSender(func(tea.Msg) {})

	msg := model.Init()()
	_, ok := msg.(tea.BatchMsg)
	assert.True(t, ok)
}

func TestInstallModelUpdateSpinnerTickWithoutFooter(t *testing.T) {
	model := newInstallModel(
		context.Background(),
		view.ColorDisabled,
		nil,
		nil,
		nil,
		func(context.Context, httpclient.Sender) installExecutionOutcome { return installExecutionOutcome{} },
		nil,
	)

	updated, cmd := model.Update(spinner.TickMsg{})
	assert.Nil(t, cmd)
	assert.Equal(t, model, updated)
}

func TestInstallModelSpinnerFrameReturnsEmptyOnError(t *testing.T) {
	model := newInstallModel(
		context.Background(),
		view.ColorDisabled,
		nil,
		nil,
		nil,
		func(context.Context, httpclient.Sender) installExecutionOutcome { return installExecutionOutcome{} },
		&RunningFooter{Render: func(RunningFooterInput) string { return "waiting" }},
	)

	model.spinner.Spinner = spinner.Spinner{}
	assert.Equal(t, "", model.spinnerFrame())
}

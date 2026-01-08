package test

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type testModel struct {
	ctx               context.Context
	cancel            context.CancelFunc
	sender            testExecSender
	execRunner        func(context.Context, testExecSender) testExecutionOutcome
	colorMode         view.ColorMode
	targetVersion     string
	items             []testItem
	indexByKey        map[string]int
	spinner           spinner.Model
	viewport          viewport.Model
	showCompatibility bool
	done              bool
	outcome           testExecutionOutcome
	windowW           int
	windowH           int
}

type testModelInput struct {
	ctx               context.Context
	targetVersion     string
	colorMode         view.ColorMode
	items             []testItem
	indexByKey        map[string]int
	showCompatibility bool
	execRunner        func(context.Context, testExecSender) testExecutionOutcome
}

type testStartMsg struct{}

type testItemUpdateMsg struct {
	key    string
	status testItemStatus
	reason string
}

type testExecutionFinishedMsg struct {
	outcome testExecutionOutcome
}

type testFinalizeMsg struct{}

func newTestModel(input testModelInput) *testModel {
	ctx, cancel := context.WithCancel(input.ctx)
	spin := spinner.New()
	if view.SupportsUnicode() {
		spin.Spinner = spinner.Dot
	} else {
		spin.Spinner = spinner.Line
	}
	model := &testModel{
		ctx:               ctx,
		cancel:            cancel,
		execRunner:        input.execRunner,
		colorMode:         input.colorMode,
		targetVersion:     input.targetVersion,
		items:             cloneTestItems(input.items),
		indexByKey:        input.indexByKey,
		spinner:           spin,
		viewport:          viewport.New(0, 0),
		showCompatibility: input.showCompatibility,
	}
	model.viewport.MouseWheelEnabled = true
	return model
}

func (model *testModel) bindSender(send func(tea.Msg)) {
	model.sender = testExecSender{send: send}
}

func (model *testModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return tea.Batch(model.spinner.Tick, model.startTestCmd(), model.startViewCmd())
}

func (model *testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.windowW = typed.Width
		model.windowH = typed.Height
		return model, nil
	case tea.KeyMsg:
		updated, cmd := model.viewport.Update(typed)
		model.viewport = updated
		return model, cmd
	case tea.MouseMsg:
		updated, cmd := model.viewport.Update(typed)
		model.viewport = updated
		return model, cmd
	case spinner.TickMsg:
		updated, cmd := model.spinner.Update(typed)
		model.spinner = updated
		return model, cmd
	case testStartMsg:
		model.showCompatibility = true
		return model, nil
	case testItemUpdateMsg:
		model.updateItem(typed.key, func(item *testItem) {
			item.Status = typed.status
			item.Reason = typed.reason
		})
		return model, nil
	case testExecutionFinishedMsg:
		model.items = typed.outcome.items
		model.outcome = typed.outcome
		model.done = true
		return model, func() tea.Msg { return testFinalizeMsg{} }
	case testFinalizeMsg:
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *testModel) View() string {
	if model.done {
		return model.renderFinalView()
	}
	if !model.showCompatibility {
		return renderTestHeader(model.targetVersion)
	}
	return model.renderRunningView()
}

func (model *testModel) startTestCmd() tea.Cmd {
	return func() tea.Msg {
		return testExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *testModel) startViewCmd() tea.Cmd {
	return func() tea.Msg { return testStartMsg{} }
}

func (model *testModel) spinnerFrame() string {
	return model.spinner.View()
}

func (model *testModel) updateItem(key string, update func(*testItem)) {
	index, ok := model.indexByKey[key]
	if !ok || index < 0 || index >= len(model.items) {
		return
	}
	item := model.items[index]
	update(&item)
	model.items[index] = item
}

func (model *testModel) renderRunningView() string {
	if model.windowH <= 0 {
		return renderTestRunningView(testViewInput{
			targetVersion: model.targetVersion,
			items:         model.items,
			colorMode:     model.colorMode,
			spinnerFrame:  model.spinnerFrame(),
		})
	}

	viewText := renderTestRunningView(testViewInput{
		targetVersion: model.targetVersion,
		items:         model.items,
		colorMode:     model.colorMode,
		spinnerFrame:  model.spinnerFrame(),
	})
	if viewText == "" {
		return viewText
	}

	model.updateViewport(viewText, model.windowH)
	return model.viewport.View()
}

func (model *testModel) renderFinalView() string {
	return renderTestFinalView(testViewInput{
		targetVersion: model.targetVersion,
		items:         model.items,
		colorMode:     model.colorMode,
	})
}

func (model *testModel) updateViewport(content string, height int) {
	model.viewport.SetContent(content)

	contentHeight := lipgloss.Height(content)
	viewportHeight := height
	if viewportHeight > contentHeight {
		viewportHeight = contentHeight
	}
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	model.viewport.Height = viewportHeight
	if model.windowW > 0 {
		model.viewport.Width = model.windowW
	}
	model.viewport.SetYOffset(model.viewport.YOffset)
}

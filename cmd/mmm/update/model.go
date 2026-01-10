package update

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type updateModel struct {
	ctx           context.Context
	cancel        context.CancelFunc
	execRunner    func(context.Context, updateExecSender) updateExecutionOutcome
	colorMode     view.ColorMode
	items         []updateItem
	indexByKey    map[int]int
	spinner       spinner.Model
	viewport      viewport.Model
	windowW       int
	windowH       int
	finalRender   bool
	userScrolled  bool
	suppressFinal bool
	outcome       updateExecutionOutcome
	sender        updateExecSender
}

type updateModelInput struct {
	ctx           context.Context
	colorMode     view.ColorMode
	items         []updateItem
	indexByKey    map[int]int
	execRunner    func(context.Context, updateExecSender) updateExecutionOutcome
	suppressFinal bool
}

type updateExecSender struct {
	send func(tea.Msg)
}

func (sender updateExecSender) Send(msg tea.Msg) {
	if sender.send == nil {
		return
	}
	sender.send(msg)
}

type updateItemStatusMsg struct {
	index       int
	status      updateItemStatus
	failReason  string
	displayName string
}

type updateItemProgressMsg struct {
	index    int
	progress updateProgress
}

type updateExecutionFinishedMsg struct {
	outcome updateExecutionOutcome
}

type updateFinalizeMsg struct{}

func newUpdateModel(input updateModelInput) *updateModel {
	ctx, cancel := context.WithCancel(input.ctx)
	spin := spinner.New()
	if view.SupportsUnicode() {
		spin.Spinner = spinner.Dot
	} else {
		spin.Spinner = spinner.Line
	}
	spin.Style = lipgloss.NewStyle()

	model := &updateModel{
		ctx:           ctx,
		cancel:        cancel,
		execRunner:    input.execRunner,
		colorMode:     input.colorMode,
		items:         cloneUpdateItems(input.items),
		indexByKey:    input.indexByKey,
		spinner:       spin,
		viewport:      viewport.New(0, 0),
		suppressFinal: input.suppressFinal,
	}
	model.viewport.MouseWheelEnabled = true
	return model
}

func (model *updateModel) bindSender(send func(tea.Msg)) {
	model.sender = updateExecSender{send: send}
}

func (model *updateModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	if updateTestModeEnabled() {
		return model.startUpdateCmd()
	}
	return tea.Batch(model.spinner.Tick, model.startUpdateCmd())
}

func (model *updateModel) startUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		if model.execRunner == nil {
			return updateExecutionFinishedMsg{outcome: updateExecutionOutcome{err: context.Canceled, errType: updateExecutionErrorCanceled}}
		}
		return updateExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *updateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.windowW = typed.Width
		model.windowH = typed.Height
		return model, nil
	case tea.KeyMsg:
		return model.handleViewportKey(typed)
	case tea.MouseMsg:
		return model.handleViewportMouse(typed)
	default:
		return model.handleUpdateMessage(typed)
	}
}

func (model *updateModel) View() string {
	if model.finalRender {
		if model.suppressFinal {
			return ""
		}
		switch model.outcome.errType {
		case updateExecutionErrorWriteLock:
			return renderUpdateWriteLockFailureView(updateErrorViewInput{
				colorMode: model.colorMode,
				lockPath:  model.outcome.lockPath,
			})
		case updateExecutionErrorWriteConfig:
			return renderUpdateWriteConfigFailureView(updateErrorViewInput{
				colorMode:  model.colorMode,
				configPath: model.outcome.configPath,
			})
		case updateExecutionErrorCanceled:
			return ""
		case updateExecutionErrorUnknown:
			return renderFinalErrorLine(model.colorMode, model.outcome.err.Error())
		default:
			return renderUpdateResultsView(updateResultsViewInput{
				items:     model.items,
				colorMode: model.colorMode,
			})
		}
	}

	content := renderUpdateRunningView(updateRunningViewInput{
		items:        model.items,
		colorMode:    model.colorMode,
		spinnerFrame: model.spinnerFrame(),
	})
	if model.windowH <= 0 || content == "" {
		return content
	}
	model.updateViewport(content, model.windowH)
	return model.viewport.View()
}

func (model *updateModel) updateViewport(content string, height int) {
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
	maxOffset := contentHeight - viewportHeight
	targetOffset := model.viewport.YOffset
	if !model.userScrolled {
		targetOffset = maxOffset
	}
	model.viewport.SetYOffset(targetOffset)
}

func (model *updateModel) spinnerFrame() string {
	frame := model.spinner.View()
	trimmed := strings.TrimSpace(frame)
	if trimmed == "" || trimmed == "(error)" {
		return ""
	}
	return frame
}

func (model *updateModel) handleViewportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		if model.cancel != nil {
			model.cancel()
		}
		return model, nil
	}
	previousOffset := model.viewport.YOffset
	updated, cmd := model.viewport.Update(msg)
	model.viewport = updated
	if model.viewport.YOffset != previousOffset || isViewportScrollKey(msg) {
		model.userScrolled = true
	}
	return model, cmd
}

func (model *updateModel) handleViewportMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	previousOffset := model.viewport.YOffset
	updated, cmd := model.viewport.Update(msg)
	model.viewport = updated
	if model.viewport.YOffset != previousOffset || isViewportScrollMouse(msg) {
		model.userScrolled = true
	}
	return model, cmd
}

func (model *updateModel) handleUpdateMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case spinner.TickMsg:
		updated, cmd := model.spinner.Update(typed)
		model.spinner = updated
		return model, cmd
	case updateItemStatusMsg:
		model.applyItemUpdate(typed)
		return model, nil
	case updateItemProgressMsg:
		model.applyItemProgress(typed)
		return model, nil
	case updateExecutionFinishedMsg:
		model.items = typed.outcome.items
		model.outcome = typed.outcome
		model.finalRender = true
		return model, func() tea.Msg { return updateFinalizeMsg{} }
	case updateFinalizeMsg:
		return model, tea.Quit
	default:
		return model, nil
	}
}

func isViewportScrollKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		return true
	default:
		switch msg.String() {
		case "j", "k", "g", "G":
			return true
		default:
			return false
		}
	}
}

func isViewportScrollMouse(msg tea.MouseMsg) bool {
	if msg.Action != tea.MouseActionPress {
		return false
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp,
		tea.MouseButtonWheelDown,
		tea.MouseButtonWheelLeft,
		tea.MouseButtonWheelRight:
		return true
	default:
		return false
	}
}

func updateTestModeEnabled() bool {
	_, present := os.LookupEnv("MMM_TEST")
	return present
}

func (model *updateModel) applyItemUpdate(msg updateItemStatusMsg) {
	index, ok := model.indexByKey[msg.index]
	if !ok || index < 0 || index >= len(model.items) {
		return
	}
	item := model.items[index]
	item.Status = msg.status
	item.FailReason = msg.failReason
	if strings.TrimSpace(msg.displayName) != "" {
		item.DisplayName = msg.displayName
	}
	if item.Status != updateItemStatusDownloading {
		item.Progress = nil
	}
	model.items[index] = item
}

func (model *updateModel) applyItemProgress(msg updateItemProgressMsg) {
	index, ok := model.indexByKey[msg.index]
	if !ok || index < 0 || index >= len(model.items) {
		return
	}
	item := model.items[index]
	item.Status = updateItemStatusDownloading
	item.Progress = &msg.progress
	model.items[index] = item
}

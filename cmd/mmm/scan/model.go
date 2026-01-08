package scan

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

type scanItemStatus int

const (
	scanItemStatusPending scanItemStatus = iota
	scanItemStatusScanning
	scanItemStatusRecognized
	scanItemStatusUnknown
	scanItemStatusUnsure
)

type scanItem struct {
	FileName string
	Status   scanItemStatus
	Match    scanMatch
}

type scanModel struct {
	ctx         context.Context
	cancel      context.CancelFunc
	sender      scanExecSender
	execRunner  func(context.Context, scanExecSender) scanExecutionOutcome
	colorMode   view.ColorMode
	items       []scanItem
	indexByKey  map[string]int
	spinner     spinner.Model
	viewport    viewport.Model
	windowW     int
	windowH     int
	finalRender bool
	outcome     scanExecutionOutcome
}

type scanModelInput struct {
	ctx        context.Context
	colorMode  view.ColorMode
	items      []scanItem
	indexByKey map[string]int
	execRunner func(context.Context, scanExecSender) scanExecutionOutcome
}

type scanItemUpdateMsg struct {
	key    string
	status scanItemStatus
	match  scanMatch
}

type scanExecutionFinishedMsg struct {
	outcome scanExecutionOutcome
}

type scanFinalizeMsg struct{}

func newScanModel(input scanModelInput) *scanModel {
	ctx, cancel := context.WithCancel(input.ctx)
	spin := spinner.New()
	if view.SupportsUnicode() {
		spin.Spinner = spinner.Dot
	} else {
		spin.Spinner = spinner.Line
	}
	spin.Style = lipgloss.NewStyle()

	model := &scanModel{
		ctx:        ctx,
		cancel:     cancel,
		execRunner: input.execRunner,
		colorMode:  input.colorMode,
		items:      cloneScanItems(input.items),
		indexByKey: input.indexByKey,
		spinner:    spin,
		viewport:   viewport.New(0, 0),
	}
	model.viewport.MouseWheelEnabled = true
	return model
}

func (model *scanModel) bindSender(send func(tea.Msg)) {
	model.sender = scanExecSender{send: send}
}

func (model *scanModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	if scanTestModeEnabled() {
		return model.startScanCmd()
	}
	return tea.Batch(model.spinner.Tick, model.startScanCmd())
}

func (model *scanModel) startScanCmd() tea.Cmd {
	return func() tea.Msg {
		return scanExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.windowW = typed.Width
		model.windowH = typed.Height
		return model, nil
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "q", "esc":
			if model.cancel != nil {
				model.cancel()
			}
			return model, nil
		}
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
	case scanItemUpdateMsg:
		model.applyItemUpdate(typed)
		return model, nil
	case scanExecutionFinishedMsg:
		model.items = typed.outcome.items
		model.outcome = typed.outcome
		model.finalRender = true
		return model, func() tea.Msg { return scanFinalizeMsg{} }
	case scanFinalizeMsg:
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *scanModel) View() string {
	content := renderScanRunningView(scanRunningViewInput{
		items:        model.items,
		colorMode:    model.colorMode,
		spinnerFrame: model.spinnerFrame(),
	})
	if model.finalRender {
		content = renderScanResultsView(scanResultsViewInput{
			matches:   model.outcome.matches,
			unknown:   model.outcome.unknown,
			unsure:    model.outcome.unsure,
			colorMode: model.colorMode,
		})
		return content
	}
	if model.windowH <= 0 || content == "" {
		return content
	}
	model.updateViewport(content, model.windowH)
	return model.viewport.View()
}

func (model *scanModel) updateViewport(content string, height int) {
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

func (model *scanModel) spinnerFrame() string {
	frame := model.spinner.View()
	trimmed := strings.TrimSpace(frame)
	if trimmed == "" || trimmed == "(error)" {
		return ""
	}
	return frame
}

func scanTestModeEnabled() bool {
	_, present := os.LookupEnv("MMM_TEST")
	return present
}

func (model *scanModel) applyItemUpdate(msg scanItemUpdateMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok || index < 0 || index >= len(model.items) {
		return
	}
	item := model.items[index]
	item.Status = msg.status
	if msg.status == scanItemStatusRecognized {
		item.Match = msg.match
	}
	model.items[index] = item
}

func scanIndexByFile(items []scanItem) map[string]int {
	index := make(map[string]int, len(items))
	for position, item := range items {
		index[item.FileName] = position
	}
	return index
}

func cloneScanItems(items []scanItem) []scanItem {
	cloned := make([]scanItem, len(items))
	copy(cloned, items)
	return cloned
}

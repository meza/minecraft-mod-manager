package scan

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
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
	ctx          context.Context
	cancel       context.CancelFunc
	sender       scanExecSender
	execRunner   func(context.Context, scanExecSender) scanExecutionOutcome
	colorMode    view.ColorMode
	items        []scanItem
	indexByKey   map[string]int
	spinner      view.Spinner
	viewport     viewport.Model
	windowW      int
	windowH      int
	userScrolled bool
	finalRender  bool
	outcome      scanExecutionOutcome
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
	spin := view.NewSpinner()

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
	return tea.Batch(model.spinner.InitCmd(), model.startScanCmd())
}

func (model *scanModel) startScanCmd() tea.Cmd {
	return func() tea.Msg {
		return scanExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updated, cmd, handled := model.spinner.Update(msg); handled {
		model.spinner = updated
		return model, cmd
	}
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.windowW = typed.Width
		model.windowH = typed.Height
		return model, nil
	case tea.KeyMsg:
		return model.handleViewportKey(typed)
	case tea.MouseMsg:
		return model.handleViewportMouse(typed)
	case scanItemUpdateMsg:
		model.applyItemUpdate(typed)
		return model, tea.WindowSize()
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
		items:     model.items,
		colorMode: model.colorMode,
		spinner:   &model.spinner,
	})
	if model.finalRender {
		content = renderScanResultsView(scanResultsViewInput{
			matches:   model.outcome.matches,
			unknown:   model.outcome.unknown,
			unsure:    model.outcome.unsure,
			colorMode: model.colorMode,
		})
		return model.renderWithStickyHeader(content, "cmd.scan.header.results")
	}
	return model.renderWithStickyHeader(content, "cmd.scan.header.running")
}

func (model *scanModel) updateViewport(content string, height int) {
	model.viewport.SetContent(content)

	contentHeight := lipgloss.Height(content)
	viewportHeight := view.ClampViewportHeight(height)

	model.viewport.Height = viewportHeight
	if model.windowW > 0 {
		model.viewport.Width = model.windowW
	}
	maxOffset := view.MaxViewportOffset(contentHeight, viewportHeight)
	targetOffset := model.viewport.YOffset
	if !model.userScrolled {
		targetOffset = maxOffset
	}
	model.viewport.SetYOffset(targetOffset)
}

func (model *scanModel) renderWithStickyHeader(content string, headerKey string) string {
	if content == "" {
		return content
	}
	headerLine, body := splitHeaderFromContent(content)
	if body == "" {
		return content
	}
	headerText := i18n.T(headerKey, nil)
	if !strings.Contains(headerLine, headerText) {
		return content
	}
	bodyHeight := lipgloss.Height(body)
	model.updateViewport(body, bodyHeight)
	rendered := view.RenderViewSections([]string{headerLine, model.viewport.View()}, view.SectionSeparatorLine)
	if model.windowH <= 0 {
		return view.RenderViewSections([]string{rendered, headerLine}, view.SectionSeparatorParagraph)
	}
	if lipgloss.Height(rendered) > model.windowH {
		return view.RenderViewSections([]string{rendered, headerLine}, view.SectionSeparatorParagraph)
	}
	return rendered
}

func splitHeaderFromContent(content string) (header string, body string) {
	parts := strings.SplitN(content, "\n", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func (model *scanModel) handleViewportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (model *scanModel) handleViewportMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	previousOffset := model.viewport.YOffset
	updated, cmd := model.viewport.Update(msg)
	model.viewport = updated
	if model.viewport.YOffset != previousOffset || isViewportScrollMouse(msg) {
		model.userScrolled = true
	}
	return model, cmd
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

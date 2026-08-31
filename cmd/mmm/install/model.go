package install

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type installViewState int

const (
	installViewRunning installViewState = iota
	installViewSuccess
	installViewDownloadFailed
	installViewWriteLockFailed
	installViewWriteConfigFailed
	installViewCanceled
	installViewFailed
)

type installSectionSeparator int

const (
	installSeparatorLine installSectionSeparator = iota
	installSeparatorParagraph
)

type installViewLayout struct {
	header      string
	listLines   []string
	footerLines []string
	separator   installSectionSeparator
}

type installModel struct {
	ctx        context.Context
	execRunner func(context.Context, httpclient.Sender) installExecutionOutcome
	cancel     func()
	colorMode  view.ColorMode
	items      []installItem
	indexByKey map[string]int
	state      installViewState
	outcome    installExecutionOutcome
	sender     installExecSender
	spinner    view.Spinner
	footer     *RunningFooter
	viewport   viewport.Model
	windowW    int
	windowH    int
	userScroll bool
}

type installExecSender struct {
	send func(tea.Msg)
}

func (sender installExecSender) Send(msg tea.Msg) {
	if sender.send == nil {
		return
	}
	sender.send(msg)
}

func newInstallModel(
	ctx context.Context,
	colorMode view.ColorMode,
	items []installItem,
	indexByKey map[string]int,
	cancel func(),
	execRunner func(context.Context, httpclient.Sender) installExecutionOutcome,
	footer *RunningFooter,
) *installModel {
	model := &installModel{
		ctx:        ctx,
		execRunner: execRunner,
		cancel:     cancel,
		colorMode:  colorMode,
		items:      items,
		indexByKey: indexByKey,
		state:      installViewRunning,
		footer:     footer,
	}
	if footer != nil {
		model.spinner = view.NewSpinner()
	}
	model.viewport = viewport.New(0, 0)
	model.viewport.MouseWheelEnabled = true
	return model
}

func (model *installModel) bindSender(send func(tea.Msg)) {
	model.sender = installExecSender{send: send}
}

func (model *installModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return tea.Batch(model.spinner.InitCmd(), model.startInstallCmd())
}

func (model *installModel) startInstallCmd() tea.Cmd {
	return func() tea.Msg {
		if model.execRunner == nil {
			return installExecutionFinishedMsg{outcome: installExecutionOutcome{
				err:     errors.New("missing execution runner"),
				errType: installExecutionErrorUnknown,
			}}
		}
		return installExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *installModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updated, cmd, handled := model.spinner.Update(msg); handled {
		model.spinner = updated
		return model, cmd
	}
	if cmd, handled := model.handleInstallMessage(msg); handled {
		return model, cmd
	}
	if cmd, handled := model.handleViewportMessage(msg); handled {
		return model, cmd
	}
	return model, nil
}

func (model *installModel) handleInstallMessage(msg tea.Msg) (tea.Cmd, bool) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		if typed.Width > 0 {
			model.windowW = typed.Width
		}
		if typed.Height > 0 {
			model.windowH = typed.Height
		}
		return nil, true
	case installItemProgressMsg:
		model.applyProgress(typed)
		return tea.WindowSize(), true
	case installItemProgressErrMsg:
		model.applyProgressError(typed)
		return tea.WindowSize(), true
	case installItemSuccessMsg:
		model.applySuccess(typed)
		return tea.WindowSize(), true
	case installItemFailureMsg:
		model.applyFailure(typed)
		return tea.WindowSize(), true
	case installItemAbortedMsg:
		model.applyAborted(typed)
		return tea.WindowSize(), true
	case installExecutionFinishedMsg:
		model.outcome = typed.outcome
		model.state = viewStateFromOutcome(typed.outcome.errType)
		return tea.Quit, true
	default:
		return nil, false
	}
}

func (model *installModel) handleViewportMessage(msg tea.Msg) (tea.Cmd, bool) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		return model.handleKeyMsg(typed), true
	case tea.MouseMsg:
		return model.handleMouseMsg(typed), true
	default:
		return nil, false
	}
}

func (model *installModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		if model.cancel != nil {
			model.cancel()
		}
	}
	return model.updateViewportFromKey(msg)
}

func (model *installModel) handleMouseMsg(msg tea.MouseMsg) tea.Cmd {
	return model.updateViewportFromMouse(msg)
}

func (model *installModel) updateViewportFromKey(msg tea.KeyMsg) tea.Cmd {
	previousOffset := model.viewport.YOffset
	updated, cmd := model.viewport.Update(msg)
	model.viewport = updated
	if model.viewport.YOffset != previousOffset || isViewportScrollKey(msg) {
		model.userScroll = true
	}
	return cmd
}

func (model *installModel) updateViewportFromMouse(msg tea.MouseMsg) tea.Cmd {
	previousOffset := model.viewport.YOffset
	updated, cmd := model.viewport.Update(msg)
	model.viewport = updated
	if model.viewport.YOffset != previousOffset || isViewportScrollMouse(msg) {
		model.userScroll = true
	}
	return cmd
}

func (model *installModel) View() string {
	return model.renderInstallViewWithLayout(model.viewLayout())
}

func (model *installModel) viewLayout() installViewLayout {
	listLines := renderInstallItemLines(model.colorMode, model.items)
	switch model.state {
	case installViewSuccess:
		return model.layoutWithSummary(
			i18n.T("cmd.install.header.success", nil),
			listLines,
			[]string{renderInstallSuccessSummary(model.colorMode)},
		)
	case installViewDownloadFailed:
		return model.layoutWithSummary(
			i18n.T("cmd.install.header.success", nil),
			listLines,
			renderInstallDownloadFailureSummaryWithHint(model.colorMode),
		)
	case installViewWriteLockFailed:
		return model.layoutWithSummary(
			i18n.T("cmd.install.header.success", nil),
			listLines,
			renderInstallWriteLockSummary(model.colorMode, model.outcome.lockPath),
		)
	case installViewWriteConfigFailed:
		return model.layoutWithSummary(
			i18n.T("cmd.install.header.success", nil),
			listLines,
			renderInstallWriteConfigSummary(model.colorMode, model.outcome.configPath),
		)
	case installViewCanceled:
		return model.layoutWithSummary(
			i18n.T("cmd.install.header.success", nil),
			listLines,
			renderInstallCanceledSummary(model.colorMode),
		)
	case installViewFailed:
		return model.layoutWithSummary(
			i18n.T("cmd.install.header.success", nil),
			listLines,
			renderInstallExecutionFailureSummary(model.colorMode, model.outcome.err),
		)
	default:
		return model.layoutForRunning(listLines)
	}
}

func (model *installModel) layoutWithSummary(header string, listLines []string, footerLines []string) installViewLayout {
	return installViewLayout{
		header:      header,
		listLines:   listLines,
		footerLines: footerLines,
		separator:   installSeparatorParagraph,
	}
}

func (model *installModel) layoutForRunning(listLines []string) installViewLayout {
	footerLine := model.runningFooterLine()
	footerLines := []string{}
	if strings.TrimSpace(footerLine) != "" {
		footerLines = []string{footerLine}
	}
	return installViewLayout{
		header:      i18n.T("cmd.install.header.success", nil),
		listLines:   listLines,
		footerLines: footerLines,
		separator:   installSeparatorLine,
	}
}

func (model *installModel) renderInstallViewWithLayout(layout installViewLayout) string {
	separatorText, separatorExtraLines := installSeparatorDetails(layout.separator)
	output := layout.header
	listText := strings.Join(layout.listLines, "\n")
	if listText != "" {
		output = strings.Join([]string{output, listText}, "\n")
	}

	listHeight := lipgloss.Height(listText)
	footerLines := model.footerLinesWithHeaderIfNeeded(layout, separatorExtraLines, listHeight)
	if len(footerLines) == 0 {
		return model.renderInstallViewWithStickyHeader(output, layout.header)
	}
	footerBlock := strings.Join(footerLines, "\n")
	return view.RenderViewSections([]string{output, footerBlock}, separatorText)
}

func installSeparatorDetails(separator installSectionSeparator) (string, int) {
	switch separator {
	case installSeparatorParagraph:
		return view.SectionSeparatorParagraph, 1
	default:
		return view.SectionSeparatorLine, 0
	}
}

func renderInstallViewWithoutViewport(layout installViewLayout, separatorText string) string {
	lines := append([]string{layout.header}, layout.listLines...)
	content := strings.Join(lines, "\n")
	if len(layout.footerLines) == 0 {
		return content
	}
	footerBlock := strings.Join(layout.footerLines, "\n")
	return view.RenderViewSections([]string{content, footerBlock}, separatorText)
}

func (model *installModel) footerLinesWithHeaderIfNeeded(layout installViewLayout, separatorExtraLines int, listHeight int) []string {
	if strings.TrimSpace(layout.header) == "" {
		return layout.footerLines
	}
	if model.windowH <= 0 {
		if len(layout.footerLines) == 0 {
			return layout.footerLines
		}
		return append([]string{layout.header}, layout.footerLines...)
	}
	contentHeight := lipgloss.Height(layout.header) + view.ClampViewportHeight(listHeight)
	footerHeight := lipgloss.Height(strings.Join(layout.footerLines, "\n"))
	totalHeight := contentHeight
	if footerHeight > 0 {
		totalHeight += footerHeight + separatorExtraLines
	}
	if totalHeight <= model.windowH {
		return layout.footerLines
	}
	return append([]string{layout.header}, layout.footerLines...)
}

func (model *installModel) renderInstallViewWithStickyHeader(output string, header string) string {
	if strings.TrimSpace(header) == "" {
		return output
	}
	if model.windowH > 0 && lipgloss.Height(output) <= model.windowH {
		return output
	}
	return view.RenderViewSections([]string{output, header}, view.SectionSeparatorParagraph)
}

func (model *installModel) updateViewport(listText string, height int) {
	model.viewport.SetContent(listText)
	contentHeight := lipgloss.Height(listText)
	viewportHeight := view.ClampViewportHeight(height)
	model.viewport.Height = viewportHeight
	if model.windowW > 0 {
		model.viewport.Width = model.windowW
	} else if model.viewport.Width == 0 {
		contentWidth := lipgloss.Width(listText)
		if contentWidth > 0 {
			model.viewport.Width = contentWidth
		}
	}

	maxOffset := view.MaxViewportOffset(contentHeight, viewportHeight)
	targetOffset := model.viewport.YOffset
	if !model.userScroll {
		targetOffset = maxOffset
	}
	model.viewport.SetYOffset(targetOffset)
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

func (model *installModel) runningFooterLine() string {
	if model.footer == nil || model.footer.Render == nil {
		return ""
	}
	return model.footer.Render(RunningFooterInput{
		SpinnerFrame: model.spinner.Frame(),
		ColorMode:    model.colorMode,
	})
}

func (model *installModel) updateItem(key string, update func(*installItem)) {
	index, ok := model.indexByKey[key]
	if !ok {
		return
	}
	item := model.items[index]
	update(&item)
	model.items[index] = item
}

func (model *installModel) applyProgress(msg installItemProgressMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemDownloading
		if item.Progress == nil {
			item.Progress = &installProgress{}
		}
		item.Progress.ratio = msg.progress.Ratio
		item.Progress.downloaded = msg.progress.Downloaded
		item.Progress.total = msg.progress.Total
	})
}

func (model *installModel) applyProgressError(msg installItemProgressErrMsg) {
	if msg.err == nil {
		return
	}
	model.updateItem(msg.key, func(item *installItem) {
		if strings.TrimSpace(item.FailureReason) == "" {
			item.FailureReason = msg.err.Error()
		}
	})
}

func (model *installModel) applySuccess(msg installItemSuccessMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemSuccess
		item.Progress = nil
		item.FailureReason = ""
		if strings.TrimSpace(msg.displayName) != "" {
			item.DisplayName = msg.displayName
		}
	})
}

func (model *installModel) applyFailure(msg installItemFailureMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemFailed
		item.Progress = nil
		item.FailureReason = msg.reason
	})
}

func (model *installModel) applyAborted(msg installItemAbortedMsg) {
	model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemAborted
		item.Progress = nil
		item.FailureReason = ""
	})
}

func viewStateFromOutcome(errType installExecutionErrorType) installViewState {
	switch errType {
	case installExecutionErrorDownload:
		return installViewDownloadFailed
	case installExecutionErrorWriteLock:
		return installViewWriteLockFailed
	case installExecutionErrorWriteConfig:
		return installViewWriteConfigFailed
	case installExecutionErrorCanceled:
		return installViewCanceled
	case installExecutionErrorUnknown:
		return installViewFailed
	default:
		return installViewSuccess
	}
}

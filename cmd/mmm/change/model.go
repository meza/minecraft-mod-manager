package change

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type changeStage int

const (
	changeStageRunning changeStage = iota
	changeStageCompatibilityFailureDetected
	changeStageSwitching
	changeStageSuccess
	changeStageCompatibilityFailed
	changeStageDownloadFailed
	changeStageSwitchFailed
	changeStageNoop
)

type changeCompatStatus int

const (
	changeCompatPending changeCompatStatus = iota
	changeCompatChecking
	changeCompatSupported
	changeCompatUnsupported
)

type changeDownloadStatus int

const (
	changeDownloadPending changeDownloadStatus = iota
	changeDownloadQueued
	changeDownloadInProgress
	changeDownloadSucceeded
	changeDownloadFailed
)

type changeSwitchStatus int

const (
	changeSwitchPending changeSwitchStatus = iota
	changeSwitchInProgress
	changeSwitchSucceeded
	changeSwitchFailed
	changeSwitchSkipped
)

type changeDownloadProgress struct {
	ratio      float64
	downloaded int64
	total      int64
}

type changeItem struct {
	Mod             models.Mod
	DisplayName     string
	CompatStatus    changeCompatStatus
	DownloadStatus  changeDownloadStatus
	SwitchStatus    changeSwitchStatus
	Download        changeDownloadProgress
	ErrorReason     string
	Skipped         bool
	ResolvedName    string
	ResolvedVersion string
}

type changeOutcome struct {
	Stage         changeStage
	TargetVersion string
	Items         []changeItem
	ForcePolicy   changeForcePolicy
	Err           error
}

type changeModel struct {
	ctx             context.Context
	cancel          context.CancelFunc
	sender          httpclient.Sender
	execRunner      func(context.Context, httpclient.Sender) changeOutcome
	colorMode       view.ColorMode
	stage           changeStage
	target          string
	items           []changeItem
	indexByKey      map[string]int
	spinner         view.Spinner
	viewport        viewport.Model
	windowW         int
	windowH         int
	finalRender     bool
	outcome         changeOutcome
	forcePolicy     changeForcePolicy
	policyPrompt    *changePolicyPromptModel
	policyAnswer    string
	policySelection chan<- changeForcePolicy
}

type viewportFocusMode int

const (
	viewportFocusPreserve viewportFocusMode = iota
	viewportFocusBottom
)

type changeModelInput struct {
	ctx             context.Context
	target          string
	colorMode       view.ColorMode
	items           []changeItem
	indexByKey      map[string]int
	execRunner      func(context.Context, httpclient.Sender) changeOutcome
	forcePolicy     changeForcePolicy
	policySelection chan<- changeForcePolicy
}

type changeExecMessage struct {
	outcome changeOutcome
}

type changeFinalizeMsg struct{}
type changeCompatResultMsg struct {
	key          string
	supported    bool
	skipped      bool
	resolvedName string
}

type changeCompatCheckingMsg struct {
	key string
}

type changeCompatFailureMsg struct{}

type changeDownloadProgressMsg struct {
	key      string
	progress httpclient.DownloadProgressMsg
}

type changeDownloadProgressErrMsg struct {
	key string
	err error
}

type changeDownloadFinishedMsg struct {
	key          string
	resolvedName string
}

type changeDownloadFailedMsg struct {
	key string
	err error
}

type changeSwitchingStartedMsg struct{}

type changeSwitchStartedMsg struct {
	key string
}

type changeSwitchFinishedMsg struct {
	key string
}

type changeSwitchFailedMsg struct {
	key string
	err error
}

type changeSwitchSkippedMsg struct {
	key string
}

type changePolicyPromptMsg struct{}

type changePolicySelectedMsg struct {
	policy changeForcePolicy
}

func newChangeModel(input changeModelInput) *changeModel {
	ctx, cancel := context.WithCancel(input.ctx)
	spin := view.NewSpinner()

	model := &changeModel{
		ctx:             ctx,
		cancel:          cancel,
		execRunner:      input.execRunner,
		colorMode:       input.colorMode,
		stage:           changeStageRunning,
		target:          input.target,
		items:           input.items,
		indexByKey:      input.indexByKey,
		spinner:         spin,
		viewport:        viewport.New(0, 0),
		forcePolicy:     input.forcePolicy,
		policySelection: input.policySelection,
	}
	model.viewport.MouseWheelEnabled = true
	model.outcome = changeOutcome{Stage: changeStageRunning, TargetVersion: input.target, Items: input.items, ForcePolicy: input.forcePolicy}
	return model
}

type changeExecSender struct {
	send func(tea.Msg)
}

func (sender changeExecSender) Send(msg tea.Msg) {
	if sender.send == nil {
		return
	}
	sender.send(msg)
}

func (model *changeModel) bindSender(sender func(tea.Msg)) {
	model.sender = changeExecSender{send: sender}
}

func (model *changeModel) Init() tea.Cmd {
	if model.sender == nil || model.execRunner == nil {
		return tea.Quit
	}
	return tea.Batch(model.spinner.InitCmd(), model.startChangeCmd())
}

func (model *changeModel) startChangeCmd() tea.Cmd {
	return func() tea.Msg {
		if model.sender == nil {
			return changeExecMessage{outcome: changeOutcome{
				Stage:       changeStageDownloadFailed,
				ForcePolicy: model.forcePolicy,
				Err:         errors.New("missing bubble tea sender"),
			}}
		}
		return changeExecMessage{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *changeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := model.updatePolicyPromptMessage(msg); handled {
		return model, cmd
	}
	if cmd, handled := model.updateViewportMessage(msg); handled {
		return model, cmd
	}
	return model.updateChangeMessage(msg)
}

func (model *changeModel) updatePolicyPromptMessage(msg tea.Msg) (tea.Cmd, bool) {
	if model.policyPrompt == nil {
		return nil, false
	}

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.windowW = typed.Width
		model.windowH = typed.Height
		model.policyPrompt.SetWidth(typed.Width)
		return nil, true
	case tea.KeyMsg, tea.MouseMsg:
		updated, cmd := model.policyPrompt.Update(msg)
		model.policyPrompt = &updated
		return cmd, true
	default:
		return nil, false
	}
}

func (model *changeModel) updateViewportMessage(msg tea.Msg) (tea.Cmd, bool) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.windowW = typed.Width
		model.windowH = typed.Height
		return nil, true
	case tea.KeyMsg:
		updated, cmd := model.viewport.Update(typed)
		model.viewport = updated
		return cmd, true
	case tea.MouseMsg:
		updated, cmd := model.viewport.Update(typed)
		model.viewport = updated
		return cmd, true
	default:
		return nil, false
	}
}

func (model *changeModel) updateChangeMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	if handled := model.updateCompatMessage(msg); handled {
		return model, nil
	}
	if handled := model.updateDownloadMessage(msg); handled {
		return model, nil
	}
	if handled := model.updateSwitchMessage(msg); handled {
		return model, nil
	}
	if updated, cmd, handled := model.spinner.Update(msg); handled {
		model.spinner = updated
		return model, cmd
	}

	switch typed := msg.(type) {
	case changePolicyPromptMsg:
		model.policyPrompt = newChangePolicyPromptModel()
		if model.windowW > 0 {
			model.policyPrompt.SetWidth(model.windowW)
		}
		return model, nil
	case changePolicySelectedMsg:
		model.forcePolicy = typed.policy
		model.policyAnswer = forcePolicyAnswerLine(typed.policy)
		if model.policySelection != nil {
			model.policySelection <- typed.policy
		}
		model.policyPrompt = nil
		return model, nil
	case changeExecMessage:
		model.outcome = typed.outcome
		model.stage = typed.outcome.Stage
		model.items = typed.outcome.Items
		model.forcePolicy = typed.outcome.ForcePolicy
		model.finalRender = true
		return model, func() tea.Msg { return changeFinalizeMsg{} }
	case changeFinalizeMsg:
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *changeModel) updateCompatMessage(msg tea.Msg) bool {
	switch typed := msg.(type) {
	case changeCompatResultMsg:
		model.applyCompatResult(typed)
		return true
	case changeCompatCheckingMsg:
		model.applyCompatChecking(typed)
		return true
	case changeCompatFailureMsg:
		if model.stage == changeStageRunning || model.stage == changeStageCompatibilityFailureDetected {
			model.stage = changeStageCompatibilityFailureDetected
		}
		return true
	default:
		return false
	}
}

func (model *changeModel) updateDownloadMessage(msg tea.Msg) bool {
	switch typed := msg.(type) {
	case changeDownloadProgressMsg:
		model.applyDownloadProgress(typed)
		return true
	case changeDownloadProgressErrMsg:
		model.applyDownloadError(typed.key, typed.err)
		return true
	case changeDownloadFinishedMsg:
		model.applyDownloadFinished(typed)
		return true
	case changeDownloadFailedMsg:
		model.applyDownloadFailed(typed)
		return true
	default:
		return false
	}
}

func (model *changeModel) updateSwitchMessage(msg tea.Msg) bool {
	switch typed := msg.(type) {
	case changeSwitchingStartedMsg:
		model.stage = changeStageSwitching
		return true
	case changeSwitchStartedMsg:
		model.applySwitchStarted(typed)
		return true
	case changeSwitchFinishedMsg:
		model.applySwitchFinished(typed)
		return true
	case changeSwitchFailedMsg:
		model.applySwitchFailed(typed)
		return true
	case changeSwitchSkippedMsg:
		model.applySwitchSkipped(typed)
		return true
	default:
		return false
	}
}

func (model *changeModel) View() string {
	sections := buildChangeSections(changeViewInput{
		stage:            model.stage,
		target:           model.target,
		items:            model.items,
		colorMode:        model.colorMode,
		spinner:          &model.spinner,
		forcePolicy:      model.forcePolicy,
		waitingForPolicy: model.policyPrompt != nil,
	})
	if model.policyPrompt != nil {
		sections = append(sections, model.policyPrompt.View())
	} else if model.policyAnswer != "" {
		sections = append(sections, model.policyAnswer)
	}
	content := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	if model.finalRender {
		return content
	}
	focusMode := viewportFocusPreserve
	if model.policyPrompt != nil {
		focusMode = viewportFocusBottom
	}
	model.updateViewport(content, focusMode)
	return model.viewport.View()
}

func (model *changeModel) updateViewport(content string, focusMode viewportFocusMode) {
	model.viewport.SetContent(content)

	contentHeight := lipgloss.Height(content)
	height := view.ViewportHeightOrContent(model.windowH, contentHeight)

	model.viewport.Height = height
	if model.windowW > 0 {
		model.viewport.Width = model.windowW
	}

	maxOffset := view.MaxViewportOffset(contentHeight, height)
	if focusMode == viewportFocusBottom {
		model.viewport.SetYOffset(maxOffset)
		return
	}
	model.viewport.SetYOffset(model.viewport.YOffset)
}

func (model *changeModel) applyCompatResult(msg changeCompatResultMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	if msg.supported {
		item.CompatStatus = changeCompatSupported
		if item.DownloadStatus == changeDownloadPending {
			item.DownloadStatus = changeDownloadQueued
		}
	} else {
		item.CompatStatus = changeCompatUnsupported
		item.Skipped = msg.skipped
	}
	if strings.TrimSpace(msg.resolvedName) != "" {
		item.DisplayName = msg.resolvedName
	}
	model.items[index] = item
}

func (model *changeModel) applyCompatChecking(msg changeCompatCheckingMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	item.CompatStatus = changeCompatChecking
	model.items[index] = item
}

func (model *changeModel) applyDownloadProgress(msg changeDownloadProgressMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	item.DownloadStatus = changeDownloadInProgress
	item.Download = changeDownloadProgress{
		ratio:      msg.progress.Ratio,
		downloaded: msg.progress.Downloaded,
		total:      msg.progress.Total,
	}
	model.items[index] = item
}

func (model *changeModel) applyDownloadError(key string, err error) {
	index, ok := model.indexByKey[key]
	if !ok {
		return
	}
	item := model.items[index]
	if item.ErrorReason == "" && err != nil {
		item.ErrorReason = err.Error()
	}
	model.items[index] = item
}

func (model *changeModel) applyDownloadFinished(msg changeDownloadFinishedMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	item.DownloadStatus = changeDownloadSucceeded
	if strings.TrimSpace(msg.resolvedName) != "" {
		item.DisplayName = msg.resolvedName
	}
	model.items[index] = item
}

func (model *changeModel) applyDownloadFailed(msg changeDownloadFailedMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	item.DownloadStatus = changeDownloadFailed
	if msg.err != nil {
		item.ErrorReason = msg.err.Error()
	}
	model.items[index] = item
}

func (model *changeModel) applySwitchStarted(msg changeSwitchStartedMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	item.SwitchStatus = changeSwitchInProgress
	model.items[index] = item
}

func (model *changeModel) applySwitchFinished(msg changeSwitchFinishedMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	if item.SwitchStatus == changeSwitchSkipped {
		model.items[index] = item
		return
	}
	item.SwitchStatus = changeSwitchSucceeded
	model.items[index] = item
}

func (model *changeModel) applySwitchFailed(msg changeSwitchFailedMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	item.SwitchStatus = changeSwitchFailed
	if msg.err != nil {
		item.ErrorReason = msg.err.Error()
	}
	model.items[index] = item
}

func (model *changeModel) applySwitchSkipped(msg changeSwitchSkippedMsg) {
	index, ok := model.indexByKey[msg.key]
	if !ok {
		return
	}
	item := model.items[index]
	item.SwitchStatus = changeSwitchSkipped
	model.items[index] = item
}

type changeViewInput struct {
	stage            changeStage
	target           string
	items            []changeItem
	colorMode        view.ColorMode
	spinner          *view.Spinner
	forcePolicy      changeForcePolicy
	waitingForPolicy bool
}

func buildChangeSections(input changeViewInput) []string {
	partition := partitionChangeItems(input.items)
	switch input.stage {
	case changeStageNoop:
		return []string{renderNoopLine(input)}
	case changeStageSuccess:
		return renderSuccessSections(input)
	case changeStageCompatibilityFailureDetected:
		return renderCompatibilityFailureRunningSections(input)
	case changeStageCompatibilityFailed:
		return renderCompatibilityErrorSections(input)
	case changeStageDownloadFailed:
		return renderDownloadErrorSections(input, partition)
	case changeStageSwitchFailed:
		return renderSwitchErrorSections(input, partition)
	case changeStageSwitching:
		return renderSwitchingSections(input, partition)
	default:
		return renderRunningSections(input, partition)
	}
}

func renderRunningSections(input changeViewInput, partition changePartition) []string {
	sections := []string{
		renderChangeHeader(input),
		renderCompatibilitySection(input, partition.compatibility),
		renderDownloadSection(input, partition.downloading),
		renderSwitchingSection(input, partition.switching),
	}
	return pruneEmptySections(sections)
}

func renderSwitchingSections(input changeViewInput, partition changePartition) []string {
	return pruneEmptySections([]string{renderSwitchingSection(input, partition.switching)})
}

func renderCompatibilityFailureRunningSections(input changeViewInput) []string {
	filtered := filterCompatibilityFailureRunningItems(input.items)
	downloadHeader := renderDownloadCancelledHeader(input)
	switchingHeader := renderSwitchingCancelledHeader(input)
	sections := []string{
		renderChangeHeader(input),
		renderCompatibilitySection(input, filtered),
		downloadHeader,
		switchingHeader,
	}
	return pruneEmptySections(sections)
}

func renderCompatibilityErrorSections(input changeViewInput) []string {
	filtered := filterCompatibilityFailureFinalItems(input.items)
	sections := []string{
		renderChangeHeader(input),
		renderCompatibilitySection(input, filtered),
		renderCompatibilityErrorFooter(input),
	}
	return pruneEmptySections(sections)
}

func renderDownloadErrorSections(input changeViewInput, partition changePartition) []string {
	sections := []string{
		renderChangeHeader(input),
		renderDownloadSection(input, partition.downloading),
		renderSwitchingSection(input, partition.switching),
		renderDownloadErrorFooter(input),
	}
	return pruneEmptySections(sections)
}

func renderSwitchErrorSections(input changeViewInput, partition changePartition) []string {
	sections := []string{
		renderSwitchingSection(input, partition.switching),
		renderSwitchErrorFooter(input),
	}
	return pruneEmptySections(sections)
}

func renderSuccessSections(input changeViewInput) []string {
	sections := []string{
		renderSuccessSection(input),
		renderNowTargetingLine(input),
	}
	if skipped := renderSkippedSection(input); skipped != "" {
		sections = append(sections, skipped)
	}
	return pruneEmptySections(sections)
}

func renderNoopLine(input changeViewInput) string {
	return view.RenderModItemLine(view.ModItemLine{
		Label:  i18n.T("cmd.change.noop", &i18n.Tvars{Data: &i18n.TData{"version": input.target}}),
		Status: view.ModItemStatusSuccess,
	}, input.colorMode)
}

func renderChangeHeader(input changeViewInput) string {
	title := i18n.T("cmd.change.header", &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
	notice := i18n.T("cmd.change.notice", nil)
	return title + "\n" + notice
}

func renderCompatibilitySection(input changeViewInput, sectionItems []changeItem) string {
	if len(sectionItems) == 0 {
		return ""
	}
	lines := []string{i18n.T("cmd.compatibility.section", nil)}
	for _, item := range sectionItems {
		lines = append(lines, renderCompatibilityLine(input, item))
	}
	return strings.Join(lines, "\n")
}

func renderCompatibilityLine(input changeViewInput, item changeItem) string {
	label := renderModLabel(input, item)
	suffix := ""
	status := view.ModItemStatusPending
	switch item.CompatStatus {
	case changeCompatChecking:
		status = view.ModItemStatusSpinning
	case changeCompatSupported:
		status = view.ModItemStatusSuccess
		if item.DownloadStatus == changeDownloadQueued {
			status = view.ModItemStatusPending
		}
	case changeCompatUnsupported:
		status = view.ModItemStatusError
		suffixKey := "cmd.change.item.unsupported"
		if item.Skipped {
			suffixKey = "cmd.change.item.unsupported_skipped"
		}
		suffix = i18n.T(suffixKey, &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
	}
	return view.RenderModItemLine(view.ModItemLine{
		Label:   label,
		Suffix:  suffix,
		Status:  status,
		Spinner: input.spinner,
	}, input.colorMode)
}

func renderDownloadSection(input changeViewInput, sectionItems []changeItem) string {
	if len(sectionItems) == 0 {
		return ""
	}
	lines := []string{i18n.T("cmd.change.section.downloading", nil)}
	for _, item := range sectionItems {
		lines = append(lines, renderDownloadLine(input, item))
	}
	return strings.Join(lines, "\n")
}

func renderDownloadLine(input changeViewInput, item changeItem) string {
	label := renderModLabel(input, item)
	suffix := ""
	status := view.ModItemStatusPending
	var progress *view.ProgressDetails

	if item.CompatStatus == changeCompatUnsupported {
		status = view.ModItemStatusError
		if item.Skipped {
			suffix = i18n.T("cmd.change.item.unsupported_skipped", &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
		} else {
			suffix = i18n.T("cmd.change.item.unsupported", &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
		}
		return view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Suffix: suffix,
			Status: status,
		}, input.colorMode)
	}

	switch item.DownloadStatus {
	case changeDownloadQueued:
		status = view.ModItemStatusSpinning
	case changeDownloadInProgress:
		status = view.ModItemStatusDownloading
		progress = &view.ProgressDetails{
			Bar:        view.NewProgressBar(),
			Ratio:      item.Download.ratio,
			Downloaded: item.Download.downloaded,
			Total:      item.Download.total,
		}
	case changeDownloadSucceeded:
		status = view.ModItemStatusSuccess
	case changeDownloadFailed:
		status = view.ModItemStatusError
		if strings.TrimSpace(item.ErrorReason) != "" {
			suffix = i18n.T("cmd.download.item.failed", &i18n.Tvars{
				Data: &i18n.TData{"reason": item.ErrorReason},
			})
		}
	}

	return view.RenderModItemLine(view.ModItemLine{
		Label:    label,
		Suffix:   suffix,
		Status:   status,
		Progress: progress,
	}, input.colorMode)
}

func renderSwitchingSection(input changeViewInput, sectionItems []changeItem) string {
	if len(sectionItems) == 0 {
		return ""
	}
	title := i18n.T("cmd.change.section.switching", nil)
	if input.stage == changeStageRunning {
		waitingKey := "cmd.change.section.switching_waiting"
		if input.waitingForPolicy {
			waitingKey = "cmd.change.section.switching_waiting_choice"
		}
		spinnerFrame := ""
		if input.spinner != nil {
			spinnerFrame = input.spinner.Frame()
		}
		waiting := i18n.T(waitingKey, &i18n.Tvars{
			Data: &i18n.TData{"spinner": spinnerFrame},
		})
		title = title + " " + view.RenderIfColorEnabled(input.colorMode, view.ParenStyle, waiting)
	}
	lines := []string{title}
	for _, item := range sectionItems {
		lines = append(lines, renderSwitchingLine(input, item))
	}
	return strings.Join(lines, "\n")
}

func renderDownloadCancelledHeader(input changeViewInput) string {
	title := i18n.T("cmd.change.section.downloading", nil)
	cancelLabel := i18n.T("cmd.change.section.cancelled_incompatibility", nil)
	return title + " " + view.RenderIfColorEnabled(input.colorMode, view.ParenStyle, cancelLabel)
}

func renderSwitchingCancelledHeader(input changeViewInput) string {
	title := i18n.T("cmd.change.section.switching", nil)
	cancelLabel := i18n.T("cmd.change.section.cancelled_incompatibility", nil)
	return title + " " + view.RenderIfColorEnabled(input.colorMode, view.ParenStyle, cancelLabel)
}

func renderSwitchingLine(input changeViewInput, item changeItem) string {
	label := renderModLabel(input, item)
	suffix := ""
	status := view.ModItemStatusPending

	switch item.SwitchStatus {
	case changeSwitchInProgress:
		status = view.ModItemStatusSpinning
	case changeSwitchSucceeded:
		status = view.ModItemStatusSuccess
	case changeSwitchFailed:
		status = view.ModItemStatusError
		if strings.TrimSpace(item.ErrorReason) != "" {
			suffix = i18n.T("cmd.change.item.switch_failed", &i18n.Tvars{
				Data: &i18n.TData{"reason": item.ErrorReason},
			})
		}
	case changeSwitchSkipped:
		status = view.ModItemStatusError
		suffix = i18n.T(skippedItemSuffixKey(input.forcePolicy), &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
	}

	return view.RenderModItemLine(view.ModItemLine{
		Label:   label,
		Suffix:  suffix,
		Status:  status,
		Spinner: input.spinner,
	}, input.colorMode)
}

type changePartition struct {
	compatibility []changeItem
	downloading   []changeItem
	switching     []changeItem
}

func partitionChangeItems(items []changeItem) changePartition {
	partition := changePartition{
		compatibility: make([]changeItem, 0, len(items)),
		downloading:   make([]changeItem, 0, len(items)),
		switching:     make([]changeItem, 0, len(items)),
	}
	for _, item := range items {
		if item.SwitchStatus != changeSwitchPending {
			partition.switching = append(partition.switching, item)
			continue
		}
		if item.CompatStatus != changeCompatSupported {
			partition.compatibility = append(partition.compatibility, item)
			continue
		}
		if item.DownloadStatus == changeDownloadSucceeded {
			partition.switching = append(partition.switching, item)
			continue
		}
		partition.downloading = append(partition.downloading, item)
	}
	return partition
}

func renderCompatibilityErrorFooter(input changeViewInput) string {
	headline := renderFinalErrorLine(input.colorMode, i18n.T("cmd.change.error.compatibility_failed", &i18n.Tvars{Data: &i18n.TData{"version": input.target}}))
	body := i18n.T("cmd.change.error.no_changes", nil)
	hint := i18n.T("cmd.change.error.compatibility_hint", &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
	return strings.Join([]string{headline, body, "", hint}, "\n")
}

func renderDownloadErrorFooter(input changeViewInput) string {
	headline := renderFinalErrorLine(input.colorMode, i18n.T("cmd.change.error.download_failed", nil))
	hint := i18n.T("cmd.change.error.download_hint", &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
	return strings.Join([]string{headline, "", hint}, "\n")
}

func renderSwitchErrorFooter(input changeViewInput) string {
	headline := renderFinalErrorLine(input.colorMode, i18n.T("cmd.change.error.switch_failed", nil))
	return headline
}

func renderSuccessSection(input changeViewInput) string {
	if len(input.items) == 0 {
		return ""
	}
	lines := make([]string, 0, len(input.items))
	for _, item := range input.items {
		lines = append(lines, renderSuccessLine(input, item))
	}
	return strings.Join(lines, "\n")
}

func renderSuccessLine(input changeViewInput, item changeItem) string {
	label := renderModLabel(input, item)
	suffix := ""
	status := view.ModItemStatusSuccess
	if item.Skipped {
		status = view.ModItemStatusError
		suffix = i18n.T(skippedItemSuffixKey(input.forcePolicy), &i18n.Tvars{Data: &i18n.TData{"version": input.target}})
	}
	return view.RenderModItemLine(view.ModItemLine{
		Label:  label,
		Suffix: suffix,
		Status: status,
	}, input.colorMode)
}

func renderNowTargetingLine(input changeViewInput) string {
	return view.RenderModItemLine(view.ModItemLine{
		Label:  i18n.T("cmd.change.success", &i18n.Tvars{Data: &i18n.TData{"version": input.target}}),
		Status: view.ModItemStatusSuccess,
	}, input.colorMode)
}

func renderSkippedSection(input changeViewInput) string {
	skipped := make([]changeItem, 0)
	for _, item := range input.items {
		if item.Skipped {
			skipped = append(skipped, item)
		}
	}
	if len(skipped) == 0 {
		return ""
	}
	lines := []string{i18n.T(skippedSectionHeader(input.forcePolicy), nil)}
	for _, item := range skipped {
		label := renderModLabel(input, item)
		lines = append(lines, view.RenderModItemLine(view.ModItemLine{
			Label:  label,
			Status: view.ModItemStatusError,
		}, input.colorMode))
	}
	return strings.Join(lines, "\n")
}

func renderModLabel(input changeViewInput, item changeItem) string {
	name := item.DisplayName
	if strings.TrimSpace(name) == "" {
		name = item.Mod.ID
	}
	return view.RenderModLabel(input.colorMode, name, item.Mod.ID, string(item.Mod.Type))
}

func filterCompatibilityFailureRunningItems(items []changeItem) []changeItem {
	filtered := make([]changeItem, 0, len(items))
	for _, item := range items {
		if item.CompatStatus == changeCompatSupported {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterCompatibilityFailureFinalItems(items []changeItem) []changeItem {
	filtered := make([]changeItem, 0, len(items))
	for _, item := range items {
		if item.CompatStatus != changeCompatUnsupported {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func pruneEmptySections(sections []string) []string {
	filtered := make([]string, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		filtered = append(filtered, section)
	}
	return filtered
}

func renderFinalErrorLine(colorMode view.ColorMode, message string) string {
	line := fmt.Sprintf("%s %s", view.FinalErrorIcon(colorMode), message)
	if colorMode.Enabled() {
		return view.ErrorStyle.Render(line)
	}
	return line
}

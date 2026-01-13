package remove

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type removeItemSuccessMsg struct {
	key string
}

type removeItemFailureMsg struct {
	key    string
	reason string
}

type removeExecutionFinishedMsg struct {
	outcome removeExecutionOutcome
}

type removeFinalizeMsg struct{}

type removeModel struct {
	ctx        context.Context
	execRunner func(context.Context, removeExecSender) removeExecutionOutcome
	colorMode  view.ColorMode
	items      []removeItem
	indexByKey map[string]int
	spinner    view.Spinner
	outcome    removeExecutionOutcome
	done       bool
	sender     removeExecSender
}

func newRemoveModel(
	ctx context.Context,
	colorMode view.ColorMode,
	items []removeItem,
	indexByKey map[string]int,
	execRunner func(context.Context, removeExecSender) removeExecutionOutcome,
) *removeModel {
	spin := view.NewSpinner()

	return &removeModel{
		ctx:        ctx,
		execRunner: execRunner,
		colorMode:  colorMode,
		items:      items,
		indexByKey: indexByKey,
		spinner:    spin,
	}
}

func (model *removeModel) bindSender(send func(tea.Msg)) {
	model.sender = removeExecSender{send: send}
}

func (model *removeModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return tea.Batch(model.spinner.InitCmd(), model.startRemoveCmd())
}

func (model *removeModel) startRemoveCmd() tea.Cmd {
	return func() tea.Msg {
		return removeExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *removeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updated, cmd, handled := model.spinner.Update(msg); handled {
		model.spinner = updated
		return model, cmd
	}
	switch typed := msg.(type) {
	case removeItemSuccessMsg:
		model.updateItem(typed.key, func(item *removeItem) {
			item.Status = removeItemSuccess
			item.FailureReason = ""
		})
		return model, nil
	case removeItemFailureMsg:
		model.updateItem(typed.key, func(item *removeItem) {
			item.Status = removeItemFailed
			item.FailureReason = typed.reason
		})
		return model, nil
	case removeExecutionFinishedMsg:
		model.outcome = typed.outcome
		model.items = typed.outcome.items
		model.done = true
		return model, func() tea.Msg { return removeFinalizeMsg{} }
	case removeFinalizeMsg:
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *removeModel) View() string {
	if model.done {
		return model.renderFinalView()
	}
	return renderRemoveRunningSection(model.colorMode, model.items, &model.spinner)
}

func (model *removeModel) updateItem(key string, update func(*removeItem)) {
	index, ok := model.indexByKey[key]
	if !ok {
		return
	}
	item := model.items[index]
	update(&item)
	model.items[index] = item
}

func (model *removeModel) renderFinalView() string {
	sections := []string{renderRemoveResultSection(model.colorMode, model.items)}
	switch model.outcome.errType {
	case removeExecutionErrorDelete:
		sections = append(sections, renderRemoveFailureSummary(model.colorMode))
	case removeExecutionErrorWriteConfig, removeExecutionErrorWriteLock, removeExecutionErrorUnknown:
		if model.outcome.err != nil {
			sections = append(sections, renderRemoveFailureLine(model.colorMode, model.outcome.err))
		}
	default:
		sections = append(sections, renderRemoveSuccessSummary(model.colorMode))
	}

	return view.RenderViewSectionsWithTrailingNewline(sections, view.SectionSeparatorParagraph)
}

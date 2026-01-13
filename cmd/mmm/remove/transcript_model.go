package remove

import (
	"context"
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type removeTranscriptModel struct {
	ctx        context.Context
	execRunner func(context.Context, removeExecSender) removeExecutionOutcome
	colorMode  view.ColorMode
	items      []removeItem
	indexByKey map[string]int
	outcome    removeExecutionOutcome
	finished   bool
	output     io.Writer
	sender     removeExecSender
}

func newRemoveTranscriptModel(
	ctx context.Context,
	colorMode view.ColorMode,
	items []removeItem,
	indexByKey map[string]int,
	output io.Writer,
	execRunner func(context.Context, removeExecSender) removeExecutionOutcome,
) *removeTranscriptModel {
	if output != nil {
		output = newLockedWriter(output)
	}
	return &removeTranscriptModel{
		ctx:        ctx,
		execRunner: execRunner,
		colorMode:  colorMode,
		items:      items,
		indexByKey: indexByKey,
		output:     output,
	}
}

func (model *removeTranscriptModel) bindSender(send func(tea.Msg)) {
	model.sender = removeExecSender{send: send}
}

func (model *removeTranscriptModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return model.startRemoveCmd()
}

func (model *removeTranscriptModel) startRemoveCmd() tea.Cmd {
	return func() tea.Msg {
		return removeExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *removeTranscriptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case removeItemSuccessMsg:
		if model.finished {
			return model, nil
		}
		if outputLine, ok := model.applyTranscriptSuccess(typed); ok {
			if err := writeOutputLine(model.output, outputLine); err != nil {
				model.outcome = removeExecutionOutcome{err: err, errType: removeExecutionErrorUnknown}
				return model, tea.Quit
			}
		}
		return model, nil
	case removeItemFailureMsg:
		if model.finished {
			return model, nil
		}
		if outputLine, ok := model.applyTranscriptFailure(typed); ok {
			if err := writeOutputLine(model.output, outputLine); err != nil {
				model.outcome = removeExecutionOutcome{err: err, errType: removeExecutionErrorUnknown}
				return model, tea.Quit
			}
		}
		return model, nil
	case removeExecutionFinishedMsg:
		model.finished = true
		model.outcome = typed.outcome
		missingLines := missingTranscriptLines(model.colorMode, model.items, typed.outcome.items)
		model.items = typed.outcome.items
		lines := append(missingLines, model.summaryLines()...)
		return model, tea.Sequence(summaryLinesCmd(model.output, lines), tea.Quit)
	case outputLineErrorMsg:
		model.outcome = removeExecutionOutcome{err: typed.Err, errType: removeExecutionErrorUnknown}
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *removeTranscriptModel) View() string {
	return ""
}

func (model *removeTranscriptModel) updateItem(key string, update func(*removeItem)) (removeItemStatus, bool) {
	index, ok := model.indexByKey[key]
	if !ok {
		return removeItemPending, false
	}
	item := model.items[index]
	previousStatus := item.Status
	update(&item)
	model.items[index] = item
	return previousStatus, true
}

func (model *removeTranscriptModel) applyTranscriptSuccess(msg removeItemSuccessMsg) (string, bool) {
	previousStatus, ok := model.updateItem(msg.key, func(item *removeItem) {
		item.Status = removeItemSuccess
		item.FailureReason = ""
	})
	if !ok || isTerminalRemoveStatus(previousStatus) {
		return "", false
	}
	return renderRemoveItemLine(model.colorMode, model.items[model.indexByKey[msg.key]], nil), true
}

func (model *removeTranscriptModel) applyTranscriptFailure(msg removeItemFailureMsg) (string, bool) {
	previousStatus, ok := model.updateItem(msg.key, func(item *removeItem) {
		item.Status = removeItemFailed
		item.FailureReason = msg.reason
	})
	if !ok || isTerminalRemoveStatus(previousStatus) {
		return "", false
	}
	return renderRemoveItemLine(model.colorMode, model.items[model.indexByKey[msg.key]], nil), true
}

func (model *removeTranscriptModel) summaryLines() []string {
	var lines []string
	switch model.outcome.errType {
	case removeExecutionErrorDelete:
		lines = []string{renderRemoveFailureSummary(model.colorMode)}
	case removeExecutionErrorWriteConfig, removeExecutionErrorWriteLock, removeExecutionErrorUnknown:
		if model.outcome.err != nil {
			lines = []string{renderRemoveFailureLine(model.colorMode, model.outcome.err)}
		}
	default:
		lines = []string{renderRemoveSuccessSummary(model.colorMode)}
	}
	if len(lines) == 0 || len(model.items) == 0 {
		return lines
	}
	return append([]string{""}, lines...)
}

func missingTranscriptLines(colorMode view.ColorMode, currentItems []removeItem, finalItems []removeItem) []string {
	if len(finalItems) == 0 {
		return nil
	}
	lines := make([]string, 0, len(finalItems))
	for index, item := range finalItems {
		if index < len(currentItems) && isTerminalRemoveStatus(currentItems[index].Status) {
			continue
		}
		lines = append(lines, renderRemoveItemLine(colorMode, item, nil))
	}
	return lines
}

func summaryLinesCmd(output io.Writer, lines []string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}
	commands := make([]tea.Cmd, 0, len(lines))
	for _, line := range lines {
		commands = append(commands, outputLineCmd(output, line))
	}
	return tea.Sequence(commands...)
}

func writeOutputLine(output io.Writer, line string) error {
	if output == nil {
		return errors.New("output writer is nil")
	}
	msg := outputLineCmd(output, line)()
	if errMsg, ok := msg.(outputLineErrorMsg); ok {
		return errMsg.Err
	}
	return nil
}

func isTerminalRemoveStatus(status removeItemStatus) bool {
	return status == removeItemSuccess || status == removeItemFailed
}

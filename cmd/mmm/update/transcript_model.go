package update

import (
	"context"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type updateTranscriptModel struct {
	ctx            context.Context
	execRunner     func(context.Context, updateExecSender) updateExecutionOutcome
	colorMode      view.ColorMode
	items          []updateItem
	indexByKey     map[int]int
	printedByIndex map[int]bool
	outcome        updateExecutionOutcome
	output         io.Writer
	sender         updateExecSender
}

func newUpdateTranscriptModel(
	ctx context.Context,
	colorMode view.ColorMode,
	items []updateItem,
	indexByKey map[int]int,
	output io.Writer,
	execRunner func(context.Context, updateExecSender) updateExecutionOutcome,
) *updateTranscriptModel {
	if output != nil {
		output = newLockedWriter(output)
	}
	return &updateTranscriptModel{
		ctx:            ctx,
		execRunner:     execRunner,
		colorMode:      colorMode,
		items:          cloneUpdateItems(items),
		indexByKey:     indexByKey,
		printedByIndex: make(map[int]bool, len(items)),
		output:         output,
	}
}

func (model *updateTranscriptModel) bindSender(send func(tea.Msg)) {
	model.sender = updateExecSender{send: send}
}

func (model *updateTranscriptModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return model.startUpdateCmd()
}

func (model *updateTranscriptModel) startUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		return updateExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *updateTranscriptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case updateItemStatusMsg:
		if outputLine, ok := model.applyTranscriptUpdate(typed); ok {
			return model, outputLineCmd(model.output, outputLine)
		}
		return model, nil
	case updateItemProgressMsg:
		return model, nil
	case updateExecutionFinishedMsg:
		model.items = typed.outcome.items
		model.outcome = typed.outcome
		return model, tea.Sequence(summaryLinesCmd(model.output, model.summaryLines()), tea.Quit)
	case outputLineErrorMsg:
		model.outcome = updateExecutionOutcome{err: typed.Err, errType: updateExecutionErrorUnknown}
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *updateTranscriptModel) View() string {
	return ""
}

func (model *updateTranscriptModel) applyTranscriptUpdate(msg updateItemStatusMsg) (string, bool) {
	_, ok := model.updateItem(msg.index, func(item *updateItem) {
		item.Status = msg.status
		item.FailReason = msg.failReason
		if strings.TrimSpace(msg.displayName) != "" {
			item.DisplayName = msg.displayName
		}
	})
	if !ok || !isTerminalUpdateStatus(msg.status) {
		return "", false
	}
	if model.printedByIndex[msg.index] {
		return "", false
	}
	model.printedByIndex[msg.index] = true
	return renderUpdateItemLine(updateRunningViewInput{
		items:     model.items,
		colorMode: model.colorMode,
	}, model.items[model.indexByKey[msg.index]]), true
}

func (model *updateTranscriptModel) updateItem(index int, update func(*updateItem)) (updateItemStatus, bool) {
	itemIndex, ok := model.indexByKey[index]
	if !ok {
		return updateItemStatusUpdating, false
	}
	item := model.items[itemIndex]
	previousStatus := item.Status
	update(&item)
	model.items[itemIndex] = item
	return previousStatus, true
}

func (model *updateTranscriptModel) summaryLines() []string {
	if model.outcome.errType == updateExecutionErrorCanceled {
		return nil
	}
	if model.outcome.errType == updateExecutionErrorWriteLock {
		return []string{renderUpdateWriteLockFailureView(updateErrorViewInput{
			colorMode: model.colorMode,
			lockPath:  model.outcome.lockPath,
		})}
	}
	if model.outcome.errType == updateExecutionErrorWriteConfig {
		return []string{renderUpdateWriteConfigFailureView(updateErrorViewInput{
			colorMode:  model.colorMode,
			configPath: model.outcome.configPath,
		})}
	}
	if model.outcome.errType == updateExecutionErrorUnknown {
		return []string{renderFinalErrorLine(model.colorMode, model.outcome.err.Error())}
	}

	lines := renderUpdateTranscriptSummary(model.colorMode, model.items)
	if hasTerminalItems(model.items) {
		return append([]string{""}, lines...)
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

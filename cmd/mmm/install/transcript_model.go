package install

import (
	"context"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type installTranscriptModel struct {
	ctx        context.Context
	execRunner func(context.Context, httpclient.Sender) installExecutionOutcome
	colorMode  view.ColorMode
	items      []installItem
	indexByKey map[string]int
	outcome    installExecutionOutcome
	output     io.Writer
	sender     installExecSender
}

func newInstallTranscriptModel(ctx context.Context, colorMode view.ColorMode, items []installItem, indexByKey map[string]int, output io.Writer, execRunner func(context.Context, httpclient.Sender) installExecutionOutcome) *installTranscriptModel {
	if output != nil {
		output = newLockedWriter(output)
	}
	return &installTranscriptModel{
		ctx:        ctx,
		execRunner: execRunner,
		colorMode:  colorMode,
		items:      items,
		indexByKey: indexByKey,
		output:     output,
	}
}

func (model *installTranscriptModel) bindSender(send func(tea.Msg)) {
	model.sender = installExecSender{send: send}
}

func (model *installTranscriptModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return model.startInstallCmd()
}

func (model *installTranscriptModel) startInstallCmd() tea.Cmd {
	return func() tea.Msg {
		return installExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *installTranscriptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case installItemSuccessMsg:
		if outputLine, ok := model.applyTranscriptSuccess(typed); ok {
			return model, outputLineCmd(model.output, outputLine)
		}
		return model, nil
	case installItemFailureMsg:
		if outputLine, ok := model.applyTranscriptFailure(typed); ok {
			return model, outputLineCmd(model.output, outputLine)
		}
		return model, nil
	case installItemAbortedMsg:
		if outputLine, ok := model.applyTranscriptAborted(typed); ok {
			return model, outputLineCmd(model.output, outputLine)
		}
		return model, nil
	case installExecutionFinishedMsg:
		model.outcome = typed.outcome
		return model, tea.Sequence(summaryLinesCmd(model.output, model.summaryLines()), tea.Quit)
	case outputLineErrorMsg:
		model.outcome = installExecutionOutcome{err: typed.Err, errType: installExecutionErrorUnknown}
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *installTranscriptModel) View() string {
	return ""
}

func (model *installTranscriptModel) updateItem(key string, update func(*installItem)) (installItemStatus, bool) {
	index, ok := model.indexByKey[key]
	if !ok {
		return installItemPending, false
	}
	item := model.items[index]
	previousStatus := item.Status
	update(&item)
	model.items[index] = item
	return previousStatus, true
}

func (model *installTranscriptModel) applyTranscriptSuccess(msg installItemSuccessMsg) (string, bool) {
	previousStatus, ok := model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemSuccess
		item.Progress = nil
		item.FailureReason = ""
		if strings.TrimSpace(msg.displayName) != "" {
			item.DisplayName = msg.displayName
		}
	})
	if !ok || isTerminalInstallStatus(previousStatus) {
		return "", false
	}
	return renderInstallItemLine(model.colorMode, model.items[model.indexByKey[msg.key]]), true
}

func (model *installTranscriptModel) applyTranscriptFailure(msg installItemFailureMsg) (string, bool) {
	previousStatus, ok := model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemFailed
		item.Progress = nil
		item.FailureReason = msg.reason
	})
	if !ok || isTerminalInstallStatus(previousStatus) {
		return "", false
	}
	return renderInstallItemLine(model.colorMode, model.items[model.indexByKey[msg.key]]), true
}

func (model *installTranscriptModel) applyTranscriptAborted(msg installItemAbortedMsg) (string, bool) {
	previousStatus, ok := model.updateItem(msg.key, func(item *installItem) {
		item.Status = installItemAborted
		item.Progress = nil
		item.FailureReason = ""
	})
	if !ok || isTerminalInstallStatus(previousStatus) {
		return "", false
	}
	return renderInstallItemLine(model.colorMode, model.items[model.indexByKey[msg.key]]), true
}

func (model *installTranscriptModel) summaryLines() []string {
	var lines []string
	switch model.outcome.errType {
	case installExecutionErrorDownload:
		lines = renderInstallDownloadFailureSummary(model.colorMode)
	case installExecutionErrorWriteLock:
		lines = renderInstallWriteLockSummary(model.colorMode, model.outcome.lockPath)
	case installExecutionErrorWriteConfig:
		lines = renderInstallWriteConfigSummary(model.colorMode, model.outcome.configPath)
	case installExecutionErrorCanceled:
		lines = renderInstallCanceledSummary(model.colorMode)
	case installExecutionErrorUnknown:
		lines = renderInstallExecutionFailureSummary(model.colorMode, model.outcome.err)
	default:
		lines = []string{renderInstallSuccessSummary(model.colorMode)}
	}
	if len(lines) == 0 || len(model.items) == 0 {
		return lines
	}
	return append([]string{""}, lines...)
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

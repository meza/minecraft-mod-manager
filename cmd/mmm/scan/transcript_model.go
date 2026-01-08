package scan

import (
	"context"
	"errors"
	"io"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type scanTranscriptModel struct {
	ctx        context.Context
	execRunner func(context.Context, scanExecSender) scanExecutionOutcome
	colorMode  view.ColorMode
	items      []scanItem
	indexByKey map[string]int
	output     io.Writer
	sender     scanExecSender
	outcome    scanExecutionOutcome
	finished   bool
	quiet      bool
	add        bool
}

func newScanTranscriptModel(
	ctx context.Context,
	colorMode view.ColorMode,
	items []scanItem,
	indexByKey map[string]int,
	output io.Writer,
	quiet bool,
	add bool,
	execRunner func(context.Context, scanExecSender) scanExecutionOutcome,
) *scanTranscriptModel {
	if output != nil {
		output = newLockedWriter(output)
	}
	return &scanTranscriptModel{
		ctx:        ctx,
		execRunner: execRunner,
		colorMode:  colorMode,
		items:      cloneScanItems(items),
		indexByKey: indexByKey,
		output:     output,
		quiet:      quiet,
		add:        add,
	}
}

func (model *scanTranscriptModel) bindSender(send func(tea.Msg)) {
	model.sender = scanExecSender{send: send}
}

func (model *scanTranscriptModel) Init() tea.Cmd {
	if model.sender.send == nil || model.execRunner == nil {
		return tea.Quit
	}
	return model.startScanCmd()
}

func (model *scanTranscriptModel) startScanCmd() tea.Cmd {
	return func() tea.Msg {
		return scanExecutionFinishedMsg{outcome: model.execRunner(model.ctx, model.sender)}
	}
}

func (model *scanTranscriptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case scanItemUpdateMsg:
		if model.finished || model.quietAdd() {
			return model, nil
		}
		outputLines := model.applyTranscriptUpdate(typed)
		for _, outputLine := range outputLines {
			if err := writeOutputLine(model.output, outputLine); err != nil {
				model.outcome = scanExecutionOutcome{err: err}
				return model, tea.Quit
			}
		}
		return model, nil
	case scanExecutionFinishedMsg:
		model.finished = true
		model.outcome = typed.outcome
		missingLines := missingScanTranscriptLines(model.colorMode, model.items, typed.outcome.items)
		if model.quiet {
			missingLines = missingScanTranscriptLinesQuiet(model.colorMode, model.items, typed.outcome.items)
		}
		model.items = typed.outcome.items

		lines := append(missingLines, model.finalTranscriptLines(missingLines)...)
		return model, tea.Sequence(summaryLinesCmd(model.output, lines), tea.Quit)
	case outputLineErrorMsg:
		model.outcome = scanExecutionOutcome{err: typed.Err}
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model *scanTranscriptModel) View() string {
	return ""
}

func (model *scanTranscriptModel) applyTranscriptUpdate(msg scanItemUpdateMsg) []string {
	previousStatus, ok := model.updateItem(msg)
	if !ok || isTerminalScanStatus(previousStatus) {
		return nil
	}
	item := model.items[model.indexByKey[msg.key]]
	if !model.shouldOutputStatus(item.Status) {
		return nil
	}
	line := renderScanTranscriptLine(model.colorMode, item)
	return []string{line}
}

func (model *scanTranscriptModel) updateItem(msg scanItemUpdateMsg) (scanItemStatus, bool) {
	index, ok := model.indexByKey[msg.key]
	if !ok || index < 0 || index >= len(model.items) {
		return scanItemStatusPending, false
	}
	item := model.items[index]
	previousStatus := item.Status
	item.Status = msg.status
	if msg.status == scanItemStatusRecognized {
		item.Match = msg.match
	}
	model.items[index] = item
	return previousStatus, true
}

func (model *scanTranscriptModel) shouldOutputStatus(status scanItemStatus) bool {
	if model.quiet {
		return status == scanItemStatusUnknown || status == scanItemStatusUnsure
	}
	return status == scanItemStatusRecognized || status == scanItemStatusUnknown || status == scanItemStatusUnsure
}

func (model *scanTranscriptModel) finalTranscriptLines(priorLines []string) []string {
	if model.quietAdd() {
		return nil
	}
	lines := []string{}
	if model.add && len(model.outcome.added) > 0 {
		section := renderScanAddedSection(scanAddedViewInput{
			added:     model.outcome.added,
			colorMode: model.colorMode,
		})
		if len(priorLines) > 0 {
			lines = append(lines, "", section)
		} else {
			lines = append(lines, section)
		}
	}
	if model.outcome.err != nil {
		return append(lines, renderScanFailureLine(model.colorMode, model.outcome.err))
	}
	return lines
}

func (model *scanTranscriptModel) quietAdd() bool {
	return model.quiet && model.add
}

func missingScanTranscriptLines(colorMode view.ColorMode, current []scanItem, final []scanItem) []string {
	return missingScanTranscriptLinesWithFilter(colorMode, current, final, func(item scanItem) bool {
		return isTerminalScanStatus(item.Status)
	})
}

func missingScanTranscriptLinesQuiet(colorMode view.ColorMode, current []scanItem, final []scanItem) []string {
	return missingScanTranscriptLinesWithFilter(colorMode, current, final, func(item scanItem) bool {
		return item.Status == scanItemStatusUnknown || item.Status == scanItemStatusUnsure
	})
}

func missingScanTranscriptLinesWithFilter(
	colorMode view.ColorMode,
	current []scanItem,
	final []scanItem,
	shouldInclude func(scanItem) bool,
) []string {
	if len(final) == 0 {
		return nil
	}
	lines := make([]string, 0, len(final))
	for index, item := range final {
		if index < len(current) && isTerminalScanStatus(current[index].Status) {
			continue
		}
		if !shouldInclude(item) {
			continue
		}
		if line := renderScanTranscriptLine(colorMode, item); line != "" {
			lines = append(lines, line)
		}
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

func isTerminalScanStatus(status scanItemStatus) bool {
	return status == scanItemStatusRecognized || status == scanItemStatusUnknown || status == scanItemStatusUnsure
}

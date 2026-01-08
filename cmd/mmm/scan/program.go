package scan

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

var runScanProgram = defaultRunScanProgram
var runScanTranscriptProgram = defaultRunScanTranscriptProgram
var runTeaProgram = defaultRunTeaProgram

func defaultRunScanProgram(model *scanModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func defaultRunScanTranscriptProgram(model *scanTranscriptModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func defaultRunTeaProgram(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

func scanOutcomeFromModel(result tea.Model) (scanExecutionOutcome, error) {
	switch typed := result.(type) {
	case *scanModel:
		return typed.outcome, nil
	case *scanTranscriptModel:
		return typed.outcome, nil
	default:
		return scanExecutionOutcome{}, errors.New("unexpected scan model")
	}
}

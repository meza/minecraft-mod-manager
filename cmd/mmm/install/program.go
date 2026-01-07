package install

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

var runInstallProgram = defaultRunInstallProgram
var runInstallTranscriptProgram = defaultRunInstallTranscriptProgram

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

func defaultRunInstallProgram(model *installModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func defaultRunInstallTranscriptProgram(model *installTranscriptModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func installOutcomeFromModel(result tea.Model) (installExecutionOutcome, error) {
	switch typed := result.(type) {
	case *installModel:
		return typed.outcome, nil
	case *installTranscriptModel:
		return typed.outcome, nil
	default:
		return installExecutionOutcome{}, errors.New("unexpected install model")
	}
}

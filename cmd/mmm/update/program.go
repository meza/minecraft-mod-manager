package update

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

var runUpdateProgram = defaultRunUpdateProgram
var runUpdateTranscriptProgram = defaultRunUpdateTranscriptProgram
var runTeaProgram = defaultRunTeaProgram

func defaultRunUpdateProgram(model *updateModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func defaultRunUpdateTranscriptProgram(model *updateTranscriptModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func defaultRunTeaProgram(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

func updateOutcomeFromModel(result tea.Model) (updateExecutionOutcome, error) {
	switch typed := result.(type) {
	case *updateModel:
		return typed.outcome, nil
	case *updateTranscriptModel:
		return typed.outcome, nil
	default:
		return updateExecutionOutcome{}, errors.New("unexpected update model")
	}
}

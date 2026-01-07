package remove

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

var runRemoveProgram = defaultRunRemoveProgram
var runRemoveTranscriptProgram = defaultRunRemoveTranscriptProgram

func defaultRunRemoveProgram(model *removeModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func defaultRunRemoveTranscriptProgram(model *removeTranscriptModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func removeOutcomeFromModel(result tea.Model) (removeExecutionOutcome, error) {
	switch typed := result.(type) {
	case *removeModel:
		return typed.outcome, nil
	case *removeTranscriptModel:
		return typed.outcome, nil
	default:
		return removeExecutionOutcome{}, errors.New("unexpected remove model")
	}
}

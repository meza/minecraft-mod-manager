package change

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

var runChangeProgram = defaultRunChangeProgram

func defaultRunChangeProgram(model *changeModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

func changeOutcomeFromModel(result tea.Model) (changeOutcome, error) {
	switch typed := result.(type) {
	case *changeModel:
		return typed.outcome, nil
	default:
		return changeOutcome{}, errors.New("unexpected change model")
	}
}

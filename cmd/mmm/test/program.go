package test

import tea "github.com/charmbracelet/bubbletea"

var runTeaProgram = defaultRunTea
var runTestProgram = defaultRunTestProgram

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	return program.Run()
}

func defaultRunTestProgram(model *testModel, options ...tea.ProgramOption) (tea.Model, error) {
	program := tea.NewProgram(model, options...)
	model.bindSender(program.Send)
	return program.Run()
}

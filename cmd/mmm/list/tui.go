package list

import tea "github.com/charmbracelet/bubbletea"

type model struct {
	view string
}

func newModel(view string) model {
	return model{view: view}
}

func (m model) Init() tea.Cmd {
	return tea.Quit
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	default:
		return m, tea.Quit
	}
}

func (m model) View() string {
	return m.view
}

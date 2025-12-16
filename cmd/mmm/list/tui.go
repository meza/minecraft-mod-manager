package list

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/perf"
)

type model struct {
	view string
}

func newModel(view string) model {
	perf.Mark("tui.list.open", nil)
	return model{view: view}
}

func (m model) Init() tea.Cmd {
	return tea.Quit
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		perf.Mark("tui.list.action.exit", nil)
		return m, tea.Quit
	default:
		perf.Mark("tui.list.action.exit", nil)
		return m, tea.Quit
	}
}

func (m model) View() string {
	return m.view
}

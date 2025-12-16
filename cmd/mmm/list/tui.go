package list

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/perf"
)

type model struct {
	view string
	span *perf.Span
}

func newModel(view string, span *perf.Span) model {
	if span != nil {
		span.AddEvent("tui.list.open")
	}
	return model{view: view, span: span}
}

func (m model) Init() tea.Cmd {
	return tea.Quit
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		if m.span != nil {
			m.span.AddEvent("tui.list.action.exit")
		}
		return m, tea.Quit
	default:
		if m.span != nil {
			m.span.AddEvent("tui.list.action.exit")
		}
		return m, tea.Quit
	}
}

func (m model) View() string {
	return m.view
}

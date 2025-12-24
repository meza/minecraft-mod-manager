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

func (listModel model) Init() tea.Cmd {
	return tea.Quit
}

func (listModel model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		if listModel.span != nil {
			listModel.span.AddEvent("tui.list.action.exit")
		}
		return listModel, tea.Quit
	default:
		if listModel.span != nil {
			listModel.span.AddEvent("tui.list.action.exit")
		}
		return listModel, tea.Quit
	}
}

func (listModel model) View() string {
	return listModel.view
}

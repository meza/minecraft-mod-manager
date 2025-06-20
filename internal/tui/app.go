package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuItem string

func (m MenuItem) FilterValue() string { return string(m) }

// AppModel is the root TUI model.
type AppModel struct {
	list list.Model
}

func NewAppModel() AppModel {
	items := []list.Item{
		MenuItem("init"),
		MenuItem("add"),
		MenuItem("install"),
		MenuItem("update"),
		MenuItem("list"),
		MenuItem("change"),
		MenuItem("test"),
		MenuItem("prune"),
		MenuItem("scan"),
		MenuItem("remove"),
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Minecraft Mod Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return AppModel{list: l}
}

func (m AppModel) Init() tea.Cmd { return nil }

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-1)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m AppModel) View() string {
	header := lipgloss.NewStyle().Bold(true).Render(m.list.Title)
	return fmt.Sprintf("%s\n%s", header, m.list.View())
}

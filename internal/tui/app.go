package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
)

type MenuItem string

func (m MenuItem) FilterValue() string { return string(m) }

// AppModel is the root TUI model.
type AppModel struct {
	list     list.Model
	width    int
	Selected MenuItem
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

	l := list.New(items, menuDelegate{}, 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	l.KeyMap = TranslatedListKeyMap()
	l.Styles.PaginationStyle = PaginationStyle
	l.Styles.HelpStyle = HelpStyle

	return AppModel{list: l}
}

func (m AppModel) Init() tea.Cmd { return nil }

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(MenuItem); ok {
				m.Selected = item
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-1)
		m.width = msg.Width
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m AppModel) View() string {
	header := renderBanner(m.width)
	return fmt.Sprintf("%s\n%s", header, m.list.View())
}

func renderBanner(width int) string {
	fig := figure.NewFigure(i18n.T("tui.header"), "", true)
	style := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)
	return style.Render(fig.String())
}

type menuDelegate struct{}

func (d menuDelegate) Height() int                             { return 1 }
func (d menuDelegate) Spacing() int                            { return 0 }
func (d menuDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d menuDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	menuItem, ok := item.(MenuItem)
	if !ok {
		return
	}
	line := fmt.Sprintf("%s", menuItem)
	if index == m.Index() {
		fmt.Fprint(w, SelectedItemStyle.Render("❯ "+line))
		return
	}
	fmt.Fprint(w, ItemStyle.Render(line))
}

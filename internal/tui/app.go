package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 18

type Component struct {
	Name  string
	Model tea.Model
}

type AppModel struct {
	width, height int
	sidebar       list.Model
	content       tea.Model
	components    []Component
}

func NewAppModel(components []Component) *AppModel {
	items := make([]list.Item, len(components))
	for i, c := range components {
		items[i] = menuItem(c.Name)
	}

	l := list.New(items, menuDelegate{}, sidebarWidth, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.Styles.Title = TitleStyle
	l.Styles.TitleBar = TitleStyle
	l.Styles.HelpStyle = HelpStyle
	l.Styles.PaginationStyle = PaginationStyle
	l.KeyMap = TranslatedListKeyMap()

	return &AppModel{
		sidebar:    l,
		content:    components[0].Model,
		components: components,
	}
}

func (m AppModel) Init() tea.Cmd { return nil }

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.SetHeight(msg.Height - 1)
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if idx := m.sidebar.Index(); idx >= 0 && idx < len(m.components) {
				m.content = m.components[idx].Model
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)
	if m.content != nil {
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	sidebar := SidebarStyle.Render(m.sidebar.View())
	main := MainStyle.Width(m.width - sidebarWidth - 2).Height(m.height - 1).Render(m.content.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	footer := FooterStyle.Width(m.width).Render("↑/↓ Navigate • ↵ Select • Q Quit • H Help")
	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

type menuItem string

func (i menuItem) FilterValue() string { return string(i) }

type menuDelegate struct{}

func (d menuDelegate) Height() int                             { return 1 }
func (d menuDelegate) Spacing() int                            { return 0 }
func (d menuDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d menuDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(menuItem)
	if !ok {
		return
	}
	label := string(item)
	if index == m.Index() {
		fmt.Fprint(w, SelectedItemStyle.Render("❯ "+label))
	} else {
		fmt.Fprint(w, ItemStyle.Render(label))
	}
}

func RunApp(components []Component) error {
	_, err := tea.NewProgram(NewAppModel(components)).Run()
	return err
}

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"time"
)

type modlistStatusMsg bool
type cwdMsg string

type model struct {
	width, height  int
	modlistPresent bool
	cwd            string
	wizard         *InitModel
}

func cwdCmd() tea.Cmd {
	return func() tea.Msg {
		wd, err := os.Getwd()
		if err != nil {
			return cwdMsg("unknown")
		}
		abs, err := filepath.Abs(wd)
		if err != nil {
			return cwdMsg(wd)
		}
		return cwdMsg(abs)
	}
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		_, err := os.Stat("modlist.json")
		return modlistStatusMsg(err == nil)
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(5*time.Second),
		cwdCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case modlistStatusMsg:
		m.modlistPresent = bool(msg)
		return m, tickCmd(5 * time.Second)
	case cwdMsg:
		m.cwd = string(msg)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyRunes:
			if msg.String() == "q" || msg.String() == "Q" {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.wizard != nil {
			contentWidth := m.width - SidebarStyle.GetWidth()
			m.wizard.SetSize(contentWidth)
		}
	}
	if m.wizard != nil {
		var cmd tea.Cmd
		var mdl tea.Model
		mdl, cmd = m.wizard.Update(msg)
		if w, ok := mdl.(*InitModel); ok {
			m.wizard = w
		}
		return m, cmd
	}
	return m, nil
}

func (m model) modlistStatusText() string {
	if m.modlistPresent {
		return "Yes"
	}
	return "No"
}

func (m model) View() string {
	header := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Padding(0).
		Render(Header(Config{
			App:     "Minecraft Mod Manager",
			Version: "1.0.0",
			Extras: []string{
				"Modlist present: " + lipgloss.NewStyle().Render(m.modlistStatusText()),
			},
		}, m.width))

	footer := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Width(m.width).
		Render(m.cwd)

	sidebar := SidebarStyle.
		Height(m.height - lipgloss.Height(header) - lipgloss.Height(footer)).
		PaddingTop(0).
		Render("Sidebar")

	contentWidth := m.width - lipgloss.Width(sidebar)
	var content string
	if m.wizard != nil {
		m.wizard.SetSize(contentWidth)
		content = m.wizard.View()
	} else {
		content = "content"
	}
	content = lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.height-lipgloss.Height(header)-lipgloss.Height(footer)).
		Align(lipgloss.Left, lipgloss.Top).
		PaddingTop(0).
		Render(content)

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	return lipgloss.JoinVertical(lipgloss.Top, header, main, footer)
}

func RunApp() *tea.Program {
	return tea.NewProgram(model{wizard: NewInitModel("", "", "", "")}, tea.WithAltScreen())
}

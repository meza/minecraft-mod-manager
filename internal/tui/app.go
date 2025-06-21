package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"time"
)

var getwd = os.Getwd
var absPath = filepath.Abs

type modlistStatusMsg bool
type cwdMsg string

type model struct {
	width, height  int
	modlistPresent bool
	cwd            string
	content        helpModel
}

type helpModel interface {
	tea.Model
	Help() string
	SetSize(width, height int)
}

func cwdCmd() tea.Cmd {
	return func() tea.Msg {
		wd, err := getwd()
		if err != nil {
			return cwdMsg("unknown")
		}
		abs, err := absPath(wd)
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
	cmds := []tea.Cmd{
		tickCmd(5 * time.Second),
		cwdCmd(),
	}
	if m.content != nil {
		cmds = append(cmds, m.content.Init())
	}
	return tea.Batch(cmds...)
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
		if m.content != nil {
			m.content.SetSize(m.width, m.height)
		}
	}
	if m.content != nil {
		var cCmd tea.Cmd
		var newModel tea.Model
		newModel, cCmd = m.content.Update(msg)
		m.content = newModel.(helpModel)
		return m, cCmd
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
	if m.content != nil {
		m.content.SetSize(contentWidth, m.height-lipgloss.Height(header)-lipgloss.Height(footer))
	}

	var contentView string
	if m.content != nil {
		if v, ok := m.content.(tea.Model); ok {
			contentView = v.View()
		}
	} else {
		contentView = "content"
	}

	content := lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.height-lipgloss.Height(header)-lipgloss.Height(footer)).
		Align(lipgloss.Left, lipgloss.Top).
		PaddingTop(0).
		Render(contentView)

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	helpText := ""
	if m.content != nil {
		helpText = m.content.Help()
	}
	footer = lipgloss.NewStyle().
		Align(lipgloss.Left).
		Width(m.width).
		Render(helpText)

	return lipgloss.JoinVertical(lipgloss.Top, header, main, footer)
}

func RunApp() *tea.Program {
	return tea.NewProgram(model{}, tea.WithAltScreen())
}

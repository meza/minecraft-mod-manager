package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.ANSIColor(termenv.ANSIBrightWhite))

	QuestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.ANSIColor(termenv.ANSIBrightGreen)).
			Bold(true)

	ItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.ANSIColor(termenv.ANSIBrightWhite))

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(0).
				Foreground(lipgloss.Color("#3a96dd")).Bold(true)

	PaginationStyle = list.DefaultStyles().PaginationStyle.
			PaddingLeft(2)

	HelpStyle = list.DefaultStyles().HelpStyle.
			PaddingLeft(2).
			PaddingBottom(1)

	PlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#767676")).
				PaddingLeft(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			Bold(true)

	SidebarStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Width(18).
			Background(lipgloss.Color("235"))

	MainStyle = lipgloss.NewStyle().
			Padding(1, 2)

	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236"))
)

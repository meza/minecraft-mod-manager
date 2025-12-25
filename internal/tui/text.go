package tui

import "github.com/charmbracelet/lipgloss"

func RenderIfColorEnabled(colorMode ColorMode, style lipgloss.Style, value string) string {
	if colorMode.Enabled() {
		return style.Render(value)
	}
	return value
}

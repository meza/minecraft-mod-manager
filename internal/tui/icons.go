// Package tui provides shared terminal UI helpers.
package tui

func SuccessIcon(colorize bool) string {
	icon := "✅"
	if colorize {
		return QuestionStyle.Render(icon)
	}
	return icon
}

func ErrorIcon(colorize bool) string {
	icon := "❌"
	if colorize {
		return ErrorStyle.Render(icon)
	}
	return icon
}

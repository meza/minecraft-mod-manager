// Package tui provides shared terminal UI helpers.
package tui

func SuccessIcon(colorMode ColorMode) string {
	icon := "✅"
	if colorMode.Enabled() {
		return QuestionStyle.Render(icon)
	}
	return icon
}

func ErrorIcon(colorMode ColorMode) string {
	icon := "❌"
	if colorMode.Enabled() {
		return ErrorStyle.Render(icon)
	}
	return icon
}

// Package tui provides shared terminal UI helpers.
package tui

func SuccessIcon(colorMode ColorMode) string {
	emojiIcon := "\u2705"
	asciiIcon := "V"
	if colorMode.Enabled() {
		return QuestionStyle.Render(emojiIcon)
	}
	return asciiIcon
}

func ErrorIcon(colorMode ColorMode) string {
	emojiIcon := "\u274C"
	asciiIcon := "X"
	if colorMode.Enabled() {
		return ErrorStyle.Render(emojiIcon)
	}
	return asciiIcon
}

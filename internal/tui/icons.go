// Package tui provides shared terminal UI helpers.
package tui

func SuccessIcon(colorMode ColorMode) string {
	asciiIcon := "V"
	icon := asciiIcon
	if colorMode.Enabled() && SupportsUnicode() {
		icon = "\u2705"
	}
	return RenderIfColorEnabled(colorMode, QuestionStyle, icon)
}

func ErrorIcon(colorMode ColorMode) string {
	asciiIcon := "X"
	icon := asciiIcon
	if colorMode.Enabled() && SupportsUnicode() {
		icon = "\u274C"
	}
	return RenderIfColorEnabled(colorMode, ErrorStyle, icon)
}

func FinalErrorIcon(colorMode ColorMode) string {
	asciiIcon := "!!"
	icon := asciiIcon
	if colorMode.Enabled() && SupportsUnicode() {
		icon = "\u203C\uFE0F"
	}
	return RenderIfColorEnabled(colorMode, ErrorStyle, icon)
}

package view

func SuccessIcon(colorMode ColorMode) string {
	asciiIcon := "V"
	icon := asciiIcon
	if SupportsUnicode() {
		icon = "\u2705"
	}
	return RenderIfColorEnabled(colorMode, QuestionStyle, icon)
}

func ErrorIcon(colorMode ColorMode) string {
	asciiIcon := "X"
	icon := asciiIcon
	if SupportsUnicode() {
		icon = "\u274C"
	}
	return RenderIfColorEnabled(colorMode, ErrorStyle, icon)
}

func FinalErrorIcon(colorMode ColorMode) string {
	asciiIcon := "!!"
	icon := asciiIcon
	if SupportsUnicode() {
		icon = "\u203C\uFE0F"
	}
	return RenderIfColorEnabled(colorMode, ErrorStyle, icon)
}

func PendingIcon(colorMode ColorMode) string {
	asciiIcon := "[~]"
	icon := asciiIcon
	if SupportsUnicode() {
		icon = "\u23F3"
	}
	return RenderIfColorEnabled(colorMode, QuestionStyle, icon)
}

func QuestionIcon(colorMode ColorMode) string {
	asciiIcon := "?"
	icon := asciiIcon
	if SupportsUnicode() {
		icon = "\u2754"
	}
	return RenderIfColorEnabled(colorMode, QuestionStyle, icon)
}

func DownloadIcon(colorMode ColorMode) string {
	asciiIcon := "->"
	icon := asciiIcon
	if SupportsUnicode() {
		icon = "\u2B07\uFE0F"
	}
	return RenderIfColorEnabled(colorMode, QuestionStyle, icon)
}

func PinnedIcon(colorMode ColorMode) string {
	asciiIcon := "+"
	icon := asciiIcon
	if SupportsUnicode() {
		icon = "\U0001F4CC"
	}
	return RenderIfColorEnabled(colorMode, QuestionStyle, icon)
}

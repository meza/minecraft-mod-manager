package view

import (
	"fmt"
	"strings"
)

type ModItemStatus int

const (
	ModItemStatusPending ModItemStatus = iota
	ModItemStatusQuestion
	ModItemStatusSpinning
	ModItemStatusDownloading
	ModItemStatusSuccess
	ModItemStatusError
	ModItemStatusSkipped
)

type ModItemLine struct {
	Label    string
	Suffix   string
	Status   ModItemStatus
	Spinner  *Spinner
	Progress *ProgressDetails
}

func RenderModLabel(colorMode ColorMode, name string, id string, platform string) string {
	idPart := RenderIfColorEnabled(colorMode, ParenStyle, id)
	platformPart := RenderIfColorEnabled(colorMode, ParenStyle, platform)
	return fmt.Sprintf("%s (%s) [%s]", name, idPart, platformPart)
}

func RenderModItemLine(line ModItemLine, colorMode ColorMode) string {
	message := line.Label
	if strings.TrimSpace(line.Suffix) != "" {
		message = fmt.Sprintf("%s %s", message, line.Suffix)
	}

	icon := iconForModItem(line, colorMode)
	lines := []string{fmt.Sprintf("%s %s", icon, message)}

	if line.Progress != nil {
		lines = append(lines,
			RenderProgressBar(line.Progress.Bar, line.Progress.Ratio),
			ProgressPercentLine(line.Progress.Ratio, line.Progress.Downloaded, line.Progress.Total),
		)
	}

	return strings.Join(lines, "\n")
}

func iconForModItem(line ModItemLine, colorMode ColorMode) string {
	switch line.Status {
	case ModItemStatusSuccess:
		return SuccessIcon(colorMode)
	case ModItemStatusError, ModItemStatusSkipped:
		return ErrorIcon(colorMode)
	case ModItemStatusDownloading:
		return DownloadIcon(colorMode)
	case ModItemStatusQuestion:
		return QuestionIcon(colorMode)
	case ModItemStatusSpinning:
		if line.Spinner != nil {
			frame := line.Spinner.Frame()
			if strings.TrimSpace(frame) != "" {
				return RenderIfColorEnabled(colorMode, QuestionStyle, frame)
			}
		}
		return PendingIcon(colorMode)
	default:
		return PendingIcon(colorMode)
	}
}

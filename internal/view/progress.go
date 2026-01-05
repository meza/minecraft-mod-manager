package view

import (
	"fmt"
	"math"

	"github.com/charmbracelet/bubbles/progress"
)

type ProgressDetails struct {
	Bar        progress.Model
	Ratio      float64
	Downloaded int64
	Total      int64
}

func NewProgressBar() progress.Model {
	bar := progress.New(
		progress.WithWidth(10),
		progress.WithoutPercentage(),
	)
	if !SupportsUnicode() {
		bar = progress.New(
			progress.WithWidth(10),
			progress.WithoutPercentage(),
			progress.WithFillCharacters('#', '-'),
		)
	}
	return bar
}

func RenderProgressBar(bar progress.Model, ratio float64) string {
	line := bar.ViewAs(ratio)
	if !SupportsUnicode() {
		line = "[" + line + "]"
	}
	return line
}

func ProgressPercentLine(ratio float64, downloaded int64, total int64) string {
	if total <= 0 {
		return fmt.Sprintf("%d%%", percentFromRatio(ratio))
	}
	return fmt.Sprintf("%d%% (%s / %s)", percentFromRatio(ratio), formatBytes(downloaded), formatBytes(total))
}

func percentFromRatio(ratio float64) int {
	if ratio <= 0 {
		return 0
	}
	if ratio >= 1 {
		return 100
	}
	return int(math.Round(ratio * 100))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	suffixes := []string{"KB", "MB", "GB", "TB"}
	for _, suffix := range suffixes {
		value = value / unit
		if value < unit {
			return fmt.Sprintf("%d %s", int64(math.Round(value)), suffix)
		}
	}
	return fmt.Sprintf("%d TB", int64(math.Round(value)))
}

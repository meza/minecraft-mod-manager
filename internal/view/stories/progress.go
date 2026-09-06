package stories

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

const (
	progressStep       = 10
	progressTotalBytes = int64(100 * 1024 * 1024)
)

// Progress presents the shared downloading mod row as an adjustable Bubble Tea model.
type Progress struct {
	details   view.ProgressDetails
	colorMode view.ColorMode
	percent   int
}

// NewProgress returns a fresh progress story at the requested percentage.
func NewProgress(colorMode view.ColorMode, initialPercent int) Progress {
	return Progress{
		details: view.ProgressDetails{
			Bar:   view.NewProgressBar(),
			Total: progressTotalBytes,
		},
		colorMode: colorMode,
		percent:   clampPercent(initialPercent),
	}
}

func (story Progress) Init() tea.Cmd {
	return nil
}

func (story Progress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMessage, ok := msg.(tea.KeyMsg)
	if !ok {
		return story, nil
	}

	switch keyMessage.Type {
	case tea.KeyLeft:
		story.percent = clampPercent(story.percent - progressStep)
	case tea.KeyRight:
		story.percent = clampPercent(story.percent + progressStep)
	}

	return story, nil
}

func (story Progress) View() string {
	ratio := float64(story.percent) / 100
	downloaded := progressTotalBytes * int64(story.percent) / 100
	story.details.Ratio = ratio
	story.details.Downloaded = downloaded
	row := view.RenderModItemLine(view.ModItemLine{
		Label:    sampleModLabel(story.colorMode),
		Status:   view.ModItemStatusDownloading,
		Progress: &story.details,
	}, story.colorMode)
	help := view.RenderIfColorEnabled(
		story.colorMode,
		view.HelpStyle,
		"Use left and right arrows to adjust progress.",
	)
	return fmt.Sprintf("%s\n\n%s", row, help)
}

func clampPercent(percent int) int {
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}

package stories

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

// Spinner presents the shared animated mod-row state as a Bubble Tea model.
type Spinner struct {
	spinner   view.Spinner
	colorMode view.ColorMode
}

// NewSpinner returns a fresh spinner story using the requested terminal color mode.
func NewSpinner(colorMode view.ColorMode) Spinner {
	return Spinner{
		spinner:   view.NewSpinner(),
		colorMode: colorMode,
	}
}

func (story Spinner) Init() tea.Cmd {
	return story.spinner.InitCmd()
}

func (story Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd, _ := story.spinner.Update(msg)
	story.spinner = updated
	return story, cmd
}

func (story Spinner) View() string {
	return view.RenderModItemLine(view.ModItemLine{
		Label:   sampleModLabel(story.colorMode),
		Status:  view.ModItemStatusSpinning,
		Spinner: &story.spinner,
	}, story.colorMode)
}

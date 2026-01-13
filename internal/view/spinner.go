package view

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner wraps the Bubbles spinner model to provide consistent configuration and test behavior.
// It owns spinner selection, frame sanitization, and MMM_TEST handling so callsites remain minimal.
type Spinner struct {
	model    spinner.Model
	testMode bool
	enabled  bool
}

// NewSpinner constructs a spinner configured for the current environment.
func NewSpinner() Spinner {
	model := spinner.New()
	if SupportsUnicode() {
		model.Spinner = spinner.Dot
	} else {
		model.Spinner = spinner.Line
	}
	model.Style = lipgloss.NewStyle()

	_, testMode := os.LookupEnv("MMM_TEST")
	return Spinner{
		model:    model,
		testMode: testMode,
		enabled:  true,
	}
}

// InitCmd returns the command that advances the spinner animation.
// When MMM_TEST is enabled or the spinner is disabled, it returns nil.
func (spin Spinner) InitCmd() tea.Cmd {
	if !spin.enabled || spin.testMode {
		return nil
	}
	return spin.model.Tick
}

// Update advances the spinner if the message is a tick.
// It returns whether the message was handled.
func (spin Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd, bool) {
	if !spin.enabled {
		return spin, nil, false
	}
	typed, ok := msg.(spinner.TickMsg)
	if !ok {
		return spin, nil, false
	}
	if spin.testMode {
		return spin, nil, true
	}
	updated, cmd := spin.model.Update(typed)
	spin.model = updated
	return spin, cmd, true
}

// Frame returns the current spinner frame, sanitized for display.
func (spin Spinner) Frame() string {
	if !spin.enabled {
		return ""
	}
	return sanitizeSpinnerFrame(spin.model.View())
}

// StaticFrame returns a deterministic frame for non-interactive output.
func (spin Spinner) StaticFrame() string {
	if !spin.enabled {
		return ""
	}
	if len(spin.model.Spinner.Frames) == 0 {
		return ""
	}
	return sanitizeSpinnerFrame(spin.model.Spinner.Frames[0])
}

func sanitizeSpinnerFrame(frame string) string {
	trimmed := strings.TrimSpace(frame)
	if trimmed == "" || trimmed == "(error)" {
		return ""
	}
	return frame
}

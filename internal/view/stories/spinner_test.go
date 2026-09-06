package stories_test

import (
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/internal/view/stories"
)

func TestSpinnerForwardsAnimationCommands(t *testing.T) {
	ensureSpinnerAnimationEnabled(t)
	setColorProfile(t, termenv.ANSI256)
	setUnicodeSupport(t, true)
	story := stories.NewSpinner(view.ColorEnabled)

	initialCommand := story.Init()
	require.NotNil(t, initialCommand)

	updated, nextCommand := story.Update(spinner.TickMsg{})
	rendered := updated.View()
	assert.Contains(t, ansi.Strip(rendered), "Sodium (AANobbMI) [modrinth]")
	assert.Contains(t, rendered, view.QuestionStyle.Render("⣽ "))
	assert.Contains(t, rendered, view.ParenStyle.Render("AANobbMI"))
	assert.NotNil(t, nextCommand)
}

func TestSpinnerDisablesStylesWhenColorIsDisabled(t *testing.T) {
	setColorProfile(t, termenv.ANSI256)
	setUnicodeSupport(t, true)
	story := stories.NewSpinner(view.ColorDisabled)

	rendered := story.View()
	assert.NotContains(t, rendered, view.QuestionStyle.Render("⣾ "))
	assert.NotContains(t, rendered, view.ParenStyle.Render("AANobbMI"))
}

func TestSpinnerCreatesIndependentInstances(t *testing.T) {
	ensureSpinnerAnimationEnabled(t)
	first := stories.NewSpinner(view.ColorDisabled)
	second := stories.NewSpinner(view.ColorDisabled)

	firstTick := requireSpinnerTick(t, first.Init())
	secondTick := requireSpinnerTick(t, second.Init())

	assert.NotEqual(t, firstTick.ID, secondTick.ID)
}

func requireSpinnerTick(t *testing.T, command func() tea.Msg) spinner.TickMsg {
	t.Helper()
	require.NotNil(t, command)
	message := command()
	tick, ok := message.(spinner.TickMsg)
	require.True(t, ok)
	return tick
}

func ensureSpinnerAnimationEnabled(t *testing.T) {
	t.Helper()
	t.Setenv("MMM_TEST", "test")
	require.NoError(t, os.Unsetenv("MMM_TEST"))
}

func setColorProfile(t *testing.T, profile termenv.Profile) {
	t.Helper()
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(profile)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(originalProfile)
	})
}

func setUnicodeSupport(t *testing.T, supported bool) {
	t.Helper()
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return supported })
	t.Cleanup(restore)
}

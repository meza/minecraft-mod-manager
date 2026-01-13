package view

import (
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinnerStaticFrameUsesUnicodeWhenAvailable(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)

	spin := NewSpinner()
	assert.Equal(t, spinner.Dot.Frames[0], spin.StaticFrame())
}

func TestSpinnerStaticFrameUsesAsciiWhenUnicodeUnavailable(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)

	spin := NewSpinner()
	assert.Equal(t, spinner.Line.Frames[0], spin.StaticFrame())
}

func TestSpinnerStaticFrameReturnsEmptyWhenNoFrames(t *testing.T) {
	spin := NewSpinner()
	spin.model.Spinner = spinner.Spinner{}

	assert.Equal(t, "", spin.StaticFrame())
}

func TestSpinnerFrameSanitizesErrorFrames(t *testing.T) {
	spin := NewSpinner()
	spin.model.Spinner = spinner.Spinner{Frames: []string{"(error)"}}

	assert.Equal(t, "", spin.Frame())
}

func TestSpinnerUpdateIgnoresNonTickMessages(t *testing.T) {
	spin := NewSpinner()

	updated, cmd, handled := spin.Update(tea.KeyMsg{})
	assert.False(t, handled)
	assert.Nil(t, cmd)
	assert.Equal(t, spin, updated)
}

func TestSpinnerInitCmdReturnsTickWhenEnabled(t *testing.T) {
	originalValue, hadValue := os.LookupEnv("MMM_TEST")
	require.NoError(t, os.Unsetenv("MMM_TEST"))
	t.Cleanup(func() {
		if hadValue {
			require.NoError(t, os.Setenv("MMM_TEST", originalValue))
		} else {
			require.NoError(t, os.Unsetenv("MMM_TEST"))
		}
	})

	spin := NewSpinner()
	assert.NotNil(t, spin.InitCmd())
}

func TestSpinnerUpdateTickReturnsCommand(t *testing.T) {
	originalValue, hadValue := os.LookupEnv("MMM_TEST")
	require.NoError(t, os.Unsetenv("MMM_TEST"))
	t.Cleanup(func() {
		if hadValue {
			require.NoError(t, os.Setenv("MMM_TEST", originalValue))
		} else {
			require.NoError(t, os.Unsetenv("MMM_TEST"))
		}
	})

	spin := NewSpinner()
	updated, cmd, handled := spin.Update(spinner.TickMsg{})
	assert.True(t, handled)
	assert.NotNil(t, cmd)
	assert.NotEqual(t, spin, updated)
}

func TestSpinnerTestModeDisablesTicking(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	spin := NewSpinner()
	assert.Nil(t, spin.InitCmd())

	updated, cmd, handled := spin.Update(spinner.TickMsg{})
	assert.True(t, handled)
	assert.Nil(t, cmd)
	assert.Equal(t, spin, updated)
}

func TestSpinnerDisabledReturnsEmptyOutput(t *testing.T) {
	var spin Spinner

	updated, cmd, handled := spin.Update(spinner.TickMsg{})
	assert.False(t, handled)
	assert.Nil(t, cmd)
	assert.Equal(t, spin, updated)
	assert.Equal(t, "", spin.Frame())
	assert.Equal(t, "", spin.StaticFrame())
	assert.Nil(t, spin.InitCmd())
}

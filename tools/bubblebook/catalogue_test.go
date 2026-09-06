package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestCatalogListsFreshColocatedStories(t *testing.T) {
	definitions := catalog(view.ColorDisabled)

	assert.Equal(t, []string{
		"Spinner",
		"Progress / 0%",
		"Progress / 50%",
		"Progress / 100%",
	}, storyNames(definitions))
	for _, definition := range definitions {
		assert.NotNil(t, definition.factory())
	}

	factory := requireStoryFactory(t, definitions, "Progress / 50%")
	first := factory()
	updated, _ := first.Update(tea.KeyMsg{Type: tea.KeyRight})
	require.Contains(t, updated.View(), "60%")

	second := factory()
	assert.Contains(t, second.View(), "50%")
}

func TestColorModeForOutputRequiresColorAndTerminalSupport(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.ANSI })
	t.Cleanup(restoreColor)

	assert.Equal(t, view.ColorEnabled, colorModeForOutput(descriptorWriter{}))

	restoreNoColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	assert.Equal(t, view.ColorDisabled, colorModeForOutput(descriptorWriter{}))
	restoreNoColor()

	restoreNotTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	assert.Equal(t, view.ColorDisabled, colorModeForOutput(descriptorWriter{}))
	restoreNotTerminal()
}

func storyNames(definitions []storyDefinition) []string {
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.name)
	}
	return names
}

func requireStoryFactory(t *testing.T, definitions []storyDefinition, name string) func() tea.Model {
	t.Helper()
	for _, definition := range definitions {
		if definition.name == name {
			return definition.factory
		}
	}
	require.FailNow(t, "story not found", name)
	return nil
}

type descriptorWriter struct{}

func (descriptorWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (descriptorWriter) Fd() uintptr {
	return 1
}

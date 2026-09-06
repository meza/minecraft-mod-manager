package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestMainRegistersCatalogueBeforeStartingBubblebook(t *testing.T) {
	originalRegister := registerBubblebookStory
	originalStart := startBubblebook
	originalOutput := bubblebookOutput
	t.Cleanup(func() {
		registerBubblebookStory = originalRegister
		startBubblebook = originalStart
		bubblebookOutput = originalOutput
	})

	var registeredNames []string
	registerBubblebookStory = func(name string, _ func() tea.Model) {
		registeredNames = append(registeredNames, name)
	}
	started := false
	startBubblebook = func() {
		started = true
	}
	bubblebookOutput = descriptorWriter{}
	main()

	assert.Equal(t, []string{
		"Spinner",
		"Progress / 0%",
		"Progress / 50%",
		"Progress / 100%",
	}, registeredNames)
	assert.True(t, started)
}

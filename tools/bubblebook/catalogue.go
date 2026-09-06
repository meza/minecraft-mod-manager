package main

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/view"
	viewstories "github.com/meza/minecraft-mod-manager/internal/view/stories"
)

type storyRegistrar func(name string, factory func() tea.Model)

type storyDefinition struct {
	name    string
	factory func() tea.Model
}

func catalog(colorMode view.ColorMode) []storyDefinition {
	return []storyDefinition{
		{name: "Spinner", factory: func() tea.Model {
			return viewstories.NewSpinner(colorMode)
		}},
		{name: "Progress / 0%", factory: func() tea.Model {
			return viewstories.NewProgress(colorMode, 0)
		}},
		{name: "Progress / 50%", factory: func() tea.Model {
			return viewstories.NewProgress(colorMode, 50)
		}},
		{name: "Progress / 100%", factory: func() tea.Model {
			return viewstories.NewProgress(colorMode, 100)
		}},
	}
}

func registerStories(register storyRegistrar, colorMode view.ColorMode) {
	for _, definition := range catalog(colorMode) {
		register(definition.name, definition.factory)
	}
}

func colorModeForOutput(output io.Writer) view.ColorMode {
	if view.SupportsColor(output) && view.SupportsControlSequences(output) {
		return view.ColorEnabled
	}
	return view.ColorDisabled
}

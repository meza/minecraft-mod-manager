package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

func KeyMap() *huh.KeyMap {
	keyMap := huh.NewDefaultKeyMap()
	keyMap.Input.AcceptSuggestion = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete"))
	keyMap.Input.Next = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "next"))

	return keyMap
}

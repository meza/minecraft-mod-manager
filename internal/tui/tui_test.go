package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
)

func TestKeyMap(t *testing.T) {
	keyMap := KeyMap()

	assert.NotNil(t, keyMap)
	assert.Equal(t, key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")), keyMap.Input.AcceptSuggestion)
	assert.Equal(t, key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "next")), keyMap.Input.Next)
}

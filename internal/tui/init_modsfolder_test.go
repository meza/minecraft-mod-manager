package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestModsFolderModel(t *testing.T) {
	m := NewModsFolderModel("")
	_ = m.Init()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.input.Value() != m.input.Placeholder {
		t.Fatalf("expected placeholder set")
	}
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command")
	}
	_ = m.View()
	_ = m.modsFolderSelected()()
	_ = m.Help()
	m = m.SetSize(10, 0)
	if m.input.Width != 10 {
		t.Fatalf("expected width set")
	}
}

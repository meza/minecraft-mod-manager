package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppModel_ViewContainsHeader(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	m := NewAppModel()
	view := m.View()
	if len(strings.TrimSpace(view)) == 0 {
		t.Errorf("view should not be empty")
	}
}

func TestMenuItem_FilterValue(t *testing.T) {
	var m MenuItem = "init"
	if m.FilterValue() != "init" {
		t.Errorf("unexpected value %s", m.FilterValue())
	}
}

func TestAppModel_Init(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	m := NewAppModel()
	if m.Init() != nil {
		t.Error("init should return nil")
	}
}

func TestAppModel_UpdateQuit(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	m := NewAppModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestAppModel_UpdateEnterInit(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	m := NewAppModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on enter")
	}
	um := updated.(AppModel)
	if um.Selected != "init" {
		t.Errorf("expected selected init got %s", um.Selected)
	}
}

func TestAppModel_UpdateWindowSize(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 25})
	um, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel")
	}
	if um.width != 80 {
		t.Errorf("expected width 80 got %d", um.width)
	}
}

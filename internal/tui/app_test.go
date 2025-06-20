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
	if !strings.Contains(view, "Minecraft Mod Manager") {
		t.Errorf("view missing header: %s", view)
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

func TestAppModel_UpdateWindowSize(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	m := NewAppModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 25})
	um, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel")
	}
	if um.list.Width() != 80 {
		t.Errorf("expected width 80 got %d", um.list.Width())
	}
}

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestModlistStatusText(t *testing.T) {
	m := model{modlistPresent: true}
	if m.modlistStatusText() != "Yes" {
		t.Errorf("expected Yes")
	}
	m.modlistPresent = false
	if m.modlistStatusText() != "No" {
		t.Errorf("expected No")
	}
}

func TestUpdateUnknownKeyDoesNotPanic(t *testing.T) {
	m := model{}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Update panicked: %v", r)
		}
	}()
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	_, _ = m.Update(modlistStatusMsg(true))
}

func TestCmdHelpers(t *testing.T) {
	cwd := cwdCmd()
	_ = cwd()
	tick := tickCmd(0)
	_ = tick()
}

func TestInitAndView(t *testing.T) {
	m := model{}
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected command")
	}
	m.width = 10
	m.height = 5
	_ = m.View()
}

func TestRunApp(t *testing.T) {
	if RunApp() == nil {
		t.Fatalf("expected program")
	}
}

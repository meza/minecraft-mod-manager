package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestInitModelStateTransitions(t *testing.T) {
	m := NewInitModel("", "", "", "")
	_ = m.Init()
	if m.state != stateLoader {
		t.Fatalf("expected loader state")
	}
	m2, _ := m.Update(LoaderSelectedMessage{})
	im := m2.(InitModel)
	if im.state != stateGameVersion {
		t.Fatalf("expected game version state")
	}
	m3, cmd := im.Update(GameVersionSelectedMessage{})
	im = m3.(InitModel)
	if im.state != done {
		t.Fatalf("expected done state")
	}
	if cmd == nil {
		t.Fatalf("expected quit command")
	}
	_ = im.View()
}

func TestLoaderModelRender(t *testing.T) {
	lm := NewLoaderModel("")
	_ = lm.View()
	_ = lm.Title()
	lm, _ = lm.Update(tea.WindowSizeMsg{Width: 10})
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if msg := lm.loaderSelected()(); msg == nil {
		t.Fatalf("expected loader selected message")
	}
}

func TestGameVersionModelRender(t *testing.T) {
	gm := GameVersionModel{}
	_ = gm.Init()
	_ = gm.View()
	_, _ = gm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_ = gm.gameVersionSelected()()
	_ = isValidMinecraftVersion("")
	_ = isValidMinecraftVersion("1.0")
}

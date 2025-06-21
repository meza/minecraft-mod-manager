package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

func TestInitModelNavigation(t *testing.T) {
	m := NewInitModel("", "", "", "")
	if m.focus != 0 {
		t.Fatalf("expected focus 0")
	}
	_ = m.Init()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	im := m2.(*InitModel)
	if im.focus != 1 {
		t.Fatalf("tab should move focus")
	}
	m3, _ := im.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	im = m3.(*InitModel)
	if im.focus != 0 {
		t.Fatalf("shift tab should move focus back")
	}
	m4, _ := im.Update(tea.KeyMsg{Type: tea.KeyRight})
	im = m4.(*InitModel)
	if im.step != stepDone {
		t.Fatalf("right should go next")
	}
	m5, _ := im.Update(tea.KeyMsg{Type: tea.KeyLeft})
	im = m5.(*InitModel)
	if im.step != stepFields {
		t.Fatalf("left should go back")
	}
	_ = im.currentHelp()
	_ = im.View()
	im.SetSize(10)
	im.focusCurrent()
	im.blurCurrent()
	im.nextField()
	im.prevField()
	_, _ = im.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
}

func TestModsFolderModelRender(t *testing.T) {
	m := NewModsFolderModel("")
	_ = m.View()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if msg := m.modsFolderSelected()(); msg == nil {
		t.Fatalf("expected message")
	}
	_ = m.HelpView()
}

func TestLoaderModelRender(t *testing.T) {
	lm := NewLoaderModel("")
	lm.Focus()
	_ = lm.View()
	lm, _ = lm.Update(tea.WindowSizeMsg{Width: 10})
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	_ = lm.HelpView()
	if msg := lm.loaderSelected()(); msg == nil {
		t.Fatalf("expected message")
	}
}

func TestGameVersionModelRender(t *testing.T) {
	gm := NewGameVersionModel("")
	gm.Focus()
	_ = gm.Init()
	_ = gm.View()
	_, _ = gm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_, _ = gm.Update(tea.KeyMsg{Type: tea.KeyTab})
	gm.input.SetValue("1.20.1")
	_, _ = gm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, _ = gm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_, _ = gm.Update(tea.KeyMsg{Type: tea.KeyTab})
	_ = gm.gameVersionSelected()()
	_ = gm.HelpView()
	_ = isValidMinecraftVersion("")
	_ = isValidMinecraftVersion("1.0")
}

func TestInitModelHelpers(t *testing.T) {
	im := NewInitModel("", "", "", "")
	im.focus = 0
	im.focusCurrent()
	im.focus = 1
	im.focusCurrent()
	im.focus = 2
	im.focusCurrent()
	im.blurCurrent()
	im.focus = 0
	im.blurCurrent()
	im.focus = 1
	im.blurCurrent()
	im.focus = 0
	im.prevField()
	if im.focus != 2 {
		t.Fatalf("wrap prevField")
	}
	im.nextField()
	if im.focus != 0 {
		t.Fatalf("nextField")
	}
	_ = im.currentHelp()
	im.focus = 1
	_ = im.currentHelp()
	im.focus = 2
	_ = im.currentHelp()
	_, _ = im.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

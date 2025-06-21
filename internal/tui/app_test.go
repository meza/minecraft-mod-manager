package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

type stubModel struct{}

func (s stubModel) Init() tea.Cmd                           { return nil }
func (s stubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return s, nil }
func (s stubModel) View() string                            { return "stub" }
func (s stubModel) Help() string                            { return "help" }
func (s stubModel) SetSize(width, height int)               {}

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

func TestModelUpdateWithoutContent(t *testing.T) {
	m := model{}
	_, _ = m.Update(modlistStatusMsg(false))
	_, _ = m.Update(cwdMsg("here"))
	_, _ = m.Update(tea.WindowSizeMsg{Width: 1, Height: 1})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_ = m.View()
}

func TestCmdHelpers(t *testing.T) {
	oldGetwd := getwd
	oldAbs := absPath
	getwd = func() (string, error) { return "", os.ErrNotExist }
	absPath = func(p string) (string, error) { return "", os.ErrNotExist }
	cwd := cwdCmd()
	_ = cwd()
	getwd = func() (string, error) { return "/tmp", nil }
	absPath = func(p string) (string, error) { return "/abs", nil }
	_ = cwdCmd()()
	getwd = oldGetwd
	absPath = oldAbs
	tick := tickCmd(0)
	_ = tick()
}

func TestInitAndView(t *testing.T) {
	m := model{content: stubModel{}}
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected command")
	}
	m.width = 10
	m.height = 5
	_ = m.View()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	m = nm.(model)
	if m.width != 20 {
		t.Fatalf("width not set")
	}
	_ = m.View()
}

func TestModelUpdateWithContent(t *testing.T) {
	m := model{content: stubModel{}}
	m.width = 5
	m.height = 5
	_, _ = m.Update(modlistStatusMsg(true))
	_, _ = m.Update(cwdMsg("/tmp"))
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_, _ = m.Update(tea.WindowSizeMsg{Width: 5, Height: 5})
	_ = m.View()
}

func TestRunApp(t *testing.T) {
	if RunApp() == nil {
		t.Fatalf("expected program")
	}
}

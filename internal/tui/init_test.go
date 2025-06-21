package tui

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"testing"
)

func TestInitModelStateTransitions(t *testing.T) {
	m := NewInitModel("", "", "", "")
	_ = m.Init()
	if m.state != stateModsFolder {
		t.Fatalf("expected mods folder state")
	}
	m.SetSize(10, 10)
	m2, _ := m.Update(ModsFolderSelectedMessage{})
	im := m2.(InitModel)
	if im.state != stateLoader {
		t.Fatalf("expected loader state")
	}
	m3, _ := im.Update(LoaderSelectedMessage{})
	im = m3.(InitModel)
	if im.state != stateGameVersion {
		t.Fatalf("expected game version state")
	}
	m4, cmd := im.Update(GameVersionSelectedMessage{})
	im = m4.(InitModel)
	if im.state != done {
		t.Fatalf("expected done state")
	}
	if cmd == nil {
		t.Fatalf("expected quit command")
	}
	_ = im.View()
	_ = im.Help()
}

func TestLoaderModelRender(t *testing.T) {
	lm := NewLoaderModel("")
	_ = lm.View()
	_ = lm.Title()
	lm = lm.SetSize(10, 0)
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	lm, cmd := lm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command")
	}
	_ = lm.View()
	_ = lm.Help()
	_ = isValidLoader(models.FABRIC)
	_ = isValidLoader(models.Loader("none"))
	d := itemDelegate{}
	if d.Height() != 1 || d.Spacing() != 0 {
		t.Fatalf("delegate dims")
	}
	_ = d.Update(nil, nil)
	var buf bytes.Buffer
	d.Render(&buf, lm.list, 0, loaderType("fabric"))
	_ = loaderType("fabric").FilterValue()
}

func TestGameVersionModelRender(t *testing.T) {
	gm := NewGameVersionModel("")
	validVersionFn = func(string, httpClient.Doer) bool { return true }
	defer func() { validVersionFn = minecraft.IsValidVersion }()
	_ = gm.Init()
	gm = gm.SetSize(10, 0)
	gm, _ = gm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	gm, cmd := gm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command")
	}
	_ = gm.View()
	_ = gm.gameVersionSelected()()
	_ = isValidMinecraftVersion("")
	validVersionFn = func(string, httpClient.Doer) bool { return false }
	_ = isValidMinecraftVersion("1.0")
	_ = gm.Help()
}

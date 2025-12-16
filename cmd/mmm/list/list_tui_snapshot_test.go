package list

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
)

func TestListViewSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	entries := []listEntry{
		{DisplayName: "Alpha Mod", ID: "mod-a", Platform: models.MODRINTH, Installed: true},
		{DisplayName: "Beta Mod", ID: "mod-b", Platform: models.CURSEFORGE, Installed: false},
	}

	view := renderListView(entries, true)
	snaps.MatchSnapshot(t, view)
}

func TestModelView(t *testing.T) {
	perf.ClearPerformanceLog()
	m := newModel("example")
	assert.Equal(t, "example", m.View())

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assertPerfMarkExistsList(t, "tui.list.open")
	assertPerfMarkExistsList(t, "tui.list.action.exit")
}

func assertPerfMarkExistsList(t *testing.T, name string) {
	t.Helper()
	for _, entry := range perf.GetPerformanceLog() {
		if entry.Type == perf.MarkType && entry.Name == name {
			return
		}
	}
	t.Fatalf("expected perf mark %q not found", name)
}

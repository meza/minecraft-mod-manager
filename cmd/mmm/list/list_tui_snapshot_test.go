package list

import (
	"context"
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
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	_, span := perf.StartSpan(context.Background(), "tui.list.session")
	m := newModel("example", span)
	assert.Equal(t, "example", m.View())

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	span.End()

	spans, err := perf.GetSpans()
	assert.NoError(t, err)

	s, ok := perf.FindSpanByName(spans, "tui.list.session")
	assert.True(t, ok)

	var eventNames []string
	for _, event := range s.Events {
		eventNames = append(eventNames, event.Name)
	}
	assert.Contains(t, eventNames, "tui.list.open")
	assert.Contains(t, eventNames, "tui.list.action.exit")
}

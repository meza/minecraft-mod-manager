package list

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
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
	m := newModel("example")
	assert.Equal(t, "example", m.View())
}

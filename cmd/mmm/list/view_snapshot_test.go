package list

import (
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestListViewSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	entries := []listEntry{
		{DisplayName: "Alpha Mod", ID: "mod-a", Platform: models.MODRINTH, Status: listEntryInstalled},
		{DisplayName: "Beta Mod", ID: "mod-b", Platform: models.CURSEFORGE, Status: listEntryMissing},
		{DisplayName: "Gamma Mod", ID: "mod-c", Platform: models.MODRINTH, Status: listEntryHashMismatch, FileName: "mod-c.jar"},
	}

	viewOutput := renderListView(entries, view.ColorEnabled)
	snaps.MatchSnapshot(t, viewOutput)
}

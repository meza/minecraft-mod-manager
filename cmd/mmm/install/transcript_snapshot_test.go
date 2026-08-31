package install

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestInstallTranscriptSnapshotSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
		},
	}
	items, indexByKey := buildInstallItems(cfg)

	buffer := &bytes.Buffer{}
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, buffer, nil)

	_, cmd := model.Update(installItemSuccessMsg{key: installModKey(cfg.Mods[0])})
	runTeaCmd(cmd)
	_, cmd = model.Update(installItemSuccessMsg{key: installModKey(cfg.Mods[1])})
	runTeaCmd(cmd)
	_, cmd = model.Update(installExecutionFinishedMsg{outcome: installExecutionOutcome{errType: installExecutionErrorNone}})
	runTeaCmd(cmd)

	snaps.MatchSnapshot(t, strings.TrimSpace(buffer.String()))
}

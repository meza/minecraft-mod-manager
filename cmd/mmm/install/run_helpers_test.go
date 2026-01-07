package install

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestInstallMaxConcurrencyHonorsSingleMod(t *testing.T) {
	assert.Equal(t, 1, installMaxConcurrency(1))
}

func TestInstallMaxConcurrencyUsesModCountWhenBelowLimit(t *testing.T) {
	previous := gomaxprocsFunc
	gomaxprocsFunc = func(int) int { return 4 }
	t.Cleanup(func() { gomaxprocsFunc = previous })

	assert.Equal(t, 2, installMaxConcurrency(2))
}

func TestInstallMaxConcurrencyUsesFallbackWhenLimitInvalid(t *testing.T) {
	previous := gomaxprocsFunc
	gomaxprocsFunc = func(int) int { return 0 }
	t.Cleanup(func() { gomaxprocsFunc = previous })

	assert.Equal(t, 1, installMaxConcurrency(10))
}

func TestSnapshotInstallItemsUsesStateWhenAvailable(t *testing.T) {
	cfg := models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}}
	items, indexByKey := buildInstallItems(cfg)

	state := &installExecutionState{items: items, indexByKey: indexByKey}
	state.setSuccess(installModKey(cfg.Mods[0]), "Alpha")

	snapshot := snapshotInstallItems(state, cfg)
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, installItemSuccess, snapshot[0].Status)
	}
}

func TestSnapshotInstallItemsBuildsWhenStateMissing(t *testing.T) {
	cfg := models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "", Type: models.MODRINTH}}}
	snapshot := snapshotInstallItems(nil, cfg)
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, "alpha", snapshot[0].DisplayName)
	}
}

func TestInstallMaxConcurrencyUsesPositiveValue(t *testing.T) {
	assert.Greater(t, installMaxConcurrency(2), 0)
}

func TestInstallMaxConcurrencyUsesProcessorCountWhenNoMods(t *testing.T) {
	assert.Greater(t, installMaxConcurrency(0), 0)
}

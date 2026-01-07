package install

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestBuildInstallItemsUsesFallbackNameAndOrdersAlphabetically(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "2", Name: "Beta", Type: models.MODRINTH},
			{ID: "1", Name: "alpha", Type: models.CURSEFORGE},
			{ID: "Gamma", Name: "", Type: models.MODRINTH},
		},
	}

	items, indexByKey := buildInstallItems(cfg)
	if assert.Len(t, items, 3) {
		assert.Equal(t, "alpha", items[0].DisplayName)
		assert.Equal(t, "Beta", items[1].DisplayName)
		assert.Equal(t, "Gamma", items[2].DisplayName)
	}

	assert.Equal(t, 0, indexByKey[installModKey(cfg.Mods[1])])
	assert.Equal(t, 1, indexByKey[installModKey(cfg.Mods[0])])
	assert.Equal(t, 2, indexByKey[installModKey(cfg.Mods[2])])
}

func TestBuildInstallItemsSortsByPlatformThenIDWhenNamesMatch(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "b", Name: "Same", Type: models.MODRINTH},
			{ID: "b", Name: "same", Type: models.CURSEFORGE},
			{ID: "a", Name: "same", Type: models.CURSEFORGE},
		},
	}

	items, _ := buildInstallItems(cfg)
	if assert.Len(t, items, 3) {
		assert.Equal(t, models.CURSEFORGE, items[0].Mod.Type)
		assert.Equal(t, "a", items[0].Mod.ID)
		assert.Equal(t, models.CURSEFORGE, items[1].Mod.Type)
		assert.Equal(t, "b", items[1].Mod.ID)
		assert.Equal(t, models.MODRINTH, items[2].Mod.Type)
	}
}

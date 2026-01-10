package update

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

func TestBuildUpdateItemsOrdersAlphabetically(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "mod-b", Name: "Beta", Type: models.MODRINTH},
			{ID: "mod-a", Name: "Alpha", Type: models.MODRINTH},
			{ID: "mod-c", Name: "", Type: models.CURSEFORGE},
			{ID: "mod-d", Name: "Alpha", Type: models.CURSEFORGE},
		},
	}

	items, indexByKey := buildUpdateItems(cfg, nil)

	assert.Equal(t, "Alpha", items[0].DisplayName)
	assert.Equal(t, models.CURSEFORGE, items[0].Mod.Type)
	assert.Equal(t, "Alpha", items[1].DisplayName)
	assert.Equal(t, models.MODRINTH, items[1].Mod.Type)
	assert.Equal(t, "Beta", items[2].DisplayName)
	assert.Equal(t, "mod-c", items[3].DisplayName)
	assert.Equal(t, 2, indexByKey[0])
	assert.Equal(t, 1, indexByKey[1])
	assert.Equal(t, 3, indexByKey[2])
	assert.Equal(t, 0, indexByKey[3])
}

func TestBuildUpdateItemsUsesIDWhenNameMissing(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "mod-id", Name: "", Type: models.MODRINTH},
		},
	}
	items, _ := buildUpdateItems(cfg, nil)
	assert.Equal(t, "mod-id", items[0].DisplayName)
}

func TestBuildUpdateItemsOrdersByIDWhenNameAndPlatformMatch(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "mod-b", Name: "Same", Type: models.MODRINTH},
			{ID: "mod-a", Name: "Same", Type: models.MODRINTH},
		},
	}
	items, _ := buildUpdateItems(cfg, nil)
	assert.Equal(t, "mod-a", items[0].Mod.ID)
	assert.Equal(t, "mod-b", items[1].Mod.ID)
}

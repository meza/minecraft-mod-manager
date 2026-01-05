package change

import (
	"bytes"
	"testing"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type fdWriter struct {
	fd uintptr
}

func (writer fdWriter) Fd() uintptr { return writer.fd }

func (writer fdWriter) Write(value []byte) (int, error) {
	return len(value), nil
}

func TestBuildChangeItemsSortsByName(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "b", Name: "Zeta", Type: models.MODRINTH},
			{ID: "a", Name: "alpha", Type: models.CURSEFORGE},
			{ID: "c", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	items, index := buildChangeItems(cfg, changeItemOrderAlphabetical)
	assert.Equal(t, 3, len(items))
	assert.Equal(t, "alpha", items[0].DisplayName)
	assert.Equal(t, models.CURSEFORGE, items[0].Mod.Type)
	assert.Equal(t, "Alpha", items[1].DisplayName)
	assert.Equal(t, models.MODRINTH, items[1].Mod.Type)
	assert.Equal(t, "Zeta", items[2].DisplayName)
	assert.Equal(t, 0, index[changeModKey(items[0].Mod)])
}

func TestBuildChangeItemsSortsByPlatformAndID(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "b", Name: "Alpha", Type: models.MODRINTH},
			{ID: "c", Name: "Alpha", Type: models.CURSEFORGE},
			{ID: "a", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	items, _ := buildChangeItems(cfg, changeItemOrderAlphabetical)
	require.Equal(t, 3, len(items))
	assert.Equal(t, models.CURSEFORGE, items[0].Mod.Type)
	assert.Equal(t, "c", items[0].Mod.ID)
	assert.Equal(t, models.MODRINTH, items[1].Mod.Type)
	assert.Equal(t, "a", items[1].Mod.ID)
	assert.Equal(t, models.MODRINTH, items[2].Mod.Type)
	assert.Equal(t, "b", items[2].Mod.ID)
}

func TestBuildChangeItemsPreservesConfigOrder(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "b", Name: "Zeta", Type: models.MODRINTH},
			{ID: "a", Name: "alpha", Type: models.CURSEFORGE},
			{ID: "c", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	items, _ := buildChangeItems(cfg, changeItemOrderConfig)
	require.Equal(t, 3, len(items))
	assert.Equal(t, "Zeta", items[0].DisplayName)
	assert.Equal(t, "alpha", items[1].DisplayName)
	assert.Equal(t, "Alpha", items[2].DisplayName)
}

func TestCloneChangeItems(t *testing.T) {
	items := []changeItem{{DisplayName: "Example"}}
	clone := cloneChangeItems(items)
	assert.Equal(t, items, clone)
	clone[0].DisplayName = "New"
	assert.NotEqual(t, items[0].DisplayName, clone[0].DisplayName)
}

func TestModDisplayNameFallsBackToID(t *testing.T) {
	mod := models.Mod{ID: "fallback", Name: " "}
	assert.Equal(t, "fallback", modDisplayName(mod))
}

func TestColorModeForOutput(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreTerminal)
	t.Cleanup(restoreProfile)

	colorMode := colorModeForOutput(fdWriter{fd: 1})
	assert.True(t, colorMode.Enabled())
}

func TestColorModeForOutputAscii(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreTerminal)
	t.Cleanup(restoreProfile)

	colorMode := colorModeForOutput(fdWriter{fd: 1})
	assert.False(t, colorMode.Enabled())
}

func TestIndexLockByMod(t *testing.T) {
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH},
		{ID: "beta", Type: models.CURSEFORGE},
	}
	index := indexLockByMod(lock)
	assert.Equal(t, lock[0], index["modrinth:alpha"])
	assert.Equal(t, lock[1], index["curseforge:beta"])
}

func TestMessageWithIcon(t *testing.T) {
	assert.Equal(t, "X message", messageWithIcon("X", "message"))
}

func TestBuildChangeItemsDefaults(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}
	items, _ := buildChangeItems(cfg, changeItemOrderAlphabetical)
	assert.Equal(t, changeCompatPending, items[0].CompatStatus)
	assert.Equal(t, changeDownloadPending, items[0].DownloadStatus)
	assert.Equal(t, changeSwitchPending, items[0].SwitchStatus)
}

func TestColorModeForOutputNonTerminal(t *testing.T) {
	colorMode := colorModeForOutput(&bytes.Buffer{})
	assert.False(t, colorMode.Enabled())
}

package test

import (
	"bytes"
	"testing"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type fakeTTYWriter struct {
	*bytes.Buffer
}

func (writer fakeTTYWriter) Fd() uintptr {
	return 1
}

func TestBuildTestItemsIndexesByKey(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.CURSEFORGE},
		},
	}

	items, indexByKey := buildTestItems(cfg)

	assert.Len(t, items, 2)
	assert.Equal(t, testItemStatusChecking, items[0].Status)
	assert.Equal(t, 0, indexByKey[testModKey(cfg.Mods[0])])
	assert.Equal(t, 1, indexByKey[testModKey(cfg.Mods[1])])
}

func TestCloneTestItemsCopiesSlice(t *testing.T) {
	items := []testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusSupported},
	}

	clone := cloneTestItems(items)
	clone[0].Status = testItemStatusUnsupported

	assert.Equal(t, testItemStatusSupported, items[0].Status)
	assert.Equal(t, testItemStatusUnsupported, clone[0].Status)
}

func TestModDisplayNameFallsBackToID(t *testing.T) {
	mod := models.Mod{ID: "alpha", Name: "   "}
	assert.Equal(t, "alpha", modDisplayName(mod))
}

func TestColorModeForOutputEnabledWhenSupported(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	writer := fakeTTYWriter{Buffer: &bytes.Buffer{}}
	assert.Equal(t, view.ColorEnabled, colorModeForOutput(writer))
}

func TestColorModeForOutputDisabledWhenNoColor(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColor)
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	writer := fakeTTYWriter{Buffer: &bytes.Buffer{}}
	assert.Equal(t, view.ColorDisabled, colorModeForOutput(writer))
}

func TestMessageWithIcon(t *testing.T) {
	assert.Equal(t, "! hello", messageWithIcon("!", "hello"))
}

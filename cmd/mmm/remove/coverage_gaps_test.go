package remove

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRemoveModelKeepsActiveHeaderVisibleAfterViewportShrinks(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
	}}, nil, nil)

	largeView := model.View()
	_, resizeCmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 100})
	assert.Nil(t, resizeCmd)
	assert.Equal(t, largeView, model.View())

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 1})
	activeHeader := i18n.T("cmd.remove.header.removing", nil)
	assert.True(t, strings.HasSuffix(model.View(), "\n\n"+activeHeader), "small viewport should repeat the active header at the bottom")

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 0})
	assert.True(t, strings.HasSuffix(model.View(), "\n\n"+activeHeader), "zero-height resize should retain the last usable height")
}

func TestRemoveModelKeepsResultHeaderVisibleAfterViewportShrinks(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod:    models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		Status: removeItemSuccess,
	}}, nil, nil)
	model.done = true
	model.outcome = removeExecutionOutcome{errType: removeExecutionErrorNone}

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 1})
	resultHeader := i18n.T("cmd.remove.header.result", nil)
	assert.True(t, strings.HasSuffix(model.View(), "\n\n"+resultHeader), "small viewport should repeat the result header at the bottom")
}

func TestRunRemoveWithMatchesReturnsConfirmationErrorWithoutRemoving(t *testing.T) {
	confirmationErr := errors.New("confirmation unavailable")
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeInteractive,
	}
	matches := []removeMatch{{mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	deps := removeDeps{runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return nil, confirmationErr
	}}

	removed, interactive, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{}, deps, runState, matches)
	require.ErrorIs(t, err, confirmationErr)
	assert.True(t, interactive)
	assert.Zero(t, removed)
}

func TestRunRemoveLeavesSelectionUntouchedWhenUnmanagedGateRejectsRun(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium", FileName: "sodium.jar"}}
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	managedPath := filepath.Join(meta.ModsFolderPath(cfg), "sodium.jar")
	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "foreign.jar")
	require.NoError(t, afero.WriteFile(fs, managedPath, []byte("managed"), 0644))
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("foreign"), 0644))

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	removed, _, err := runRemove(context.Background(), cmd, removeOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
		Lookups:    []string{"sodium"},
		LockSync:   locksync.PolicyFlags{Skip: true},
	}, removeDeps{fs: fs, runTea: defaultRunTea})
	require.ErrorIs(t, err, interaction.ErrUnmanagedFiles)
	assert.True(t, clierrors.IsHandled(err))
	assert.Zero(t, removed)
	assert.Contains(t, out.String(), "foreign.jar")

	updatedCfg, readErr := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, readErr)
	assert.Equal(t, cfg.Mods, updatedCfg.Mods)
	updatedLock, readErr := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, readErr)
	assert.Equal(t, lock, updatedLock)
	for _, path := range []string{managedPath, unmanagedPath} {
		exists, existsErr := afero.Exists(fs, path)
		require.NoError(t, existsErr)
		assert.True(t, exists, "%s should remain", path)
	}
}

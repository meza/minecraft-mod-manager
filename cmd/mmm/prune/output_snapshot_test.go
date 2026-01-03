package prune

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

func TestPrunePromptOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	input := &fakeTerminalReader{}
	_, err := input.WriteString("n\n")
	require.NoError(t, err)

	out := &fakeTerminalWriter{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(input)
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		normalizeSnapshotOutput(stripANSI(strings.TrimRight(out.String(), "\n"))),
		strings.TrimRight(errOut.String(), "\n"),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func TestPruneUnattendedOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	out := &fakeTerminalWriter{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Unattended: true,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		normalizeSnapshotOutput(stripANSI(strings.TrimRight(out.String(), "\n"))),
		strings.TrimRight(errOut.String(), "\n"),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func TestPrunePromptDisabledOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := tui.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(out, errOut, true, false),
		output: output.New(out, errOut, true),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	assert.True(t, clierrors.IsHandled(err))
	assert.Equal(t, 0, deletedCount)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		normalizeSnapshotOutput(strings.TrimRight(out.String(), "\n")),
		strings.TrimRight(errOut.String(), "\n"),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func normalizeSnapshotOutput(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}

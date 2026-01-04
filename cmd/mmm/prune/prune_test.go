package prune

import (
	"bytes"
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

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
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

type fakeTerminalWriter struct {
	bytes.Buffer
}

func (writer *fakeTerminalWriter) Fd() uintptr {
	return 1
}

type fakeTerminalReader struct {
	bytes.Buffer
}

func (reader *fakeTerminalReader) Fd() uintptr {
	return 0
}

func TestRunPruneNoUnmanagedLogsNotice(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

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
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)
	assert.Equal(t, "cmd.prune.no_unmanaged\n", out.String())
	assert.Empty(t, errOut.String())
}

func TestRunPruneUnattendedWithoutForceSkipsDeletion(t *testing.T) {
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

	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "extra.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	out := &fakeTerminalWriter{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Unattended: true,
		Quiet:      true,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(out, errOut, true, false),
		output: output.New(out, errOut, true),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)
	expected := "❌ cmd.prune.unmanaged.entry, Arg 1: {Count: 0, Data: &map[file:" + unmanagedPath + "]}\n"
	assert.Equal(t, expected, stripANSI(out.String()))
	assert.Equal(t, "cmd.prune.error.prompt_disabled\n", errOut.String())

	exists, existsErr := afero.Exists(fs, unmanagedPath)
	require.NoError(t, existsErr)
	assert.True(t, exists)
}

func TestRunPrunePromptNoKeepsFiles(t *testing.T) {
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

	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "extra.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	input := &fakeTerminalReader{}
	_, err := input.WriteString("n\n")
	require.NoError(t, err)
	outputWriter := &fakeTerminalWriter{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(input)
	cmd.SetOut(outputWriter)
	cmd.SetErr(errOut)

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(outputWriter, errOut, false, false),
		output: output.New(outputWriter, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)

	exists, existsErr := afero.Exists(fs, unmanagedPath)
	require.NoError(t, existsErr)
	assert.True(t, exists)

	expected := "❌ cmd.prune.unmanaged.entry, Arg 1: {Count: 0, Data: &map[file:" + unmanagedPath + "]}\n" +
		"? cmd.prune.confirm "
	assert.Equal(t, expected, stripANSI(outputWriter.String()))
	assert.Empty(t, errOut.String())
}

func TestRunPrunePromptYesDeletesFiles(t *testing.T) {
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

	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "extra.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	input := &fakeTerminalReader{}
	_, err := input.WriteString("y\n")
	require.NoError(t, err)
	outputWriter := &fakeTerminalWriter{}
	cmd := &cobra.Command{}
	cmd.SetIn(input)
	cmd.SetOut(outputWriter)
	cmd.SetErr(&bytes.Buffer{})

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(outputWriter, outputWriter, false, false),
		output: output.New(outputWriter, outputWriter, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, deletedCount)
	assert.Contains(t, outputWriter.String(), "cmd.prune.confirm")

	exists, existsErr := afero.Exists(fs, unmanagedPath)
	require.NoError(t, existsErr)
	assert.False(t, exists)
}

func TestRunPruneForceDeletesAndRespectsIgnore(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

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
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "managed.jar"},
	}))

	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "managed.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "ignored.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("ignored.jar\n"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Unattended: true,
		Force:      true,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, deletedCount)

	expectedLine := "cmd.prune.deleted, Arg 1: {Count: 0, Data: &map[file:" +
		filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar") + "]}\n"
	assert.Equal(t, expectedLine, out.String())
	assert.Empty(t, errOut.String())

	managedExists, managedErr := afero.Exists(fs, filepath.Join(meta.ModsFolderPath(cfg), "managed.jar"))
	require.NoError(t, managedErr)
	assert.True(t, managedExists)

	ignoredExists, ignoredErr := afero.Exists(fs, filepath.Join(meta.ModsFolderPath(cfg), "ignored.jar"))
	require.NoError(t, ignoredErr)
	assert.True(t, ignoredExists)

	unmanagedExists, unmanagedErr := afero.Exists(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"))
	require.NoError(t, unmanagedErr)
	assert.False(t, unmanagedExists)
}

func TestRunPruneQuietForceIsSilent(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

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
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Force:      true,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(out, errOut, true, false),
		output: output.New(out, errOut, true),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, deletedCount)
	assert.Empty(t, out.String())
	assert.Empty(t, errOut.String())
}

func TestRunPruneLockMissingErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

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
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	assert.True(t, clierrors.IsHandled(err))
	assert.Equal(t, 0, deletedCount)
	assert.Empty(t, out.String())
	assert.True(t, strings.Contains(errOut.String(), "cmd.prune.error.lock_missing"))
	assert.True(t, strings.Contains(errOut.String(), meta.LockPath()))
	assert.True(t, strings.Contains(errOut.String(), "mmm install"))
}

var ansiPattern = regexp.MustCompile("\u001b\\[[0-9;]*[A-Za-z]")

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

package list

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRunListPrintsInstalledAndMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
			{ID: "mod-b", Name: "Mod B", Type: models.CURSEFORGE},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))

	installedContents := []byte("installed")
	installedHash := fmt.Sprintf("%x", sha1.Sum(installedContents))

	lock := []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: installedHash},
		{ID: "mod-b", Type: models.CURSEFORGE, FileName: "mod-b.jar", Hash: "missing"},
	}
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, lock))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), installedContents, 0644))

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(errBuffer)

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, errBuffer, false),
		logger:    logger.New(outBuffer, errBuffer, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"\u2705 Mod A (mod-a) [modrinth]\n" +
		"\u274C Mod B (mod-b) [curseforge] cmd.list.entry.missing_suffix\n"
	assert.Equal(t, expected, outBuffer.String())
	assert.Empty(t, errBuffer.String())
}

func TestRunListShowsHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))

	installedContents := []byte("installed")
	installedHash := fmt.Sprintf("%x", sha1.Sum(installedContents))
	otherHash := fmt.Sprintf("%x", sha1.Sum([]byte("different")))

	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), installedContents, 0644))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: otherHash},
	}))

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(errBuffer)

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, errBuffer, false),
		logger:    logger.New(outBuffer, errBuffer, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"\u274C Mod A (mod-a) [modrinth] cmd.list.entry.hash_mismatch_suffix, Arg 1: {Count: 0, Data: &map[fix_command:mmm install]}\n"
	assert.Equal(t, expected, outBuffer.String())
	assert.Empty(t, errBuffer.String())
	assert.NotEqual(t, installedHash, otherHash)
}

func TestRunListShowsEmptyMessageWhenNoMods(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{}))

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(errBuffer)

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, errBuffer, false),
		logger:    logger.New(outBuffer, errBuffer, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.NoError(t, err)
	assert.Equal(t, "cmd.list.empty\n", outBuffer.String())
}

func TestRunListQuietStillPrints(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{}))

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(errBuffer)

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: true}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, errBuffer, true),
		logger:    logger.New(outBuffer, errBuffer, true, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.NoError(t, err)
	expected := "cmd.list.header\n" +
		"\u274C Mod A (mod-a) [modrinth] cmd.list.entry.missing_suffix\n"
	assert.Equal(t, expected, outBuffer.String())
}

func TestRunListLogsInvalidLockFileName(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{
		{ID: "mod-a", Name: " ", Type: models.MODRINTH, FileName: "mods/mod-a.jar"},
	}))

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(errBuffer)

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, errBuffer, false),
		logger:    logger.New(outBuffer, errBuffer, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.NoError(t, err)
	assert.Contains(t, outBuffer.String(), "cmd.list.error.invalid_filename_lock")
}

func TestRunListInvalidLockWarningWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{
		{ID: "mod-a", Name: " ", Type: models.MODRINTH, FileName: "mods/mod-a.jar"},
	}))

	writeErr := errors.New("write failed")
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(outputLinesModel); ok {
				return outputLinesModel{Err: writeErr}, nil
			}
			return model, nil
		},
	})

	assert.ErrorIs(t, err, writeErr)
}

func TestRunListMissingLockErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(errBuffer)

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, errBuffer, false),
		logger:    logger.New(outBuffer, errBuffer, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.failed")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.lock_missing")
}

func TestRunListMissingLockReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	writeErr := errors.New("write failed")
	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(io.Discard, io.Discard, false),
		logger:    logger.New(io.Discard, io.Discard, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return outputLinesModel{Err: writeErr}, nil
		},
	})

	assert.ErrorIs(t, err, writeErr)
}

func TestRunListInvalidConfigErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, meta.ConfigPath, []byte("{invalid"), 0644))

	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(errBuffer)

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, errBuffer, false),
		logger:    logger.New(outBuffer, errBuffer, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.Error(t, err)
	assert.Contains(t, outBuffer.String(), "cmd.list.error.failed")
}

func TestRunListMissingConfigUnattendedOutputsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(&bytes.Buffer{})

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{unattended: true}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, &bytes.Buffer{}, false),
		logger:    logger.New(outBuffer, &bytes.Buffer{}, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.config_missing")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.config_missing_hint")
}

func TestRunListMissingConfigOutputWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	writeErr := errors.New("write failed")
	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return outputLinesModel{Err: writeErr}, nil
		},
	})

	assert.ErrorIs(t, err, writeErr)
}

func TestRunListMissingConfigDeclinesInit(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetErr(&bytes.Buffer{})

	entriesCount, usedInteractive, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false, canceled: false}, nil
		},
	})

	assert.NoError(t, err)
	assert.True(t, usedInteractive)
	assert.Equal(t, 0, entriesCount)
}

func TestRunListMissingConfigPromptFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetErr(&bytes.Buffer{})

	runErr := errors.New("prompt failed")
	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	})

	assert.ErrorIs(t, err, runErr)
}

func TestRunListMissingConfigRequiresInitRunner(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetErr(&bytes.Buffer{})

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	})

	assert.ErrorContains(t, err, "missing init runner")
}

func TestRunListMissingConfigInitCanceled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetErr(&bytes.Buffer{})

	entriesCount, usedInteractive, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return initCmd.ErrInitCanceled
		},
	})

	assert.NoError(t, err)
	assert.True(t, usedInteractive)
	assert.Equal(t, 0, entriesCount)
}

func TestRunListMissingConfigInitCreatesInvalidConfig(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: outBuffer})
	command.SetErr(&bytes.Buffer{})

	runInit := func(_ context.Context, _ *cobra.Command, request initRequest) error {
		requestMeta := config.NewMetadata(request.configPath)
		if err := fileSystem.MkdirAll(requestMeta.Dir(), 0755); err != nil {
			return err
		}
		return afero.WriteFile(fileSystem, requestMeta.ConfigPath, []byte("{invalid"), 0644)
	}

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(configInitModel); ok {
				return configInitModel{confirmed: true}, nil
			}
			return defaultRunTea(model, options...)
		},
		runInit: runInit,
	})

	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.failed")
}

func TestRunListMissingConfigInitCreatesMissingLock(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: outBuffer})
	command.SetErr(&bytes.Buffer{})

	runInit := func(ctx context.Context, _ *cobra.Command, request initRequest) error {
		cfg := models.ModsJSON{
			Loader:                     models.FABRIC,
			GameVersion:                "1.20.1",
			DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
			ModsFolder:                 "mods",
			Mods:                       []models.Mod{},
		}
		requestMeta := config.NewMetadata(request.configPath)
		if err := fileSystem.MkdirAll(requestMeta.Dir(), 0755); err != nil {
			return err
		}
		if err := fileSystem.MkdirAll(requestMeta.ModsFolderPath(cfg), 0755); err != nil {
			return err
		}
		return config.WriteConfig(ctx, fileSystem, requestMeta, cfg)
	}

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(configInitModel); ok {
				return configInitModel{confirmed: true}, nil
			}
			return defaultRunTea(model, options...)
		},
		runInit: runInit,
	})

	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.lock_missing")
}

func TestRunListMissingConfigInitFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetErr(&bytes.Buffer{})

	runErr := errors.New("init failed")
	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return runErr
		},
	})

	assert.ErrorIs(t, err, runErr)
}

func TestRunListMissingConfigRunsInitAndContinues(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetErr(&bytes.Buffer{})

	runInit := func(ctx context.Context, _ *cobra.Command, request initRequest) error {
		cfg := models.ModsJSON{
			Loader:                     models.FABRIC,
			GameVersion:                "1.20.1",
			DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
			ModsFolder:                 "mods",
			Mods:                       []models.Mod{},
		}
		requestMeta := config.NewMetadata(request.configPath)
		if err := fileSystem.MkdirAll(requestMeta.Dir(), 0755); err != nil {
			return err
		}
		if err := fileSystem.MkdirAll(requestMeta.ModsFolderPath(cfg), 0755); err != nil {
			return err
		}
		if err := config.WriteConfig(ctx, fileSystem, requestMeta, cfg); err != nil {
			return err
		}
		return config.WriteLock(ctx, fileSystem, requestMeta, []models.ModInstall{})
	}

	outBuffer := &bytes.Buffer{}
	command.SetOut(fakeTTY{Buffer: outBuffer})

	entriesCount, usedInteractive, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(configInitModel); ok {
				return configInitModel{confirmed: true}, nil
			}
			return defaultRunTea(model, options...)
		},
		runInit: runInit,
	})

	assert.NoError(t, err)
	assert.True(t, usedInteractive)
	assert.Equal(t, 0, entriesCount)
	assert.Contains(t, outBuffer.String(), "cmd.list.empty")
}

func TestRunListIncludesUnmanagedNotice(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged-a.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged-b.jar"), []byte("data"), 0644))

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(&bytes.Buffer{})

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{quiet: false}, listDeps{
		fs:        fileSystem,
		output:    output.New(outBuffer, &bytes.Buffer{}, false),
		logger:    logger.New(outBuffer, &bytes.Buffer{}, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.NoError(t, err)
	assert.Contains(t, outBuffer.String(), "cmd.list.unmanaged.header")
	assert.Contains(t, outBuffer.String(), "cmd.list.unmanaged.cta")
	assert.Contains(t, outBuffer.String(), "unmanaged-a.jar")
	assert.Contains(t, outBuffer.String(), "unmanaged-b.jar")
}

func TestRunListUnmanagedNoticeOutputWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged-a.jar"), []byte("data"), 0644))

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	writeErr := errors.New("write failed")
	callCount := 0
	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(outputLinesModel); ok {
				callCount++
				if callCount == 2 {
					return outputLinesModel{Err: writeErr}, nil
				}
			}
			return model, nil
		},
	})

	assert.ErrorIs(t, err, writeErr)
}

func TestRunListReturnsListOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{}))

	writeErr := errors.New("write failed")
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(outputLinesModel); ok {
				return outputLinesModel{Err: writeErr}, nil
			}
			return model, nil
		},
	})

	assert.ErrorIs(t, err, writeErr)
}

func TestRunListReturnsModsFolderErrorBeforeOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}

	modPath := filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar")
	fileSystem := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: modPath,
		err:      errors.New("stat failed"),
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{
		{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: "hash"},
	}))

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(outBuffer)
	command.SetErr(&bytes.Buffer{})

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder_hint")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder_hint_secondary")
}

func TestRunListReturnsUnmanagedPartialError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fileSystem := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/cfg/.mmmignore"),
		err:      errors.New("stat failed"),
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fileSystem, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "one.jar"), []byte("data"), 0644))

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	outBuffer := &bytes.Buffer{}
	command.SetOut(outBuffer)
	command.SetErr(&bytes.Buffer{})

	_, _, err := runList(context.Background(), command, meta.ConfigPath, runListOptions{}, listDeps{
		fs:        fileSystem,
		telemetry: func(telemetry.CommandTelemetry) {},
		runTea:    defaultRunTea,
	})

	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder_partial")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder_partial_notice")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder_partial_hint")
}

func TestEntryStatusReturnsMissingWhenHashEmpty(t *testing.T) {
	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}

	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("installed"), 0644))

	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: ""}}
	status, statusErr := entryStatus(mod, lock, meta, cfg, fileSystem)

	assert.NoError(t, statusErr)
	assert.Equal(t, listEntryMissing, status.Status)
}

func TestEntryStatusReturnsErrorWhenHashReadFails(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}

	fileSystem := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"),
		err:      errors.New("open failed"),
	}

	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("installed"), 0644))

	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: "expected"}}
	_, statusErr := entryStatus(mod, lock, meta, cfg, fileSystem)

	assert.Error(t, statusErr)
	var readErr *modsFolderReadError
	assert.ErrorAs(t, statusErr, &readErr)
}

func TestSha1ForFileReturnsErrorOnOpenFailure(t *testing.T) {
	fileSystem := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/mod-a.jar"),
		err:      errors.New("open failed"),
	}

	require.NoError(t, fileSystem.MkdirAll(filepath.FromSlash("/mods"), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.FromSlash("/mods/mod-a.jar"), []byte("installed"), 0644))

	_, err := sha1ForFile(fileSystem, filepath.FromSlash("/mods/mod-a.jar"))
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnReadFailure(t *testing.T) {
	baseFileSystem := afero.NewMemMapFs()
	filePath := filepath.FromSlash("/mods/mod-a.jar")
	require.NoError(t, baseFileSystem.MkdirAll(filepath.FromSlash("/mods"), 0755))
	require.NoError(t, afero.WriteFile(baseFileSystem, filePath, []byte("installed"), 0644))

	fileSystem := readErrorFs{
		Fs:       baseFileSystem,
		failPath: filePath,
		err:      errors.New("read failed"),
	}

	_, err := sha1ForFile(fileSystem, filePath)
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnCloseFailure(t *testing.T) {
	baseFileSystem := afero.NewMemMapFs()
	filePath := filepath.FromSlash("/mods/mod-a.jar")
	require.NoError(t, baseFileSystem.MkdirAll(filepath.FromSlash("/mods"), 0755))
	require.NoError(t, afero.WriteFile(baseFileSystem, filePath, []byte("installed"), 0644))

	fileSystem := closeErrorFs{
		Fs:       baseFileSystem,
		failPath: filePath,
		err:      errors.New("close failed"),
	}

	_, err := sha1ForFile(fileSystem, filePath)
	assert.Error(t, err)
}

func TestRenderListViewReturnsEmptyOnWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(io.Writer, string) error {
		return errors.New("write failed")
	}

	output := renderListView([]listEntry{
		{ID: "mod-a", DisplayName: "Mod A"},
	}, view.ColorDisabled)

	assert.Equal(t, "", output)
}

func TestRenderListViewReturnsEmptyOnNewlineWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	callCount := 0
	view.WriteString = func(io.Writer, string) error {
		callCount++
		if callCount == 2 {
			return errors.New("write failed")
		}
		return nil
	}

	output := renderListView([]listEntry{
		{ID: "mod-a", DisplayName: "Mod A"},
	}, view.ColorDisabled)

	assert.Equal(t, "", output)
}

func TestRenderListViewReturnsEmptyOnEntryWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	callCount := 0
	view.WriteString = func(io.Writer, string) error {
		callCount++
		if callCount == 3 {
			return errors.New("write failed")
		}
		return nil
	}

	output := renderListView([]listEntry{
		{ID: "mod-a", DisplayName: "Mod A"},
	}, view.ColorDisabled)

	assert.Equal(t, "", output)
}

func TestRenderListViewReturnsEmptyOnEntrySeparatorWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	callCount := 0
	view.WriteString = func(io.Writer, string) error {
		callCount++
		if callCount == 4 {
			return errors.New("write failed")
		}
		return nil
	}

	output := renderListView([]listEntry{
		{ID: "mod-a", DisplayName: "Mod A"},
		{ID: "mod-b", DisplayName: "Mod B"},
	}, view.ColorDisabled)

	assert.Equal(t, "", output)
}

func TestRenderListViewEmptyReturnsMessage(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	output := renderListView(nil, view.ColorDisabled)
	assert.Equal(t, "cmd.list.empty", output)
}

func TestReadLockRequiredReturnsErrorOnStatFailure(t *testing.T) {
	fileSystem := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/cfg/modlist-lock.json"),
		err:      errors.New("stat failed"),
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, err := readLockRequired(context.Background(), fileSystem, meta)
	assert.Error(t, err)
}

func TestReadLockRequiredReturnsLockMissingError(t *testing.T) {
	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, err := readLockRequired(context.Background(), fileSystem, meta)
	assert.Error(t, err)

	var missingErr *lockMissingError
	assert.ErrorAs(t, err, &missingErr)
}

func TestReadLockRequiredReturnsReadError(t *testing.T) {
	readErr := errors.New("read failed")
	fileSystem := readErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/cfg/modlist-lock.json"),
		err:      readErr,
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, meta.LockPath(), []byte("{}"), 0644))

	_, err := readLockRequired(context.Background(), fileSystem, meta)
	assert.ErrorIs(t, err, readErr)
}

func TestReadLockRequiredReturnsLockWhenPresent(t *testing.T) {
	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteLock(context.Background(), fileSystem, meta, []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar", Hash: "hash"},
	}))

	lock, err := readLockRequired(context.Background(), fileSystem, meta)
	require.NoError(t, err)
	assert.Len(t, lock, 1)
}

func TestBuildEntriesUsesIDWhenNameBlank(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "mod-a", Name: " ", Type: models.MODRINTH},
		},
	}

	entries, err := buildEntries(cfg, nil, config.NewMetadata("modlist.json"), afero.NewMemMapFs())
	require.NoError(t, err)
	if assert.Len(t, entries, 1) {
		assert.Equal(t, "mod-a", entries[0].DisplayName)
	}
}

func TestBuildEntriesReturnsStatusError(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"},
	}

	failPath := filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar")
	fileSystem := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: failPath,
		err:      errors.New("read failed"),
	}

	_, err := buildEntries(cfg, lock, meta, fileSystem)
	var readErr *modsFolderReadError
	assert.ErrorAs(t, err, &readErr)
}

func TestBuildEntriesSortsByPlatformAndID(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "b", Name: "Alpha", Type: models.MODRINTH},
			{ID: "c", Name: "Alpha", Type: models.CURSEFORGE},
			{ID: "a", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	entries, err := buildEntries(cfg, nil, config.NewMetadata("modlist.json"), afero.NewMemMapFs())
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, models.CURSEFORGE, entries[0].Platform)
	assert.Equal(t, "c", entries[0].ID)
	assert.Equal(t, models.MODRINTH, entries[1].Platform)
	assert.Equal(t, "a", entries[1].ID)
	assert.Equal(t, models.MODRINTH, entries[2].Platform)
	assert.Equal(t, "b", entries[2].ID)
}

func TestIsInstalledReturnsFalseWhenFileNameMissing(t *testing.T) {
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: ""}}

	assert.False(t, isInstalled(mod, lock, config.NewMetadata("modlist.json"), models.ModsJSON{}, afero.NewMemMapFs()))
}

func TestIsInstalledReturnsFalseWhenFileNameInvalid(t *testing.T) {
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mods/mod-a.jar"}}

	assert.False(t, isInstalled(mod, lock, config.NewMetadata("modlist.json"), models.ModsJSON{}, afero.NewMemMapFs()))
}

func TestIsInstalledReturnsFalseWhenStatFails(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "mod-a", Type: models.MODRINTH}
	lock := []models.ModInstall{{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"}}

	failPath := filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar")
	fileSystem := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: failPath,
		err:      errors.New("stat failed"),
	}

	assert.False(t, isInstalled(mod, lock, meta, cfg, fileSystem))
}

func TestListUnmanagedFilesIgnoresManagedAndIgnored(t *testing.T) {
	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.Dir(), ".mmmignore"), []byte("ignored.jar\n"), 0644))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "managed.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "ignored.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	lock := []models.ModInstall{{ID: "managed", FileName: "managed.jar"}}

	files, err := listUnmanagedFiles(fileSystem, meta, cfg, lock)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")}, files)
}

func TestConfigMissingPromptErrorAllowsPrompting(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	err := configMissingPromptError(runListOptions{}, command, meta)
	assert.NoError(t, err)
}

func TestConfigMissingPromptErrorReturnsErrorWithoutTTY(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	err := configMissingPromptError(runListOptions{}, command, meta)
	assert.Error(t, err)
}

func TestListJarFilesReturnsErrorWhenIgnorePatternsFail(t *testing.T) {
	fileSystem := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/cfg/.mmmignore"),
		err:      errors.New("stat failed"),
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "one.jar"), []byte("data"), 0644))

	_, err := listJarFiles(fileSystem, meta, cfg)
	assert.Error(t, err)
}

func TestListJarFilesReturnsErrorWhenReadDirFails(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	fileSystem := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: meta.ModsFolderPath(cfg),
		err:      errors.New("open failed"),
	}

	_, err := listJarFiles(fileSystem, meta, cfg)
	assert.Error(t, err)
}

func TestListJarFilesHandlesMissingIgnoreFile(t *testing.T) {
	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	files, err := listJarFiles(fileSystem, meta, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(meta.ModsFolderPath(cfg), "mod.jar")}, files)
}

func TestRenderUnmanagedNoticeReturnsEmptyForNoFiles(t *testing.T) {
	output := renderUnmanagedNotice(nil, view.ColorDisabled)
	assert.Equal(t, "", output)
}

func TestRenderUnmanagedNoticeReturnsEmptyOnWriteErrors(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})

	for failAt := 1; failAt <= 7; failAt++ {
		callCount := 0
		view.WriteString = func(io.Writer, string) error {
			callCount++
			if callCount == failAt {
				return errors.New("write failed")
			}
			return nil
		}

		output := renderUnmanagedNotice([]string{"one.jar"}, view.ColorDisabled)
		assert.Equal(t, "", output)
	}
}

func TestRenderUnmanagedNoticeReturnsEmptyOnSeparatorWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})

	callCount := 0
	view.WriteString = func(io.Writer, string) error {
		callCount++
		if callCount == 4 {
			return errors.New("write failed")
		}
		return nil
	}

	output := renderUnmanagedNotice([]string{"one.jar", "two.jar"}, view.ColorDisabled)
	assert.Equal(t, "", output)
}

func TestRenderUnmanagedNoticeWithColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)

	output := renderUnmanagedNotice([]string{"one.jar"}, view.ColorEnabled)
	assert.Contains(t, output, "cmd.list.unmanaged.cta")
}

func TestRenderUnmanagedNoticeIncludesMultipleFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	output := renderUnmanagedNotice([]string{"one.jar", "two.jar"}, view.ColorDisabled)
	assert.Contains(t, output, "one.jar")
	assert.Contains(t, output, "two.jar")
}

func TestListJarFilesSkipsDirectoriesAndNonJar(t *testing.T) {
	fileSystem := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fileSystem.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, fileSystem.MkdirAll(filepath.Join(meta.ModsFolderPath(cfg), "subdir"), 0755))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "readme.txt"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fileSystem, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	files, err := listJarFiles(fileSystem, meta, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(meta.ModsFolderPath(cfg), "mod.jar")}, files)
}

func TestColorModeForWriterRespectsTerminalSupport(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	command := &cobra.Command{}
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	assert.Equal(t, view.ColorEnabled, colorModeForWriter(command))
}

func TestColorModeForWriterDisablesWhenNoCommand(t *testing.T) {
	assert.Equal(t, view.ColorDisabled, colorModeForWriter(nil))
}

func TestResolveExecutionModeUsesPromptingSupport(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	command := &cobra.Command{}
	command.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	command.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: false,
		In:         command.InOrStdin(),
		Out:        command.OutOrStdout(),
	})
	assert.Equal(t, interaction.ExecutionModeInteractive, mode)
}

func TestModsFolderReadErrorFormatsMessage(t *testing.T) {
	baseErr := errors.New("read failed")
	readErr := &modsFolderReadError{path: filepath.FromSlash("/mods"), err: baseErr}
	assert.Contains(t, readErr.Error(), "could not read")
	assert.ErrorIs(t, readErr, baseErr)
}

func TestWriteInvalidLockWarningsReturnsError(t *testing.T) {
	writeErr := errors.New("write failed")
	command := &cobra.Command{}
	command.SetOut(errorWriter{err: writeErr})
	command.SetErr(&bytes.Buffer{})

	err := writeInvalidLockWarnings(command, listDeps{runTea: defaultRunTea}, []string{"warning"})
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleListModsFolderFailureFallsBackToGenericError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(outBuffer)
	command.SetErr(&bytes.Buffer{})

	err := handleListModsFolderFailure(command, listDeps{runTea: defaultRunTea}, errors.New("boom"))
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.failed")
}

func TestHandleListModsFolderFailureReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	command := &cobra.Command{}
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	writeErr := errors.New("write failed")
	err := handleListModsFolderFailure(command, listDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return outputLinesModel{Err: writeErr}, nil
		},
	}, &modsFolderReadError{path: filepath.FromSlash("/mods"), err: errors.New("read failed")})
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleListModsFolderPartialFailureFallsBackToGenericError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(outBuffer)
	command.SetErr(&bytes.Buffer{})

	err := handleListModsFolderPartialFailure(command, listDeps{runTea: defaultRunTea}, errors.New("boom"))
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, outBuffer.String(), "cmd.list.error.failed")
}

func TestHandleListModsFolderPartialFailureReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	command := &cobra.Command{}
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	writeErr := errors.New("write failed")
	err := handleListModsFolderPartialFailure(command, listDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return outputLinesModel{Err: writeErr}, nil
		},
	}, &modsFolderReadError{path: filepath.FromSlash("/mods/mod.jar"), err: errors.New("read failed")})
	assert.ErrorIs(t, err, writeErr)
}

func TestWriteListModsFolderFailureColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(fakeTTY{Buffer: outBuffer})
	command.SetErr(&bytes.Buffer{})

	err := writeListModsFolderFailure(command, listDeps{runTea: defaultRunTea}, filepath.FromSlash("/mods"), errors.New("read failed"))
	assert.NoError(t, err)
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder")
}

func TestWriteListModsFolderPartialFailureColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(fakeTTY{Buffer: outBuffer})
	command.SetErr(&bytes.Buffer{})

	err := writeListModsFolderPartialFailure(command, listDeps{runTea: defaultRunTea}, filepath.FromSlash("/mods/mod.jar"), errors.New("read failed"))
	assert.NoError(t, err)
	assert.Contains(t, outBuffer.String(), "cmd.list.error.mods_folder_partial")
}

func TestDefaultListDepsCreatesRunner(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	deps := defaultListDeps(command, listCommandOptions{})
	assert.NotNil(t, deps.fs)
	assert.NotNil(t, deps.runInit)
	assert.NotNil(t, deps.runTea)
}

func TestDefaultListDepsRunInitUsesRunner(t *testing.T) {
	originalRunner := runInteractiveInit
	t.Cleanup(func() {
		runInteractiveInit = originalRunner
	})

	called := false
	runInteractiveInit = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		called = true
		return nil
	}

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})

	deps := defaultListDeps(command, listCommandOptions{})
	err := deps.runInit(context.Background(), command, initRequest{configPath: "/cfg/modlist.json"})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestWriteListFailureOutputWritesHint(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(fakeTTY{Buffer: outBuffer})
	command.SetErr(&bytes.Buffer{})

	err := writeListFailureOutput(command, listDeps{runTea: defaultRunTea}, errors.New("boom"))
	assert.NoError(t, err)
	assert.Contains(t, outBuffer.String(), "cmd.list.error.failed")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.failed_hint")
}

func TestWriteConfigMissingOutputWritesHint(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	outBuffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(fakeTTY{Buffer: outBuffer})
	command.SetErr(&bytes.Buffer{})

	err := writeConfigMissingOutput(command, listDeps{runTea: defaultRunTea}, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
	assert.Contains(t, outBuffer.String(), "cmd.list.error.config_missing")
	assert.Contains(t, outBuffer.String(), "cmd.list.error.config_missing_hint")
}

func TestResolveExecutionModeUnattended(t *testing.T) {
	command := &cobra.Command{}
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: true,
		In:         command.InOrStdin(),
		Out:        command.OutOrStdout(),
	})
	assert.Equal(t, interaction.ExecutionModeUnattended, mode)
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

type openErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

type readErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

type closeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Stat(name)
}

func (filesystem openErrorFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Open(name)
}

type readErrorFile struct {
	afero.File
	err error
}

func (file readErrorFile) Read([]byte) (int, error) {
	return 0, file.err
}

func (filesystem readErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return readErrorFile{File: file, err: filesystem.err}, nil
	}
	return file, nil
}

type closeErrorFile struct {
	afero.File
	err error
}

func (file closeErrorFile) Close() error {
	return file.err
}

func (filesystem closeErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return closeErrorFile{File: file, err: filesystem.err}, nil
	}
	return file, nil
}

type fakeTTY struct {
	*bytes.Buffer
}

func (tty fakeTTY) Fd() uintptr { return 0 }

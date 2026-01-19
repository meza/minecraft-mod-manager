package prune

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
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
	assert.Equal(t, "cmd.prune.no_unmanaged\n\ncmd.prune.summary.success_hint\n", out.String())
	assert.Empty(t, errOut.String())
}

func TestRunPruneUnattendedQuietWithoutForceSkipsDeletion(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
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

	assert.NoError(t, err)
	assert.Equal(t, 0, deletedCount)
	expected := "cmd.prune.header.unmanaged\n❌ extra.jar\n"
	assert.Equal(t, expected, stripANSI(out.String()))
	assert.Empty(t, errOut.String())

	exists, existsErr := afero.Exists(fs, unmanagedPath)
	require.NoError(t, existsErr)
	assert.True(t, exists)
}

func TestRunPrunePromptNoKeepsFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
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
	_, err := input.WriteString("cmd.init.prompt.option.no.short\n")
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
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(pruneConfirmDeleteModel); ok {
				typed.prompt.value = typed.prompt.noOption.short
				typed.prompt.input.SetValue(typed.prompt.value)
				model = typed
			}
			if _, writeErr := fmt.Fprint(outputWriter, model.View()); writeErr != nil {
				return model, writeErr
			}
			return model, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)

	exists, existsErr := afero.Exists(fs, unmanagedPath)
	require.NoError(t, existsErr)
	assert.True(t, exists)

	expected := "cmd.prune.header.unmanaged\n❌ extra.jar\n\n" +
		"? cmd.prune.confirm cmd.init.prompt.confirm.suffix cmd.init.prompt.option.no.short"
	assert.Equal(t, expected, stripANSI(outputWriter.String()))
	assert.Empty(t, errOut.String())
}

func TestRunPrunePromptCanceledSkipsDelete(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
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

	outputWriter := &fakeTerminalWriter{}
	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(outputWriter)
	cmd.SetErr(&bytes.Buffer{})

	deletedCount, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(outputWriter, outputWriter, false, false),
		output: output.New(outputWriter, outputWriter, false),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(pruneConfirmDeleteModel); ok {
				typed.canceled = true
				model = typed
			}
			if _, writeErr := fmt.Fprint(outputWriter, model.View()); writeErr != nil {
				return model, writeErr
			}
			return model, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, deletedCount)

	exists, existsErr := afero.Exists(fs, unmanagedPath)
	require.NoError(t, existsErr)
	assert.True(t, exists)
}

func TestRunPrunePromptYesDeletesFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
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
	_, err := input.WriteString("cmd.init.prompt.option.yes.short\n")
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
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(pruneConfirmDeleteModel); ok {
				results, deleteErr := typed.deleteFn()
				typed.results = results
				typed.deleteErr = deleteErr
				typed.confirmed = true
				typed.done = true
				model = typed
			}
			if _, writeErr := fmt.Fprint(outputWriter, model.View()); writeErr != nil {
				return model, writeErr
			}
			return model, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, deletedCount)
	assert.Contains(t, outputWriter.String(), "cmd.prune.header.deleted")

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

	expectedLine := strings.Join([]string{
		"cmd.lock_sync.header",
		"cmd.lock_sync.header.detail",
		"",
		"\u2754 mod-a (mod-a) [modrinth] cmd.lock_sync.entry.present_suffix, Arg 1: {Count: 0, Data: &map[file:managed.jar]}",
		"",
		"\u23F3 cmd.common.no_changes",
		"-------------------------------------------------------------",
		"cmd.prune.header.deleted",
		"\u2705 unmanaged.jar",
		"",
		"\u2705 cmd.prune.summary.success",
		"cmd.prune.summary.success_hint",
	}, "\n") + "\n"
	assert.Equal(t, expectedLine, stripANSI(out.String()))
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

func TestRunPruneLockSyncReturnsPolicyError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)

	_, err := runPruneLockSync(context.Background(), cmd, pruneOptions{
		LockSync: locksync.PolicyFlags{Add: true, Delete: true},
	}, pruneDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
		runTea: runTeaProgram,
	}, meta, cfg)
	assert.Error(t, err)
}

func TestRunPruneLockSyncStopsOnPromptCancel(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0o755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTY{Buffer: bytes.NewBuffer(nil)})

	state, err := runPruneLockSync(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
			return updated, nil
		},
	}, meta, cfg)
	require.NoError(t, err)
	assert.False(t, state.ShouldContinue)
}

type fakeTTY struct {
	*bytes.Buffer
}

func (tty fakeTTY) Fd() uintptr { return 0 }

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
	assert.True(t, strings.Contains(out.String(), "cmd.prune.error.lock_missing"))
	assert.True(t, strings.Contains(out.String(), meta.LockPath()))
	assert.True(t, strings.Contains(out.String(), "cmd.prune.error.lock_missing_hint"))
	assert.Empty(t, errOut.String())
}

var ansiPattern = regexp.MustCompile("\u001b\\[[0-9;]*[A-Za-z]")

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

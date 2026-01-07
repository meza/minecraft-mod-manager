package prune

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs statErrorFs) Stat(name string) (os.FileInfo, error) {
	if name == fs.failPath {
		return nil, fs.err
	}
	return fs.Fs.Stat(name)
}

type removeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs removeErrorFs) Remove(name string) error {
	if name == fs.failPath {
		return fs.err
	}
	return fs.Fs.Remove(name)
}

func TestApplyPruneCommandErrorPolicy(t *testing.T) {
	cmd := &cobra.Command{}
	applyPruneCommandErrorPolicy(cmd, nil)
	assert.False(t, cmd.SilenceErrors)
	assert.False(t, cmd.SilenceUsage)

	handledErr := clierrors.MarkHandled(errors.New("handled"))
	cmd = &cobra.Command{}
	applyPruneCommandErrorPolicy(cmd, handledErr)
	assert.True(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)

	cmd = &cobra.Command{}
	applyPruneCommandErrorPolicy(cmd, errors.New("unhandled"))
	assert.False(t, cmd.SilenceErrors)
	assert.True(t, cmd.SilenceUsage)
}

func TestRecordPruneTelemetryExitCodes(t *testing.T) {
	var payloads []telemetry.CommandTelemetry
	recorder := func(payload telemetry.CommandTelemetry) {
		payloads = append(payloads, payload)
	}

	recordPruneTelemetry(recorder, pruneOptions{Force: true}, 2, true, nil)
	recordPruneTelemetry(recorder, pruneOptions{}, 0, false, errors.New("boom"))

	require.Len(t, payloads, 2)
	assert.Equal(t, 0, payloads[0].ExitCode)
	assert.Equal(t, 1, payloads[1].ExitCode)
	assert.Equal(t, 2, payloads[0].Extra["deletedCount"])
	assert.True(t, payloads[0].Interactive)
	assert.False(t, payloads[1].Interactive)
}

func TestRunPruneNoUnmanagedReturnsOutputError(t *testing.T) {
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

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	outErr := errors.New("write failed")
	_, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: outErr}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, outErr)
}

func TestShouldDeleteUnmanagedNonInteractiveReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	outErr := errors.New("write failed")
	result, err := shouldDeleteUnmanaged(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: outErr}, nil
		},
	}, interaction.ExecutionModeNonTTY, view.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.False(t, result)
	assert.ErrorIs(t, err, outErr)
}

func TestShouldDeleteUnmanagedInteractiveReturnsPromptError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	promptErr := errors.New("prompt failed")
	result, err := shouldDeleteUnmanaged(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, promptErr
		},
	}, interaction.ExecutionModeInteractive, view.ColorDisabled, []string{"/mods/unmanaged.jar"})

	assert.False(t, result)
	assert.ErrorIs(t, err, promptErr)
}

func TestRunConfigInitPromptUsesRunTeaResult(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	deps := pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(command, deps, config.NewMetadata("/cfg/modlist.json"))
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestConfigInitResultHandlesPointerModel(t *testing.T) {
	model := &configInitModel{confirmed: true, canceled: false}
	confirmed, canceled, err := configInitResult(model)
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunConfigInitPromptReturnsErrorWhenRunTeaFails(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	runErr := errors.New("run tea failed")
	deps := pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}

	_, _, err := runConfigInitPrompt(command, deps, config.NewMetadata("/cfg/modlist.json"))
	assert.ErrorIs(t, err, runErr)
}

func TestRunDeletePromptReturnsErrorWhenRunTeaFails(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	runErr := errors.New("run tea failed")
	deps := pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}

	_, _, err := runDeletePrompt(command, deps, view.ColorDisabled, []string{"/mods/unmanaged.jar"})
	assert.ErrorIs(t, err, runErr)
}

func TestReadLockRequiredReturnsErrorOnInvalidJSON(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, afero.WriteFile(fs, meta.LockPath(), []byte("not-json"), 0644))

	_, err := readLockRequired(context.Background(), fs, meta)
	assert.Error(t, err)
}

func TestRunPruneConfigReadError(t *testing.T) {
	fs := afero.NewMemMapFs()

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	_, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: "/cfg/missing.json"}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.True(t, clierrors.IsHandled(err))
}

func TestEnsurePruneConfigInvalidConfigReturnsHandled(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	_, _, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, meta, interaction.ExecutionModeInteractive)

	assert.True(t, clierrors.IsHandled(err))
}

func TestEnsurePruneConfigMissingWriteConfigMissingOutputError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	_, _, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{Unattended: true}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, meta, interaction.ExecutionModeUnattended)

	assert.ErrorIs(t, err, writeErr)
}

func TestEnsurePruneConfigRunInitError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	runErr := errors.New("init failed")
	_, _, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return runErr
		},
	}, meta, interaction.ExecutionModeInteractive)

	assert.ErrorIs(t, err, runErr)
}

func TestEnsurePruneConfigMissingAfterInitHandled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	_, _, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return nil
		},
	}, meta, interaction.ExecutionModeInteractive)

	assert.True(t, clierrors.IsHandled(err))
}

func TestEnsurePruneConfigLockMissingAfterInitHandled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	_, _, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
			require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
			require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
			return nil
		},
	}, meta, interaction.ExecutionModeInteractive)

	assert.True(t, clierrors.IsHandled(err))
}

func TestRunPruneListUnmanagedError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	_, _, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
	}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.True(t, clierrors.IsHandled(err))
}

func TestRunPruneDeleteError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	wrapped := removeErrorFs{Fs: fs, failPath: unmanagedPath, err: errors.New("remove failed")}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	_, _, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
	}, pruneDeps{
		fs: wrapped,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.True(t, clierrors.IsHandled(err))
}

func TestRunPruneForceInteractiveDeletingRunTeaError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	runErr := errors.New("run tea failed")
	_, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath, Force: true}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, runErr)
}

func TestRunPruneDeleteErrorOutputFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	wrapped := removeErrorFs{Fs: fs, failPath: unmanagedPath, err: errors.New("remove failed")}

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	writeErr := errors.New("write failed")
	_, _, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
	}, pruneDeps{
		fs: wrapped,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, writeErr)
}

func TestRunPruneDeleteSuccessOutputFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	writeErr := errors.New("write failed")
	_, _, err := runPrune(context.Background(), cmd, pruneOptions{
		ConfigPath: meta.ConfigPath,
		Force:      true,
	}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, writeErr)
}

func TestRunPrunePromptDisabledReturnsHandledError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	_, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.True(t, clierrors.IsHandled(err))
}

func TestRunPrunePromptDeclinedSkipsDelete(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	unmanagedPath := filepath.Join(meta.ModsFolderPath(cfg), "extra.jar")
	require.NoError(t, afero.WriteFile(fs, unmanagedPath, []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	deletedCount, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return pruneConfirmDeleteModel{confirmed: false}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, deletedCount)
	exists, statErr := afero.Exists(fs, unmanagedPath)
	require.NoError(t, statErr)
	assert.True(t, exists)
}

func TestRunPrunePromptErrorReturnsError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	promptErr := errors.New("prompt failed")
	_, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, promptErr
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, promptErr)
}

func TestRunPruneConfirmDeleteRunTeaErrorReturnsError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	runErr := errors.New("run tea failed")
	_, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, runErr)
}

func TestRunPruneConfirmDeleteReturnsHandledDeleteError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	deleteErr := errors.New("delete failed")
	_, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: meta.ConfigPath}, pruneDeps{
		fs: fs,
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return pruneConfirmDeleteModel{
				confirmed: true,
				results: []pruneFileResult{
					{Path: filepath.Join(meta.ModsFolderPath(cfg), "extra.jar"), Status: pruneFileStatusFailed, Err: deleteErr},
				},
				deleteErr: deleteErr,
				done:      true,
			}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.True(t, clierrors.IsHandled(err))
	assert.ErrorIs(t, err, deleteErr)
}

func TestRunPruneConfigMissingPromptDeclined(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	deletedCount, _, err := runPrune(context.Background(), cmd, pruneOptions{ConfigPath: "/cfg/modlist.json"}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, deletedCount)
}

func TestEnsurePruneConfigPromptReturnsError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(io.Discard)

	promptErr := errors.New("prompt failed")
	_, _, err := ensurePruneConfig(context.Background(), cmd, pruneOptions{}, pruneDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, promptErr
		},
	}, meta, interaction.ExecutionModeInteractive)

	assert.ErrorIs(t, err, promptErr)
}

func TestHandleLockReadErrorReturnsOriginalError(t *testing.T) {
	original := errors.New("original")
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(io.Discard)

	err := handleLockReadError(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{}, nil
		},
	}, original)
	assert.True(t, clierrors.IsHandled(err))
	assert.ErrorIs(t, err, original)
}

func TestHandleLockReadErrorReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	err := handleLockReadError(&cobra.Command{}, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: writeErr}, nil
		},
	}, &lockMissingError{message: "lock missing"})
	assert.ErrorIs(t, err, writeErr)
}

func TestListUnmanagedFilesReturnsErrorOnMissingModsDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	_, err := listUnmanagedFiles(fs, meta, cfg, []models.ModInstall{})
	assert.Error(t, err)
}

func TestListJarFilesSkipsDirsNonJarAndIgnores(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, fs.MkdirAll(filepath.Join(meta.ModsFolderPath(cfg), "nested"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "note.txt"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "ignored.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("ignored.jar\n"), 0644))

	files, err := listJarFiles(fs, meta, cfg)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), files[0])
}

func TestListJarFilesReturnsAbsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	originalAbs := absPath
	absPath = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	t.Cleanup(func() {
		absPath = originalAbs
	})

	_, err := listJarFiles(fs, meta, cfg)
	assert.Error(t, err)
}

func TestListJarFilesReturnsIgnorePatternError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("data"), 0644))

	wrapped := statErrorFs{
		Fs:       fs,
		failPath: filepath.Join(meta.Dir(), ".mmmignore"),
		err:      errors.New("stat failed"),
	}

	_, err := listJarFiles(wrapped, meta, cfg)
	assert.Error(t, err)
}

func TestDeleteUnmanagedFilesRemoveError(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/unmanaged.jar")
	require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	wrapped := removeErrorFs{Fs: fs, failPath: path, err: errors.New("remove failed")}
	results, err := deleteUnmanagedFiles(pruneDeps{fs: wrapped}, []string{path})

	assert.Error(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, pruneFileStatusFailed, results[0].Status)
}

func TestRemoveFileForceMissingFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	missingPath := filepath.FromSlash("/mods/missing.jar")
	assert.NoError(t, removeFileForce(fs, missingPath))
}

func TestRenderDeleteFailureSummaryUsesErrorStyle(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	summary := renderDeleteFailureSummary(view.ColorEnabled)
	assert.True(t, strings.Contains(summary, view.FinalErrorIcon(view.ColorEnabled)))
}

package update

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRunUpdateQuietNoFailures(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "mod.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "hash",
			DownloadURL: "https://example.invalid/mod.jar",
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("x"), 0644))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	counts, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath, Quiet: true}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Configured",
				FileName:    "mod.jar",
				ReleaseDate: "2024-01-01T00:00:00Z",
				Hash:        "hash",
				DownloadURL: "https://example.invalid/mod.jar",
			}, nil
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, counts.failed)
}

func TestRunUpdateQuietWithFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	counts, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath, Quiet: true}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unexpected fetch")
		},
	})
	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 1, counts.failed)
}

func TestRunUpdateNoModsConfiguredSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{Mods: []models.Mod{}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when no mods are configured")
			return install.Result{}, nil
		},
	})
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.list.empty")
}

func TestRunUpdateNoModsConfiguredReportsUnmanagedFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{Mods: []models.Mod{}, ModsFolder: "mods"}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when no mods are configured")
			return install.Result{}, nil
		},
	})
	assert.ErrorIs(t, err, interaction.ErrUnmanagedFiles)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "cmd.list.unmanaged.header")
	assert.Contains(t, out.String(), "cmd.list.unmanaged.cta")
}

func TestRunUpdateNoModsConfiguredReportsUnmanagedReadError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{Mods: []models.Mod{}, ModsFolder: "mods"}
	require.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))

	fileSystem := openErrorFs{Fs: baseFs, failPath: meta.ModsFolderPath(cfg), err: errors.New("read failed")}

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fileSystem,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when no mods are configured")
			return install.Result{}, nil
		},
	})
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "read failed")
}

func TestRunUpdateNoModsConfiguredUnmanagedReadErrorWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{Mods: []models.Mod{}, ModsFolder: "mods"}
	require.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))

	fileSystem := openErrorFs{Fs: baseFs, failPath: meta.ModsFolderPath(cfg), err: errors.New("read failed")}

	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fileSystem,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when no mods are configured")
			return install.Result{}, nil
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestRunUpdateQuietNoModsConfiguredIsSilent(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{Mods: []models.Mod{}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath, Quiet: true}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when no mods are configured")
			return install.Result{}, nil
		},
	})
	assert.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestRunUpdateStopsWhenInitCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})

	counts, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return initCmd.ErrInitCanceled
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, updateCounts{}, counts)
}

func TestRunUpdateTranscriptErrorPropagation(t *testing.T) {
	original := runUpdateTranscriptProgram
	runUpdateTranscriptProgram = func(*updateTranscriptModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("transcript failed")
	}
	t.Cleanup(func() { runUpdateTranscriptProgram = original })

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "proj-1", Name: "Alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{{ID: "proj-1", Name: "Alpha", Type: models.MODRINTH}}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
	})
	assert.Error(t, err)
}

func TestRunUpdateUsesInteractiveModeWhenPromptingSupported(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "proj-1", Name: "Alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{
		{ID: "proj-1", Name: "Alpha", Type: models.MODRINTH},
	}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unexpected fetch")
		},
	})
	assert.ErrorIs(t, err, errUpdateFailures)
}

func TestEnsureInstallForUpdateHandlesUnmanagedFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	err := ensureInstallForUpdate(context.Background(), cmd, updateOptions{}, updateDeps{
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			_, writeErr := cmd.OutOrStdout().Write([]byte("install output\n"))
			require.NoError(t, writeErr)
			return install.Result{}, clierrors.MarkHandled(interaction.ErrUnmanagedFiles)
		},
	}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, interaction.ErrUnmanagedFiles)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "cmd.update.header.installing_potentially_missing")
}

func TestEnsureInstallForUpdateReturnsOutputErrorOnInstallFailure(t *testing.T) {
	installErr := errors.New("install failed")
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: installErr})

	err := ensureInstallForUpdate(context.Background(), cmd, updateOptions{}, updateDeps{
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, installErr
		},
	}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, installErr)
}

func TestRunUpdateInstallFailureOutputsInstallAndUpdateErrorsNonTTY(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			_, writeErr := cmd.OutOrStdout().Write([]byte("install failed output\n"))
			require.NoError(t, writeErr)
			return install.Result{}, errors.New("install failed")
		},
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "cmd.update.header.installing_potentially_missing")
	assert.Contains(t, out.String(), "install failed output")
	assert.Contains(t, out.String(), "cmd.update.error.install_failed")
}

func TestRunUpdateInstallFailureOutputsInstallAndUpdateErrorsTTY(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &terminalWriter{}
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			_, writeErr := cmd.OutOrStdout().Write([]byte("install failed output\n"))
			require.NoError(t, writeErr)
			return install.Result{}, errors.New("install failed")
		},
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "install failed output")
	assert.Contains(t, out.String(), "cmd.update.error.install_failed")
}

func TestRunUpdateReturnsConfigReadError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{Mods: []models.Mod{}}
	require.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))

	fs := openErrorFs{Fs: baseFs, failPath: meta.ConfigPath, err: errors.New("read failed")}
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			return install.Result{}, nil
		},
	})
	assert.ErrorContains(t, err, "read failed")
}

func TestRunUpdateMissingConfigNonTTYOutputsConfigMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return false })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: "missing.json"}, updateDeps{
		fs: fs,
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when config is missing")
			return install.Result{}, nil
		},
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "cmd.config.error.missing")
	assert.Contains(t, out.String(), "cmd.config.error.missing_hint")
}

func TestRunUpdateMissingConfigInteractiveRunsInitAndContinues(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	out := &terminalWriter{}
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)

	initCalled := false
	deps := updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(ctx context.Context, _ *cobra.Command, request initRequest) error {
			initCalled = true
			cfg := models.ModsJSON{ModsFolder: "mods", Mods: []models.Mod{}}
			if err := fs.MkdirAll(meta.Dir(), 0755); err != nil {
				return err
			}
			return config.WriteConfig(ctx, fs, config.NewMetadata(request.ConfigPath), cfg)
		},
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when no mods are configured")
			return install.Result{}, nil
		},
	}

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)
	assert.True(t, initCalled)
	assert.Contains(t, out.String(), "cmd.list.empty")
}

func TestRunUpdateStopsWhenConfigInitDeclined(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})
	cmd.SetErr(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: "missing.json"}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false, canceled: false}, nil
		},
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when config init is declined")
			return install.Result{}, nil
		},
	})

	assert.NoError(t, err)
}

func TestRunUpdateMissingConfigUnattendedSkipsPrompt(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	out := &terminalWriter{}
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(out)
	cmd.SetErr(io.Discard)

	_, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: "missing.json", Unattended: true}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			t.Fatal("prompt should not run in unattended mode")
			return nil, errors.New("unexpected prompt")
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			t.Fatal("init should not run in unattended mode")
			return nil
		},
		install: func(context.Context, *cobra.Command, install.RunOptions) (install.Result, error) {
			t.Fatal("install should not run when config is missing")
			return install.Result{}, nil
		},
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "cmd.config.error.missing")
	assert.Contains(t, out.String(), "cmd.config.error.missing_hint")
}

func TestEnsureUpdateConfigReturnsErrorWhenInitRunnerMissing(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})

	_, err := ensureUpdateConfig(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}, meta)
	assert.ErrorContains(t, err, "missing init runner")
}

func TestEnsureUpdateConfigStopsWhenInitCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})

	state, err := ensureUpdateConfig(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return initCmd.ErrInitCanceled
		},
	}, meta)

	require.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestEnsureUpdateConfigReturnsPromptError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})

	promptErr := errors.New("prompt failed")
	_, err := ensureUpdateConfig(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return nil, promptErr
		},
	}, meta)
	assert.ErrorIs(t, err, promptErr)
}

func TestEnsureUpdateConfigReturnsInitError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})

	initErr := errors.New("init failed")
	_, err := ensureUpdateConfig(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return initErr
		},
	}, meta)
	assert.ErrorIs(t, err, initErr)
}

func TestEnsureUpdateConfigReturnsOutputErrorWhenConfigMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return false })
	t.Cleanup(restoreTerminal)

	originalRunTeaProgram := runTeaProgram
	runTeaProgram = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return view.OutputLinesModel{Err: errors.New("write failed")}, nil
	}
	t.Cleanup(func() { runTeaProgram = originalRunTeaProgram })

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := ensureUpdateConfig(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
	}, meta)
	assert.ErrorContains(t, err, "write failed")
}

func TestEnsureUpdateConfigReturnsConfigReadErrorAfterInit(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cmd := &cobra.Command{}
	cmd.SetIn(terminalReader{Reader: bytes.NewBuffer(nil)})
	cmd.SetOut(&terminalWriter{})

	_, err := ensureUpdateConfig(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return nil
		},
	}, meta)
	assert.Error(t, err)
}

func TestBuildUpdateItemsMarksPinned(t *testing.T) {
	version := "1.0.0"
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Pinned", Type: models.MODRINTH, Version: &version},
		},
	}
	items, _ := buildUpdateItems(cfg, nil)
	assert.Equal(t, updateItemStatusSkipped, items[0].Status)
}

func TestNotifySkippedItemsWithNilSender(t *testing.T) {
	items := []updateItem{{ConfigIndex: 0, Status: updateItemStatusSkipped}}
	notifySkippedItems(items, updateExecSender{})
}

func TestRunUpdateInteractiveMissingRunner(t *testing.T) {
	original := runUpdateProgram
	runUpdateProgram = nil
	t.Cleanup(func() { runUpdateProgram = original })

	_, err := runUpdateInteractive(context.Background(), &cobra.Command{}, updateExecutionInput{})
	assert.Error(t, err)
}

func TestRunUpdateInteractiveReturnsProgramError(t *testing.T) {
	original := runUpdateProgram
	runUpdateProgram = func(*updateModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("program failed")
	}
	t.Cleanup(func() { runUpdateProgram = original })

	_, err := runUpdateInteractive(context.Background(), &cobra.Command{}, updateExecutionInput{})
	assert.ErrorContains(t, err, "program failed")
}

func TestRunUpdateInteractiveReturnsModelError(t *testing.T) {
	original := runUpdateProgram
	runUpdateProgram = func(*updateModel, ...tea.ProgramOption) (tea.Model, error) {
		return outputLinesModel{}, nil
	}
	t.Cleanup(func() { runUpdateProgram = original })

	_, err := runUpdateInteractive(context.Background(), &cobra.Command{}, updateExecutionInput{})
	assert.Error(t, err)
}

func TestRunUpdateInteractiveReturnsOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	originalProgram := runUpdateProgram
	runUpdateProgram = func(*updateModel, ...tea.ProgramOption) (tea.Model, error) {
		return &updateModel{
			items:     []updateItem{},
			colorMode: view.ColorDisabled,
			outcome:   updateExecutionOutcome{errType: updateExecutionErrorNone},
		}, nil
	}
	t.Cleanup(func() { runUpdateProgram = originalProgram })

	originalTea := runTeaProgram
	runTeaProgram = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return outputLinesModel{Err: errors.New("output failed")}, nil
	}
	t.Cleanup(func() { runTeaProgram = originalTea })

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	_, err := runUpdateInteractive(context.Background(), cmd, updateExecutionInput{
		colorMode: view.ColorDisabled,
	})
	assert.ErrorContains(t, err, "output failed")
}

func TestRunUpdateTranscriptMissingRunner(t *testing.T) {
	original := runUpdateTranscriptProgram
	runUpdateTranscriptProgram = nil
	t.Cleanup(func() { runUpdateTranscriptProgram = original })

	_, err := runUpdateTranscript(context.Background(), &cobra.Command{}, updateExecutionInput{})
	assert.Error(t, err)
}

func TestHandleUpdateOutcomeBranches(t *testing.T) {
	_, err := handleUpdateOutcome(updateExecutionOutcome{
		errType: updateExecutionErrorWriteLock,
		err:     errors.New("lock failed"),
	})
	assert.Error(t, err)

	_, err = handleUpdateOutcome(updateExecutionOutcome{
		errType: updateExecutionErrorWriteConfig,
		err:     errors.New("config failed"),
	})
	assert.Error(t, err)

	_, err = handleUpdateOutcome(updateExecutionOutcome{
		errType: updateExecutionErrorUnknown,
		err:     errors.New("unknown"),
	})
	assert.Error(t, err)

	_, err = handleUpdateOutcome(updateExecutionOutcome{
		errType: updateExecutionErrorCanceled,
		err:     context.Canceled,
	})
	assert.ErrorIs(t, err, context.Canceled)

	_, err = handleUpdateOutcome(updateExecutionOutcome{
		items: []updateItem{{ConfigIndex: 0, Status: updateItemStatusFailed}},
	})
	assert.ErrorIs(t, err, errUpdateFailures)
}

func TestTranscriptSummaryLinesBranches(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	model := newUpdateTranscriptModel(context.Background(), view.ColorDisabled, []updateItem{
		{ConfigIndex: 0, Status: updateItemStatusFailed},
	}, map[int]int{0: 0}, io.Discard, nil)

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorWriteLock, lockPath: "/lock"}
	assert.Contains(t, strings.Join(model.summaryLines(), "\n"), "cmd.update.error.write_lock")

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorWriteConfig, configPath: "/config"}
	assert.Contains(t, strings.Join(model.summaryLines(), "\n"), "cmd.update.error.write_config")

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorUnknown, err: errors.New("boom")}
	assert.Contains(t, strings.Join(model.summaryLines(), "\n"), "boom")

	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorCanceled}
	assert.Nil(t, model.summaryLines())

	model.items = []updateItem{{ConfigIndex: 0, Status: updateItemStatusUpToDate}}
	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorNone}
	lines := model.summaryLines()
	assert.Equal(t, "", lines[0])

	model.items = []updateItem{{ConfigIndex: 0, Status: updateItemStatusUpdating}}
	model.outcome = updateExecutionOutcome{errType: updateExecutionErrorNone}
	lines = model.summaryLines()
	assert.NotEqual(t, "", lines[0])
}

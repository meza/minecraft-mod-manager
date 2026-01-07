package install

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestPrepareInstallRunStateUsesExistingConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "."}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	loadedCfg, readErr := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, readErr)
	require.Equal(t, cfg.ModsFolder, loadedCfg.ModsFolder)
	_, statErr := fs.Stat(meta.ModsFolderPath(cfg))
	require.NoError(t, statErr)
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	runState, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, nil))
	require.NoError(t, err)
	assert.True(t, runState.shouldContinue)
	assert.Equal(t, cfg.ModsFolder, runState.cfg.ModsFolder)
}

func TestPrepareInstallRunStateHandlesConfigReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	require.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{invalid"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, nil))
	assert.True(t, clierrors.IsHandled(err))
}

func TestPrepareInstallRunStateReturnsEnsureLockError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	readOnlyFs := afero.NewReadOnlyFs(baseFs)

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(readOnlyFs, nil))
	assert.True(t, clierrors.IsHandled(err))
}

func TestPrepareInstallRunStateReturnsPromptErrorWhenUnattended(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath, Unattended: true}, baseInstallDeps(fs, nil))
	assert.True(t, clierrors.IsHandled(err))
}

func TestPrepareInstallRunStateReturnsPromptErrorWhenNonTTY(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, nil))
	assert.True(t, clierrors.IsHandled(err))
}

func TestPrepareInstallRunStateReturnsConfigMissingOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	writeErr := errors.New("write failed")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	deps := baseInstallDeps(fs, nil)
	deps.runTea = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return view.OutputLinesModel{Err: writeErr}, nil
	}

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath, Unattended: true}, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestPrepareInstallRunStateStopsOnPromptCancel(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	runState, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, configInitModel{canceled: true}))
	require.NoError(t, err)
	assert.False(t, runState.shouldContinue)
}

func TestPrepareInstallRunStateStopsOnPromptDecline(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	runState, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, configInitModel{confirmed: false}))
	require.NoError(t, err)
	assert.False(t, runState.shouldContinue)
}

func TestPrepareInstallRunStateErrorsWhenRunInitMissing(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, configInitModel{confirmed: true}))
	assert.ErrorContains(t, err, "missing init runner")
}

func TestPrepareInstallRunStateStopsOnInitCanceled(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, configInitModel{confirmed: true})
	deps.runInit = func(context.Context, *cobra.Command, initRequest) error {
		return initCmd.ErrInitCanceled
	}

	runState, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	require.NoError(t, err)
	assert.False(t, runState.shouldContinue)
}

func TestPrepareInstallRunStateReturnsInitError(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, configInitModel{confirmed: true})
	deps.runInit = func(context.Context, *cobra.Command, initRequest) error {
		return errors.New("boom")
	}

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorContains(t, err, "boom")
}

func TestPrepareInstallRunStateReloadsAfterInit(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, configInitModel{confirmed: true})
	deps.runInit = func(ctx context.Context, _ *cobra.Command, request initRequest) error {
		if request.configPath != meta.ConfigPath {
			return errors.New("unexpected config path")
		}
		return config.WriteConfig(ctx, fs, meta, cfg)
	}

	runState, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	require.NoError(t, err)
	assert.True(t, runState.shouldContinue)
	assert.Equal(t, cfg.ModsFolder, runState.cfg.ModsFolder)
}

func TestPrepareInstallRunStateHandlesConfigMissingAfterInit(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, configInitModel{confirmed: true})
	deps.runInit = func(context.Context, *cobra.Command, initRequest) error {
		return nil
	}

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.True(t, clierrors.IsHandled(err))
}

func TestPrepareInstallRunStateReturnsPromptErrorOnRunTeaFailure(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, nil)
	deps.runTea = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("tea failed")
	}

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorContains(t, err, "tea failed")
}

func TestPrepareInstallRunStateHandlesEnsureLockFailureAfterInit(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	fs := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}
	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, configInitModel{confirmed: true})
	deps.runInit = func(ctx context.Context, _ *cobra.Command, request initRequest) error {
		if request.configPath != meta.ConfigPath {
			return errors.New("unexpected config path")
		}
		return config.WriteConfig(ctx, fs, meta, cfg)
	}

	_, err := prepareInstallRunState(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.True(t, clierrors.IsHandled(err))
}

func TestConfigMissingPromptErrorReturnsNilWhenInteractive(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	err := configMissingPromptError(installOptions{}, cmd, meta)
	assert.NoError(t, err)
}

func TestHandleInstallFailureMarksHandled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	deps := baseInstallDeps(afero.NewMemMapFs(), nil)
	err := handleInstallFailure(cmd, deps, errors.New("boom"))
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandleInstallFailureReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	deps := baseInstallDeps(afero.NewMemMapFs(), nil)
	deps.runTea = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return view.OutputLinesModel{Err: errors.New("write failed")}, nil
	}

	err := handleInstallFailure(cmd, deps, errors.New("boom"))
	assert.ErrorContains(t, err, "write failed")
}

func TestWriteInstallFailureOutputWithColor(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(func() {
		restoreColor()
		restoreTerminal()
	})

	t.Setenv("MMM_TEST", "true")
	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	err := writeInstallFailureOutput(cmd, baseInstallDeps(afero.NewMemMapFs(), nil), errors.New("boom"))
	assert.NoError(t, err)
}

func TestWriteConfigMissingOutputWithColor(t *testing.T) {
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(func() {
		restoreColor()
		restoreTerminal()
	})

	t.Setenv("MMM_TEST", "true")
	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	err := writeConfigMissingOutput(cmd, baseInstallDeps(afero.NewMemMapFs(), nil), config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestWriteInstallUnresolvedOutputVariants(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	deps := baseInstallDeps(afero.NewMemMapFs(), nil)
	assert.NoError(t, writeInstallUnresolvedOutput(cmd, deps, nil))
	assert.NoError(t, writeInstallUnresolvedOutput(cmd, deps, []string{"line 1"}))
}

func TestRunInstallStopsOnCanceledPrompt(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, configInitModel{canceled: true})
	result, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.InstalledCount)
}

func TestRunInstallReturnsPrepareStateError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	require.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{invalid"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, nil))
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunInstallReturnsUnresolvedOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	writeErr := errors.New("write failed")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "foreign.jar")
	require.NoError(t, afero.WriteFile(fs, jarPath, []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(errorWriter{err: writeErr})

	deps := baseInstallDeps(fs, nil)
	deps.runTea = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return view.OutputLinesModel{Err: writeErr}, nil
	}
	deps.curseforgeFingerprint = func(string) uint32 { return 0 }
	deps.curseforgeFingerprintMatch = func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
		return &curseforge.FingerprintResult{}, nil
	}
	deps.curseforgeProjectName = func(context.Context, string, httpclient.Doer) (string, error) {
		return "", nil
	}
	deps.modrinthVersionForSha = func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
		return &modrinth.Version{ProjectID: "proj-1"}, nil
	}
	deps.modrinthProjectTitle = func(context.Context, string, httpclient.Doer) (string, error) {
		return "Configured", nil
	}

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestRunInstallReturnsPreflightOutputError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	writeErr := errors.New("write failed")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "foreign.jar"), []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(errorWriter{err: writeErr})

	deps := baseInstallDeps(fs, nil)
	deps.runTea = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return view.OutputLinesModel{Err: writeErr}, nil
	}
	deps.curseforgeFingerprint = func(string) uint32 { return 1 }
	deps.curseforgeFingerprintMatch = func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
		return &curseforge.FingerprintResult{
			Matches: []curseforge.File{{ProjectID: 123, Fingerprint: 1}},
		}, nil
	}
	deps.curseforgeProjectName = func(context.Context, string, httpclient.Doer) (string, error) {
		return "Foreign", nil
	}
	deps.modrinthVersionForSha = func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
		return nil, &modrinth.VersionNotFoundError{}
	}
	deps.modrinthProjectTitle = func(context.Context, string, httpclient.Doer) (string, error) {
		return "", nil
	}

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestRunInstallHandlesPreflightFailure(t *testing.T) {
	fs := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/cfg/mods"),
		err:      errors.New("open failed"),
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, baseInstallDeps(fs, nil))
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunInstallUsesInteractivePath(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	restoreProgram := runInstallProgram
	runInstallProgram = func(model *installModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = installExecutionOutcome{errType: installExecutionErrorNone}
		return model, nil
	}
	t.Cleanup(func() { runInstallProgram = restoreProgram })

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "."}
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := baseInstallDeps(fs, nil)
	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)
}

func baseInstallDeps(fs afero.Fs, promptModel tea.Model) installDeps {
	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if promptModel != nil {
			return promptModel, nil
		}
		return model, nil
	}

	return installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		output:  output.New(io.Discard, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		runTea:  runTea,
	}
}

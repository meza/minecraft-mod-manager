package change

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGameVersionDefaultsToLatest(t *testing.T) {
	assert.Equal(t, "latest", resolveGameVersion(nil))
	assert.Equal(t, "1.20.4", resolveGameVersion([]string{"1.20.4"}))
}

func TestResolveTargetVersionNoop(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	deps := changeDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return true, nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	version, noop, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "1.20.1"}, deps, runState)
	assert.NoError(t, err)
	assert.True(t, noop)
	assert.Equal(t, "1.20.1", version)
}

func TestResolveTargetVersionNoopSkipsValidation(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	validationCalled := false
	deps := changeDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			validationCalled = true
			return false, errors.New("boom")
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	version, noop, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "1.20.1"}, deps, runState)
	assert.NoError(t, err)
	assert.True(t, noop)
	assert.Equal(t, "1.20.1", version)
	assert.False(t, validationCalled)
}

func TestResolveTargetVersionLatestError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return true, nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	_, _, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "latest"}, deps, runState)
	assert.True(t, clierrors.IsHandled(err))
}

func TestResolveTargetVersionInvalid(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return false, nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	_, _, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "bad"}, deps, runState)
	assert.True(t, clierrors.IsHandled(err))
}

func TestResolveTargetVersionValidationError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return false, errors.New("boom") },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	_, _, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "bad"}, deps, runState)
	assert.True(t, clierrors.IsHandled(err))
}

func TestResolveTargetVersionNoopQuiet(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	deps := changeDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return true, nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	version, noop, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "1.20.1", Quiet: true}, deps, runState)
	assert.NoError(t, err)
	assert.True(t, noop)
	assert.Equal(t, "1.20.1", version)
}

func TestResolveTargetVersionNoopOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	deps := changeDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return true, nil },
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("write failed")
		},
	}

	_, _, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "1.20.1"}, deps, runState)
	assert.Error(t, err)
}

func TestResolveTargetVersionLatestSuccess(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	runState := changeRunState{cfg: models.ModsJSON{GameVersion: "1.20.1"}, meta: meta}

	deps := changeDeps{
		latestVersion: func(context.Context, httpclient.Doer) (string, error) { return "1.21.1", nil },
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	version, noop, err := resolveTargetVersion(context.Background(), cmd, changeOptions{GameVersion: "latest"}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, noop)
	assert.Equal(t, "1.21.1", version)
}

func TestPrepareChangeRunStateMissingConfigUnattended(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	deps := changeDeps{
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{}, &config.ConfigFileNotFoundException{Path: meta.ConfigPath}
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	_, err := prepareChangeRunState(context.Background(), cmd, changeOptions{ConfigPath: meta.ConfigPath, Unattended: true}, deps)
	assert.True(t, clierrors.IsHandled(err))
}

func TestLoadChangeConfigReturnsLockError(t *testing.T) {
	deps := changeDeps{
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{}, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return nil, errors.New("lock failed")
		},
	}

	_, err := loadChangeConfig(context.Background(), deps, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestEnsureChangeConfigHandlesReadError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{}, errors.New("boom")
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}
	_, err := ensureChangeConfig(context.Background(), cmd, changeOptions{}, deps, runState)
	assert.True(t, clierrors.IsHandled(err))
}

func TestConfigMissingPromptErrorAllowsPrompt(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	err := configMissingPromptError(changeOptions{}, cmd, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestConfigMissingPromptErrorUnattended(t *testing.T) {
	cmd := &cobra.Command{}
	err := configMissingPromptError(changeOptions{Unattended: true}, cmd, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestConfigMissingPromptErrorNoTTY(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})
	err := configMissingPromptError(changeOptions{}, cmd, config.NewMetadata("/cfg/modlist.json"))
	assert.Error(t, err)
}

func TestHandleMissingChangeConfigPromptErrorOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("write failed")
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	_, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{Unattended: true}, deps, runState)
	assert.Error(t, err)
}

func TestHandleMissingChangeConfigPromptErrorHandled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	_, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{Unattended: true}, deps, runState)
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandleMissingChangeConfigPromptRunnerError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("prompt failed")
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	_, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{}, deps, runState)
	assert.Error(t, err)
}

func TestHandleMissingChangeConfigCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{canceled: true}, nil
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	state, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{
		LockSync: locksync.PolicyFlags{Skip: true},
	}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestHandleMissingChangeConfigNotConfirmed(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false}, nil
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	state, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{
		LockSync: locksync.PolicyFlags{Skip: true},
	}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestHandleMissingChangeConfigSuccess(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runInitInteractive
	defer func() { runInitInteractive = original }()
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1"}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH}}
	fileSystem := afero.NewMemMapFs()

	deps := changeDeps{
		fs: fileSystem,
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return cfg, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return lock, nil
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}
	runState := changeRunState{meta: meta}

	state, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{
		LockSync: locksync.PolicyFlags{Skip: true},
	}, deps, runState)
	assert.NoError(t, err)
	assert.True(t, state.shouldContinue)
	assert.Equal(t, cfg, state.cfg)
	assert.Equal(t, lock, state.lock)
}

func TestHandleMissingChangeConfigRunsLockSyncAfterInit(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runInitInteractive
	defer func() { runInitInteractive = original }()
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1"}
	lock := []models.ModInstall{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}
	fileSystem := afero.NewMemMapFs()
	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))

	deps := changeDeps{
		fs: fileSystem,
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return cfg, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return lock, nil
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(configInitModel); ok {
				return configInitModel{confirmed: true}, nil
			}
			return model, nil
		},
	}
	runState := changeRunState{
		meta: meta,
		mode: interaction.ExecutionModeNonTTY,
	}

	state, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{
		LockSync: locksync.PolicyFlags{Add: true},
	}, deps, runState)
	require.NoError(t, err)
	require.True(t, state.shouldContinue)
	require.Len(t, state.cfg.Mods, 1)
	assert.Equal(t, "alpha", state.cfg.Mods[0].ID)
	assert.Equal(t, models.MODRINTH, state.cfg.Mods[0].Type)
	assert.Equal(t, lock, state.lock)
}

func TestHandleMissingChangeConfigLockSyncError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runInitInteractive
	defer func() { runInitInteractive = original }()
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1"}
	lock := []models.ModInstall{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}
	fileSystem := afero.NewMemMapFs()
	require.NoError(t, fileSystem.MkdirAll(meta.Dir(), 0755))

	deps := changeDeps{
		fs: fileSystem,
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return cfg, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return lock, nil
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(configInitModel); ok {
				return configInitModel{confirmed: true}, nil
			}
			return model, nil
		},
	}
	runState := changeRunState{
		meta: meta,
		mode: interaction.ExecutionModeNonTTY,
	}

	_, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{
		LockSync: locksync.PolicyFlags{Add: true, Delete: true},
	}, deps, runState)
	assert.Error(t, err)
}

func TestHandleMissingChangeConfigInitCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runInitInteractive
	defer func() { runInitInteractive = original }()
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return initCmd.ErrInitCanceled
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	state, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

func TestHandleMissingChangeConfigInitError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runInitInteractive
	defer func() { runInitInteractive = original }()
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return errors.New("boom")
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	_, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{}, deps, runState)
	assert.Error(t, err)
}

func TestHandleMissingChangeConfigReloadErrorHandled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runInitInteractive
	defer func() { runInitInteractive = original }()
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{}, errors.New("load failed")
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}
	runState := changeRunState{meta: config.NewMetadata("/cfg/modlist.json")}

	_, err := handleMissingChangeConfig(context.Background(), cmd, changeOptions{}, deps, runState)
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunInteractiveChangeHandlesOutcomeError(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()

	runChangeProgram = func(model *changeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = changeOutcome{Stage: changeStageCompatibilityFailed, Err: errCompatibilityFailed}
		return model, nil
	}

	deps := changeDeps{}
	cfg := models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Type: models.MODRINTH}}}
	items, index := buildChangeItems(cfg, changeItemOrderAlphabetical)

	_, err := runInteractiveChange(context.Background(), &cobra.Command{}, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         items,
		indexByKey:    index,
	})
	assert.True(t, clierrors.IsHandled(err))
}

func TestOutputAndHandleChangeErrorReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("write failed")
		},
	}
	err := outputAndHandleChangeError(cmd, deps, "message", errInvalidVersion)
	assert.Error(t, err)
}

func TestHandleChangeFailureMarksHandled(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}
	err := handleChangeFailure(cmd, deps, errors.New("boom"))
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandleChangeFailureReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("write failed")
		},
	}

	err := handleChangeFailure(cmd, deps, errors.New("boom"))
	assert.Error(t, err)
}

func TestWriteConfigMissingOutput(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}
	meta := config.NewMetadata("/cfg/modlist.json")
	assert.NoError(t, writeConfigMissingOutput(cmd, deps, meta))
}

func TestWriteChangeFailureOutputColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreTerminal)
	t.Cleanup(restoreProfile)

	cmd := &cobra.Command{}
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	err := writeChangeFailureOutput(cmd, deps, errors.New("boom"))
	assert.NoError(t, err)
}

func TestWriteConfigMissingOutputColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreTerminal)
	t.Cleanup(restoreProfile)

	cmd := &cobra.Command{}
	cmd.SetOut(fdWriter{fd: 1})
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	err := writeConfigMissingOutput(cmd, deps, config.NewMetadata("/cfg/modlist.json"))
	assert.NoError(t, err)
}

func TestRunChangeNoop(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1"}

	deps := changeDeps{
		fs: afero.NewMemMapFs(),
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return cfg, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return nil, nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return true, nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	result, err := runChange(context.Background(), cmd, changeOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.1",
		Unattended:  true,
	}, deps)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunChangePrepareStateError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")

	deps := changeDeps{
		fs: afero.NewMemMapFs(),
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{}, errors.New("read failed")
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	result, err := runChange(context.Background(), cmd, changeOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.True(t, clierrors.IsHandled(err))
	assert.Equal(t, 1, result.ExitCode)
}

func TestRunChangeResolveVersionError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1"}

	deps := changeDeps{
		fs: afero.NewMemMapFs(),
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return cfg, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return nil, nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return false, nil
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	result, err := runChange(context.Background(), cmd, changeOptions{ConfigPath: meta.ConfigPath, GameVersion: "bad"}, deps)
	assert.True(t, clierrors.IsHandled(err))
	assert.Equal(t, 1, result.ExitCode)
}

func TestRunChangeConfigInitCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(fdWriter{fd: 1})
	meta := config.NewMetadata("/cfg/modlist.json")

	deps := changeDeps{
		fs: afero.NewMemMapFs(),
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return models.ModsJSON{}, &config.ConfigFileNotFoundException{Path: meta.ConfigPath}
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{canceled: true}, nil
		},
	}

	result, err := runChange(context.Background(), cmd, changeOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunChangeWithTargetError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}

	deps := changeDeps{
		fs: afero.NewMemMapFs(),
		readConfig: func(context.Context, afero.Fs, config.Metadata) (models.ModsJSON, error) {
			return cfg, nil
		},
		ensureLock: func(context.Context, afero.Fs, config.Metadata) ([]models.ModInstall, error) {
			return nil, nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return true, nil },
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unsupported")
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
		removeAll: func(afero.Fs, string) error { return nil },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
	}

	result, err := runChange(context.Background(), cmd, changeOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.21.1",
		Unattended:  true,
	}, deps)
	assert.True(t, clierrors.IsHandled(err))
	assert.Equal(t, 1, result.ExitCode)
}

func TestRunChangeWithTargetQuietNoMods(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	deps := changeDeps{
		fs:          afero.NewMemMapFs(),
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeAll:   func(afero.Fs, string) error { return nil },
		mkdirAll:    func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	runState := changeRunState{
		meta: meta,
		cfg:  cfg,
		mode: interaction.ExecutionModeUnattended,
	}

	result, err := runChangeWithTarget(context.Background(), cmd, changeOptions{
		Force: true,
		Quiet: true,
	}, deps, runState, "1.21.1")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunChangeWithTargetInteractive(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()

	runChangeProgram = func(model *changeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = changeOutcome{Stage: changeStageSuccess}
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	deps := changeDeps{
		fs:          afero.NewMemMapFs(),
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeAll:   func(afero.Fs, string) error { return nil },
		mkdirAll:    func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	runState := changeRunState{
		meta: meta,
		cfg:  cfg,
		mode: interaction.ExecutionModeInteractive,
	}

	result, err := runChangeWithTarget(context.Background(), cmd, changeOptions{}, deps, runState, "1.21.1")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunChangeWithTargetInteractiveError(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = nil

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		removeAll: func(afero.Fs, string) error { return nil },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	runState := changeRunState{
		meta: meta,
		cfg:  cfg,
		mode: interaction.ExecutionModeInteractive,
	}

	result, err := runChangeWithTarget(context.Background(), cmd, changeOptions{}, deps, runState, "1.21.1")
	assert.Error(t, err)
	assert.Equal(t, 1, result.ExitCode)
}
func TestRunChangeWithTargetNonInteractive(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	deps := changeDeps{
		fs:          afero.NewMemMapFs(),
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeAll:   func(afero.Fs, string) error { return nil },
		mkdirAll:    func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	runState := changeRunState{
		meta: meta,
		cfg:  cfg,
		mode: interaction.ExecutionModeUnattended,
	}

	result, err := runChangeWithTarget(context.Background(), cmd, changeOptions{}, deps, runState, "1.21.1")
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunChangeWithTargetUsesInteractiveInInteractiveMode(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()

	interactiveRan := false
	runChangeProgram = func(model *changeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		interactiveRan = true
		model.outcome = changeOutcome{Stage: changeStageSuccess}
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	deps := changeDeps{
		fs:          afero.NewMemMapFs(),
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeAll:   func(afero.Fs, string) error { return nil },
		mkdirAll:    func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	runState := changeRunState{
		meta: meta,
		cfg:  cfg,
		mode: interaction.ExecutionModeInteractive,
	}

	result, err := runChangeWithTarget(context.Background(), cmd, changeOptions{}, deps, runState, "1.21.1")
	assert.NoError(t, err)
	assert.True(t, interactiveRan)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunNonInteractiveChangeCompatibilityFailed(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	items, index := buildChangeItems(cfg, changeItemOrderAlphabetical)

	deps := changeDeps{
		fs:      afero.NewMemMapFs(),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unsupported")
		},
		removeAll: func(afero.Fs, string) error { return nil },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	_, err := runNonInteractiveChange(context.Background(), cmd, deps, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         items,
		indexByKey:    index,
	}, interaction.ExecutionModeUnattended)
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunNonInteractiveChangeSuccess(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{},
	}
	deps := changeDeps{
		fs:          afero.NewMemMapFs(),
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeAll:   func(afero.Fs, string) error { return nil },
		mkdirAll:    func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	_, err := runNonInteractiveChange(context.Background(), cmd, deps, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         []changeItem{},
		indexByKey:    map[string]int{},
	}, interaction.ExecutionModeUnattended)
	assert.NoError(t, err)
}

func TestRunQuietChangeSuccess(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{},
	}
	deps := changeDeps{
		fs:          afero.NewMemMapFs(),
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeAll:   func(afero.Fs, string) error { return nil },
		mkdirAll:    func(afero.Fs, string, os.FileMode) error { return nil },
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
	}

	_, err := runQuietChange(context.Background(), cmd, deps, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         []changeItem{},
		indexByKey:    map[string]int{},
	})
	assert.NoError(t, err)
}

func TestRunQuietChangeHandledError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}
	items, index := buildChangeItems(cfg, changeItemOrderAlphabetical)

	deps := changeDeps{
		fs:      afero.NewMemMapFs(),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("unsupported")
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
		removeAll: func(afero.Fs, string) error { return nil },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
	}

	_, err := runQuietChange(context.Background(), cmd, deps, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         items,
		indexByKey:    index,
	})
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunNonInteractiveChangeWriteError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods", Mods: []models.Mod{{ID: "alpha", Type: models.MODRINTH}}}
	items, index := buildChangeItems(cfg, changeItemOrderAlphabetical)

	deps := changeDeps{
		fs:      afero.NewMemMapFs(),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "Alpha", FileName: "alpha.jar", Hash: "abc"}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("write failed")
		},
		removeAll: func(afero.Fs, string) error { return nil },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
	}

	_, err := runNonInteractiveChange(context.Background(), cmd, deps, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         items,
		indexByKey:    index,
	}, interaction.ExecutionModeUnattended)
	assert.Error(t, err)
}

func TestChangeItemOrderForMode(t *testing.T) {
	assert.Equal(t, changeItemOrderAlphabetical, changeItemOrderForMode(interaction.ExecutionModeNonTTY))
	assert.Equal(t, changeItemOrderAlphabetical, changeItemOrderForMode(interaction.ExecutionModeInteractive))
	assert.Equal(t, changeItemOrderAlphabetical, changeItemOrderForMode(interaction.ExecutionModeUnattended))
}

func TestShouldRunInteractiveChangeSkipsWhenUnattended(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	should := shouldRunInteractiveChange(changeOptions{Unattended: true}, interaction.ExecutionModeNonTTY)
	assert.False(t, should)
}

func TestShouldRunInteractiveChangeSkipsWhenQuiet(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	should := shouldRunInteractiveChange(changeOptions{Quiet: true}, interaction.ExecutionModeNonTTY)
	assert.False(t, should)
}

func TestShouldRunInteractiveChangeRequiresInteractiveMode(t *testing.T) {
	should := shouldRunInteractiveChange(changeOptions{}, interaction.ExecutionModeNonTTY)
	assert.False(t, should)
}

func TestShouldRunInteractiveChangeAcceptsInteractiveMode(t *testing.T) {
	should := shouldRunInteractiveChange(changeOptions{}, interaction.ExecutionModeInteractive)
	assert.True(t, should)
}

func TestRunQuietChangeWriteError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods", Mods: []models.Mod{{ID: "alpha", Type: models.MODRINTH}}}
	items, index := buildChangeItems(cfg, changeItemOrderAlphabetical)

	deps := changeDeps{
		fs:      afero.NewMemMapFs(),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "Alpha", FileName: "alpha.jar", Hash: "abc"}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("write failed")
		},
		removeAll: func(afero.Fs, string) error { return nil },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
	}

	_, err := runQuietChange(context.Background(), cmd, deps, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         items,
		indexByKey:    index,
	})
	assert.Error(t, err)
}

func TestRunInteractiveChangeMissingRunner(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = nil

	_, err := runInteractiveChange(context.Background(), &cobra.Command{}, changeExecutionInput{})
	assert.Error(t, err)
}

func TestRunInteractiveChangeRunnerError(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = func(*changeModel, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("boom")
	}

	_, err := runInteractiveChange(context.Background(), &cobra.Command{}, changeExecutionInput{})
	assert.Error(t, err)
}

func TestRunInteractiveChangeRunnerReturnsModelAndError(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = func(model *changeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return model, errors.New("boom")
	}

	_, err := runInteractiveChange(context.Background(), &cobra.Command{}, changeExecutionInput{})
	assert.Error(t, err)
}

func TestRunInteractiveChangeOutcomeFromModelError(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = func(*changeModel, ...tea.ProgramOption) (tea.Model, error) {
		return unexpectedModel{}, nil
	}

	_, err := runInteractiveChange(context.Background(), &cobra.Command{}, changeExecutionInput{})
	assert.Error(t, err)
}

func TestRunInteractiveChangeSuccess(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = func(model *changeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = changeOutcome{Stage: changeStageSuccess}
		return model, nil
	}

	outcome, err := runInteractiveChange(context.Background(), &cobra.Command{}, changeExecutionInput{})
	assert.NoError(t, err)
	assert.Equal(t, changeStageSuccess, outcome.Stage)
}

func TestRunInteractiveChangeExecRunnerInvoked(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = func(model *changeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = model.execRunner(model.ctx, nil)
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{GameVersion: "1.20.1", ModsFolder: "mods"}

	deps := changeDeps{
		fs:          afero.NewMemMapFs(),
		writeConfig: func(context.Context, afero.Fs, config.Metadata, models.ModsJSON) error { return nil },
		writeLock:   func(context.Context, afero.Fs, config.Metadata, []models.ModInstall) error { return nil },
		removeAll:   func(afero.Fs, string) error { return nil },
		mkdirAll:    func(afero.Fs, string, os.FileMode) error { return nil },
	}

	outcome, err := runInteractiveChange(context.Background(), cmd, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         []changeItem{},
		indexByKey:    map[string]int{},
	})
	assert.NoError(t, err)
	assert.Equal(t, changeStageSuccess, outcome.Stage)
}

func TestChangeResultFromOutcome(t *testing.T) {
	items := []changeItem{{DownloadStatus: changeDownloadSucceeded}}
	result := changeResultFromOutcome("1.21.1", items, interaction.ExecutionModeUnattended, changeOutcome{Items: items}, 1)
	assert.Equal(t, 1, result.ExitCode)
	assert.Equal(t, 1, result.DownloadedMods)
}

func TestWriteInteractiveTranscriptIfNeededAlwaysWrites(t *testing.T) {
	model := buildTranscriptTestModel()
	model.windowH = 1

	teaInvoked := false
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			teaInvoked = true
			return model, nil
		},
	}

	err := writeInteractiveTranscriptIfNeeded(&cobra.Command{}, changeExecutionInput{deps: deps}, model)
	assert.NoError(t, err)
	assert.True(t, teaInvoked)
}

func TestWriteInteractiveTranscriptIfNeeded(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	teaInvoked := false
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			teaInvoked = true
			return model, nil
		},
	}

	model := buildTranscriptTestModel()
	model.windowH = 1

	err := writeInteractiveTranscriptIfNeeded(cmd, changeExecutionInput{deps: deps}, model)
	assert.NoError(t, err)
	assert.True(t, teaInvoked)

	teaInvoked = false
	model.windowH = 200
	err = writeInteractiveTranscriptIfNeeded(cmd, changeExecutionInput{deps: deps}, model)
	assert.NoError(t, err)
	assert.True(t, teaInvoked)
}

func TestWriteInteractiveTranscriptIfNeededSkipsNilModel(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	teaInvoked := false
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			teaInvoked = true
			return model, nil
		},
	}

	err := writeInteractiveTranscriptIfNeeded(cmd, changeExecutionInput{deps: deps}, nil)
	assert.NoError(t, err)
	assert.False(t, teaInvoked)
}

func TestWriteInteractiveTranscriptIfNeededSkipsEmptyContent(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	teaInvoked := false
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			teaInvoked = true
			return model, nil
		},
	}

	model := buildEmptyTranscriptTestModel()
	model.stage = changeStageSwitching
	model.windowH = 1

	err := writeInteractiveTranscriptIfNeeded(cmd, changeExecutionInput{deps: deps}, model)
	assert.NoError(t, err)
	assert.False(t, teaInvoked)
}

func TestWriteInteractiveTranscriptIncludesPolicyAnswer(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	var capturedLines []string
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			outputModel, ok := model.(view.OutputLinesModel)
			if ok {
				capturedLines = outputModel.Lines
			}
			return model, nil
		},
	}

	model := buildTranscriptTestModel()
	model.windowH = 1
	model.policyAnswer = "answer line"

	err := writeInteractiveTranscriptIfNeeded(cmd, changeExecutionInput{deps: deps}, model)
	assert.NoError(t, err)
	assert.Contains(t, capturedLines, "answer line")
}

func TestRunInteractiveChangeTranscriptWriteError(t *testing.T) {
	original := runChangeProgram
	defer func() { runChangeProgram = original }()
	runChangeProgram = func(model *changeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.windowH = 1
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, errors.New("write failed")
		},
	}

	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod, DisplayName: mod.Name}}
	indexByKey := map[string]int{changeModKey(mod): 0}

	_, err := runInteractiveChange(context.Background(), cmd, changeExecutionInput{
		deps:          deps,
		targetVersion: "1.21.1",
		items:         items,
		indexByKey:    indexByKey,
	})
	assert.Error(t, err)
}

func buildTranscriptTestModel() *changeModel {
	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}
	items := []changeItem{{Mod: mod, DisplayName: mod.Name}}
	indexByKey := map[string]int{changeModKey(mod): 0}
	return newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.21.1",
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: indexByKey,
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})
}

func buildEmptyTranscriptTestModel() *changeModel {
	return newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.21.1",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome { return changeOutcome{} },
	})
}

func TestRunChangeCommandReturnsFlagError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	called := false
	err := runChangeCommand(cmd, nil, func(context.Context, *cobra.Command, changeOptions, changeDeps) (changeResult, error) {
		called = true
		return changeResult{}, nil
	})
	assert.Error(t, err)
	assert.False(t, called)
}

func TestEnsureChangeConfigReturnsPolicyError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := ensureChangeConfig(context.Background(), cmd, changeOptions{
		LockSync: locksync.PolicyFlags{Add: true, Delete: true},
	}, changeDeps{
		fs:         fs,
		runTea:     runTeaProgram,
		readConfig: config.ReadConfig,
		ensureLock: config.EnsureLock,
	}, changeRunState{
		meta: meta,
		mode: interaction.ExecutionModeNonTTY,
	})
	assert.Error(t, err)
}

func TestEnsureChangeConfigStopsOnPromptCancel(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"}}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(&bytes.Buffer{})

	state, err := ensureChangeConfig(context.Background(), cmd, changeOptions{}, changeDeps{
		fs:         fs,
		readConfig: config.ReadConfig,
		ensureLock: config.EnsureLock,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
			return updated, nil
		},
	}, changeRunState{
		meta: meta,
		mode: interaction.ExecutionModeInteractive,
	})
	require.NoError(t, err)
	assert.False(t, state.shouldContinue)
}

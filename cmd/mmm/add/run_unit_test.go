package add

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestDownloadFailureErrorError(t *testing.T) {
	err := downloadFailureError{err: errors.New("boom")}
	assert.Equal(t, "boom", err.Error())
}

func TestWriteAddFailureOutputWrites(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	buffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buffer)
	cmd.SetErr(buffer)

	deps := addDeps{runTea: runTeaProgram}
	assert.NoError(t, writeAddFailureOutput(cmd, deps, errors.New("boom")))
	assert.Contains(t, buffer.String(), "cmd.add.error.failed")
	assert.Contains(t, buffer.String(), "cmd.add.error.failed_hint")
}

func TestWriteAddFailureOutputColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.ANSI })
	t.Cleanup(restoreColor)

	outputBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTYWriter{Buffer: outputBuffer})
	cmd.SetErr(outputBuffer)

	deps := addDeps{runTea: runTeaProgram}
	assert.NoError(t, writeAddFailureOutput(cmd, deps, errors.New("boom")))
	assert.Contains(t, outputBuffer.String(), "cmd.add.error.failed")
}

func TestHandleAddFailureMarksHandled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	buffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buffer)
	cmd.SetErr(buffer)

	deps := addDeps{runTea: runTeaProgram}
	err := handleAddFailure(cmd, deps, errors.New("boom"))
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandleAddFailureReturnsOutputError(t *testing.T) {
	outputErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	deps := addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(outputLinesModel); ok {
				typed.Err = outputErr
				return typed, nil
			}
			return model, nil
		},
	}

	err := handleAddFailure(cmd, deps, errors.New("boom"))
	assert.ErrorIs(t, err, outputErr)
}

func TestWriteConfigMissingOutputWrites(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	buffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buffer)
	cmd.SetErr(buffer)

	deps := addDeps{runTea: runTeaProgram}
	meta := config.NewMetadata("modlist.json")
	assert.NoError(t, writeConfigMissingOutput(cmd, deps, meta))
	assert.Contains(t, buffer.String(), "cmd.config.error.missing")
	assert.Contains(t, buffer.String(), "cmd.config.error.missing_hint")
}

func TestWriteConfigMissingOutputColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.ANSI })
	t.Cleanup(restoreColor)

	outputBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTYWriter{Buffer: outputBuffer})
	cmd.SetErr(outputBuffer)

	deps := addDeps{runTea: runTeaProgram}
	meta := config.NewMetadata("modlist.json")
	assert.NoError(t, writeConfigMissingOutput(cmd, deps, meta))
	assert.Contains(t, outputBuffer.String(), "cmd.config.error.missing")
}

func TestConfigMissingPromptErrorUnattended(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))

	err := configMissingPromptError(addOptions{Unattended: true}, cmd, config.NewMetadata("modlist.json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.config.error.missing")
}

func TestConfigMissingPromptErrorNoTTY(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})

	err := configMissingPromptError(addOptions{}, cmd, config.NewMetadata("modlist.json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.config.error.missing")
}

func TestLoadAddConfigLockError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	readOnly := afero.NewReadOnlyFs(fs)
	_, err := loadAddConfig(context.Background(), addDeps{fs: readOnly}, meta)
	assert.Error(t, err)
}

func TestEnsureAddConfigCanceled(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{
		meta: config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		mode: interaction.ExecutionModeInteractive,
	}
	deps := addDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.canceled = true
				return typed, nil
			}
			return model, nil
		},
	}

	configState, err := ensureAddConfig(context.Background(), cmd, addOptions{}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, configState.shouldContinue)
}

func TestEnsureAddConfigExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, models.ModsJSON{ModsFolder: "mods"}))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	configState, err := ensureAddConfig(context.Background(), cmd, addOptions{}, addDeps{fs: fs, runTea: runTeaProgram}, addRunState{meta: meta})
	assert.NoError(t, err)
	assert.True(t, configState.shouldContinue)
}

func TestEnsureAddConfigInvalidConfigReturnsHandledError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{invalid"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := ensureAddConfig(context.Background(), cmd, addOptions{}, addDeps{fs: fs, runTea: runTeaProgram}, addRunState{meta: meta})
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandleMissingAddConfigUnattendedWrites(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	buffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(buffer)
	cmd.SetErr(buffer)

	runState := addRunState{meta: config.NewMetadata("modlist.json")}
	_, err := handleMissingAddConfig(context.Background(), cmd, addOptions{Unattended: true}, addDeps{runTea: runTeaProgram}, runState)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, buffer.String(), "cmd.config.error.missing")
}

func TestHandleMissingAddConfigOutputError(t *testing.T) {
	outputErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{meta: config.NewMetadata("modlist.json")}
	_, err := handleMissingAddConfig(context.Background(), cmd, addOptions{Unattended: true}, addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(outputLinesModel); ok {
				typed.Err = outputErr
				return typed, nil
			}
			return model, nil
		},
	}, runState)
	assert.ErrorIs(t, err, outputErr)
}

func TestHandleMissingAddConfigInitCanceled(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)
	originalRunInit := runInitInteractive
	t.Cleanup(func() { runInitInteractive = originalRunInit })
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return initCmd.ErrInitCanceled
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{meta: config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))}
	deps := addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.confirmed = true
				return typed, nil
			}
			return model, nil
		},
	}

	configState, err := handleMissingAddConfig(context.Background(), cmd, addOptions{}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, configState.shouldContinue)
}

func TestHandleMissingAddConfigPromptError(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{meta: config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))}
	runErr := errors.New("run failed")
	_, err := handleMissingAddConfig(context.Background(), cmd, addOptions{}, addDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}, runState)
	assert.ErrorIs(t, err, runErr)
}

func TestHandleMissingAddConfigInitError(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)
	originalRunInit := runInitInteractive
	t.Cleanup(func() { runInitInteractive = originalRunInit })
	runErr := errors.New("init failed")
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return runErr
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{meta: config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))}
	deps := addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.confirmed = true
				return typed, nil
			}
			return model, nil
		},
	}

	_, err := handleMissingAddConfig(context.Background(), cmd, addOptions{}, deps, runState)
	assert.ErrorIs(t, err, runErr)
}

func TestHandleMissingAddConfigDeclined(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{meta: config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))}
	deps := addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.confirmed = false
				return typed, nil
			}
			return model, nil
		},
	}

	configState, err := handleMissingAddConfig(context.Background(), cmd, addOptions{}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, configState.shouldContinue)
}

func TestHandleMissingAddConfigCanceled(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{meta: config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))}
	deps := addDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.canceled = true
				return typed, nil
			}
			return model, nil
		},
	}

	configState, err := handleMissingAddConfig(context.Background(), cmd, addOptions{}, deps, runState)
	assert.NoError(t, err)
	assert.False(t, configState.shouldContinue)
}

func TestHandleMissingAddConfigLoadConfigError(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	originalRunInit := runInitInteractive
	t.Cleanup(func() { runInitInteractive = originalRunInit })
	runInitInteractive = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{meta: config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))}
	deps := addDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			switch typed := model.(type) {
			case configInitModel:
				typed.confirmed = true
				return typed, nil
			case outputLinesModel:
				return typed, nil
			default:
				return model, nil
			}
		},
	}

	_, err := handleMissingAddConfig(context.Background(), cmd, addOptions{}, deps, runState)
	assert.True(t, clierrors.IsHandled(err))
}

func TestHandleDownloadFailureNonInteractive(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	buffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buffer)
	cmd.SetErr(buffer)

	runState := addRunState{
		mode: interaction.ExecutionModeNonTTY,
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
	}
	outcome := handleDownloadFailure(cmd, runState, addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
	}, models.MODRINTH, "abc", downloadFailureError{err: errors.New("boom")})

	assert.False(t, outcome.recovered)
	assert.True(t, clierrors.IsHandled(outcome.err))
	assert.Contains(t, buffer.String(), "cmd.add.error.download_failed")
}

func TestHandleDownloadFailureNonInteractiveOutputError(t *testing.T) {
	outputErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	runState := addRunState{
		mode: interaction.ExecutionModeNonTTY,
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
	}
	outcome := handleDownloadFailure(cmd, runState, addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(outputLinesModel); ok {
				typed.Err = outputErr
				return typed, nil
			}
			return model, nil
		},
	}, models.MODRINTH, "abc", downloadFailureError{err: errors.New("boom")})

	assert.False(t, outcome.recovered)
	assert.ErrorIs(t, outcome.err, outputErr)
}

func TestHandleDownloadFailureInteractive(t *testing.T) {
	runState := addRunState{
		mode: interaction.ExecutionModeInteractive,
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome := handleDownloadFailure(cmd, runState, addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return recoveryFlowModel{
				selectedPlatform: models.CURSEFORGE,
				selectedProject:  "xyz",
			}, nil
		},
	}, models.MODRINTH, "abc", downloadFailureError{err: errors.New("boom")})

	assert.True(t, outcome.recovered)
	assert.NoError(t, outcome.err)
	assert.Equal(t, models.CURSEFORGE, outcome.platformValue)
	assert.Equal(t, "xyz", outcome.projectID)
}

func TestHandleDownloadFailurePromptError(t *testing.T) {
	runState := addRunState{
		mode: interaction.ExecutionModeInteractive,
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	runErr := errors.New("boom")
	outcome := handleDownloadFailure(cmd, runState, addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}, models.MODRINTH, "abc", downloadFailureError{err: runErr})

	assert.False(t, outcome.recovered)
	assert.True(t, clierrors.IsHandled(outcome.err))
}

func TestFinishAddQuietReturnsTelemetry(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	runState := addRunState{
		meta: meta,
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		lock: nil,
		mode: interaction.ExecutionModeNonTTY,
		setupCoordinator: modsetup.NewSetupCoordinator(fs, fakeDoer{}, modsetup.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return nil
		})),
	}

	telemetryPayload, err := finishAdd(context.Background(), &cobra.Command{}, addOptions{Quiet: true}, addDeps{
		fs:     fs,
		output: output.New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false),
	}, runState, view.ColorDisabled, resolveStepOutcome{
		resolved: resolvedRemoteMod{platform: models.MODRINTH, projectID: "abc"},
		remoteMod: platform.RemoteMod{
			Name:        "Example",
			FileName:    "mod.jar",
			Hash:        "hash",
			ReleaseDate: "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/mod.jar",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "add", telemetryPayload.Command)
}

func TestFinishAddOutputError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	runState := addRunState{
		meta: meta,
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		lock: nil,
		mode: interaction.ExecutionModeNonTTY,
		setupCoordinator: modsetup.NewSetupCoordinator(fs, fakeDoer{}, modsetup.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return nil
		})),
	}

	outputErr := errors.New("write failed")
	_, err := finishAdd(context.Background(), &cobra.Command{}, addOptions{}, addDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(outputLinesModel); ok {
				typed.Err = outputErr
				return typed, nil
			}
			return model, nil
		},
	}, runState, view.ColorDisabled, resolveStepOutcome{
		resolved: resolvedRemoteMod{platform: models.MODRINTH, projectID: "abc"},
		remoteMod: platform.RemoteMod{
			Name:        "Example",
			FileName:    "mod.jar",
			Hash:        "hash",
			ReleaseDate: "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/mod.jar",
		},
	})

	assert.ErrorIs(t, err, outputErr)
}

func TestFinishAddPersistError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	readOnly := afero.NewReadOnlyFs(fs)
	runState := addRunState{
		meta: meta,
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		lock: nil,
		mode: interaction.ExecutionModeNonTTY,
		setupCoordinator: modsetup.NewSetupCoordinator(readOnly, fakeDoer{}, modsetup.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return nil
		})),
	}

	_, err := finishAdd(context.Background(), &cobra.Command{}, addOptions{Quiet: true}, addDeps{fs: readOnly}, runState, view.ColorDisabled, resolveStepOutcome{
		resolved: resolvedRemoteMod{platform: models.MODRINTH, projectID: "abc"},
		remoteMod: platform.RemoteMod{
			Name:        "Example",
			FileName:    "mod.jar",
			Hash:        "hash",
			ReleaseDate: "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/mod.jar",
		},
	})

	assert.Error(t, err)
}

func TestEnsureRemoteFileInteractiveUnexpectedModel(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(_ *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return outputLinesModel{}, nil
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	installer := modinstall.NewInstaller(fs, modinstall.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return nil
	}))
	_, err := ensureRemoteFileInteractive(ensureRemoteFileInput{
		ctx:              context.Background(),
		installer:        installer,
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		remoteMod:        platform.RemoteMod{Name: "Example", FileName: "mod.jar", Hash: "hash", DownloadURL: "https://example.com/mod.jar"},
		resolvedPlatform: models.MODRINTH,
		resolvedID:       "abc",
		deps:             addDeps{clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0))},
		colorMode:        view.ColorDisabled,
		input:            bytes.NewBuffer(nil),
		output:           bytes.NewBuffer(nil),
	})
	assert.Error(t, err)
}

func TestEnsureRemoteFileInteractiveErrorResult(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(_ *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return &downloadProgressModel{err: errors.New("boom")}, nil
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	installer := modinstall.NewInstaller(fs, modinstall.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return nil
	}))
	_, err := ensureRemoteFileInteractive(ensureRemoteFileInput{
		ctx:              context.Background(),
		installer:        installer,
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		remoteMod:        platform.RemoteMod{Name: "Example", FileName: "mod.jar", Hash: "hash", DownloadURL: "https://example.com/mod.jar"},
		resolvedPlatform: models.MODRINTH,
		resolvedID:       "abc",
		deps:             addDeps{clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0))},
		colorMode:        view.ColorDisabled,
		input:            bytes.NewBuffer(nil),
		output:           bytes.NewBuffer(nil),
	})
	assert.Error(t, err)
}

func TestEnsureRemoteFileInteractiveRunnerError(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(_ *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("runner failed")
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	installer := modinstall.NewInstaller(fs, modinstall.Downloader(func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return nil
	}))
	_, err := ensureRemoteFileInteractive(ensureRemoteFileInput{
		ctx:              context.Background(),
		installer:        installer,
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		remoteMod:        platform.RemoteMod{Name: "Example", FileName: "mod.jar", Hash: "hash", DownloadURL: "https://example.com/mod.jar"},
		resolvedPlatform: models.MODRINTH,
		resolvedID:       "abc",
		deps:             addDeps{clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0))},
		colorMode:        view.ColorDisabled,
		input:            bytes.NewBuffer(nil),
		output:           bytes.NewBuffer(nil),
	})
	assert.Error(t, err)
}

func TestRunResolvedAddRecoversFromNotFound(t *testing.T) {
	runState := addRunState{
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeInteractive,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome, err := runResolvedAdd(context.Background(), nil, cmd, addOptions{}, addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
		logger: logger.New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false, true),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return recoveryFlowModel{selectedPlatform: models.CURSEFORGE, selectedProject: "xyz"}, nil
		},
	}, runState, view.ColorDisabled, addIdentifiers{platformValue: models.MODRINTH, projectID: "abc"})

	assert.NoError(t, err)
	assert.True(t, outcome.recovered)
	assert.Equal(t, models.CURSEFORGE, outcome.platformValue)
	assert.Equal(t, "xyz", outcome.projectID)
}

func TestRunResolvedAddRecoversFromDownloadFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	runState := addRunState{
		meta: meta,
		cfg:  cfg,
		lock: nil,
		mode: interaction.ExecutionModeInteractive,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome, err := runResolvedAdd(context.Background(), nil, cmd, addOptions{Quiet: true}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "mod.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.invalid/mod.jar",
			}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
		logger: logger.New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false, true),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return recoveryFlowModel{selectedPlatform: models.CURSEFORGE, selectedProject: "xyz"}, nil
		},
	}, runState, view.ColorDisabled, addIdentifiers{platformValue: models.MODRINTH, projectID: "abc"})

	assert.NoError(t, err)
	assert.True(t, outcome.recovered)
	assert.Equal(t, models.CURSEFORGE, outcome.platformValue)
	assert.Equal(t, "xyz", outcome.projectID)
}

func TestResolveAddModReturnsError(t *testing.T) {
	runState := addRunState{
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeNonTTY,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome, err := resolveAddMod(context.Background(), nil, cmd, addOptions{}, addDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("boom")
		},
		logger: logger.New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false, true),
	}, runState, addIdentifiers{platformValue: models.MODRINTH, projectID: "abc"})

	assert.Error(t, err)
	assert.Equal(t, models.MODRINTH, outcome.platformValue)
	assert.Equal(t, "abc", outcome.projectID)
}

func TestResolveAddModSuccess(t *testing.T) {
	runState := addRunState{
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeNonTTY,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome, err := resolveAddMod(context.Background(), nil, cmd, addOptions{}, addDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:     "Example",
				FileName: "mod.jar",
			}, nil
		},
		logger: logger.New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false, true),
	}, runState, addIdentifiers{platformValue: models.MODRINTH, projectID: "abc"})

	assert.NoError(t, err)
	assert.False(t, outcome.recovered)
	assert.Equal(t, "Example", outcome.remoteMod.Name)
	assert.Equal(t, models.MODRINTH, outcome.platformValue)
}

func TestResolveAddModRecovery(t *testing.T) {
	runState := addRunState{
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeInteractive,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome, err := resolveAddMod(context.Background(), nil, cmd, addOptions{}, addDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false, true),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return recoveryFlowModel{selectedPlatform: models.CURSEFORGE, selectedProject: "xyz"}, nil
		},
	}, runState, addIdentifiers{platformValue: models.MODRINTH, projectID: "abc"})

	assert.NoError(t, err)
	assert.True(t, outcome.recovered)
	assert.Equal(t, models.CURSEFORGE, outcome.platformValue)
	assert.Equal(t, "xyz", outcome.projectID)
}

func TestRunResolvedAddReturnsError(t *testing.T) {
	runState := addRunState{
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeNonTTY,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runResolvedAdd(context.Background(), nil, cmd, addOptions{}, addDeps{
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("boom")
		},
		logger: logger.New(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false, true),
	}, runState, view.ColorDisabled, addIdentifiers{platformValue: models.MODRINTH, projectID: "abc"})

	assert.Error(t, err)
}

func TestRunAddReturnsWhenConfigMissingCanceled(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	ctx, span := context.Background(), (*perf.Span)(nil)
	deps := addDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if typed, ok := model.(configInitModel); ok {
				typed.canceled = true
				return typed, nil
			}
			return model, nil
		},
		logger: logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		output: output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false),
	}

	telemetryPayload, err := runAdd(ctx, span, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: filepath.FromSlash("/cfg/modlist.json"),
	}, deps)

	assert.NoError(t, err)
	assert.Equal(t, "add", telemetryPayload.Command)
}

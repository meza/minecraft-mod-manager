package add

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type errAfterWriter struct {
	remaining int
	err       error
}

func (writer *errAfterWriter) Write(value []byte) (int, error) {
	if writer.remaining == 0 {
		return 0, writer.err
	}
	writer.remaining--
	return len(value), nil
}

type fakeDoer struct{}

func (fakeDoer) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("fake http doer error")
}

type progressSender struct{}

func (progressSender) Send(tea.Msg) {}

func TestModNameForConfig_ReturnsProjectIDWhenMissing(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	assert.Equal(t, "xyz", modNameForConfig(cfg, models.MODRINTH, "xyz"))
	assert.Equal(t, "Example", modNameForConfig(cfg, models.MODRINTH, "abc"))
}

func TestNormalizePlatform(t *testing.T) {
	assert.Equal(t, models.MODRINTH, normalizePlatform("Modrinth"))
	assert.Equal(t, models.CURSEFORGE, normalizePlatform("CURSEFORGE"))
	assert.Equal(t, models.Platform("unknown"), normalizePlatform("unknown"))
}

func TestNormalizedAddIdentifiers(t *testing.T) {
	platformValue, projectID := normalizedAddIdentifiers(addOptions{Platform: "Modrinth", ProjectID: "abc"})
	assert.Equal(t, models.MODRINTH, platformValue)
	assert.Equal(t, "abc", projectID)
}

func TestNormalizeRemoteModFileNameRejectsInvalid(t *testing.T) {
	remote := platform.RemoteMod{Name: "Example", FileName: "mods/mod.jar"}
	_, err := normalizeRemoteModFileName(remote)
	assert.Error(t, err)
}

func TestNormalizeRemoteModFileNameAcceptsValid(t *testing.T) {
	remote := platform.RemoteMod{Name: "Example", FileName: "mod.jar"}
	normalized, err := normalizeRemoteModFileName(remote)
	assert.NoError(t, err)
	assert.Equal(t, "mod.jar", normalized.FileName)
}

func TestIntegrityErrorMessage(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message, ok := integrityErrorMessage(modinstall.MissingHashError{FileName: "mod.jar"}, "Example")
	assert.True(t, ok)
	assert.Contains(t, message, "cmd.add.error.missing_hash_remote")

	message, ok = integrityErrorMessage(modinstall.HashMismatchError{FileName: "mod.jar"}, "Example")
	assert.True(t, ok)
	assert.Contains(t, message, "cmd.add.error.hash_mismatch")

	message, ok = integrityErrorMessage(modpath.OutsideRootError{ResolvedPath: "/tmp/mod.jar", Root: "/mods"}, "Example")
	assert.True(t, ok)
	assert.Contains(t, message, "cmd.add.error.symlink_outside_mods")

	message, ok = integrityErrorMessage(errors.New("boom"), "Example")
	assert.False(t, ok)
	assert.Empty(t, message)
}

func TestLogFetchFailureReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true)

	err := logFetchFailure(log, models.MODRINTH, "abc", errors.New("fetch failed"))
	assert.ErrorIs(t, err, writeErr)
}

func TestLogFetchFailureLogsWrappedError(t *testing.T) {
	log := logger.New(io.Discard, io.Discard, false, true)
	err := logFetchFailure(log, models.MODRINTH, "abc", modfilename.Error{Value: "mods/mod.jar", Reason: modfilename.ReasonSeparator})
	assert.NoError(t, err)
}

func TestLogFetchFailureReturnsErrorOnInnerLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &errAfterWriter{remaining: 1, err: writeErr}
	log := logger.New(writer, writer, false, true)

	err := logFetchFailure(log, models.MODRINTH, "abc", fmt.Errorf("outer: %w", errors.New("inner")))
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsLogFailureError(t *testing.T) {
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true)

	deps := addDeps{
		logger:  log,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("fetch failed")
		},
	}

	_, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx:           context.Background(),
		cfg:           models.ModsJSON{},
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		mode:          interaction.ExecutionModeNonTTY,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsRemoteOnSuccess(t *testing.T) {
	remote := platform.RemoteMod{Name: "Example"}
	deps := addDeps{
		logger:  logger.New(io.Discard, io.Discard, false, true),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
	}

	resolved, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx:           context.Background(),
		cfg:           models.ModsJSON{},
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		mode:          interaction.ExecutionModeNonTTY,
	})
	assert.NoError(t, err)
	assert.Equal(t, remote, resolved.remoteMod)
}

func TestRetryCountForClient(t *testing.T) {
	assert.Equal(t, 0, retryCountForClient(fakeDoer{}))

	assert.Equal(t, 3, retryCountForClient(&httpclient.RLHTTPClient{}))

	assert.Equal(t, 5, retryCountForClient(&httpclient.RLHTTPClient{
		RetryConfig: &httpclient.RetryConfig{MaxRetries: 5},
	}))
}

func TestFindLockInstallFindsMatch(t *testing.T) {
	lock := []models.ModInstall{
		{Type: models.MODRINTH, ID: "abc"},
	}
	install, found := findLockInstall(lock, models.MODRINTH, "abc")
	assert.True(t, found)
	assert.Equal(t, "abc", install.ID)
}

func TestNormalizeExistingInstallFileNameRejectsInvalid(t *testing.T) {
	input := existingInstallInput{
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		install:       models.ModInstall{FileName: "mods/invalid.jar"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
	}

	_, err := normalizeExistingInstallFileName(input)
	assert.Error(t, err)
}

func TestNormalizeExistingInstallFileNameAcceptsValid(t *testing.T) {
	input := existingInstallInput{
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		install:       models.ModInstall{FileName: "mod.jar"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
	}

	install, err := normalizeExistingInstallFileName(input)
	assert.NoError(t, err)
	assert.Equal(t, "mod.jar", install.FileName)
}

func TestEnsureExistingInstallNonInteractiveSkipsProgress(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           cfg,
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      fs,
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
		mode: interaction.ExecutionModeNonTTY,
	}

	result, err := ensureExistingInstall(input, models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")})
	assert.NoError(t, err)
	assert.Equal(t, modinstall.EnsureReasonAlreadyPresent, result.Reason)
}

func TestEnsureExistingInstallInteractiveReturnsResult(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(model *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.result = modinstall.EnsureResult{Reason: modinstall.EnsureReasonMissing}
		return model, nil
	}

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      afero.NewMemMapFs(),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
		},
		mode: interaction.ExecutionModeInteractive,
		in:   bytes.NewBuffer(nil),
		out:  bytes.NewBuffer(nil),
	}

	result, err := ensureExistingInstall(input, models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")})
	assert.NoError(t, err)
	assert.Equal(t, modinstall.EnsureReasonMissing, result.Reason)
}

func TestEnsureExistingInstallInteractiveReturnsProgressError(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(_ *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return &downloadProgressModel{err: errors.New("boom")}, nil
	}

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      afero.NewMemMapFs(),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
		},
		mode: interaction.ExecutionModeInteractive,
		in:   bytes.NewBuffer(nil),
		out:  bytes.NewBuffer(nil),
	}

	_, err := ensureExistingInstall(input, models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")})
	assert.Error(t, err)
}

func TestEnsureExistingInstallInteractiveUnexpectedModel(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(_ *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return outputLinesModel{}, nil
	}

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      afero.NewMemMapFs(),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
		},
		mode: interaction.ExecutionModeInteractive,
		in:   bytes.NewBuffer(nil),
		out:  bytes.NewBuffer(nil),
	}

	_, err := ensureExistingInstall(input, models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")})
	assert.Error(t, err)
}

func TestEnsureExistingInstallInteractiveRunnerError(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(_ *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("runner failed")
	}

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      afero.NewMemMapFs(),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
		},
		mode: interaction.ExecutionModeInteractive,
		in:   bytes.NewBuffer(nil),
		out:  bytes.NewBuffer(nil),
	}

	_, err := ensureExistingInstall(input, models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")})
	assert.Error(t, err)
}

func TestEnsureExistingInstallInteractiveExecutesDownload(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(model *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		result, downloadErr := model.download(progressSender{})
		if downloadErr != nil {
			model.err = downloadErr
		} else {
			model.result = result
		}
		return model, nil
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           cfg,
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      fs,
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
		},
		mode: interaction.ExecutionModeInteractive,
		in:   bytes.NewBuffer(nil),
		out:  bytes.NewBuffer(nil),
	}

	result, err := ensureExistingInstall(input, models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")})
	assert.NoError(t, err)
	assert.Equal(t, modinstall.EnsureReasonAlreadyPresent, result.Reason)
}

func TestRecordExistingInstallTelemetryReturnsError(t *testing.T) {
	writeErr := errors.New("write failed")
	input := existingInstallInput{
		deps: addDeps{
			logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{},
	}

	_, err := recordExistingInstallTelemetry(input, modinstall.EnsureReasonMissing)
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleExistingInstallNonInteractiveWritesOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	output := &bytes.Buffer{}
	telemetryPayload, err := handleExistingInstall(existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           cfg,
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      fs,
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
			logger: logger.New(io.Discard, io.Discard, false, true),
			runTea: runTeaProgram,
		},
		mode: interaction.ExecutionModeNonTTY,
		out:  output,
	})

	assert.NoError(t, err)
	assert.Equal(t, "add", telemetryPayload.Command)
	assert.Contains(t, output.String(), "cmd.add.success")
}

func TestHandleExistingInstallInteractiveWritesOutput(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(model *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.result = modinstall.EnsureResult{Downloaded: false, Reason: modinstall.EnsureReasonAlreadyPresent}
		return model, nil
	}

	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	output := &bytes.Buffer{}
	telemetryPayload, err := handleExistingInstall(existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           cfg,
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      fs,
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
			logger: logger.New(io.Discard, io.Discard, false, true),
			runTea: runTeaProgram,
		},
		mode: interaction.ExecutionModeInteractive,
		in:   bytes.NewBuffer(nil),
		out:  output,
	})

	assert.NoError(t, err)
	assert.Equal(t, "add", telemetryPayload.Command)
	assert.Contains(t, output.String(), "cmd.add.success")
}

func TestHandleExistingInstallEnsureError(t *testing.T) {
	restore := runProgressProgram
	t.Cleanup(func() { runProgressProgram = restore })
	runProgressProgram = func(_ *downloadProgressModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("runner failed")
	}

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      afero.NewMemMapFs(),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
		},
		mode: interaction.ExecutionModeInteractive,
		in:   bytes.NewBuffer(nil),
		out:  bytes.NewBuffer(nil),
	}

	_, err := handleExistingInstall(input)
	assert.Error(t, err)
}

func TestHandleExistingInstallTelemetryError(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           cfg,
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: true},
		deps: addDeps{
			fs:      fs,
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			logger:  logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		},
		mode: interaction.ExecutionModeNonTTY,
		out:  bytes.NewBuffer(nil),
	}

	_, err := handleExistingInstall(input)
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleExistingInstallInvalidFileName(t *testing.T) {
	input := existingInstallInput{
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		install:       models.ModInstall{FileName: "mods/invalid.jar"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps:          addDeps{},
		mode:          interaction.ExecutionModeNonTTY,
	}

	_, err := handleExistingInstall(input)
	assert.Error(t, err)
}

func TestHandleExistingInstallOutputError(t *testing.T) {
	outputErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("data"), 0644))

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           cfg,
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: false},
		deps: addDeps{
			fs:      fs,
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
			runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
				if typed, ok := model.(outputLinesModel); ok {
					typed.Err = outputErr
					return typed, nil
				}
				return model, nil
			},
		},
		mode: interaction.ExecutionModeNonTTY,
		out:  bytes.NewBuffer(nil),
	}

	_, err := handleExistingInstall(input)
	assert.ErrorIs(t, err, outputErr)
}

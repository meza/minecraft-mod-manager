package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestModNameForConfig_ReturnsProjectIDWhenMissing(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	assert.Equal(t, "xyz", modNameForConfig(cfg, models.MODRINTH, "xyz"))
	assert.Equal(t, "Example", modNameForConfig(cfg, models.MODRINTH, "abc"))
}

func TestDownloadClientPrefersCurseforge(t *testing.T) {
	modrinthClient := platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)).Modrinth
	curseforgeClient := platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)).Curseforge

	clients := platform.Clients{Modrinth: modrinthClient}
	assert.Equal(t, modrinthClient, platform.PreferredDownloadClient(clients))

	clients.Curseforge = curseforgeClient
	assert.Equal(t, curseforgeClient, platform.PreferredDownloadClient(clients))
}

func TestResolveRemoteMod_WrappedErrorReturnsError(t *testing.T) {
	ctx := context.Background()
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	deps := addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(io.Discard, io.Discard, true, false),
		output:  output.New(io.Discard, io.Discard, true),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, fmt.Errorf("outer: %w", errors.New("inner"))
		},
	}

	_, err := resolveRemoteMod(ctx, addResolveInputs{
		ctx:           ctx,
		commandSpan:   nil,
		cfg:           cfg,
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		useTUI:        false,
		in:            strings.NewReader(""),
		out:           io.Discard,
	})
	assert.Error(t, err)
}

func TestResolveUnknownPlatformReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	inputs := addResolveInputs{
		ctx:           context.Background(),
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps: addDeps{
			output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, true),
		},
		useTUI: false,
	}

	_, err := resolveUnknownPlatform(context.Background(), inputs, &platform.UnknownPlatformError{Platform: "unknown"})
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveModNotFoundReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	inputs := addResolveInputs{
		ctx:           context.Background(),
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps: addDeps{
			output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, true),
		},
		useTUI: false,
	}

	_, err := resolveModNotFound(context.Background(), inputs, errors.New("missing"))
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveNoCompatibleFileReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	inputs := addResolveInputs{
		ctx:           context.Background(),
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps: addDeps{
			output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, true),
		},
		useTUI: false,
	}

	_, err := resolveNoCompatibleFile(context.Background(), inputs, errors.New("no file"))
	assert.ErrorIs(t, err, writeErr)
}

func TestLogFetchFailureReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true)

	err := logFetchFailure(log, models.MODRINTH, "abc", errors.New("fetch failed"))
	assert.ErrorIs(t, err, writeErr)
}

func TestLogFetchFailureLogsWrappedError(t *testing.T) {
	log := logger.New(io.Discard, io.Discard, false, true)

	err := logFetchFailure(log, models.MODRINTH, "abc", fmt.Errorf("outer: %w", errors.New("inner")))
	assert.NoError(t, err)
}

func TestLogFetchFailureReturnsNilForSimpleError(t *testing.T) {
	log := logger.New(io.Discard, io.Discard, false, true)

	err := logFetchFailure(log, models.MODRINTH, "abc", errors.New("boom"))
	assert.NoError(t, err)
}

func TestLogFetchFailureReturnsErrorOnInnerLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &errAfterWriter{remaining: 1, err: writeErr}
	log := logger.New(writer, writer, false, true)

	err := logFetchFailure(log, models.MODRINTH, "abc", fmt.Errorf("outer: %w", errors.New("inner")))
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsLogFetchFailureError(t *testing.T) {
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true)

	deps := addDeps{
		logger:  log,
		output:  output.New(io.Discard, io.Discard, true),
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
		useTUI:        false,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsLogFetchFailureErrorAfterInitialDebug(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &errAfterWriter{remaining: 1, err: writeErr}

	deps := addDeps{
		logger:  logger.New(writer, writer, false, true),
		output:  output.New(io.Discard, io.Discard, true),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("fetch failed")
		},
	}

	_, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx: context.Background(),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		useTUI:        false,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := addDeps{
		logger:  logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		output:  output.New(io.Discard, io.Discard, true),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
	}

	_, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx: context.Background(),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		useTUI:        false,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModHandlesModNotFound(t *testing.T) {
	deps := addDeps{
		logger:  logger.New(io.Discard, io.Discard, false, true),
		output:  output.New(io.Discard, io.Discard, true),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
	}

	_, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx:           context.Background(),
		cfg:           models.ModsJSON{},
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		useTUI:        false,
	})
	assert.Error(t, err)
}

func TestResolveRemoteModHandlesUnknownPlatform(t *testing.T) {
	deps := addDeps{
		logger:  logger.New(io.Discard, io.Discard, false, true),
		output:  output.New(io.Discard, io.Discard, true),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.UnknownPlatformError{Platform: "invalid"}
		},
	}

	_, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx:           context.Background(),
		cfg:           models.ModsJSON{},
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		useTUI:        false,
	})
	assert.Error(t, err)
}

func TestResolveRemoteModHandlesNoCompatibleFile(t *testing.T) {
	deps := addDeps{
		logger:  logger.New(io.Discard, io.Discard, false, true),
		output:  output.New(io.Discard, io.Discard, true),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
	}

	_, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx:           context.Background(),
		cfg:           models.ModsJSON{},
		opts:          addOptions{Quiet: true},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		useTUI:        false,
	})
	assert.Error(t, err)
}

func TestResolveRemoteModUsesTuiWhenInteractive(t *testing.T) {
	deps := addDeps{
		logger:  logger.New(io.Discard, io.Discard, false, true),
		output:  output.New(io.Discard, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.UnknownPlatformError{Platform: "invalid"}
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{
				state:            addTUIStateDone,
				remoteMod:        platform.RemoteMod{FileName: "example.jar"},
				resolvedPlatform: models.MODRINTH,
				resolvedProject:  "abc",
			}, nil
		},
	}

	resolved, err := resolveRemoteMod(context.Background(), addResolveInputs{
		ctx:           context.Background(),
		cfg:           models.ModsJSON{},
		opts:          addOptions{Quiet: false},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		deps:          deps,
		useTUI:        true,
		in:            strings.NewReader(""),
		out:           io.Discard,
	})
	assert.NoError(t, err)
	assert.Equal(t, models.MODRINTH, resolved.platform)
	assert.Equal(t, "abc", resolved.projectID)
}

func TestResolveRemoteModReturnsRemoteOnSuccess(t *testing.T) {
	remote := platform.RemoteMod{Name: "Example"}
	deps := addDeps{
		logger:  logger.New(io.Discard, io.Discard, false, true),
		output:  output.New(io.Discard, io.Discard, true),
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
		useTUI:        false,
	})
	assert.NoError(t, err)
	assert.Equal(t, remote, resolved.remoteMod)
	assert.Equal(t, models.MODRINTH, resolved.platform)
	assert.Equal(t, "abc", resolved.projectID)
}

func TestRecordExistingInstallTelemetryReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	input := existingInstallInput{
		deps: addDeps{
			logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{},
		useTUI:        false,
	}

	_, err := recordExistingInstallTelemetry(input, modinstall.EnsureReasonAlreadyPresent)
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleExistingInstallReturnsErrorOnInvalidFilename(t *testing.T) {
	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		install:       models.ModInstall{FileName: "mods/invalid.jar"},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: true},
		deps: addDeps{
			logger: logger.New(io.Discard, io.Discard, false, false),
		},
		useTUI: false,
	}

	_, err := handleExistingInstall(input)
	assert.Error(t, err)
}

func TestHandleExistingInstallReturnsErrorOnEnsureFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: ""},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: true},
		deps: addDeps{
			fs:      fs,
			logger:  logger.New(io.Discard, io.Discard, false, false),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
		useTUI: false,
	}

	_, err := handleExistingInstall(input)
	assert.Error(t, err)
}

func TestHandleExistingInstallReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           models.ModsJSON{ModsFolder: "mods"},
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: true},
		deps: addDeps{
			fs: fs,
			downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
				return afero.WriteFile(filesystems[0], destination, []byte("data"), 0644)
			},
			logger:  logger.New(io.Discard, io.Discard, false, false),
			output:  output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
		useTUI: false,
	}

	_, err := handleExistingInstall(input)
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleExistingInstallReturnsTelemetryOnSuccess(t *testing.T) {
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
			logger:  logger.New(io.Discard, io.Discard, false, true),
			output:  output.New(io.Discard, io.Discard, false),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
		useTUI: false,
	}

	telemetryPayload, err := handleExistingInstall(input)
	assert.NoError(t, err)
	assert.Equal(t, "add", telemetryPayload.Command)
}

func TestHandleExistingInstallLogsHashMismatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod.jar"), []byte("bad"), 0644))

	input := existingInstallInput{
		ctx:           context.Background(),
		meta:          meta,
		cfg:           cfg,
		install:       models.ModInstall{FileName: "mod.jar", DownloadURL: "https://example.invalid", Hash: sha1Hex("data")},
		platformValue: models.MODRINTH,
		projectID:     "abc",
		opts:          addOptions{Quiet: true},
		deps: addDeps{
			fs: fs,
			downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
				return afero.WriteFile(filesystems[0], destination, []byte("data"), 0644)
			},
			logger:  logger.New(io.Discard, io.Discard, false, true),
			output:  output.New(io.Discard, io.Discard, false),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
		useTUI: false,
	}

	_, err := handleExistingInstall(input)
	assert.NoError(t, err)
}

func TestHandleExistingInstallReturnsLoggerError(t *testing.T) {
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
			logger:  logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
			output:  output.New(io.Discard, io.Discard, false),
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
		useTUI: false,
	}

	_, err := handleExistingInstall(input)
	assert.ErrorIs(t, err, writeErr)
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

func TestFinalizeAddReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	setupCoordinator := modsetup.NewSetupCoordinator(fs, nil, nil)
	remoteMod := platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.invalid/example.jar",
		ReleaseDate: "1970-01-01T00:00:00Z",
	}

	_, err := finalizeAdd(finalizeAddInput{
		ctx:              context.Background(),
		meta:             meta,
		cfg:              cfg,
		lock:             nil,
		remoteMod:        remoteMod,
		resolvedPlatform: models.MODRINTH,
		resolvedID:       "abc",
		opts:             addOptions{ConfigPath: meta.ConfigPath},
		setupCoordinator: setupCoordinator,
		output:           output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false),
	})
	assert.ErrorIs(t, err, writeErr)
}

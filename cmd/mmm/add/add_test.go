package add

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	tuiinternal "github.com/meza/minecraft-mod-manager/internal/tui"
)

var noopTelemetry = func(telemetry.CommandTelemetry) {}

func TestRunAdd_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	var telemetryCalled telemetry.CommandTelemetry
	downloaded := false

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	err := runAdd(cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        "abc",
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ string, path string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			downloaded = true
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
		telemetry: func(data telemetry.CommandTelemetry) {
			telemetryCalled = data
		},
	})

	assert.NoError(t, err)
	assert.True(t, downloaded)
	configAfter, err := config.ReadConfig(fs, meta)
	assert.NoError(t, err)
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, "abc", configAfter.Mods[0].ID)
	assert.Equal(t, "Example", configAfter.Mods[0].Name)
	assert.Nil(t, configAfter.Mods[0].AllowVersionFallback)
	assert.Nil(t, configAfter.Mods[0].Version)

	lock, err := config.ReadLock(fs, meta)
	assert.NoError(t, err)
	assert.Len(t, lock, 1)
	assert.Equal(t, "example.jar", lock[0].FileName)

	assert.True(t, telemetryCalled.Success)
	assert.Equal(t, "add", telemetryCalled.Command)
}

func TestRunAdd_DuplicateSkipsWork(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	telemetryCalls := 0
	downloaded := false

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	err := runAdd(cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:       fs,
		clients:  platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:   logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		fetchMod: platform.FetchMod,
		downloader: func(string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			downloaded = true
			return nil
		},
		telemetry: func(_ telemetry.CommandTelemetry) {
			telemetryCalls++
		},
	})

	assert.NoError(t, err)
	assert.False(t, downloaded)
	assert.Equal(t, 1, telemetryCalls)
}

func TestAddCommand_MissingArgsShowsUsage(t *testing.T) {
	cmd := Command()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	output := out.String()
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "add <platform> <id>")
}

func TestRunAdd_UnknownPlatformQuiet(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	errBuf := bytes.NewBuffer(nil)
	cmd.SetErr(errBuf)
	noopTelemetry := func(telemetry.CommandTelemetry) {}

	err := runAdd(cmd, addOptions{
		Platform:   "invalid",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), true, false),
		telemetry: noopTelemetry,
		fetchMod: func(models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.UnknownPlatformError{Platform: "invalid"}
		},
	})

	assert.Error(t, err)
	assert.Contains(t, errBuf.String(), "Unknown platform")
}

func TestRunAdd_ModNotFoundCancelled(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	err := runAdd(cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry: noopTelemetry,
		fetchMod: func(models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{state: addTUIStateAborted}, nil
		},
	})

	assert.NoError(t, err)
	configAfter, _ := config.ReadConfig(fs, meta)
	assert.Len(t, configAfter.Mods, 0)
}

func TestRunAdd_ModNotFoundRetry(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	err := runAdd(cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry: noopTelemetry,
		fetchMod: func(p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: p, ProjectID: id}
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{
				state:            addTUIStateDone,
				resolvedPlatform: models.CURSEFORGE,
				resolvedProject:  "def",
				remoteMod: platform.RemoteMod{
					Name:        "Retry",
					FileName:    "retry.jar",
					Hash:        "hash",
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.com/retry.jar",
				},
			}, nil
		},
		downloader: func(_ string, path string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, _ := config.ReadConfig(fs, meta)
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, "def", configAfter.Mods[0].ID)
	assert.Equal(t, models.CURSEFORGE, configAfter.Mods[0].Type)
}

func TestRunAdd_NoFileRetryAlternate(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))
	noopTelemetry := func(telemetry.CommandTelemetry) {}

	err := runAdd(cmd, addOptions{
		Platform:   "curseforge",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry: noopTelemetry,
		fetchMod: func(p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: p, ProjectID: id}
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{
				state:            addTUIStateDone,
				resolvedPlatform: models.MODRINTH,
				resolvedProject:  "zzz",
				remoteMod: platform.RemoteMod{
					Name:        "Retry",
					FileName:    "retry.jar",
					Hash:        "hash",
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.com/retry.jar",
				},
			}, nil
		},
		downloader: func(_ string, path string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, _ := config.ReadConfig(fs, meta)
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, models.MODRINTH, configAfter.Mods[0].Type)
	assert.Equal(t, "zzz", configAfter.Mods[0].ID)
}

func TestRunAdd_DownloadFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	noopTelemetry := func(telemetry.CommandTelemetry) {}

	err := runAdd(cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry: noopTelemetry,
		fetchMod: func(models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        "abc",
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
	})

	assert.Error(t, err)
	configAfter, _ := config.ReadConfig(fs, meta)
	assert.Len(t, configAfter.Mods, 0)
}

func TestRunAdd_CreatesConfigWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.RemoveAll(meta.ConfigPath))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	minecraft.ClearManifestCache()

	err := runAdd(cmd, addOptions{
		Platform:             "modrinth",
		ProjectID:            "abc",
		ConfigPath:           meta.ConfigPath,
		AllowVersionFallback: true,
	}, addDeps{
		fs:              fs,
		clients:         platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		minecraftClient: manifestDoer{body: `{"latest":{"release":"1.21.1","snapshot":""},"versions":[]}`},
		logger:          logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry:       func(telemetry.CommandTelemetry) {},
		fetchMod: func(models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        "abc",
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ string, path string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, err := config.ReadConfig(fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, "1.21.1", configAfter.GameVersion)
}

type manifestDoer struct {
	body string
}

func (m manifestDoer) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(m.body)),
		Header:     make(http.Header),
	}, nil
}

func TestRunAdd_UnknownPlatformInteractiveRetry(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	err := runAdd(cmd, addOptions{
		Platform:   "invalid",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry: noopTelemetry,
		fetchMod: func(p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.UnknownPlatformError{Platform: "invalid"}
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{
				state:            addTUIStateDone,
				resolvedPlatform: models.CURSEFORGE,
				resolvedProject:  "abc",
				remoteMod: platform.RemoteMod{
					Name:        "Retry",
					FileName:    "retry.jar",
					Hash:        "hash",
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.com/retry.jar",
				},
			}, nil
		},
		downloader: func(_ string, path string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, _ := config.ReadConfig(fs, meta)
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, models.CURSEFORGE, configAfter.Mods[0].Type)
}

type fakeTTYReader struct {
	*bytes.Buffer
}

func (fakeTTYReader) Fd() uintptr { return 0 }

type fakeTTYWriter struct {
	*bytes.Buffer
}

func (fakeTTYWriter) Fd() uintptr { return 1 }

func TestRunAdd_ModNotFoundQuiet(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)

	err := runAdd(cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), true, false),
		telemetry: noopTelemetry,
		fetchMod: func(models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "Mod \"abc\" for modrinth does not exist")
}

func TestRunAdd_FetchOptionsPropagateVersionAndFallback(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	var gotOpts platform.FetchOptions

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	err := runAdd(cmd, addOptions{
		Platform:             "modrinth",
		ProjectID:            "abc",
		ConfigPath:           meta.ConfigPath,
		Version:              "1.2.3",
		AllowVersionFallback: true,
	}, addDeps{
		fs:        fs,
		clients:   platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:    logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry: noopTelemetry,
		fetchMod: func(_ models.Platform, _ string, opts platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			gotOpts = opts
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        "abc",
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ string, path string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, gotOpts.AllowedReleaseTypes)
	assert.True(t, gotOpts.AllowFallback)
	assert.Equal(t, "1.2.3", gotOpts.FixedVersion)
}

func TestRunAdd_TelemetryOnFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(fs, meta, cfg))
	assert.NoError(t, config.WriteLock(fs, meta, nil))

	var telemetryCalled telemetry.CommandTelemetry

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	err := runAdd(cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		telemetry: func(data telemetry.CommandTelemetry) {
			telemetryCalled = data
		},
		fetchMod: func(models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("boom")
		},
	})

	assert.Error(t, err)
	assert.False(t, telemetryCalled.Success)
	assert.Equal(t, "add", telemetryCalled.Command)
}

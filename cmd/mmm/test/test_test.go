package test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"sync"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type noopDoer struct{}

func (n noopDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     http.Header{},
	}, nil
}

func TestCommandHasCorrectUsageAndAliases(t *testing.T) {
	cmd := Command()
	assert.Equal(t, "test [game_version]", cmd.Use)
	assert.Contains(t, cmd.Aliases, "t")
	assert.False(t, cmd.SilenceUsage)
}

func TestExitCode0WhenAllModsSupported(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SupportedMod", Type: models.MODRINTH},
			{ID: "proj-2", Name: "AnotherMod", Type: models.CURSEFORGE},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "TestMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return version == "1.22.0" || version == "1.21.1"
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, out.String(), "cmd.test.success")
}

func TestExitCode1WhenSomeModsUnsupported(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SupportedMod", Type: models.MODRINTH},
			{ID: "proj-2", Name: "UnsupportedMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			if id == "proj-2" {
				return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: p, ProjectID: id}
			}
			return platform.RemoteMod{Name: "SupportedMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	var exitErr *exitCodeError
	assert.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, out.String(), "cmd.test.missing_support_header")
	assert.Contains(t, out.String(), "UnsupportedMod (proj-2)")
	assert.Contains(t, out.String(), "cmd.test.cannot_upgrade")
}

func TestExitCode2WhenVersionMatchesCurrent(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SomeMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.21.1",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when version matches current")
			return platform.RemoteMod{}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	// Exit code 2 is now returned via exitCodeError for proper process exit code propagation
	assert.Error(t, err)
	var exitErr *exitCodeError
	assert.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 2, exitErr.code)
	assert.Equal(t, 2, exitCode)
	assert.Contains(t, out.String(), "cmd.test.same_version")
}

func TestLatestVersionResolution(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SomeMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	fetchedVersion := ""
	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "latest",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			fetchedVersion = opts.GameVersion
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "1.22.0", fetchedVersion)
}

func TestInvalidVersionHandling(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SomeMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "invalid-version",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called for invalid version")
			return platform.RemoteMod{}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return version == "1.22.0" || version == "1.21.1"
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, errInvalidVersion)
}

func TestQuietFlagBehavior(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SomeMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
		Quiet:       true,
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, true, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	// In quiet mode, no output is produced unless forced
	assert.Empty(t, out.String())
}

func TestDebugFlagBehavior(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SomeMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
		Debug:       true,
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, true),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, out.String(), "cmd.test.debug.checking")
}

func TestAllowVersionFallbackHonored(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	trueVal := true
	falseVal := false
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "FallbackMod", Type: models.MODRINTH, AllowVersionFallback: &trueVal},
			{ID: "proj-2", Name: "NoFallbackMod", Type: models.MODRINTH, AllowVersionFallback: &falseVal},
			{ID: "proj-3", Name: "DefaultMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	var fetchOptionsMu sync.Mutex
	fetchOptions := make(map[string]platform.FetchOptions)
	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			fetchOptionsMu.Lock()
			fetchOptions[id] = opts
			fetchOptionsMu.Unlock()
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.True(t, fetchOptions["proj-1"].AllowFallback, "FallbackMod should have AllowFallback=true")
	assert.False(t, fetchOptions["proj-2"].AllowFallback, "NoFallbackMod should have AllowFallback=false")
	assert.False(t, fetchOptions["proj-3"].AllowFallback, "DefaultMod should have AllowFallback=false")
}

func TestPinnedModsAreChecked(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	pinnedVersion := "1.0.0"
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "PinnedMod", Type: models.MODRINTH, Version: &pinnedVersion},
			{ID: "proj-2", Name: "NormalMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	var checkedModsMu sync.Mutex
	checkedMods := make(map[string]bool)
	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			checkedModsMu.Lock()
			checkedMods[id] = true
			checkedModsMu.Unlock()
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.True(t, checkedMods["proj-1"], "Pinned mod should be checked")
	assert.True(t, checkedMods["proj-2"], "Normal mod should be checked")
}

func TestModNotFoundError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "MissingMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: p, ProjectID: id}
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	var exitErr *exitCodeError
	assert.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, out.String(), "MissingMod (proj-1)")
}

func TestConfigFileNotFound(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when config is missing")
			return platform.RemoteMod{}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestLatestVersionResolutionError(t *testing.T) {
	// Per ADR 0006: when latest version manifest is unavailable in non-interactive mode,
	// we cannot determine "latest" and must prompt the user to provide an explicit version.
	// This test verifies the command fails gracefully with an informative error.
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SomeMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "latest",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when latest version resolution fails")
			return platform.RemoteMod{}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "", errors.New("network error")
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, errLatestVersionRequired)
	// Verify user-friendly error message is logged
	assert.Contains(t, errOut.String(), "cmd.test.error.latest_unavailable")
}

func TestEmptyModListSuccess(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called for empty mod list")
			return platform.RemoteMod{}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, out.String(), "cmd.test.success")
}

func TestRunTestReturnsCorrectExitCodeForTelemetry(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "SomeMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	// runTest returns exitCode and err which Command() uses for telemetry:
	// Success = err == nil && exitCode == 0
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	// This verifies telemetry would have Success=true and ExitCode=0
}

func TestParallelModChecks(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Mod1", Type: models.MODRINTH},
			{ID: "proj-2", Name: "Mod2", Type: models.MODRINTH},
			{ID: "proj-3", Name: "Mod3", Type: models.MODRINTH},
			{ID: "proj-4", Name: "Mod4", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	var callCountMu sync.Mutex
	callCount := 0
	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			callCountMu.Lock()
			callCount++
			callCountMu.Unlock()
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, 4, callCount)
}

func TestMixedPlatformMods(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "ModrinthMod", Type: models.MODRINTH},
			{ID: "proj-2", Name: "CurseforgeMod", Type: models.CURSEFORGE},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	var platformCallsMu sync.Mutex
	platformCalls := make(map[models.Platform]int)
	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			platformCallsMu.Lock()
			platformCalls[p]++
			platformCallsMu.Unlock()
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, 1, platformCalls[models.MODRINTH])
	assert.Equal(t, 1, platformCalls[models.CURSEFORGE])
}

func TestCustomAllowedReleaseTypes(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "CustomMod", Type: models.MODRINTH, AllowedReleaseTypes: []models.ReleaseType{models.Alpha, models.Beta}},
			{ID: "proj-2", Name: "DefaultMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	var fetchOptionsMu sync.Mutex
	fetchOptions := make(map[string]platform.FetchOptions)
	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			fetchOptionsMu.Lock()
			fetchOptions[id] = opts
			fetchOptionsMu.Unlock()
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, []models.ReleaseType{models.Alpha, models.Beta}, fetchOptions["proj-1"].AllowedReleaseTypes)
	assert.Equal(t, []models.ReleaseType{models.Release}, fetchOptions["proj-2"].AllowedReleaseTypes)
}

func TestGenericFetchErrorLogsToErrorOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "FailingMod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	exitCode, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("network timeout")
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) bool {
			return true
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	var exitErr *exitCodeError
	assert.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, errOut.String(), "network timeout")
}

func TestFormatMissingModEntryWithColorization(t *testing.T) {
	mod := models.Mod{
		ID:   "test-mod-id",
		Name: "TestMod",
		Type: models.MODRINTH,
	}

	result := formatMissingModEntry(mod, true)

	assert.Contains(t, result, "TestMod")
	assert.Contains(t, result, "test-mod-id")
}

func TestFormatMissingModEntryWithoutColorization(t *testing.T) {
	mod := models.Mod{
		ID:   "test-mod-id",
		Name: "TestMod",
		Type: models.MODRINTH,
	}

	result := formatMissingModEntry(mod, false)

	assert.Equal(t, "❌ TestMod (test-mod-id)", result)
}

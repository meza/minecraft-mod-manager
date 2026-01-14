package test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopDoer struct{}

func (doer noopDoer) Do(*http.Request) (*http.Response, error) {
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
		Unattended:  true,
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		minecraftClient: noopDoer{},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "TestMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return version == "1.22.0" || version == "1.21.1", nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, out.String(), "cmd.test.header")
	assert.Contains(t, out.String(), "cmd.test.section.compatible")
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, result.ExitCode)
	assert.Contains(t, out.String(), "cmd.test.section.compatible")
	assert.Contains(t, out.String(), "cmd.test.section.not_compatible")
	assert.Contains(t, out.String(), "UnsupportedMod (proj-2)")
	assert.Contains(t, out.String(), "cmd.test.summary.unsupported")
	assert.Empty(t, errOut.String())
}

func TestExitCode0WhenVersionMatchesCurrent(t *testing.T) {
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.21.1",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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
	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "latest",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return version == "1.22.0" || version == "1.21.1", nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, errInvalidVersion)
	assert.Contains(t, out.String(), "cmd.test.error.invalid_version")
	assert.Contains(t, out.String(), "cmd.test.error.invalid_version_hint")
}

func TestVersionValidationFailureReturnsError(t *testing.T) {
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
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when version validation fails")
			return platform.RemoteMod{}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return false, errors.New("manifest unavailable")
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, errVersionValidationUnavailable)
	assert.Contains(t, out.String(), "cmd.minecraft.version.unavailable")
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
		Quiet:       true,
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, true, false),
		output: output.New(out, errOut, true),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
		Debug:       true,
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, true),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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
	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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
	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: p, ProjectID: id}
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, result.ExitCode)
	assert.Contains(t, out.String(), "cmd.test.section.not_compatible")
	assert.Contains(t, out.String(), "MissingMod (proj-1)")
	assert.Empty(t, errOut.String())
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
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "cmd.config.error.missing")
	assert.Contains(t, out.String(), "cmd.config.error.missing_hint")
}

func TestLatestVersionResolutionError(t *testing.T) {
	// Per ADR 0006: when latest version manifest is unavailable in unattended mode,
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
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, errLatestVersionRequired)
	assert.Contains(t, out.String(), "cmd.minecraft.version.latest_unavailable")
	assert.Contains(t, out.String(), "cmd.test.error.latest_unavailable_hint")
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "SomeMod"}, nil
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	// runTest returns result.ExitCode and err which Command() uses for telemetry:
	// Success = err == nil && result.ExitCode == 0
	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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
	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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
	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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
	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
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
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
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

	result, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.22.0",
	}, testDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		fetchMod: func(ctx context.Context, p models.Platform, id string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("network timeout")
		},
		latestVersion: func(ctx context.Context, client httpclient.Doer) (string, error) {
			return "1.22.0", nil
		},
		isValidVersion: func(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, result.ExitCode)
	assert.Contains(t, out.String(), "cmd.test.section.inconclusive")
	assert.Contains(t, out.String(), "cmd.platform.error.reason.unknown")
	assert.Empty(t, errOut.String())
}

func TestRenderTestItemLineSupported(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	mod := models.Mod{
		ID:   "test-mod-id",
		Name: "TestMod",
		Type: models.MODRINTH,
	}

	result := renderTestItemLine(view.ColorDisabled, testItem{
		Mod:    mod,
		Status: testItemStatusSupported,
	})

	assert.Equal(t, "✅ TestMod (test-mod-id) [modrinth]", result)
}

func TestRenderTestItemLineInconclusiveIncludesReason(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	mod := models.Mod{
		ID:   "test-mod-id",
		Name: "TestMod",
		Type: models.MODRINTH,
	}

	result := renderTestItemLineWithReason(view.ColorDisabled, testItem{
		Mod:    mod,
		Status: testItemStatusInconclusive,
		Reason: "network error",
	})

	assert.Equal(t, "❔ TestMod (test-mod-id) [modrinth] network error", result)
}

func TestFormatReleaseTypesEmptyReturnsNone(t *testing.T) {
	assert.Equal(t, "none", formatReleaseTypes(nil))
}

func TestFormatReleaseTypesFormatsEntries(t *testing.T) {
	assert.Equal(t, "release,beta", formatReleaseTypes([]models.ReleaseType{models.Release, models.Beta}))
}

func TestFetchFailureDebugEventUsesResponseErrorDetails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	mod := models.Mod{ID: "proj-1", Name: "Example", Type: models.MODRINTH}
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	opts := platform.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       false,
	}

	event, ok := fetchFailureDebugEvent(&httpclient.ResponseError{
		Method:     http.MethodGet,
		URL:        "https://example.invalid",
		StatusCode: http.StatusForbidden,
	}, mod, cfg, "1.20.1", opts)

	assert.True(t, ok)
	assert.Equal(t, logEventKindDebug, event.Kind)
	assert.Contains(t, event.Message, "cmd.test.debug.platform_error")
	assert.Contains(t, event.Message, "status=403")
}

func TestFetchFailureDetailEventUsesResponseErrorDetails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	mod := models.Mod{ID: "proj-1", Name: "Example", Type: models.MODRINTH}

	event, ok := fetchFailureDetailEvent(&httpclient.ResponseError{
		Method:     http.MethodGet,
		URL:        "https://example.invalid",
		StatusCode: http.StatusForbidden,
	}, mod)

	assert.True(t, ok)
	assert.Equal(t, logEventKindError, event.Kind)
	assert.Contains(t, event.Message, "cmd.test.error.platform_details")
	assert.Contains(t, event.Message, "status=403")
}

func TestFetchFailureDetailEventReturnsFalseForNil(t *testing.T) {
	event, ok := fetchFailureDetailEvent(nil, models.Mod{})

	assert.False(t, ok)
	assert.Equal(t, logEvent{}, event)
}

func TestFetchFailureDetailEventSkipsNotFound(t *testing.T) {
	mod := models.Mod{ID: "proj-1", Name: "Example", Type: models.MODRINTH}

	event, ok := fetchFailureDetailEvent(&platform.ModNotFoundError{
		Platform:  models.MODRINTH,
		ProjectID: "proj-1",
	}, mod)

	assert.False(t, ok)
	assert.Equal(t, logEvent{}, event)
}

func TestFetchFailureDetailEventSkipsNoCompatibleFile(t *testing.T) {
	mod := models.Mod{ID: "proj-1", Name: "Example", Type: models.MODRINTH}

	event, ok := fetchFailureDetailEvent(&platform.NoCompatibleFileError{
		Platform:  models.MODRINTH,
		ProjectID: "proj-1",
	}, mod)

	assert.False(t, ok)
	assert.Equal(t, logEvent{}, event)
}

func TestFetchFailureDetailEventReturnsFalseForEmptyDetails(t *testing.T) {
	mod := models.Mod{ID: "proj-1", Name: "Example", Type: models.MODRINTH}

	event, ok := fetchFailureDetailEvent(emptyError{}, mod)

	assert.False(t, ok)
	assert.Equal(t, logEvent{}, event)
}

type emptyError struct{}

func (emptyError) Error() string { return "" }

func TestRunTestReturnsContextErrorWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fs := afero.NewMemMapFs()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")
	meta := config.NewMetadata(configPath)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.0",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Name: "Mod One", ID: "mod-one", Type: models.MODRINTH},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(ctx, fs, meta, cfg))

	deps := testDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, true, false),
		output: output.New(io.Discard, io.Discard, true),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called after cancellation")
			return platform.RemoteMod{}, errors.New("unexpected")
		},
		latestVersion:  func(context.Context, httpclient.Doer) (string, error) { return "1.20.1", nil },
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) { return true, nil },
	}

	opts := testOptions{ConfigPath: configPath, GameVersion: "1.20.1"}
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runTest(ctx, cmd, opts, deps)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRunTestExecutionBoundsConcurrency(t *testing.T) {
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		Mods: []models.Mod{
			{Name: "Mod One", ID: "mod-one", Type: models.MODRINTH},
			{Name: "Mod Two", ID: "mod-two", Type: models.MODRINTH},
			{Name: "Mod Three", ID: "mod-three", Type: models.MODRINTH},
			{Name: "Mod Four", ID: "mod-four", Type: models.MODRINTH},
			{Name: "Mod Five", ID: "mod-five", Type: models.MODRINTH},
			{Name: "Mod Six", ID: "mod-six", Type: models.MODRINTH},
		},
	}
	items, indexByKey := buildTestItems(cfg)
	readyCh := make(chan struct{}, len(cfg.Mods))
	releaseCh := make(chan struct{})
	var currentInFlight int64
	var maxInFlight int64

	deps := testDeps{
		fetchMod: func(ctx context.Context, platformName models.Platform, projectID string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			current := atomic.AddInt64(&currentInFlight, 1)
			for {
				observed := atomic.LoadInt64(&maxInFlight)
				if current <= observed || atomic.CompareAndSwapInt64(&maxInFlight, observed, current) {
					break
				}
			}
			readyCh <- struct{}{}
			<-releaseCh
			atomic.AddInt64(&currentInFlight, -1)
			return platform.RemoteMod{}, errors.New("fetch failed")
		},
	}

	resultCh := make(chan testExecutionOutcome, 1)
	concurrencyLimit := defaultTestMaxConcurrency
	go func() {
		resultCh <- runTestExecution(context.Background(), testExecutionInput{
			cfg:           cfg,
			targetVersion: "1.20.1",
			items:         items,
			indexByKey:    indexByKey,
			deps:          deps,
		}, testExecSender{})
	}()

	for index := 0; index < concurrencyLimit; index++ {
		select {
		case <-readyCh:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for bounded concurrency")
		}
	}

	if observed := atomic.LoadInt64(&maxInFlight); observed > int64(concurrencyLimit) {
		t.Fatalf("expected max concurrency %d, got %d", concurrencyLimit, observed)
	}

	close(releaseCh)

	select {
	case outcome := <-resultCh:
		assert.NoError(t, outcome.err)
	case <-time.After(time.Second):
		t.Fatal("runTestExecution did not finish")
	}
}

func TestRunTestExecutionReturnsContextErrorWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := models.ModsJSON{
		Mods: []models.Mod{{Name: "Mod One", ID: "mod-one", Type: models.MODRINTH}},
	}
	items, indexByKey := buildTestItems(cfg)

	outcome := runTestExecution(ctx, testExecutionInput{
		cfg:           cfg,
		targetVersion: "1.20.1",
		items:         items,
		indexByKey:    indexByKey,
		deps:          testDeps{},
	}, testExecSender{})

	assert.Error(t, outcome.err)
	assert.ErrorIs(t, outcome.err, context.Canceled)
}

func TestRunTestExecutionReturnsContextErrorWhenCanceledDuringWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := models.ModsJSON{
		Mods: []models.Mod{{Name: "Mod One", ID: "mod-one", Type: models.MODRINTH}},
	}
	items, indexByKey := buildTestItems(cfg)

	readyCh := make(chan struct{}, 1)
	deps := testDeps{
		fetchMod: func(ctx context.Context, platformName models.Platform, projectID string, opts platform.FetchOptions, clients platform.Clients) (platform.RemoteMod, error) {
			readyCh <- struct{}{}
			<-ctx.Done()
			return platform.RemoteMod{}, ctx.Err()
		},
	}

	resultCh := make(chan testExecutionOutcome, 1)
	go func() {
		resultCh <- runTestExecution(ctx, testExecutionInput{
			cfg:           cfg,
			targetVersion: "1.20.1",
			items:         items,
			indexByKey:    indexByKey,
			deps:          deps,
		}, testExecSender{})
	}()

	select {
	case <-readyCh:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for worker")
	}

	cancel()

	select {
	case outcome := <-resultCh:
		assert.Error(t, outcome.err)
		assert.ErrorIs(t, outcome.err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("runTestExecution did not finish")
	}
}

func TestRunTestReturnsOutputErrorWhenNoMods(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, models.ModsJSON{ModsFolder: "mods"}))

	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	_, err := runTest(context.Background(), cmd, testOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.1",
	}, testDeps{
		fs: fs,
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveTargetVersionReturnsOutputErrorOnValidationUnavailable(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	_, err := resolveTargetVersion(context.Background(), cmd, testOptions{GameVersion: "1.20.1"}, testDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return false, errors.New("boom")
		},
	}, models.ModsJSON{}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveTargetVersionReturnsOutputErrorOnInvalidVersion(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	_, err := resolveTargetVersion(context.Background(), cmd, testOptions{GameVersion: "1.20.1"}, testDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return false, nil
		},
	}, models.ModsJSON{}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveTargetVersionReturnsOutputErrorOnSameVersionLog(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: writeErr})

	_, err := resolveTargetVersion(context.Background(), cmd, testOptions{GameVersion: "1.20.1"}, testDeps{
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
	}, models.ModsJSON{GameVersion: "1.20.1"}, interaction.ExecutionModeNonTTY)
	assert.ErrorIs(t, err, writeErr)
}

func TestLogEventsReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := testDeps{
		logger: logger.New(errorWriter{err: writeErr}, io.Discard, false, true),
	}

	err := logEvents(deps, []logEvent{{Kind: logEventKindDebug, Message: "boom"}})
	assert.ErrorIs(t, err, writeErr)
}

func TestEvaluateTestOutcomeReportsUnsupported(t *testing.T) {
	exitCode, err := evaluateTestOutcome(testExecutionOutcome{
		items: []testItem{{Status: testItemStatusUnsupported}},
	})
	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, exitCode)
}

func TestEvaluateTestOutcomeReportsInconclusive(t *testing.T) {
	exitCode, err := evaluateTestOutcome(testExecutionOutcome{
		items: []testItem{{Status: testItemStatusInconclusive}},
	})
	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, exitCode)
}

func TestEvaluateTestOutcomeSuccess(t *testing.T) {
	exitCode, err := evaluateTestOutcome(testExecutionOutcome{
		items: []testItem{{Status: testItemStatusSupported}},
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestToUnsupportedModsFiltersUnsupported(t *testing.T) {
	unsupported := toUnsupportedMods([]testItem{
		{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Status: testItemStatusUnsupported},
		{Mod: models.Mod{ID: "beta", Type: models.MODRINTH}, Status: testItemStatusSupported},
	})

	if assert.Len(t, unsupported, 1) {
		assert.Equal(t, "alpha", unsupported[0].ID)
	}
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

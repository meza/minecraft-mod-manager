package test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

func TestNewDepsUsesCommonValues(t *testing.T) {
	fs := afero.NewMemMapFs()
	common := cmddeps.CommonDeps{
		FS:     fs,
		Output: output.New(io.Discard, io.Discard, false),
		Logger: logger.New(io.Discard, io.Discard, false, false),
	}

	deps := NewDeps(common)
	assert.Equal(t, fs, deps.fs)
	assert.NotNil(t, deps.logger)
	assert.NotNil(t, deps.output)
}

func TestRunWithDepsReturnsSuccessForEmptyMods(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		Mods:        []models.Mod{},
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := testDeps{
		fs:      fs,
		output:  output.New(io.Discard, io.Discard, false),
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.1", nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	result, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, deps, tui.ColorDisabled)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestHandleForcedTestResultError(t *testing.T) {
	writeErr := errors.New("write failed")
	_, err := handleForcedTestResult("1.20.1", []modCheckOutcome{
		{Mod: models.Mod{ID: "abc", Name: "Example", Type: models.MODRINTH}},
	}, testDeps{
		output: output.New(errorWriter{err: writeErr}, io.Discard, false),
	}, tui.ColorDisabled)

	assert.ErrorIs(t, err, writeErr)
}

func TestRunWithDepsForcePath(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := testDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
		logger: logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{
			Modrinth: httpclient.NewRLClient(httpclient.DefaultLimiter()),
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "Alpha"}, nil
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.1", nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	result, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
		Force:       true,
	}, deps, tui.ColorDisabled)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Empty(t, result.UnsupportedMods)
}

func TestRunWithDepsForceReturnsErrorWhenReportingUnsupportedModsFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	writeErr := errors.New("write failed")

	deps := testDeps{
		fs:     fs,
		output: output.New(errorWriter{err: writeErr}, io.Discard, false),
		logger: logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{
			Modrinth: httpclient.NewRLClient(httpclient.DefaultLimiter()),
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "alpha"}
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.2", nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	result, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
		Force:       true,
	}, deps, tui.ColorDisabled)

	assert.Equal(t, writeErr, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "1.20.2", result.TargetVersion)
}

func TestRunWithDepsReturnsLogError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := testDeps{
		fs:     fs,
		output: output.New(io.Discard, errorWriter{err: assert.AnError}, false),
		logger: logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{
			Modrinth: httpclient.NewRLClient(httpclient.DefaultLimiter()),
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, assert.AnError
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.2", nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	result, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, deps, tui.ColorDisabled)

	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestRunWithDepsSupportedMods(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := testDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
		logger: logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{
			Modrinth: httpclient.NewRLClient(httpclient.DefaultLimiter()),
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{Name: "Alpha"}, nil
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.1", nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	result, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, deps, tui.ColorDisabled)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Empty(t, result.UnsupportedMods)
}

func TestRunWithDepsInvalidVersion(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := testDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
		logger: logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{
			Modrinth: httpclient.NewRLClient(httpclient.DefaultLimiter()),
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.3", nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return false, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	_, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.3",
	}, deps, tui.ColorDisabled)

	assert.ErrorIs(t, err, errInvalidVersion)
}

func TestRunWithDepsMissingConfig(t *testing.T) {
	fs := afero.NewMemMapFs()

	deps := testDeps{
		fs:        fs,
		output:    output.New(io.Discard, io.Discard, false),
		logger:    logger.New(io.Discard, io.Discard, false, false),
		clients:   platform.Clients{},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	_, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  "/cfg/modlist.json",
		GameVersion: "1.20.1",
	}, deps, tui.ColorDisabled)

	assert.Error(t, err)
}

func TestRunWithDepsUnsupportedMods(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := testDeps{
		fs:      fs,
		output:  output.New(io.Discard, io.Discard, false),
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, assert.AnError
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.1", nil
		},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	}

	result, err := RunWithDeps(context.Background(), &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, deps, tui.ColorDisabled)

	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, result.ExitCode)
	assert.Len(t, result.UnsupportedMods, 1)
}

func TestRunWithDepsContextCanceled(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := testDeps{
		fs:        fs,
		output:    output.New(io.Discard, io.Discard, false),
		logger:    logger.New(io.Discard, io.Discard, false, false),
		clients:   platform.Clients{},
		telemetry: func(telemetry.CommandTelemetry) {},
		isValidVersion: func(context.Context, string, httpclient.Doer) (bool, error) {
			return true, nil
		},
		latestVersion: func(context.Context, httpclient.Doer) (string, error) {
			return "1.20.1", nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunWithDeps(ctx, &cobra.Command{}, Options{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.20.2",
	}, deps, tui.ColorDisabled)

	assert.Error(t, err)
}

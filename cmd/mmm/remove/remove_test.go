package remove

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func TestResolveModsToRemoveMatchesIDsAndNamesAndPreservesOrder(t *testing.T) {
	cfg := models.ModsJson{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "AANobbMI", Name: "Sodium"},
			{Type: models.CURSEFORGE, ID: "123", Name: "Fabric API"},
			{Type: models.MODRINTH, ID: "iris", Name: "Iris Shaders"},
		},
	}

	mods, err := resolveModsToRemove([]string{"*bmi", "fabric*", "sod*"}, cfg)
	require.NoError(t, err)
	require.Len(t, mods, 2)
	assert.Equal(t, "AANobbMI", mods[0].ID)
	assert.Equal(t, "123", mods[1].ID)
}

func TestResolveModsToRemoveErrorsOnInvalidPattern(t *testing.T) {
	cfg := models.ModsJson{Mods: []models.Mod{{Type: models.MODRINTH, ID: "x", Name: "y"}}}
	_, err := resolveModsToRemove([]string{"["}, cfg)
	assert.Error(t, err)
}

func TestRunRemoveDryRunPrintsHeaderAndWouldHaveLines(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "iris", Name: "Iris Shaders"},
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}

	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		DryRun:     true,
		Lookups:    []string{"*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)

	assert.Equal(t, "Running in dry-run mode. Nothing will actually be removed.\nWould have removed Iris Shaders\nWould have removed Sodium\n", out.String())
}

func TestRunRemoveDryRunDoesNotCreateLockFileWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}

	meta := config.NewMetadata("config/modlist.json")
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	lockPath := meta.LockPath()
	exists, err := afero.Exists(fs, lockPath)
	require.NoError(t, err)
	require.False(t, exists)

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		DryRun:     true,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)

	exists, err = afero.Exists(fs, lockPath)
	require.NoError(t, err)
	assert.False(t, exists)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	require.Len(t, updatedCfg.Mods, 1)
	assert.Equal(t, "sodium", updatedCfg.Mods[0].ID)
}

func TestRunRemoveQuietSuppressesNormalOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, true, false)

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Type: models.MODRINTH, Id: "sodium", Name: "Sodium", FileName: "missing.jar"},
	}))

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		DryRun:     false,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.Equal(t, "", out.String())
}

func TestRunRemoveDeletesFilesUpdatesLockAndConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
			{Type: models.CURSEFORGE, ID: "fabric-api", Name: "Fabric API"},
		},
	}
	meta := config.NewMetadata("config/modlist.json")
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	modsDir := meta.ModsFolderPath(cfg)
	require.NoError(t, fs.MkdirAll(modsDir, 0755))

	lock := []models.ModInstall{
		{Type: models.MODRINTH, Id: "sodium", Name: "Sodium", FileName: "sodium.jar"},
		{Type: models.CURSEFORGE, Id: "fabric-api", Name: "Fabric API", FileName: "fabric-api.jar"},
	}
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modsDir, "sodium.jar"), []byte("x"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modsDir, "fabric-api.jar"), []byte("x"), 0644))

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		DryRun:     false,
		Lookups:    []string{"sod*", "fabric*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 2, removed)

	exists, err := afero.Exists(fs, filepath.Join(modsDir, "sodium.jar"))
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = afero.Exists(fs, filepath.Join(modsDir, "fabric-api.jar"))
	require.NoError(t, err)
	assert.False(t, exists)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedLock)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedCfg.Mods)

	assert.Equal(t, "✅ Removed Sodium\n✅ Removed Fabric API\n", out.String())
}

func TestRunRemoveSkipsMissingFilesWithoutFailing(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	lock := []models.ModInstall{
		{Type: models.MODRINTH, Id: "sodium", Name: "Sodium", FileName: "missing.jar"},
	}
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		DryRun:     false,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedLock)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedCfg.Mods)

	assert.Equal(t, "✅ Removed Sodium\n", out.String())
}

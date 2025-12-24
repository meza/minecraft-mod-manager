package remove

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
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
	cfg := models.ModsJSON{
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
	cfg := models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "x", Name: "y"}}}
	_, err := resolveModsToRemove([]string{"["}, cfg)
	assert.Error(t, err)
}

func TestGlobMatchesReturnsFalseOnInvalidPattern(t *testing.T) {
	assert.False(t, globMatches("[", "value"))
}

func TestResolveModsToRemoveSkipsBlanksAndDedupes(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}

	mods, err := resolveModsToRemove([]string{"", "sod*", "SOD*"}, cfg)
	require.NoError(t, err)
	require.Len(t, mods, 1)
	assert.Equal(t, "sodium", mods[0].ID)
}

func TestRunRemoveDryRunPrintsHeaderAndWouldHaveLines(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
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

	cfg := models.ModsJSON{
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

	cfg := models.ModsJSON{
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
		{Type: models.MODRINTH, ID: "sodium", Name: "Sodium", FileName: "missing.jar"},
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

	cfg := models.ModsJSON{
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
		{Type: models.MODRINTH, ID: "sodium", Name: "Sodium", FileName: "sodium.jar"},
		{Type: models.CURSEFORGE, ID: "fabric-api", Name: "Fabric API", FileName: "fabric-api.jar"},
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

func TestRemoveConfigEntryNoMatchDoesNothing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/tmp/modlist.json")
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "one"},
		},
	}

	err := removeConfigEntry(context.Background(), meta, &cfg, models.Mod{Type: models.CURSEFORGE, ID: "two"}, removeDeps{fs: fs})
	assert.NoError(t, err)
	assert.Len(t, cfg.Mods, 1)
	assert.Equal(t, "one", cfg.Mods[0].ID)
}

func TestRunRemoveSkipsMissingFilesWithoutFailing(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
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
		{Type: models.MODRINTH, ID: "sodium", Name: "Sodium", FileName: "missing.jar"},
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

func TestRunRemoveReturnsZeroWhenNoMatches(t *testing.T) {
	fs := afero.NewMemMapFs()
	var out bytes.Buffer
	log := logger.New(&out, &out, false, false)

	cfg := models.ModsJSON{
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

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"does-not-match"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
}

func TestRunRemoveReturnsErrorWhenConfigMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	log := logger.New(io.Discard, io.Discard, false, false)

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	_, err := runRemove(context.Background(), removeOptions{
		ConfigPath: "missing.json",
		Lookups:    []string{"mod"},
	}, deps)
	assert.Error(t, err)
}

func TestRunRemoveReturnsErrorWhenReadLockFails(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))

	fs := afero.NewReadOnlyFs(baseFs)
	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	_, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)
}

func TestRunRemoveReturnsErrorWhenResolveModsFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
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

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	_, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		DryRun:     true,
		Lookups:    []string{"["},
	}, deps)
	assert.Error(t, err)
}

func TestRunRemoveReturnsErrorWhenWriteLockFails(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{
		{Type: models.MODRINTH, ID: "sodium", FileName: "missing.jar"},
	}))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}
	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	_, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)
}

func TestRunRemoveReturnsErrorWhenWriteConfigFails(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{
		{Type: models.MODRINTH, ID: "sodium", FileName: "missing.jar"},
	}))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.ConfigPath, err: errors.New("rename failed")}
	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	_, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)
}

func TestRunRemoveSkipsLockWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	log := logger.New(io.Discard, io.Discard, false, false)

	cfg := models.ModsJSON{
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

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	updated, readErr := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, readErr)
	assert.Empty(t, updated.Mods)

	lock, lockErr := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, lockErr)
	assert.Empty(t, lock)
}

func TestRunRemoveSkipsFileRemovalWhenFileNameEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	log := logger.New(out, errOut, false, false)

	cfg := models.ModsJSON{
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
		{Type: models.MODRINTH, ID: "sodium", FileName: ""},
	}))

	deps := removeDeps{
		fs:     fs,
		logger: log,
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	removed, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.Contains(t, errOut.String(), "cmd.remove.error.invalid_filename_lock")
}

func TestRunRemoveReturnsErrorWhenFileRemovalFails(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{
		{Type: models.MODRINTH, ID: "sodium", FileName: "bad.jar"},
	}))
	require.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "bad.jar"), []byte("x"), 0644))

	fs := removeErrorFs{
		Fs:       baseFs,
		failPath: filepath.Join(meta.ModsFolderPath(cfg), "bad.jar"),
		err:      errors.New("remove failed"),
	}

	deps := removeDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		telemetry: func(_ telemetry.CommandTelemetry) {
		},
	}

	_, err := runRemove(context.Background(), removeOptions{
		ConfigPath: meta.ConfigPath,
		Lookups:    []string{"sod*"},
	}, deps)
	assert.Error(t, err)
}

func TestReadLockForRemoveErrorsOnEnsureLockFailure(t *testing.T) {
	readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, err := readLockForRemove(context.Background(), readOnlyFs, meta, false)
	assert.Error(t, err)
}

func TestReadLockForRemoveReturnsErrorOnStatFailure(t *testing.T) {
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/cfg/modlist-lock.json"),
		err:      errors.New("stat failed"),
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, err := readLockForRemove(context.Background(), fs, meta, true)
	assert.Error(t, err)
}

func TestReadLockForRemoveReadsExistingLockOnDryRun(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "proj-1"}}
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	readLock, err := readLockForRemove(context.Background(), fs, meta, true)
	require.NoError(t, err)
	assert.Len(t, readLock, 1)
	assert.Equal(t, "proj-1", readLock[0].ID)
}

func TestLockAndConfigIndexForReturnMinusOneWhenMissing(t *testing.T) {
	mod := models.Mod{Type: models.MODRINTH, ID: "missing"}
	assert.Equal(t, -1, lockIndexFor(mod, nil))
	assert.Equal(t, -1, configIndexFor(mod, nil))
}

func TestRemoveFileForceReturnsErrorOnRemoveFailure(t *testing.T) {
	fs := removeErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/mod.jar"),
		err:      errors.New("remove failed"),
	}

	assert.Error(t, removeFileForce(fs, filepath.FromSlash("/mods/mod.jar")))
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Stat(name)
}

type removeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem removeErrorFs) Remove(name string) error {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return filesystem.err
	}
	return filesystem.Fs.Remove(name)
}

type renameErrorFs struct {
	afero.Fs
	failNew string
	err     error
}

func (filesystem renameErrorFs) Rename(oldname, newname string) error {
	if filepath.Clean(newname) == filepath.Clean(filesystem.failNew) {
		return filesystem.err
	}
	return filesystem.Fs.Rename(oldname, newname)
}

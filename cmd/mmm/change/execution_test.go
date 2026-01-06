package change

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestRunChangeExecutionSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("old"), 0o644))

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			filesystem := afero.NewOsFs()
			if len(filesystems) > 0 {
				filesystem = filesystems[0]
			}
			return afero.WriteFile(filesystem, destination, []byte("new"), 0o644)
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		lock:          lock,
		targetVersion: "1.21.1",
		force:         false,
		deps:          deps,
		items:         buildItemsForTest(cfg),
		indexByKey:    map[string]int{"modrinth:alpha": 0},
	})

	assert.Equal(t, changeStageSuccess, outcome.Stage)
	assert.Equal(t, changeForcePolicyUnset, outcome.ForcePolicy)
	assert.NoError(t, outcome.Err)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Equal(t, "1.21.1", updatedCfg.GameVersion)
	assert.Equal(t, "Alpha Remote", updatedCfg.Mods[0].Name)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	require.Len(t, updatedLock, 1)
	assert.Equal(t, "alpha-new.jar", updatedLock[0].FileName)
	assert.Equal(t, sha1Hex("new"), updatedLock[0].Hash)
}

func TestRunChangeExecutionCompatibilityFailed(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.MODRINTH},
		},
	}

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(_ context.Context, _ models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			if id == "beta" {
				return platform.RemoteMod{}, errors.New("unsupported")
			}
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			filesystem := afero.NewOsFs()
			if len(filesystems) > 0 {
				filesystem = filesystems[0]
			}
			return afero.WriteFile(filesystem, destination, []byte("new"), 0o644)
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		lock:          []models.ModInstall{},
		targetVersion: "1.21.1",
		force:         false,
		deps:          deps,
		items:         buildItemsForTest(cfg),
		indexByKey:    map[string]int{"modrinth:alpha": 0, "modrinth:beta": 1},
	})

	assert.Equal(t, changeStageCompatibilityFailed, outcome.Stage)
	assert.ErrorIs(t, outcome.Err, errCompatibilityFailed)
	require.Len(t, outcome.Items, 2)
	assert.Equal(t, changeCompatSupported, outcome.Items[0].CompatStatus)
	assert.Equal(t, changeCompatUnsupported, outcome.Items[1].CompatStatus)
}

func TestRunChangeExecutionCompatibilityFailedCancelsDownloads(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.MODRINTH},
		},
	}

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	downloadStarted := make(chan struct{})
	downloadCanceled := make(chan struct{})
	var downloadStartOnce sync.Once
	var downloadCancelOnce sync.Once

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		limiter: rate.NewLimiter(rate.Inf, 2),
		fetchMod: func(ctx context.Context, _ models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			if id == "beta" {
				<-downloadStarted
				return platform.RemoteMod{}, errors.New("unsupported")
			}
			return remote, nil
		},
		downloader: func(ctx context.Context, _ string, _ string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			downloadStartOnce.Do(func() { close(downloadStarted) })
			<-ctx.Done()
			downloadCancelOnce.Do(func() { close(downloadCanceled) })
			return ctx.Err()
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))

	done := make(chan changeOutcome, 1)
	go func() {
		done <- runChangeExecution(context.Background(), nil, changeExecutionInput{
			meta:          meta,
			cfg:           cfg,
			lock:          []models.ModInstall{},
			targetVersion: "1.21.1",
			force:         false,
			deps:          deps,
			items:         buildItemsForTest(cfg),
			indexByKey:    map[string]int{"modrinth:alpha": 0, "modrinth:beta": 1},
		})
	}()

	select {
	case outcome := <-done:
		assert.Equal(t, changeStageCompatibilityFailed, outcome.Stage)
		assert.ErrorIs(t, outcome.Err, errCompatibilityFailed)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for change execution")
	}

	select {
	case <-downloadCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected downloads to be canceled after compatibility failure")
	}
}

func TestRunChangeExecutionForceDefaultsToKeepConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
		{ID: "beta", Type: models.MODRINTH, FileName: "beta.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("old"), 0o644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "beta.jar"), []byte("old"), 0o644))

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(_ context.Context, _ models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			if id == "beta" {
				return platform.RemoteMod{}, errors.New("unsupported")
			}
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			filesystem := afero.NewOsFs()
			if len(filesystems) > 0 {
				filesystem = filesystems[0]
			}
			return afero.WriteFile(filesystem, destination, []byte("new"), 0o644)
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		lock:          lock,
		targetVersion: "1.21.1",
		force:         true,
		deps:          deps,
		items:         buildItemsForTest(cfg),
		indexByKey:    map[string]int{"modrinth:alpha": 0, "modrinth:beta": 1},
	})

	assert.Equal(t, changeStageSuccess, outcome.Stage)
	assert.Equal(t, changeForcePolicyKeepConfig, outcome.ForcePolicy)

	updatedConfig, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Equal(t, "1.21.1", updatedConfig.GameVersion)
	require.Len(t, updatedConfig.Mods, 2)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	require.Len(t, updatedLock, 1)
	assert.Equal(t, "alpha-new.jar", updatedLock[0].FileName)

	betaPath := filepath.Join(meta.Dir(), "mods", "beta.jar")
	exists, err := afero.Exists(fs, betaPath)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRunChangeExecutionForcePrunesConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
		{ID: "beta", Type: models.MODRINTH, FileName: "beta.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("old"), 0o644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "beta.jar"), []byte("old"), 0o644))

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(_ context.Context, _ models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			if id == "beta" {
				return platform.RemoteMod{}, errors.New("unsupported")
			}
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			filesystem := afero.NewOsFs()
			if len(filesystems) > 0 {
				filesystem = filesystems[0]
			}
			return afero.WriteFile(filesystem, destination, []byte("new"), 0o644)
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		lock:          lock,
		targetVersion: "1.21.1",
		force:         true,
		forcePolicy:   changeForcePolicyPruneConfig,
		deps:          deps,
		items:         buildItemsForTest(cfg),
		indexByKey:    map[string]int{"modrinth:alpha": 0, "modrinth:beta": 1},
	})

	assert.Equal(t, changeStageSuccess, outcome.Stage)

	updatedConfig, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	require.Len(t, updatedConfig.Mods, 1)
	assert.Equal(t, "alpha", updatedConfig.Mods[0].ID)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	require.Len(t, updatedLock, 1)
	assert.Equal(t, "alpha-new.jar", updatedLock[0].FileName)

	betaPath := filepath.Join(meta.Dir(), "mods", "beta.jar")
	exists, err := afero.Exists(fs, betaPath)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRunChangeExecutionForceDisablesSkipped(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
			{ID: "beta", Name: "Beta", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
		{ID: "beta", Type: models.MODRINTH, FileName: "beta.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("old"), 0o644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "beta.jar"), []byte("old"), 0o644))

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	policySelections := make(chan changeForcePolicy, 1)
	policySelections <- changeForcePolicyDisableSkipped

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(_ context.Context, _ models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			if id == "beta" {
				return platform.RemoteMod{}, errors.New("unsupported")
			}
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			filesystem := afero.NewOsFs()
			if len(filesystems) > 0 {
				filesystem = filesystems[0]
			}
			return afero.WriteFile(filesystem, destination, []byte("new"), 0o644)
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:            meta,
		cfg:             cfg,
		lock:            lock,
		targetVersion:   "1.21.1",
		force:           true,
		policySelection: policySelections,
		deps:            deps,
		items:           buildItemsForTest(cfg),
		indexByKey:      map[string]int{"modrinth:alpha": 0, "modrinth:beta": 1},
	})

	assert.Equal(t, changeStageSuccess, outcome.Stage)
	assert.Equal(t, changeForcePolicyDisableSkipped, outcome.ForcePolicy)

	updatedConfig, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	require.Len(t, updatedConfig.Mods, 2)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	require.Len(t, updatedLock, 1)
	assert.Equal(t, "alpha-new.jar", updatedLock[0].FileName)

	betaPath := filepath.Join(meta.Dir(), "mods", "beta.jar")
	betaDisabled := filepath.Join(meta.Dir(), "mods", "beta.jar.disabled")
	exists, err := afero.Exists(fs, betaPath)
	require.NoError(t, err)
	assert.False(t, exists)
	disabledExists, err := afero.Exists(fs, betaDisabled)
	require.NoError(t, err)
	assert.True(t, disabledExists)
}

func TestRunChangeExecutionDownloadFailed(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		lock:          []models.ModInstall{},
		targetVersion: "1.21.1",
		force:         false,
		deps:          deps,
		items:         buildItemsForTest(cfg),
		indexByKey:    map[string]int{"modrinth:alpha": 0},
	})

	assert.Equal(t, changeStageDownloadFailed, outcome.Stage)
	assert.Error(t, outcome.Err)
}

func TestRunChangeExecutionSwitchFailedRollback(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("old"), 0o644))

	remote := platform.RemoteMod{
		Name:        "Alpha Remote",
		FileName:    "alpha-new.jar",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			filesystem := afero.NewOsFs()
			if len(filesystems) > 0 {
				filesystem = filesystems[0]
			}
			return afero.WriteFile(filesystem, destination, []byte("new"), 0o644)
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile: func(fs afero.Fs, source string, dest string) error {
			if filepath.Base(dest) == "alpha-new.jar" {
				return errors.New("rename failed")
			}
			return fs.Rename(source, dest)
		},
		removeAll: func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:  func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		lock:          lock,
		targetVersion: "1.21.1",
		force:         false,
		deps:          deps,
		items:         buildItemsForTest(cfg),
		indexByKey:    map[string]int{"modrinth:alpha": 0},
	})

	assert.Equal(t, changeStageSwitchFailed, outcome.Stage)
	assert.Error(t, outcome.Err)

	exists, err := afero.Exists(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"))
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRunChangeExecutionNoModsUpdatesConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{},
	}
	lock := []models.ModInstall{
		{ID: "alpha", Type: models.MODRINTH, FileName: "alpha.jar"},
	}

	require.NoError(t, fs.MkdirAll(filepath.Join(meta.Dir(), "mods"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), "mods", "alpha.jar"), []byte("old"), 0o644))

	deps := changeDeps{
		fs:      fs,
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return nil
		},
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile:  func(fs afero.Fs, path string) error { return fs.Remove(path) },
		renameFile:  func(fs afero.Fs, source string, dest string) error { return fs.Rename(source, dest) },
		removeAll:   func(fs afero.Fs, path string) error { return fs.RemoveAll(path) },
		mkdirAll:    func(fs afero.Fs, path string, perm os.FileMode) error { return fs.MkdirAll(path, perm) },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          meta,
		cfg:           cfg,
		lock:          lock,
		targetVersion: "1.21.1",
		force:         false,
		deps:          deps,
		items:         []changeItem{},
		indexByKey:    map[string]int{},
	})

	assert.Equal(t, changeStageSuccess, outcome.Stage)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Equal(t, "1.21.1", updatedCfg.GameVersion)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	require.NoError(t, err)
	assert.Empty(t, updatedLock)
}

func TestRunChangeExecutionResetStagingFailure(t *testing.T) {
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{{ID: "alpha", Type: models.MODRINTH}},
	}

	deps := changeDeps{
		fs:        afero.NewMemMapFs(),
		removeAll: func(afero.Fs, string) error { return errors.New("cleanup failed") },
		mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
	}

	outcome := runChangeExecution(context.Background(), nil, changeExecutionInput{
		meta:          config.NewMetadata("/cfg/modlist.json"),
		cfg:           cfg,
		targetVersion: "1.21.1",
		deps:          deps,
		items:         buildItemsForTest(cfg),
		indexByKey:    map[string]int{"modrinth:alpha": 0},
	})

	assert.Equal(t, changeStageDownloadFailed, outcome.Stage)
	assert.Error(t, outcome.Err)
}

func TestResolveForcePolicyForRunVariants(t *testing.T) {
	baseItem := changeItem{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}}
	index := map[string]int{changeModKey(baseItem.Mod): 0}

	input := changeExecutionInput{items: []changeItem{baseItem}, indexByKey: index}
	state := newChangeExecutionState(input, nil)
	policy, err := resolveForcePolicyForRun(context.Background(), input, state)
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyUnset, policy)

	input.force = true
	state = newChangeExecutionState(input, nil)
	policy, err = resolveForcePolicyForRun(context.Background(), input, state)
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyKeepConfig, policy)

	input.forcePolicy = changeForcePolicyPruneConfig
	state = newChangeExecutionState(input, nil)
	policy, err = resolveForcePolicyForRun(context.Background(), input, state)
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyPruneConfig, policy)

	skippedItem := baseItem
	skippedItem.Skipped = true
	input.items = []changeItem{skippedItem}
	input.forcePolicy = changeForcePolicyDisableSkipped
	state = newChangeExecutionState(input, nil)
	policy, err = resolveForcePolicyForRun(context.Background(), input, state)
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyDisableSkipped, policy)

	input.forcePolicy = changeForcePolicyUnset
	input.policySelection = nil
	state = newChangeExecutionState(input, nil)
	policy, err = resolveForcePolicyForRun(context.Background(), input, state)
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyKeepConfig, policy)

	selection := make(chan changeForcePolicy, 1)
	selection <- changeForcePolicyPruneConfig
	input.policySelection = selection
	state = newChangeExecutionState(input, nil)
	policy, err = resolveForcePolicyForRun(context.Background(), input, state)
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyPruneConfig, policy)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	input.policySelection = make(chan changeForcePolicy)
	state = newChangeExecutionState(input, nil)
	policy, err = resolveForcePolicyForRun(ctx, input, state)
	assert.Error(t, err)
	assert.Equal(t, changeForcePolicyKeepConfig, policy)
}

func TestRunChangeExecutionPolicySelectionCanceled(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := models.ModsJSON{
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
		Mods:        []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	outcome := runChangeExecution(ctx, nil, changeExecutionInput{
		meta:            config.NewMetadata("/cfg/modlist.json"),
		cfg:             cfg,
		targetVersion:   "1.21.1",
		force:           true,
		policySelection: make(chan changeForcePolicy),
		deps: changeDeps{
			fs:        fs,
			removeAll: func(afero.Fs, string) error { return nil },
			mkdirAll:  func(afero.Fs, string, os.FileMode) error { return nil },
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return nil
			},
		},
		items:      []changeItem{{Mod: models.Mod{ID: "alpha", Type: models.MODRINTH}, Skipped: true}},
		indexByKey: map[string]int{"modrinth:alpha": 0},
	})

	assert.Equal(t, changeStageSwitchFailed, outcome.Stage)
	assert.Error(t, outcome.Err)
}

func buildItemsForTest(cfg models.ModsJSON) []changeItem {
	items, _ := buildChangeItems(cfg, changeItemOrderAlphabetical)
	return items
}

func sha1Hex(value string) string {
	hash := sha1.Sum([]byte(value))
	return hex.EncodeToString(hash[:])
}

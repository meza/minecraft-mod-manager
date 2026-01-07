package install

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestRunInstallExecutionSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	items, indexByKey := buildInstallItems(cfg)
	outcome := runInstallExecution(context.Background(), installExecutionInput{
		meta:       meta,
		cfg:        cfg,
		lock:       nil,
		deps:       installDeps{fs: fs, logger: logger.New(io.Discard, io.Discard, false, false)},
		items:      items,
		indexByKey: indexByKey,
	}, nil)

	assert.NoError(t, outcome.err)
	assert.Equal(t, installExecutionErrorNone, outcome.errType)
}

func TestRunInstallExecutionReturnsUnknownErrorType(t *testing.T) {
	logErr := errors.New("log failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	items, indexByKey := buildInstallItems(cfg)
	outcome := runInstallExecution(context.Background(), installExecutionInput{
		meta:       meta,
		cfg:        cfg,
		lock:       nil,
		deps:       installDeps{fs: fs, logger: logger.New(errorWriter{err: logErr}, io.Discard, false, true)},
		items:      items,
		indexByKey: indexByKey,
	}, nil)

	assert.ErrorIs(t, outcome.err, logErr)
	assert.Equal(t, installExecutionErrorUnknown, outcome.errType)
}

func TestRunInstallExecutionDownloadFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	items, indexByKey := buildInstallItems(cfg)
	outcome := runInstallExecution(context.Background(), installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: []models.ModInstall{{
			Type:        models.MODRINTH,
			ID:          "alpha",
			Name:        "Alpha",
			FileName:    "alpha.jar",
			Hash:        "",
			DownloadURL: "https://example.invalid/alpha.jar",
		}},
		deps:       installDeps{fs: fs, logger: logger.New(io.Discard, io.Discard, false, false)},
		items:      items,
		indexByKey: indexByKey,
	}, nil)

	assert.ErrorIs(t, outcome.err, errInstallFailures)
	assert.Equal(t, installExecutionErrorDownload, outcome.errType)
}

func TestRunInstallExecutionWriteLockFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}
	items, indexByKey := buildInstallItems(cfg)
	outcome := runInstallExecution(context.Background(), installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: nil,
		deps: installDeps{
			fs:     fs,
			logger: logger.New(io.Discard, io.Discard, false, false),
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{
					Name:        "Alpha Remote",
					FileName:    "alpha.jar",
					Hash:        sha1Hex("data"),
					DownloadURL: "https://example.invalid/alpha.jar",
				}, nil
			},
			downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
				var target afero.Fs = fs
				if len(filesystems) > 0 {
					target = filesystems[0]
				}
				return afero.WriteFile(target, destination, []byte("data"), 0644)
			},
		},
		items:      items,
		indexByKey: indexByKey,
	}, nil)

	assert.Error(t, outcome.err)
	assert.Equal(t, installExecutionErrorWriteLock, outcome.errType)
}

func TestRunInstallExecutionWriteConfigFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	fs := renameErrorFs{
		Fs:      baseFs,
		failNew: meta.ConfigPath,
		err:     errors.New("rename failed"),
	}

	items, indexByKey := buildInstallItems(cfg)
	outcome := runInstallExecution(context.Background(), installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: nil,
		deps: installDeps{
			fs:     fs,
			logger: logger.New(io.Discard, io.Discard, false, false),
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{
					Name:        "Alpha Remote",
					FileName:    "alpha.jar",
					Hash:        sha1Hex("data"),
					DownloadURL: "https://example.invalid/alpha.jar",
				}, nil
			},
			downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
				var target afero.Fs = fs
				if len(filesystems) > 0 {
					target = filesystems[0]
				}
				return afero.WriteFile(target, destination, []byte("data"), 0644)
			},
		},
		items:      items,
		indexByKey: indexByKey,
	}, nil)

	assert.ErrorContains(t, outcome.err, "rename failed")
	assert.Equal(t, installExecutionErrorWriteConfig, outcome.errType)
}

func TestRunInstallExecutionReturnsCanceledError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods: []models.Mod{
			{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	items, indexByKey := buildInstallItems(cfg)
	outcome := runInstallExecution(ctx, installExecutionInput{
		meta: meta,
		cfg:  cfg,
		lock: nil,
		deps: installDeps{
			fs:     fs,
			logger: logger.New(io.Discard, io.Discard, false, false),
			fetchMod: func(ctx context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, ctx.Err()
			},
		},
		items:      items,
		indexByKey: indexByKey,
	}, nil)

	assert.ErrorIs(t, outcome.err, context.Canceled)
	assert.Equal(t, installExecutionErrorCanceled, outcome.errType)
	if assert.Len(t, outcome.items, 1) {
		assert.Equal(t, installItemAborted, outcome.items[0].Status)
	}
}

func TestIsContextCancellation(t *testing.T) {
	assert.False(t, isContextCancellation(nil))
	assert.False(t, isContextCancellation(errors.New("boom")))
	assert.True(t, isContextCancellation(context.Canceled))
}

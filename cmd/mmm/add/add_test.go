package add

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	tuiinternal "github.com/meza/minecraft-mod-manager/internal/tui"
)

func TestRunAdd_Success(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	downloaded := false

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	telemetryCalled, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			downloaded = true
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.True(t, downloaded)
	configAfter, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, "abc", configAfter.Mods[0].ID)
	assert.Equal(t, "Example", configAfter.Mods[0].Name)
	assert.Nil(t, configAfter.Mods[0].AllowVersionFallback)
	assert.Nil(t, configAfter.Mods[0].Version)

	lock, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, lock, 1)
	assert.Equal(t, "example.jar", lock[0].FileName)

	assert.True(t, telemetryCalled.Success)
	assert.Equal(t, "add", telemetryCalled.Command)

	assertPerfSpanExists(t, "app.command.add.stage.prepare")
	assertPerfSpanExists(t, "app.command.add.stage.resolve")
	assertPerfSpanExists(t, "app.command.add.resolve.attempt")
	assertPerfSpanExists(t, "app.command.add.stage.download")
	assertPerfSpanExists(t, "app.command.add.stage.persist")
}

func TestRunAdd_SkipsDownloadWhenFileAlreadyMatchesRemoteHash(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "example.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(jarPath), 0755))
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	downloaded := false
	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			assert.Equal(t, models.MODRINTH, p)
			assert.Equal(t, "abc", id)
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        "a17c9aaa61e80a1bf71d0d850af4e5baa9800bbd",
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			downloaded = true
			return nil
		},
	})

	assert.NoError(t, err)
	assert.False(t, downloaded)
}

func TestRunAdd_QuietMissingConfigReturnsError(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), true, false),
	})

	assert.Error(t, err)
}

func TestRunAdd_DuplicateSkipsWorkWhenFilePresent(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	data := []byte("existing")
	sum := sha1.Sum(data)
	hash := fmt.Sprintf("%x", sum[:])

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "abc",
			Name:        "Existing",
			FileName:    "existing.jar",
			Hash:        hash,
			ReleasedOn:  "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/existing.jar",
		},
	}))

	managedPath := filepath.Join(meta.ModsFolderPath(cfg), "existing.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(managedPath), 0755))
	assert.NoError(t, afero.WriteFile(fs, managedPath, data, 0644))

	downloaded := false

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:       fs,
		clients:  platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:   logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		fetchMod: platform.FetchMod,
		downloader: func(_ context.Context, _ string, _ string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			downloaded = true
			return nil
		},
	})

	assert.NoError(t, err)
	assert.False(t, downloaded)
}

func TestRunAdd_DuplicateDownloadsWhenFileMissing(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "abc",
			Name:        "Existing",
			FileName:    "existing.jar",
			Hash:        sha1Hex("downloaded"),
			ReleasedOn:  "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/existing.jar",
		},
	}))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystem ...afero.Fs) error {
			return afero.WriteFile(filesystem[0], destination, []byte("downloaded"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.Contains(t, cmd.OutOrStdout().(*bytes.Buffer).String(), "downloading from modrinth")
}

func TestRunAdd_PersistFailureReturnsError(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	base := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, base.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), base, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), base, meta, nil))

	fs := renameFailFs{
		Fs: base,
		failTargets: map[string]struct{}{
			filepath.Clean(meta.ConfigPath): {},
		},
	}

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("downloaded"),
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystem ...afero.Fs) error {
			return afero.WriteFile(filesystem[0], destination, []byte("downloaded"), 0644)
		},
	})

	assert.Error(t, err)
}

func TestRunAdd_DuplicateEnsureLockedFileError(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "abc",
			Name:        "Existing",
			FileName:    "existing.jar",
			Hash:        sha1Hex("downloaded"),
			ReleasedOn:  "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/existing.jar",
		},
	}))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
	})

	assert.Error(t, err)
}

func TestRunAdd_DuplicateInvalidLockFileNameReturnsFriendlyError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "abc",
			Name:        "Existing",
			FileName:    "mods/existing.jar",
			Hash:        sha1Hex("downloaded"),
			ReleasedOn:  "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/existing.jar",
		},
	}))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when lock filename is invalid")
			return nil
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.add.error.invalid_filename_lock")
}

func TestRunAdd_DuplicateHashMismatchDownloads(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	existing := []byte("existing")
	existingHash := fmt.Sprintf("%x", sha1.Sum(existing))
	updatedHash := fmt.Sprintf("%x", sha1.Sum([]byte("updated")))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "abc",
			Name:        "Existing",
			FileName:    "existing.jar",
			Hash:        updatedHash,
			ReleasedOn:  "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/existing.jar",
		},
	}))

	managedPath := filepath.Join(meta.ModsFolderPath(cfg), "existing.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(managedPath), 0755))
	assert.NoError(t, afero.WriteFile(fs, managedPath, existing, 0644))

	downloaded := false
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			downloaded = true
			return afero.WriteFile(fs, path, []byte("updated"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.True(t, downloaded)
	assert.NotEqual(t, existingHash, updatedHash)
}

func TestRunAdd_ConfigPresentLockMissingBackfillsLock(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	downloaded := false

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			downloaded = true
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.True(t, downloaded)

	configAfter, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, "abc", configAfter.Mods[0].ID)

	lockAfter, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, lockAfter, 1)
	assert.Equal(t, "abc", lockAfter[0].ID)
}

func TestRunAdd_ConfigAndLockPresentButFileMissingDownloads(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "abc",
			Name:        "Existing",
			FileName:    "existing.jar",
			Hash:        sha1Hex("downloaded"),
			ReleasedOn:  "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/existing.jar",
		},
	}))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	downloaded := false
	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("fetch should not be called")
		},
		downloader: func(_ context.Context, _ string, dest string, _ httpclient.Doer, _ httpclient.Sender, filesystem ...afero.Fs) error {
			downloaded = true
			useFS := fs
			if len(filesystem) > 0 {
				useFS = filesystem[0]
			}
			return afero.WriteFile(useFS, dest, []byte("downloaded"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.True(t, downloaded)
}

func TestRunAdd_DownloadsWhenModsFolderMissingOnOsFs(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewOsFs()
	root := t.TempDir()
	configPath := filepath.Join(root, "cfg", "modlist.json")
	meta := config.NewMetadata(configPath)
	assert.NoError(t, os.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "abc",
			Name:        "Existing",
			FileName:    "existing.jar",
			Hash:        sha1Hex("downloaded"),
			ReleasedOn:  "2024-01-01T00:00:00Z",
			DownloadURL: "https://example.com/existing.jar",
		},
	}))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	downloaded := false
	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, true),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("fetch should not be called")
		},
		downloader: func(_ context.Context, _ string, dest string, _ httpclient.Doer, _ httpclient.Sender, filesystem ...afero.Fs) error {
			downloaded = true
			useFS := fs
			if len(filesystem) > 0 {
				useFS = filesystem[0]
			}
			return afero.WriteFile(useFS, dest, []byte("downloaded"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.True(t, downloaded)

	installedPath := filepath.Join(meta.ModsFolderPath(cfg), "existing.jar")
	exists, err := afero.Exists(fs, installedPath)
	assert.NoError(t, err)
	assert.True(t, exists)
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
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	errBuf := bytes.NewBuffer(nil)
	cmd.SetErr(errBuf)

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "invalid",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), true, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.UnknownPlatformError{Platform: "invalid"}
		},
	})

	assert.Error(t, err)
	assert.Contains(t, errBuf.String(), "Unknown platform")
}

func TestRunAdd_ModNotFoundCancelled(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{state: addTUIStateAborted}, nil
		},
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, errAborted))
	configAfter, readErr := config.ReadConfig(context.Background(), fs, meta)
	if !assert.NoError(t, readErr) {
		return
	}
	assert.Len(t, configAfter.Mods, 0)
}

func TestRunAdd_ModNotFoundRetry(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
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
					Hash:        sha1Hex("data"),
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.com/retry.jar",
				},
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, readErr := config.ReadConfig(context.Background(), fs, meta)
	if !assert.NoError(t, readErr) {
		return
	}
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, "def", configAfter.Mods[0].ID)
	assert.Equal(t, models.CURSEFORGE, configAfter.Mods[0].Type)

	commandSpan.End()
	assertPerfSpanExists(t, "app.command.add.resolve.attempt")
	assertPerfEventExists(t, "app.command.add", "app.command.add.tui.open")
}

func TestRunAdd_NoFileRetryAlternate(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "curseforge",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
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
					Hash:        sha1Hex("data"),
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.com/retry.jar",
				},
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, readErr := config.ReadConfig(context.Background(), fs, meta)
	if !assert.NoError(t, readErr) {
		return
	}
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, models.MODRINTH, configAfter.Mods[0].Type)
	assert.Equal(t, "zzz", configAfter.Mods[0].ID)

	commandSpan.End()
	assertPerfSpanExists(t, "app.command.add.resolve.attempt")
	assertPerfEventExists(t, "app.command.add", "app.command.add.tui.open")
}

func TestRunAdd_DownloadFailure(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, _ string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return errors.New("download failed")
		},
	})

	assert.Error(t, err)
	configAfter, readErr := config.ReadConfig(context.Background(), fs, meta)
	if !assert.NoError(t, readErr) {
		return
	}
	assert.Len(t, configAfter.Mods, 0)
}

func TestRunAdd_MissingHashReturnsFriendlyError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        "",
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when hash is missing")
			return nil
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.add.error.missing_hash_remote")
}

func TestRunAdd_InvalidFileNameReturnsFriendlyError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "mods/example.jar",
				Hash:        sha1Hex("expected"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when filename is invalid")
			return nil
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.add.error.invalid_filename_remote")
}

func TestRunAdd_HashMismatchReturnsFriendlyError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("expected"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("actual"), 0644)
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cmd.add.error.hash_mismatch")
}

func TestIntegrityErrorMessage_SymlinkOutsideMods(t *testing.T) {
	outsidePath := filepath.FromSlash("/outside/path")
	rootPath := filepath.FromSlash("/mods")
	message, ok := integrityErrorMessage(modpath.OutsideRootError{
		ResolvedPath: outsidePath,
		Root:         rootPath,
	}, "Example Mod")
	assert.True(t, ok)
	assert.Contains(t, message, outsidePath)
	assert.Contains(t, message, rootPath)
}

func TestRunAdd_PersistErrorReturnsError(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, nil))

	fileData := []byte("existing")
	fileHash := fmt.Sprintf("%x", sha1.Sum(fileData))
	managedPath := filepath.Join(meta.ModsFolderPath(cfg), "example.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.Dir(managedPath), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, managedPath, fileData, 0644))

	fs := afero.NewReadOnlyFs(baseFs)

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	downloaded := false
	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        fileHash,
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			downloaded = true
			return nil
		},
	})

	assert.Error(t, err)
	assert.False(t, downloaded)
}

func TestRunAdd_CreatesConfigWhenMissing(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.RemoveAll(meta.ConfigPath))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	minecraft.ClearManifestCache()

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:             "modrinth",
		ProjectID:            "abc",
		ConfigPath:           meta.ConfigPath,
		AllowVersionFallback: true,
	}, addDeps{
		fs:              fs,
		clients:         platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		minecraftClient: manifestDoer{body: `{"latest":{"release":"1.21.1","snapshot":""},"versions":[]}`},
		logger:          logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, "1.21.1", configAfter.GameVersion)
}

type manifestDoer struct {
	body string
}

func (doer manifestDoer) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(doer.body)),
		Header:     make(http.Header),
	}, nil
}

func TestRunAdd_UnknownPlatformInteractiveRetry(t *testing.T) {
	restoreTTY := tuiinternal.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restoreTTY()

	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTYReader{Buffer: bytes.NewBuffer(nil)})
	cmd.SetOut(fakeTTYWriter{Buffer: bytes.NewBuffer(nil)})
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "invalid",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
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
					Hash:        sha1Hex("data"),
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.com/retry.jar",
				},
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	configAfter, readErr := config.ReadConfig(context.Background(), fs, meta)
	if !assert.NoError(t, readErr) {
		return
	}
	assert.Len(t, configAfter.Mods, 1)
	assert.Equal(t, models.CURSEFORGE, configAfter.Mods[0].Type)

	commandSpan.End()
	assertPerfSpanExists(t, "app.command.add.resolve.attempt")
	assertPerfEventExists(t, "app.command.add", "app.command.add.tui.open")
}

type fakeTTYReader struct {
	*bytes.Buffer
}

func (fakeTTYReader) Fd() uintptr { return 0 }

type fakeTTYWriter struct {
	*bytes.Buffer
}

func (fakeTTYWriter) Fd() uintptr { return 1 }

func sha1Hex(data string) string {
	sum := sha1.Sum([]byte(data))
	return fmt.Sprintf("%x", sum[:])
}

type renameFailFs struct {
	afero.Fs
	failTargets map[string]struct{}
}

func (filesystem renameFailFs) Rename(oldname, newname string) error {
	if _, ok := filesystem.failTargets[filepath.Clean(newname)]; ok {
		return errors.New("rename failed")
	}
	return filesystem.Fs.Rename(oldname, newname)
}

func TestRunAdd_ModNotFoundQuiet(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	out := bytes.NewBuffer(nil)
	cmd.SetOut(out)
	cmd.SetErr(out)

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), true, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "Mod \"abc\" for modrinth does not exist")
}

func TestResolveRemoteMod_NoFileQuiet(t *testing.T) {
	ctx := context.Background()
	out := &bytes.Buffer{}
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	deps := addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(out, out, true, false),
		fetchMod: func(_ context.Context, p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: p, ProjectID: id}
		},
	}

	_, _, _, err := resolveRemoteMod(ctx, nil, cfg, addOptions{Quiet: true}, models.MODRINTH, "abc", deps, false, strings.NewReader(""), io.Discard)
	assert.Error(t, err)
}

func TestRunAdd_FetchOptionsPropagateVersionAndFallback(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	var gotOpts platform.FetchOptions

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	_, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:             "modrinth",
		ProjectID:            "abc",
		ConfigPath:           meta.ConfigPath,
		Version:              "1.2.3",
		AllowVersionFallback: true,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, opts platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			gotOpts = opts
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.com/example.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, gotOpts.AllowedReleaseTypes)
	assert.True(t, gotOpts.AllowFallback)
	assert.Equal(t, "1.2.3", gotOpts.FixedVersion)
}

func TestRunAdd_TelemetryOnFailure(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	telemetryCalled, err := runAdd(ctx, commandSpan, cmd, addOptions{
		Platform:   "modrinth",
		ProjectID:  "abc",
		ConfigPath: meta.ConfigPath,
	}, addDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("boom")
		},
	})

	assert.Error(t, err)
	assert.False(t, telemetryCalled.Success)
	assert.Equal(t, "add", telemetryCalled.Command)
}

func TestResolveRemoteModWithTUI_RecordsAttempt(t *testing.T) {
	ctx, commandSpan := startAddPerf(t)

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}

	var fetchCalls int
	deps := addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(io.Discard, io.Discard, false, false),
		fetchMod: func(_ context.Context, p models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			fetchCalls++
			return platform.RemoteMod{Name: "Example", FileName: "example.jar"}, nil
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			typed, ok := model.(addTUIModel)
			assert.True(t, ok)
			cmd := typed.fetchCmd(models.CURSEFORGE, "abc")
			msg := cmd().(addTUIFetchResultMsg)
			updated, _ := typed.Update(msg)
			return updated, nil
		},
	}

	remote, resolvedPlatform, resolvedProject, err := resolveRemoteModWithTUI(ctx, commandSpan, addTUIStateUnknownPlatformSelect, cfg, addOptions{}, models.Platform("invalid"), "abc", deps, strings.NewReader(""), io.Discard)
	assert.NoError(t, err)
	assert.Equal(t, "example.jar", remote.FileName)
	assert.Equal(t, models.CURSEFORGE, resolvedPlatform)
	assert.Equal(t, "abc", resolvedProject)
	assert.Equal(t, 1, fetchCalls)

	commandSpan.End()
	assertPerfSpanExists(t, "app.command.add.resolve.attempt")
}

func TestResolveRemoteModWithTUIFetchError(t *testing.T) {
	ctx := context.Background()
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	deps := addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(io.Discard, io.Discard, false, false),
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("boom")
		},
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			typed := model.(addTUIModel)
			cmd := typed.fetchCmd(models.MODRINTH, "abc")
			msg := cmd().(addTUIFetchResultMsg)
			updated, _ := typed.Update(msg)
			return updated, nil
		},
	}

	_, _, _, err := resolveRemoteModWithTUI(ctx, nil, addTUIStateUnknownPlatformSelect, cfg, addOptions{}, models.MODRINTH, "abc", deps, strings.NewReader(""), io.Discard)
	assert.Error(t, err)
}

func startAddPerf(t *testing.T) (context.Context, *perf.Span) {
	t.Helper()
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	ctx, span := perf.StartSpan(context.Background(), "app.command.add")
	return ctx, span
}

func assertPerfSpanExists(t *testing.T, name string) {
	t.Helper()
	spans, err := perf.GetSpans()
	assert.NoError(t, err)

	_, ok := perf.FindSpanByName(spans, name)
	assert.True(t, ok, "expected span %q", name)
}

func assertPerfEventExists(t *testing.T, spanName string, eventName string) {
	t.Helper()
	spans, err := perf.GetSpans()
	assert.NoError(t, err)

	span, ok := perf.FindSpanByName(spans, spanName)
	assert.True(t, ok, "expected span %q", spanName)

	for _, e := range span.Events {
		if e.Name == eventName {
			return
		}
	}
	t.Fatalf("expected event %q on span %q", eventName, spanName)
}

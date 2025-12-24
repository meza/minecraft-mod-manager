package modsetup

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestEnsureConfigAndLock_ReturnsExistingConfigAndLock(t *testing.T) {
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

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	gotCfg, gotLock, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)

	assert.NoError(t, err)
	assert.Equal(t, cfg.GameVersion, gotCfg.GameVersion)
	assert.Len(t, gotLock, 0)
}

func TestEnsureConfigAndLock_MissingConfigQuietReturnsError(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	_, _, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, true)

	assert.Error(t, err)
	var notFound *config.ConfigFileNotFoundException
	assert.True(t, errors.As(err, &notFound))
}

func TestEnsureConfigAndLock_MissingConfigInteractiveInitializes(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.RemoveAll(meta.ConfigPath))

	setupCoordinator := NewSetupCoordinator(fs, manifestDoer{body: `{"latest":{"release":"1.21.1","snapshot":""},"versions":[]}`}, nil)
	cfg, lock, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)

	assert.NoError(t, err)
	assert.Equal(t, "1.21.1", cfg.GameVersion)
	assert.Len(t, lock, 0)
}

func TestEnsureConfigAndLock_InitConfigFailureReturnsError(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.RemoveAll(meta.ConfigPath))

	setupCoordinator := NewSetupCoordinator(fs, failingDoer{}, nil)
	_, _, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)

	assert.Error(t, err)
}

func TestEnsureConfigAndLock_MissingConfigInteractiveWithoutMinecraftClientReturnsError(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.RemoveAll(meta.ConfigPath))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	_, _, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minecraftClient")
}

func TestEnsureConfigAndLock_InvalidConfigReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{not-json"), 0644))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	_, _, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)

	assert.Error(t, err)
}

func TestEnsureConfigAndLock_LockCreateFailureReturnsError(t *testing.T) {
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

	setupCoordinator := NewSetupCoordinator(failingRenameFs{Fs: fs, failTarget: meta.LockPath()}, nil, nil)
	_, _, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)

	assert.Error(t, err)
}

func TestEnsureDownloaded_CreatesModsFolderAndDownloads(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{ModsFolder: "mods"}
	remote := platform.RemoteMod{
		FileName:    "example.jar",
		DownloadURL: "https://example.com/example.jar",
		Hash:        sha1Hex("data"),
	}

	downloadCalled := false
	setupCoordinator := NewSetupCoordinator(fs, nil, func(_ context.Context, url string, path string, _ httpclient.Doer, _ httpclient.Sender, filesystem ...afero.Fs) error {
		downloadCalled = true
		assert.Equal(t, remote.DownloadURL, url)
		assert.Equal(t, meta.ModsFolderPath(cfg), filepath.Dir(path))
		assert.True(t, strings.HasPrefix(filepath.Base(path), "example.jar.mmm."))
		assert.True(t, strings.HasSuffix(filepath.Base(path), ".tmp"))
		return afero.WriteFile(filesystem[0], path, []byte("data"), 0644)
	})

	destination, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, cfg, remote, nil)
	assert.NoError(t, err)
	assert.True(t, downloadCalled)
	assert.Equal(t, filepath.Join(meta.ModsFolderPath(cfg), remote.FileName), destination)
}

func TestEnsureDownloaded_DownloadFailureReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	setupCoordinator := NewSetupCoordinator(fs, nil, func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return errors.New("download failed")
	})

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{
		FileName:    "x.jar",
		DownloadURL: "https://example.com/x.jar",
		Hash:        sha1Hex("data"),
	}, nil)
	assert.Error(t, err)
}

func TestEnsureDownloaded_ValidatesRemote(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{}, nil)
	assert.Error(t, err)
}

func TestEnsureDownloaded_WhitespaceFieldsReturnError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{
		FileName:    "   ",
		DownloadURL: "https://example.com/x.jar",
	}, nil)
	assert.Error(t, err)

	_, err = setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{
		FileName:    "x.jar",
		DownloadURL: "   ",
	}, nil)
	assert.Error(t, err)
}

func TestEnsureDownloaded_TrimsFileName(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{ModsFolder: "mods"}
	remote := platform.RemoteMod{
		FileName:    " example.jar ",
		DownloadURL: "https://example.com/example.jar",
		Hash:        sha1Hex("data"),
	}

	downloadCalled := false
	setupCoordinator := NewSetupCoordinator(fs, nil, func(_ context.Context, url string, path string, _ httpclient.Doer, _ httpclient.Sender, filesystem ...afero.Fs) error {
		downloadCalled = true
		assert.Equal(t, meta.ModsFolderPath(cfg), filepath.Dir(path))
		assert.True(t, strings.HasPrefix(filepath.Base(path), "example.jar.mmm."))
		assert.True(t, strings.HasSuffix(filepath.Base(path), ".tmp"))
		assert.Equal(t, remote.DownloadURL, url)
		return afero.WriteFile(filesystem[0], path, []byte("data"), 0644)
	})

	destination, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, cfg, remote, nil)
	assert.NoError(t, err)
	assert.True(t, downloadCalled)
	assert.Equal(t, filepath.Join(meta.ModsFolderPath(cfg), "example.jar"), destination)
}

func TestEnsureDownloaded_InvalidFileNameReturnsError(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), config.NewMetadata("modlist.json"), models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{
		FileName:    "mods/example.jar",
		DownloadURL: "https://example.com/example.jar",
	}, nil)
	assert.Error(t, err)
}

func TestEnsureDownloaded_MissingURLReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{FileName: "x.jar"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download url")
}

func TestEnsureDownloaded_MissingHashReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	setupCoordinator := NewSetupCoordinator(fs, nil, func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return nil
	})

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{
		FileName:    "x.jar",
		DownloadURL: "https://example.com/x.jar",
	}, nil)
	assert.Error(t, err)
}

func TestEnsureDownloaded_ReturnsErrorForSymlinkOutsideMods(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	meta := config.NewMetadata(filepath.Join(root, "modlist.json"))
	assert.NoError(t, os.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, os.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	outside := t.TempDir()
	target := filepath.Join(outside, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	linkPath := filepath.Join(meta.ModsFolderPath(cfg), "example.jar")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	setupCoordinator := NewSetupCoordinator(fs, nil, func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return nil
	})

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, cfg, platform.RemoteMod{
		FileName:    "example.jar",
		DownloadURL: "https://example.com/example.jar",
		Hash:        sha1Hex("data"),
	}, nil)
	assert.Error(t, err)
}

func TestEnsureDownloaded_MissingDownloaderReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{
		FileName:    "x.jar",
		DownloadURL: "https://example.com/x.jar",
		Hash:        sha1Hex("data"),
	}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "downloader")
}

func TestEnsureDownloaded_MkdirFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, base.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	readOnly := afero.NewReadOnlyFs(base)
	setupCoordinator := NewSetupCoordinator(readOnly, nil, func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
		return nil
	})

	_, err := setupCoordinator.EnsureDownloaded(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, platform.RemoteMod{
		FileName:    "x.jar",
		DownloadURL: "https://example.com/x.jar",
		Hash:        sha1Hex("data"),
	}, nil)
	assert.Error(t, err)
}

func TestEnsurePersisted_AppendsAndWrites(t *testing.T) {
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

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	cfgBefore, lockBefore, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)
	assert.NoError(t, err)

	remote := platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}

	cfgAfter, lockAfter, result, err := setupCoordinator.EnsurePersisted(context.Background(), meta, cfgBefore, lockBefore, models.MODRINTH, "abc", remote, EnsurePersistOptions{
		Version:              "1.2.3",
		AllowVersionFallback: true,
	})

	assert.NoError(t, err)
	assert.True(t, result.ConfigAdded)
	assert.True(t, result.LockAdded)
	assert.Len(t, cfgAfter.Mods, 1)
	assert.Equal(t, "abc", cfgAfter.Mods[0].ID)
	assert.NotNil(t, cfgAfter.Mods[0].AllowVersionFallback)
	assert.NotNil(t, cfgAfter.Mods[0].Version)
	assert.Len(t, lockAfter, 1)
	assert.Equal(t, "example.jar", lockAfter[0].FileName)

	onDiskConfig, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, onDiskConfig.Mods, 1)

	onDiskLock, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, onDiskLock, 1)
}

func TestEnsurePersisted_ConfigWriteFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	readOnly := afero.NewReadOnlyFs(base)

	setupCoordinator := NewSetupCoordinator(readOnly, nil, nil)
	cfg := models.ModsJSON{ModsFolder: "mods"}

	_, _, _, err := setupCoordinator.EnsurePersisted(context.Background(), config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")), cfg, nil, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.Error(t, err)
}

func TestEnsurePersisted_LockWriteFailureReturnsError(t *testing.T) {
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

	setupCoordinator := NewSetupCoordinator(failingRenameFs{Fs: fs, failTarget: meta.LockPath()}, nil, nil)
	_, _, _, err := setupCoordinator.EnsurePersisted(context.Background(), meta, cfg, nil, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.Error(t, err)
}

func TestEnsurePersisted_DuplicateIsNoOp(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Existing"},
		},
	}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "abc"}}

	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)
	cfgAfter, lockAfter, result, err := setupCoordinator.EnsurePersisted(context.Background(), config.NewMetadata("modlist.json"), cfg, lock, models.MODRINTH, "abc", platform.RemoteMod{Name: "Example"}, EnsurePersistOptions{})

	assert.NoError(t, err)
	assert.False(t, result.ConfigAdded)
	assert.False(t, result.LockAdded)
	assert.Equal(t, cfg, cfgAfter)
	assert.Equal(t, lock, lockAfter)
}

func TestEnsurePersisted_ConfigPresentLockMissingAddsLockOnly(t *testing.T) {
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

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	cfgBefore, lockBefore, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)
	assert.NoError(t, err)

	cfgAfter, lockAfter, result, err := setupCoordinator.EnsurePersisted(context.Background(), meta, cfgBefore, lockBefore, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.NoError(t, err)
	assert.False(t, result.ConfigAdded)
	assert.True(t, result.LockAdded)
	assert.Len(t, cfgAfter.Mods, 1)
	assert.Len(t, lockAfter, 1)

	onDiskConfig, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, onDiskConfig.Mods, 1)
}

func TestEnsurePersisted_LockPresentConfigMissingAddsConfigOnly(t *testing.T) {
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
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:     models.MODRINTH,
			ID:       "abc",
			Name:     "Existing",
			FileName: "existing.jar",
		},
	}))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	cfgBefore, lockBefore, err := setupCoordinator.EnsureConfigAndLock(context.Background(), meta, false)
	assert.NoError(t, err)

	cfgAfter, lockAfter, result, err := setupCoordinator.EnsurePersisted(context.Background(), meta, cfgBefore, lockBefore, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{
		Version: "1.2.3",
	})

	assert.NoError(t, err)
	assert.True(t, result.ConfigAdded)
	assert.False(t, result.LockAdded)
	assert.Len(t, cfgAfter.Mods, 1)
	assert.Len(t, lockAfter, 1)

	onDiskLock, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, onDiskLock, 1)
	assert.Equal(t, "existing.jar", onDiskLock[0].FileName)
}

func TestEnsurePersisted_MissingRemoteFieldsReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	_, _, _, err := setupCoordinator.EnsurePersisted(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, nil, models.MODRINTH, "abc", platform.RemoteMod{
		Name: "Example",
	}, EnsurePersistOptions{})

	assert.Error(t, err)
}

func TestEnsurePersisted_InvalidFileNameReturnsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	setupCoordinator := NewSetupCoordinator(fs, nil, nil)
	_, _, _, err := setupCoordinator.EnsurePersisted(context.Background(), meta, models.ModsJSON{ModsFolder: "mods"}, nil, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "mods/example.jar",
		Hash:        "hash",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.Error(t, err)
}

func TestEnsurePersisted_EmptyResolvedPlatformReturnsError(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)
	_, _, _, err := setupCoordinator.EnsurePersisted(context.Background(), config.NewMetadata("modlist.json"), models.ModsJSON{}, nil, "", "abc", platform.RemoteMod{}, EnsurePersistOptions{})
	assert.Error(t, err)
}

func TestEnsurePersisted_EmptyResolvedIDReturnsError(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)
	_, _, _, err := setupCoordinator.EnsurePersisted(context.Background(), config.NewMetadata("modlist.json"), models.ModsJSON{}, nil, models.MODRINTH, "", platform.RemoteMod{}, EnsurePersistOptions{})
	assert.Error(t, err)
}

func TestUpsertConfigAndLock_AddsMissingEntries(t *testing.T) {
	cfg := models.ModsJSON{ModsFolder: "mods", Mods: nil}
	lock := []models.ModInstall{}

	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)

	updatedCfg, updatedLock, result, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "sha",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.NoError(t, err)
	assert.True(t, result.ConfigAdded)
	assert.True(t, result.LockAdded)
	assert.Len(t, updatedCfg.Mods, 1)
	assert.Len(t, updatedLock, 1)
}

func TestUpsertConfigAndLock_InvalidFileNameReturnsError(t *testing.T) {
	cfg := models.ModsJSON{ModsFolder: "mods", Mods: nil}
	lock := []models.ModInstall{}

	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)

	_, _, _, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "mods/example.jar",
		Hash:        "sha",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.Error(t, err)
}

func TestUpsertConfigAndLock_UpdatesNameAndLockWhenDifferent(t *testing.T) {
	cfg := models.ModsJSON{ModsFolder: "mods", Mods: []models.Mod{
		{Type: models.MODRINTH, ID: "abc", Name: "Old"},
	}}
	lock := []models.ModInstall{{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Old",
		FileName:    "old.jar",
		Hash:        "old",
		ReleasedOn:  "2020-01-01T00:00:00Z",
		DownloadURL: "https://example.com/old.jar",
	}}

	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)

	updatedCfg, updatedLock, result, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "New",
		FileName:    "new.jar",
		Hash:        "NEW",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/new.jar",
	}, EnsurePersistOptions{})

	assert.NoError(t, err)
	assert.True(t, result.ConfigUpdated)
	assert.True(t, result.LockUpdated)
	assert.Equal(t, "New", updatedCfg.Mods[0].Name)
	assert.Equal(t, "new.jar", updatedLock[0].FileName)
}

func TestUpsertConfigAndLock_NoChangesDoesNotWrite(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewReadOnlyFs(afero.NewMemMapFs()), nil, nil)

	cfg := models.ModsJSON{ModsFolder: "mods", Mods: []models.Mod{
		{Type: models.MODRINTH, ID: "abc", Name: "Example"},
	}}
	lock := []models.ModInstall{{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleasedOn:  "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}}

	updatedCfg, updatedLock, result, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.NoError(t, err)
	assert.False(t, result.ConfigAdded)
	assert.False(t, result.ConfigUpdated)
	assert.False(t, result.LockAdded)
	assert.False(t, result.LockUpdated)
	assert.Equal(t, cfg, updatedCfg)
	assert.Equal(t, lock, updatedLock)
}

func TestUpsertConfigAndLock_HashCaseDifferenceIsNoOp(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)

	cfg := models.ModsJSON{ModsFolder: "mods", Mods: []models.Mod{
		{Type: models.MODRINTH, ID: "abc", Name: "Example"},
	}}
	lock := []models.ModInstall{{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "ABC",
		ReleasedOn:  "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}}

	_, _, result, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}, EnsurePersistOptions{})

	assert.NoError(t, err)
	assert.False(t, result.LockUpdated)
}

func TestUpsertConfigAndLock_LockPresentMissingRemoteFieldsReturnsError(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)

	cfg := models.ModsJSON{ModsFolder: "mods", Mods: []models.Mod{
		{Type: models.MODRINTH, ID: "abc", Name: "Example"},
	}}
	lock := []models.ModInstall{{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleasedOn:  "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/example.jar",
	}}

	_, _, _, err := setupCoordinator.UpsertConfigAndLock(cfg, lock, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "",
	}, EnsurePersistOptions{})

	assert.Error(t, err)
}

func TestUpsertConfigAndLock_LockMissingMissingRemoteFieldsReturnsError(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)

	_, _, _, err := setupCoordinator.UpsertConfigAndLock(models.ModsJSON{ModsFolder: "mods"}, nil, models.MODRINTH, "abc", platform.RemoteMod{
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "",
	}, EnsurePersistOptions{})

	assert.Error(t, err)
}

func TestUpsertConfigAndLock_MissingResolvedPlatformReturnsError(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)
	_, _, _, err := setupCoordinator.UpsertConfigAndLock(models.ModsJSON{}, nil, "", "abc", platform.RemoteMod{}, EnsurePersistOptions{})
	assert.Error(t, err)
}

func TestUpsertConfigAndLock_MissingResolvedIDReturnsError(t *testing.T) {
	setupCoordinator := NewSetupCoordinator(afero.NewMemMapFs(), nil, nil)
	_, _, _, err := setupCoordinator.UpsertConfigAndLock(models.ModsJSON{}, nil, models.MODRINTH, "", platform.RemoteMod{}, EnsurePersistOptions{})
	assert.Error(t, err)
}

func TestModExists_ReturnsTrueWhenPresent(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc"},
		},
	}

	assert.True(t, ModExists(cfg, models.MODRINTH, "abc"))
	assert.False(t, ModExists(cfg, models.MODRINTH, "def"))
	assert.False(t, ModExists(cfg, models.CURSEFORGE, "abc"))
}

func TestValidateRemoteForLock_MissingName(t *testing.T) {
	_, err := validateRemoteForLock(platform.RemoteMod{
		FileName:    "x.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/x.jar",
	})
	assert.Error(t, err)
}

func TestValidateRemoteForLock_MissingHash(t *testing.T) {
	_, err := validateRemoteForLock(platform.RemoteMod{
		Name:        "Example",
		FileName:    "x.jar",
		ReleaseDate: "2024-01-01T00:00:00Z",
		DownloadURL: "https://example.com/x.jar",
	})
	assert.Error(t, err)
}

func TestValidateRemoteForLock_MissingReleaseDate(t *testing.T) {
	_, err := validateRemoteForLock(platform.RemoteMod{
		Name:        "Example",
		FileName:    "x.jar",
		Hash:        "abc",
		DownloadURL: "https://example.com/x.jar",
	})
	assert.Error(t, err)
}

func TestValidateRemoteForLock_MissingDownloadURL(t *testing.T) {
	_, err := validateRemoteForLock(platform.RemoteMod{
		Name:        "Example",
		FileName:    "x.jar",
		Hash:        "abc",
		ReleaseDate: "2024-01-01T00:00:00Z",
	})
	assert.Error(t, err)
}

func TestOptionalBool_FalseReturnsNil(t *testing.T) {
	assert.Nil(t, optionalBool(false))
}

func TestOptionalString_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, optionalString(""))
	assert.Nil(t, optionalString("   "))
}

func TestNoopSender_SendIsNoOp(t *testing.T) {
	var sender noopSender
	sender.Send(nil)
}

type failingRenameFs struct {
	afero.Fs
	failTarget string
}

func (filesystem failingRenameFs) Rename(oldname, newname string) error {
	if filepath.Clean(newname) == filepath.Clean(filesystem.failTarget) {
		return errors.New("rename blocked")
	}
	return filesystem.Fs.Rename(oldname, newname)
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

type failingDoer struct{}

func (doer failingDoer) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("do failed")
}

func sha1Hex(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

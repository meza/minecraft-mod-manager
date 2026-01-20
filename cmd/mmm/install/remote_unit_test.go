package install

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestInstallFromLockReturnsInstallerError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}

	outcome, err := installFromLock(context.Background(), meta, cfg, mod, models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "alpha",
		Name:        "Alpha",
		FileName:    "alpha.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}, installDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
	}, nil)

	assert.Error(t, err)
	assert.False(t, outcome.failed)
}

func TestInstallFromLockHandlesInvalidFilename(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}

	outcome, err := installFromLock(context.Background(), meta, cfg, mod, models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "alpha",
		Name:        "Alpha",
		FileName:    "",
		Hash:        "hash",
		DownloadURL: "https://example.invalid/alpha.jar",
	}, installDeps{}, nil)

	assert.NoError(t, err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.install.error.invalid_filename_lock")
}

func TestInstallFromLockHandlesMissingHash(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}

	outcome, err := installFromLock(context.Background(), meta, cfg, mod, models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "alpha",
		Name:        "Alpha",
		FileName:    "alpha.jar",
		Hash:        "",
		DownloadURL: "https://example.invalid/alpha.jar",
	}, installDeps{fs: fs, downloader: httpclient.DownloadFile}, nil)

	assert.NoError(t, err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.install.error.missing_hash_lock")
}

func TestInstallFromLockSkipsDownloadWhenHashMatches(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), []byte("data"), 0644))

	outcome, err := installFromLock(context.Background(), meta, cfg, mod, models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "alpha",
		Name:        "Alpha",
		FileName:    "alpha.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}, installDeps{fs: fs}, nil)

	assert.NoError(t, err)
	assert.False(t, outcome.failed)
}

func TestInstallFromLockDownloadsMissingFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	mod := models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}

	outcome, err := installFromLock(context.Background(), meta, cfg, mod, models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "alpha",
		Name:        "Alpha",
		FileName:    "alpha.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.invalid/alpha.jar",
	}, installDeps{
		fs:      fs,
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			target := fs
			if len(filesystems) > 0 {
				target = filesystems[0]
			}
			return afero.WriteFile(target, destination, []byte("data"), 0644)
		},
	}, nil)

	assert.NoError(t, err)
	assert.False(t, outcome.failed)
}

func TestInstallFromRemoteHandlesMissingHash(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	outcome, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{
					Name:        "Alpha",
					FileName:    "alpha.jar",
					Hash:        "",
					DownloadURL: "https://example.invalid/alpha.jar",
				}, nil
			},
		},
	})

	assert.NoError(t, err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.mods.error.missing_hash_remote")
}

func TestNormalizeRemoteForInstallRejectsInvalidFilename(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	remote := platform.RemoteMod{Name: "Alpha", FileName: "mods/alpha.jar", Hash: "hash"}
	mod := models.Mod{Name: "Alpha", Type: models.MODRINTH}

	_, outcome := normalizeRemoteForInstall(remote, mod)

	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.install.error.invalid_filename_remote")
}

func TestResolveRemoteDestinationSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	remote := platform.RemoteMod{Name: "Alpha", FileName: "alpha.jar"}
	mod := models.Mod{Name: "Alpha", Type: models.MODRINTH}

	destination, outcome, err := resolveRemoteDestination(meta, cfg, remote, mod, installDeps{fs: fs})

	assert.NoError(t, err)
	assert.False(t, outcome.failed)
	assert.Equal(t, filepath.Join(meta.ModsFolderPath(cfg), "alpha.jar"), destination)
}

func TestResolveRemoteDestinationReturnsErrorWhenRootMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	meta := config.NewMetadata(filepath.Join(root, "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	remote := platform.RemoteMod{Name: "Alpha", FileName: "alpha.jar"}
	mod := models.Mod{Name: "Alpha", Type: models.MODRINTH}

	fs := symlinkStubFs{Fs: afero.NewOsFs(), symlinks: map[string]string{}}

	_, outcome, err := resolveRemoteDestination(meta, cfg, remote, mod, installDeps{fs: fs})

	assert.Error(t, err)
	assert.False(t, outcome.failed)
}

func TestInstallFromRemoteHandlesNotFound(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	outcome, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "alpha"}
			},
		},
	})

	assert.NoError(t, err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.install.error.mod_not_found")
}

func TestInstallFromRemoteHandlesNoCompatibleFile(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	outcome, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "alpha"}
			},
		},
	})

	assert.NoError(t, err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.install.error.no_file")
}

func TestInstallFromRemoteReturnsFetchError(t *testing.T) {
	_, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, errors.New("boom")
			},
		},
	})

	assert.ErrorContains(t, err, "boom")
}

func TestInstallFromRemoteSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	remote := platform.RemoteMod{
		Name:        "Alpha",
		FileName:    "alpha.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.invalid/alpha.jar",
		ReleaseDate: "2024-01-01T00:00:00Z",
	}

	items, indexByKey := buildInstallItems(models.ModsJSON{Mods: []models.Mod{{ID: "alpha", Name: "Alpha", Type: models.MODRINTH}}})
	state := &installExecutionState{items: items, indexByKey: indexByKey}

	outcome, err := installFromRemote(installModInputs{
		ctx:   context.Background(),
		meta:  meta,
		cfg:   cfg,
		mod:   models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		state: state,
		deps: installDeps{
			fs: fs,
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return remote, nil
			},
			downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
				return afero.WriteFile(fs, destination, []byte("data"), 0644)
			},
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
	})

	assert.NoError(t, err)
	assert.False(t, outcome.failed)
	assert.NotNil(t, outcome.lockEntry)
	assert.Equal(t, remote.FileName, outcome.lockEntry.FileName)
}

func TestInstallFromRemoteReturnsResolveErrorWhenRootMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	_, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: meta,
		cfg:  cfg,
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fs: fs,
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{
					Name:        "Alpha",
					FileName:    "alpha.jar",
					Hash:        sha1Hex("data"),
					DownloadURL: "https://example.invalid/alpha.jar",
				}, nil
			},
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
	})

	assert.Error(t, err)
}

func TestInstallFromRemoteHandlesResolveOutsideRoot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	root := t.TempDir()
	meta := config.NewMetadata(filepath.Join(root, "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	fs := symlinkStubFs{Fs: afero.NewOsFs(), symlinks: map[string]string{}}
	modsRoot := meta.ModsFolderPath(cfg)
	assert.NoError(t, fs.MkdirAll(modsRoot, 0755))

	linkPath := filepath.Join(modsRoot, "alpha.jar")
	fs.symlinks[filepath.Clean(linkPath)] = filepath.Join(t.TempDir(), "alpha.jar")

	outcome, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: meta,
		cfg:  cfg,
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fs: fs,
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{
					Name:        "Alpha",
					FileName:    "alpha.jar",
					Hash:        sha1Hex("data"),
					DownloadURL: "https://example.invalid/alpha.jar",
				}, nil
			},
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
	})

	assert.NoError(t, err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.mods.error.symlink_outside_mods")
}

func TestInstallFromRemoteReturnsDownloadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	_, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: meta,
		cfg:  cfg,
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fs: fs,
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{
					Name:        "Alpha",
					FileName:    "alpha.jar",
					Hash:        sha1Hex("data"),
					DownloadURL: "https://example.invalid/alpha.jar",
				}, nil
			},
			downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
				return errors.New("download failed")
			},
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
	})

	assert.ErrorContains(t, err, "download failed")
}

func TestInstallFromRemoteHandlesHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	outcome, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: meta,
		cfg:  cfg,
		mod:  models.Mod{ID: "alpha", Name: "Alpha", Type: models.MODRINTH},
		deps: installDeps{
			fs: fs,
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{
					Name:        "Alpha",
					FileName:    "alpha.jar",
					Hash:        sha1Hex("expected"),
					DownloadURL: "https://example.invalid/alpha.jar",
				}, nil
			},
			downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
				return afero.WriteFile(fs, destination, []byte("wrong"), 0644)
			},
			clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		},
	})

	assert.NoError(t, err)
	assert.True(t, outcome.failed)
	assert.Contains(t, outcome.failureReason, "cmd.install.error.hash_mismatch")
}

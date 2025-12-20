package install

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestRunInstallReturnsErrorOnConfigReadFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{invalid"), 0644))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnEnsureLockFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))

	readOnlyFs := afero.NewReadOnlyFs(baseFs)

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     readOnlyFs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnPreflightFailure(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	baseFs := afero.NewMemMapFs()
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))

	fs := openErrorFs{
		Fs:       baseFs,
		failPath: meta.ModsFolderPath(cfg),
		err:      errors.New("open failed"),
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsUnresolvedFilesError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.CURSEFORGE, ID: "123", Name: "Example"},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	filePath := filepath.Join(meta.ModsFolderPath(cfg), "unknown.jar")
	require.NoError(t, afero.WriteFile(fs, filePath, []byte("data"), 0644))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:                    fs,
		logger:                logger.New(io.Discard, io.Discard, false, false),
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectId: 123, Fingerprint: 1}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "Example", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.Sha1)}
		},
	})
	assert.ErrorIs(t, err, errUnresolvedFiles)
}

func TestRunInstallEnsuresExistingLockEntry(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	filePath := filepath.Join(meta.ModsFolderPath(cfg), "example.jar")
	require.NoError(t, afero.WriteFile(fs, filePath, []byte("data"), 0644))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Type: models.MODRINTH, Id: "abc", FileName: "example.jar", Hash: sha1Hex("data"), DownloadUrl: "https://example.invalid/example.jar"},
	}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called")
			return nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called")
			return platform.RemoteMod{}, nil
		},
	})
	assert.NoError(t, err)
}

func TestRunInstallFetchesWhenLockMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	remote := platform.RemoteMod{
		Name:        "Remote",
		FileName:    "remote.jar",
		DownloadURL: "https://example.invalid/remote.jar",
		Hash:        sha1Hex("data"),
	}

	result, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpClient.Doer, _ httpClient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("data"), 0644)
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, result.InstalledCount)
}

func TestRunInstallHandlesExpectedFetchError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
	})
	assert.NoError(t, err)
}

func TestRunInstallReturnsErrorOnFetchFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("boom")
		},
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnDownloadFailureWithDefaultClients(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	remote := platform.RemoteMod{
		Name:        "Remote",
		FileName:    "remote.jar",
		DownloadURL: "https://example.invalid/remote.jar",
		Hash:        sha1Hex("data"),
	}

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnWriteLockFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnWriteConfigFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.ConfigPath, err: errors.New("rename failed")}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnMkdirFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))
	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	fs := mkdirErrorFs{Fs: baseFs, failPath: meta.ModsFolderPath(cfg), err: errors.New("mkdir failed")}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnEnsureLockInstallFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	version := "1.2.3"
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium", Version: &version},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Type: models.MODRINTH, Id: "sodium", FileName: "sodium.jar"},
	}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{},
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnFetchUnexpected(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, errors.New("boom")
		},
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnDownloadFailureWithCreatedModsFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		},
	}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.Clients{},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Sodium",
				FileName:    "sodium.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.invalid/sodium.jar",
			}, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
	})
	assert.Error(t, err)
}

func TestRunInstallReturnsErrorOnLockWriteFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	})
	assert.Error(t, err)
}

func TestRunInstallReportsUnmanagedFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	filePath := filepath.Join(meta.ModsFolderPath(cfg), "unknown.jar")
	require.NoError(t, afero.WriteFile(fs, filePath, []byte("data"), 0644))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	result, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:                    fs,
		logger:                logger.New(io.Discard, io.Discard, false, false),
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectId: 456, Fingerprint: 1}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "Unknown", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.Sha1)}
		},
	})

	assert.NoError(t, err)
	assert.True(t, result.UnmanagedFound)
}

type renameErrorFs struct {
	afero.Fs
	failNew string
	err     error
}

func (r renameErrorFs) Rename(oldname, newname string) error {
	if filepath.Clean(newname) == filepath.Clean(r.failNew) {
		return r.err
	}
	return r.Fs.Rename(oldname, newname)
}

type mkdirErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (m mkdirErrorFs) MkdirAll(path string, perm os.FileMode) error {
	if filepath.Clean(path) == filepath.Clean(m.failPath) {
		return m.err
	}
	return m.Fs.MkdirAll(path, perm)
}

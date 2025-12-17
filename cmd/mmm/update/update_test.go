package update

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type noopDoer struct{}

func (n noopDoer) Do(*http.Request) (*http.Response, error) { return nil, nil }

type renameNoOverwriteFs struct {
	afero.Fs
}

func (r renameNoOverwriteFs) Rename(oldname, newname string) error {
	exists, err := afero.Exists(r.Fs, newname)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("cannot overwrite existing destination")
	}
	return r.Fs.Rename(oldname, newname)
}

type failRemoveBackupFs struct {
	afero.Fs
}

func (f failRemoveBackupFs) Remove(name string) error {
	if strings.Contains(name, ".mmm.bak") {
		return errors.New("cannot remove backup")
	}
	return f.Fs.Remove(name)
}

func TestDownloadAndSwapInPlaceLogsBackupDeletionFailureAtDebugLevel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := failRemoveBackupFs{Fs: baseFs}

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	installedPath := filepath.Join(meta.ModsFolderPath(cfg), "same.jar")
	assert.NoError(t, afero.WriteFile(fs, installedPath, []byte("old"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	log := logger.New(out, errOut, false, true)

	deps := updateDeps{
		fs:     fs,
		logger: log,
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpClient.Doer, _ httpClient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.NoError(t, downloadAndSwap(context.Background(), deps, installedPath, installedPath, "https://example.invalid/same.jar"))
	assert.Contains(t, out.String(), "cmd.update.debug.backup_cleanup_failed")
}

func TestCommandIsNoArgsAndDoesNotSilenceUsageByDefault(t *testing.T) {
	cmd := Command()
	assert.False(t, cmd.SilenceUsage)
	assert.NotNil(t, cmd.Args)
	assert.Error(t, cmd.Args(cmd, []string{"unexpected"}))
}

func TestRunUpdateAbortsWhenInstallReportsUnmanagedFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{UnmanagedFound: true}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when unmanaged files are detected")
			return platform.RemoteMod{}, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when unmanaged files are detected")
			return nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUnmanagedFiles)
	assert.Contains(t, errOut.String(), "cmd.update.error.unmanaged_found")
}

func TestRunUpdateReturnsErrorWhenInstallFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	sentinel := errors.New("install failed")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, sentinel
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when install fails")
			return platform.RemoteMod{}, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when install fails")
			return nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})
	assert.ErrorIs(t, err, sentinel)
}

func TestRunUpdateSkipsPinnedModsWithoutNetwork(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	version := "1.2.3"
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Pinned", Type: models.MODRINTH, Version: &version},
		},
	}

	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			Id:          "proj-1",
			Name:        "Pinned Name",
			FileName:    "pinned.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "hash",
			DownloadUrl: "https://example.invalid/pinned.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "pinned.jar"), []byte("x"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called for pinned mods")
			return platform.RemoteMod{}, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called for pinned mods")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 0, failed)

	updatedCfg, readErr := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, readErr)
	assert.Equal(t, "Pinned Name", updatedCfg.Mods[0].Name)
	assert.Contains(t, out.String(), "cmd.update.no_updates")
}

func TestRunUpdateFailsWhenLockEntryIsMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when lock entry is missing")
			return platform.RemoteMod{}, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when lock entry is missing")
			return nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.missing_lock_entry")
}

func TestRunUpdateReturnsNonZeroWhenFetchReturnsExpectedErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Missing", Type: models.MODRINTH},
			{ID: "proj-2", Name: "NoFile", Type: models.MODRINTH},
		},
	}

	lock := []models.ModInstall{
		{Type: models.MODRINTH, Id: "proj-1", Name: "Missing", FileName: "a.jar", ReleasedOn: "2024-01-01T00:00:00Z", Hash: "a", DownloadUrl: "https://example.invalid/a.jar"},
		{Type: models.MODRINTH, Id: "proj-2", Name: "NoFile", FileName: "b.jar", ReleasedOn: "2024-01-01T00:00:00Z", Hash: "b", DownloadUrl: "https://example.invalid/b.jar"},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "a.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "b.jar"), []byte("x"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	call := 0
	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			call++
			if call == 1 {
				return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "proj-1"}
			}
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "proj-2"}
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when fetchMod fails")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 2, failed)
	assert.NotContains(t, out.String(), "cmd.update.no_updates")
	assert.Contains(t, out.String(), "cmd.update.error.mod_not_found")
	assert.Contains(t, out.String(), "cmd.update.error.no_file")
}

func TestRunUpdateFailsWhenLockedFileIsMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}

	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			Id:          "proj-1",
			Name:        "Configured",
			FileName:    "missing.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadUrl: "https://example.invalid/missing.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Remote Name",
		FileName:    "new.jar",
		ReleaseDate: "2024-01-02T00:00:00Z",
		Hash:        "newhash",
		DownloadURL: "https://example.invalid/new.jar",
	}

	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when locked file is missing")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.locked_file_missing")
}

func TestRunUpdateFailsWhenInstalledTimestampIsInvalid(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}

	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			Id:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "not-a-time",
			Hash:        "oldhash",
			DownloadUrl: "https://example.invalid/old.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "old.jar"), []byte("old"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Remote Name",
		FileName:    "new.jar",
		ReleaseDate: "2024-01-02T00:00:00Z",
		Hash:        "newhash",
		DownloadURL: "https://example.invalid/new.jar",
	}

	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when timestamp is invalid")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.invalid_timestamp")
}

func TestRunUpdateDownloadsAndSwapsWhenNewerReleaseExists(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}

	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			Id:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadUrl: "https://example.invalid/old.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	oldPath := filepath.Join(meta.ModsFolderPath(cfg), "old.jar")
	assert.NoError(t, afero.WriteFile(fs, oldPath, []byte("old"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Remote Name",
		FileName:    "new.jar",
		ReleaseDate: "2024-01-02T00:00:00Z",
		Hash:        "newhash",
		DownloadURL: "https://example.invalid/new.jar",
	}

	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpClient.Doer, _ httpClient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, updated)
	assert.Equal(t, 0, failed)

	exists, _ := afero.Exists(fs, oldPath)
	assert.False(t, exists)

	newPath := filepath.Join(meta.ModsFolderPath(cfg), "new.jar")
	exists, _ = afero.Exists(fs, newPath)
	assert.True(t, exists)

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, lockErr)
	assert.Equal(t, "new.jar", updatedLock[0].FileName)
	assert.Equal(t, "newhash", updatedLock[0].Hash)
	assert.Equal(t, "2024-01-02T00:00:00Z", updatedLock[0].ReleasedOn)
	assert.Equal(t, "Remote Name", updatedLock[0].Name)

	updatedCfg, cfgErr := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, cfgErr)
	assert.Equal(t, "Remote Name", updatedCfg.Mods[0].Name)

	assert.Contains(t, out.String(), "cmd.update.has_update")
}

func TestRunUpdateKeepsPreviousFileAndLockWhenDownloadFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	downloadErr := errors.New("download failed")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}

	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			Id:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadUrl: "https://example.invalid/old.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	oldPath := filepath.Join(meta.ModsFolderPath(cfg), "old.jar")
	assert.NoError(t, afero.WriteFile(fs, oldPath, []byte("old"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Remote Name",
		FileName:    "new.jar",
		ReleaseDate: "2024-01-02T00:00:00Z",
		Hash:        "newhash",
		DownloadURL: "https://example.invalid/new.jar",
	}

	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return downloadErr
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)

	exists, _ := afero.Exists(fs, oldPath)
	assert.True(t, exists)

	newPath := filepath.Join(meta.ModsFolderPath(cfg), "new.jar")
	exists, _ = afero.Exists(fs, newPath)
	assert.False(t, exists)

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, lockErr)
	assert.Equal(t, "old.jar", updatedLock[0].FileName)
	assert.Equal(t, "oldhash", updatedLock[0].Hash)
}

func TestDownloadAndSwapReplacesInPlaceWithoutRenamingOverExistingFile(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := renameNoOverwriteFs{Fs: baseFs}

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	installedPath := filepath.Join(meta.ModsFolderPath(cfg), "same.jar")
	assert.NoError(t, afero.WriteFile(fs, installedPath, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpClient.Doer, _ httpClient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.NoError(t, downloadAndSwap(context.Background(), deps, installedPath, installedPath, "https://example.invalid/same.jar"))

	content, err := afero.ReadFile(fs, installedPath)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), content)

	exists, _ := afero.Exists(fs, installedPath+".mmm.tmp")
	assert.False(t, exists)
}

func TestDownloadAndSwapInPlaceDoesNotFailWhenBackupDeletionFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := failRemoveBackupFs{Fs: baseFs}

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	installedPath := filepath.Join(meta.ModsFolderPath(cfg), "same.jar")
	assert.NoError(t, afero.WriteFile(fs, installedPath, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		logger:  logger.New(&bytes.Buffer{}, &bytes.Buffer{}, false, false),
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpClient.Doer, _ httpClient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.NoError(t, downloadAndSwap(context.Background(), deps, installedPath, installedPath, "https://example.invalid/same.jar"))

	content, err := afero.ReadFile(fs, installedPath)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), content)

	exists, _ := afero.Exists(fs, installedPath+".mmm.bak")
	assert.True(t, exists)
}

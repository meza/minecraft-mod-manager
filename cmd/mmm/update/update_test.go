package update

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/cmd/mmm/install"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type noopDoer struct{}

func (n noopDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     http.Header{},
	}, nil
}

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

type removeMatchErrorFs struct {
	afero.Fs
	failPaths map[string]error
	failMatch []string
}

func (r removeMatchErrorFs) Remove(name string) error {
	if err, ok := r.failPaths[filepath.Clean(name)]; ok {
		if err != nil {
			return err
		}
		return errors.New("remove failed")
	}
	for _, match := range r.failMatch {
		if strings.Contains(filepath.Clean(name), match) {
			return errors.New("remove failed")
		}
	}
	return r.Fs.Remove(name)
}

type closeErrorFile struct {
	afero.File
	closeErr error
}

func (c closeErrorFile) Close() error {
	_ = c.File.Close()
	if c.closeErr != nil {
		return c.closeErr
	}
	return nil
}

type closeErrorFs struct {
	afero.Fs
	closeErr error
}

func (c closeErrorFs) Create(name string) (afero.File, error) {
	file, err := c.Fs.Create(name)
	if err != nil {
		return nil, err
	}
	return closeErrorFile{File: file, closeErr: c.closeErr}, nil
}

func (c closeErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, err := c.Fs.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return closeErrorFile{File: file, closeErr: c.closeErr}, nil
}

func (c closeErrorFs) Open(name string) (afero.File, error) {
	file, err := c.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	return closeErrorFile{File: file, closeErr: c.closeErr}, nil
}

type failingReadFs struct {
	afero.Fs
	failPath     string
	failContains string
}

func (f failingReadFs) Open(name string) (afero.File, error) {
	file, err := f.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if f.failPath != "" && filepath.Clean(name) == filepath.Clean(f.failPath) {
		return failingReaderFile{File: file}, nil
	}
	if f.failContains != "" && strings.Contains(filepath.Clean(name), f.failContains) {
		return failingReaderFile{File: file}, nil
	}
	return file, nil
}

type failingReaderFile struct {
	afero.File
}

func (f failingReaderFile) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestDownloadAndSwapInPlaceLogsBackupDeletionFailureAtDebugLevel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := failRemoveBackupFs{Fs: baseFs}

	cfg := models.ModsJSON{
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
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.NoError(t, downloadAndSwap(context.Background(), deps, installedPath, installedPath, meta.ModsFolderPath(cfg), "https://example.invalid/same.jar", sha1Hex("new")))
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

	cfg := models.ModsJSON{
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
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

	cfg := models.ModsJSON{
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
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
	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Pinned Name",
			FileName:    "pinned.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "hash",
			DownloadURL: "https://example.invalid/pinned.jar",
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
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

	cfg := models.ModsJSON{
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
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

	cfg := models.ModsJSON{
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
		{Type: models.MODRINTH, ID: "proj-1", Name: "Missing", FileName: "a.jar", ReleasedOn: "2024-01-01T00:00:00Z", Hash: "a", DownloadURL: "https://example.invalid/a.jar"},
		{Type: models.MODRINTH, ID: "proj-2", Name: "NoFile", FileName: "b.jar", ReleasedOn: "2024-01-01T00:00:00Z", Hash: "b", DownloadURL: "https://example.invalid/b.jar"},
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

	updated, failed, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(_ context.Context, _ models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			switch id {
			case "proj-1":
				return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "proj-1"}
			case "proj-2":
				return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "proj-2"}
			default:
				return platform.RemoteMod{}, errors.New("unexpected project id")
			}
		},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
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

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "missing.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/missing.jar",
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
		Hash:        sha1Hex("new"),
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
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

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "not-a-time",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/old.jar",
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
		Hash:        sha1Hex("new"),
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
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

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/old.jar",
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
		Hash:        sha1Hex("new"),
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
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, updated)
	assert.Equal(t, 0, failed)

	exists, err := afero.Exists(fs, oldPath)
	assert.NoError(t, err)
	assert.False(t, exists)

	newPath := filepath.Join(meta.ModsFolderPath(cfg), "new.jar")
	exists, err = afero.Exists(fs, newPath)
	assert.NoError(t, err)
	assert.True(t, exists)

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, lockErr)
	assert.Equal(t, "new.jar", updatedLock[0].FileName)
	assert.Equal(t, sha1Hex("new"), updatedLock[0].Hash)
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

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/old.jar",
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
		Hash:        sha1Hex("new"),
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return downloadErr
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)

	exists, err := afero.Exists(fs, oldPath)
	assert.NoError(t, err)
	assert.True(t, exists)

	newPath := filepath.Join(meta.ModsFolderPath(cfg), "new.jar")
	exists, err = afero.Exists(fs, newPath)
	assert.NoError(t, err)
	assert.False(t, exists)

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, lockErr)
	assert.Equal(t, "old.jar", updatedLock[0].FileName)
	assert.Equal(t, "oldhash", updatedLock[0].Hash)
}

func TestRunUpdateReportsMissingHash(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/old.jar",
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
		Hash:        "",
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when hash is missing")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.missing_hash_remote")
}

func TestRunUpdateReportsInvalidRemoteFileName(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/old.jar",
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
		FileName:    "mods/new.jar",
		ReleaseDate: "2024-01-02T00:00:00Z",
		Hash:        sha1Hex("new"),
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when filename is invalid")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.invalid_filename_remote")
}

func TestRunUpdateReportsInvalidLockFileName(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "mods/old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/old.jar",
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
		Hash:        sha1Hex("new"),
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when lock filename is invalid")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.invalid_filename_lock")
}

func TestRunUpdateReportsMissingInstalledHash(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "",
			DownloadURL: "https://example.invalid/old.jar",
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
		Hash:        sha1Hex("new"),
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
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when installed hash is missing")
			return nil
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.missing_hash_lock")
}

func TestRunUpdateReportsHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "old.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "oldhash",
			DownloadURL: "https://example.invalid/old.jar",
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
		Hash:        sha1Hex("expected"),
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
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("actual"), 0644)
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.hash_mismatch")

	exists, err := afero.Exists(fs, oldPath)
	assert.NoError(t, err)
	assert.True(t, exists)
	exists, err = afero.Exists(fs, filepath.Join(meta.ModsFolderPath(cfg), "new.jar"))
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDownloadAndSwapReplacesInPlaceWithoutRenamingOverExistingFile(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := renameNoOverwriteFs{Fs: baseFs}

	cfg := models.ModsJSON{
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
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.NoError(t, downloadAndSwap(context.Background(), deps, installedPath, installedPath, meta.ModsFolderPath(cfg), "https://example.invalid/same.jar", sha1Hex("new")))

	content, err := afero.ReadFile(fs, installedPath)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), content)

	entries, entriesErr := afero.ReadDir(fs, meta.ModsFolderPath(cfg))
	assert.NoError(t, entriesErr)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), ".tmp"))
	}
}

func TestDownloadAndSwapInPlaceDoesNotFailWhenBackupDeletionFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := failRemoveBackupFs{Fs: baseFs}

	cfg := models.ModsJSON{
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
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.NoError(t, downloadAndSwap(context.Background(), deps, installedPath, installedPath, meta.ModsFolderPath(cfg), "https://example.invalid/same.jar", sha1Hex("new")))

	content, err := afero.ReadFile(fs, installedPath)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), content)

	exists, err := afero.Exists(fs, installedPath+".mmm.bak")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestDownloadAndSwapReturnsJoinedErrorOnTempCloseCleanupFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := removeMatchErrorFs{
		Fs:        baseFs,
		failMatch: []string{".mmm."},
	}
	closeFs := closeErrorFs{Fs: fs, closeErr: errors.New("close failed")}

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, closeFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, closeFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	deps := updateDeps{
		fs:      closeFs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return nil
		},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.Join(meta.ModsFolderPath(cfg), "old.jar"), filepath.Join(meta.ModsFolderPath(cfg), "new.jar"), meta.ModsFolderPath(cfg), "https://example.invalid/new.jar", sha1Hex("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestDownloadAndSwapReturnsCloseErrorWhenTempCloseFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := closeErrorFs{Fs: baseFs, closeErr: errors.New("close failed")}

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return nil
		},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.Join(meta.ModsFolderPath(cfg), "old.jar"), filepath.Join(meta.ModsFolderPath(cfg), "new.jar"), meta.ModsFolderPath(cfg), "https://example.invalid/new.jar", sha1Hex("data"))
	assert.ErrorContains(t, err, "close failed")
}

func TestDownloadAndSwapReturnsJoinedErrorOnDownloadCleanupFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := removeMatchErrorFs{
		Fs:        baseFs,
		failMatch: []string{".mmm."},
	}

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.Join(meta.ModsFolderPath(cfg), "old.jar"), filepath.Join(meta.ModsFolderPath(cfg), "new.jar"), meta.ModsFolderPath(cfg), "https://example.invalid/new.jar", sha1Hex("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestDownloadAndSwapReturnsJoinedErrorOnHashReadCleanupFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := removeMatchErrorFs{
		Fs:        baseFs,
		failMatch: []string{".mmm."},
	}
	readFailFs := failingReadFs{Fs: fs, failContains: ".mmm."}

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, readFailFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, readFailFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	deps := updateDeps{
		fs:      readFailFs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("data"), 0644)
		},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.Join(meta.ModsFolderPath(cfg), "old.jar"), filepath.Join(meta.ModsFolderPath(cfg), "new.jar"), meta.ModsFolderPath(cfg), "https://example.invalid/new.jar", sha1Hex("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestDownloadAndSwapReturnsJoinedErrorOnHashMismatchCleanupFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	fs := removeMatchErrorFs{
		Fs:        baseFs,
		failMatch: []string{".mmm."},
	}

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("actual"), 0644)
		},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.Join(meta.ModsFolderPath(cfg), "old.jar"), filepath.Join(meta.ModsFolderPath(cfg), "new.jar"), meta.ModsFolderPath(cfg), "https://example.invalid/new.jar", sha1Hex("expected"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestDownloadAndSwapReturnsJoinedErrorOnReplaceCleanupFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	newPath := filepath.FromSlash("/cfg/mods/new.jar")
	renameFs := renameOnNewErrorFs{Fs: baseFs, failNew: newPath, err: errors.New("rename failed")}
	fs := removeMatchErrorFs{
		Fs:        renameFs,
		failMatch: []string{".mmm."},
	}

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("data"), 0644)
		},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.FromSlash("/cfg/mods/old.jar"), newPath, meta.ModsFolderPath(cfg), "https://example.invalid/new.jar", sha1Hex("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestDownloadAndSwapReturnsJoinedErrorOnOldFileCleanupFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	oldPath := filepath.FromSlash("/cfg/mods/old.jar")
	newPath := filepath.FromSlash("/cfg/mods/new.jar")
	fs := removeMatchErrorFs{
		Fs: baseFs,
		failPaths: map[string]error{
			filepath.Clean(oldPath): errors.New("remove old failed"),
			filepath.Clean(newPath): errors.New("remove new failed"),
		},
	}

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, oldPath, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	err := downloadAndSwap(context.Background(), deps, oldPath, newPath, meta.ModsFolderPath(cfg), "https://example.invalid/new.jar", sha1Hex("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove new file")
}

func TestSha1ForFileReturnsReadError(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/file.jar")
	assert.NoError(t, afero.WriteFile(base, path, []byte("content"), 0644))

	fs := failingReadFs{Fs: base, failPath: path}
	_, err := sha1ForFile(fs, path)
	assert.ErrorContains(t, err, "read failed")
}

func TestSha1ForFileReturnsCloseError(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/file.jar")
	assert.NoError(t, afero.WriteFile(base, path, []byte("content"), 0644))

	fs := closeErrorFs{Fs: base, closeErr: errors.New("close failed")}
	_, err := sha1ForFile(fs, path)
	assert.ErrorContains(t, err, "close failed")
}

func TestRunUpdateLogsNoUpdatesWhenNothingChanges(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "same.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "samehash",
			DownloadURL: "https://example.invalid/same.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "same.jar"), []byte("x"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Configured",
		FileName:    "same.jar",
		ReleaseDate: "2024-01-01T00:00:00Z",
		Hash:        "samehash",
		DownloadURL: "https://example.invalid/same.jar",
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
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 0, failed)
	assert.Contains(t, out.String(), "cmd.update.no_updates")
}

func TestRunUpdateReturnsErrorOnUnexpectedFetchError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "same.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "samehash",
			DownloadURL: "https://example.invalid/same.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "same.jar"), []byte("x"), 0644))

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
			return platform.RemoteMod{}, errors.New("boom")
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "boom")
}

func TestRunUpdateReturnsErrorOnRemoteTimestampInvalid(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "same.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "samehash",
			DownloadURL: "https://example.invalid/same.jar",
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "same.jar"), []byte("x"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Configured",
		FileName:    "same.jar",
		ReleaseDate: "not-a-date",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/same.jar",
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
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.ErrorIs(t, err, errUpdateFailures)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, failed)
	assert.Contains(t, errOut.String(), "cmd.update.error.invalid_timestamp")
}

func TestRunUpdateReturnsErrorWhenLockWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "same.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "samehash",
			DownloadURL: "https://example.invalid/same.jar",
		},
	}

	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, lock))
	assert.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "same.jar"), []byte("x"), 0644))

	fs := renameOnNewErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Configured",
		FileName:    "same.jar",
		ReleaseDate: "2024-01-02T00:00:00Z",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/same.jar",
	}

	_, _, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunUpdateReturnsErrorWhenConfigWriteFails(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
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
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "same.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "samehash",
			DownloadURL: "https://example.invalid/same.jar",
		},
	}

	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, lock))
	assert.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "same.jar"), []byte("x"), 0644))

	fs := renameOnNewErrorFs{Fs: baseFs, failNew: meta.ConfigPath, err: errors.New("rename failed")}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	remote := platform.RemoteMod{
		Name:        "Configured",
		FileName:    "same.jar",
		ReleaseDate: "2024-01-02T00:00:00Z",
		Hash:        sha1Hex("new"),
		DownloadURL: "https://example.invalid/same.jar",
	}

	_, _, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: meta.ConfigPath}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return remote, nil
		},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		clients:   platform.Clients{Modrinth: noopDoer{}},
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunUpdateReturnsErrorWhenConfigMissing(t *testing.T) {
	fs := afero.NewMemMapFs()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, _, err := runUpdate(context.Background(), cmd, updateOptions{ConfigPath: "missing.json"}, updateDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		install: func(context.Context, *cobra.Command, string, bool, bool) (install.Result, error) {
			return install.Result{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})
	assert.Error(t, err)
}

func TestRunUpdateReturnsErrorWhenLockMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Configured", Type: models.MODRINTH},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

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
			return install.Result{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
	})
	assert.Error(t, err)
}

func TestProcessModReturnsErrorOnExistsFailure(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods", Loader: models.FABRIC, GameVersion: "1.21.1"}
	mod := models.Mod{ID: "proj-1", Name: "Configured", Type: models.MODRINTH}
	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "mod.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "abc",
			DownloadURL: "https://example.invalid/mod.jar",
		},
	}

	oldPath := filepath.Join(meta.ModsFolderPath(cfg), "mod.jar")
	fs := statErrorFs{Fs: afero.NewMemMapFs(), failPath: oldPath, err: errors.New("stat failed")}

	outcome := processMod(context.Background(), meta, cfg, lock, modUpdateCandidate{ConfigIndex: 0, Mod: mod}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Configured",
				FileName:    "mod.jar",
				ReleaseDate: "2024-01-02T00:00:00Z",
				Hash:        sha1Hex("new"),
				DownloadURL: "https://example.invalid/mod.jar",
			}, nil
		},
	}, false)

	assert.Error(t, outcome.Error)
}

func TestProcessModReturnsUnchangedWhenHashMatches(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods", Loader: models.FABRIC, GameVersion: "1.21.1"}
	mod := models.Mod{ID: "proj-1", Name: "Configured", Type: models.MODRINTH}
	lock := []models.ModInstall{
		{
			Type:        models.MODRINTH,
			ID:          "proj-1",
			Name:        "Configured",
			FileName:    "mod.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "samehash",
			DownloadURL: "https://example.invalid/mod.jar",
		},
	}

	oldPath := filepath.Join(meta.ModsFolderPath(cfg), "mod.jar")
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, oldPath, []byte("data"), 0644))

	outcome := processMod(context.Background(), meta, cfg, lock, modUpdateCandidate{ConfigIndex: 0, Mod: mod}, updateDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Remote",
				FileName:    "mod.jar",
				ReleaseDate: "2024-01-02T00:00:00Z",
				Hash:        "samehash",
				DownloadURL: "https://example.invalid/mod.jar",
			}, nil
		},
	}, false)

	assert.False(t, outcome.Updated)
	assert.Empty(t, outcome.NewInstall.FileName)
	assert.NoError(t, outcome.Error)
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

type renameOnNewErrorFs struct {
	afero.Fs
	failNew string
	err     error
}

func (r renameOnNewErrorFs) Rename(oldname, newname string) error {
	if filepath.Clean(newname) == filepath.Clean(r.failNew) {
		return r.err
	}
	return r.Fs.Rename(oldname, newname)
}

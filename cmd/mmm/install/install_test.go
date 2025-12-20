package install

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunInstallHaltsWhenPreflightFindsUnsureHashMismatch(t *testing.T) {
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
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Id: "proj-1", Type: models.MODRINTH, Hash: "expected", FileName: "managed.jar", DownloadUrl: "https://example.invalid/managed.jar"},
	}))

	foreignPath := filepath.Join(meta.ModsFolderPath(cfg), "foreign.jar")
	assert.NoError(t, afero.WriteFile(fs, foreignPath, []byte("local-content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{ProjectId: "proj-1"}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "Matched Project", nil
		},
	}

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorIs(t, err, errUnresolvedFiles)

	assert.Contains(t, out.String(), "cmd.install.unsure.hash_mismatch")
	assert.Contains(t, errOut.String(), "cmd.install.error.unresolved")
}

func TestRunInstallReportsUnmanagedButDoesNotHalt(t *testing.T) {
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
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	_, err := config.EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)

	foreignPath := filepath.Join(meta.ModsFolderPath(cfg), "foreign.jar")
	assert.NoError(t, afero.WriteFile(fs, foreignPath, []byte("x"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{ProjectId: "unmanaged-proj"}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "Some Project", nil
		},
	}

	result, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)
	assert.True(t, result.UnmanagedFound)

	assert.Contains(t, out.String(), "cmd.install.unmanaged.found")
	assert.Contains(t, out.String(), "cmd.install.success")
	assert.Empty(t, errOut.String())
}

func TestRunInstallPreflightRespectsMmmignoreAndDisabledFiles(t *testing.T) {
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
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	_, err := config.EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("mods/ignored-*.jar\n"), 0644))

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "keep.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "ignored-1.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "keep.jar.disabled"), []byte("x"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	fingerprintCalls := 0
	modrinthCalls := 0

	deps := installDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth:   nil,
			Curseforge: nil,
		},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 {
			fingerprintCalls++
			return 123
		},
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			modrinthCalls++
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err = runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)

	assert.Equal(t, 1, fingerprintCalls, "expected only keep.jar to be scanned")
	assert.Equal(t, 1, modrinthCalls, "expected only keep.jar to be scanned")
}

func TestRunInstallSilentlyIgnoresFilesWithNoPlatformHits(t *testing.T) {
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
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	_, err := config.EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)

	foreignPath := filepath.Join(meta.ModsFolderPath(cfg), "foreign.jar")
	assert.NoError(t, afero.WriteFile(fs, foreignPath, []byte("x"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 { return 123 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err = runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)

	assert.Contains(t, out.String(), "cmd.install.success")
	assert.NotContains(t, out.String(), "cmd.install.unmanaged.found")
	assert.NotContains(t, out.String(), "cmd.install.unsure.")
	assert.Empty(t, errOut.String())
}

func TestRunInstallDownloadsMissingManagedFileFromLock(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Mod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Id: "proj-1", Type: models.MODRINTH, Hash: sha1Hex("downloaded"), FileName: "managed.jar", DownloadUrl: "https://example.invalid/managed.jar"},
	}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	downloaded := false
	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(_ context.Context, _ string, dest string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
			downloaded = true
			useFS := fs
			if len(filesystem) > 0 {
				useFS = filesystem[0]
			}
			return afero.WriteFile(useFS, dest, []byte("downloaded"), 0644)
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)
	assert.True(t, downloaded)

	assert.Contains(t, out.String(), "cmd.install.download.missing")
	assert.Contains(t, out.String(), "cmd.install.success")
	assert.Empty(t, errOut.String())
}

func TestRunInstallDownloadsWhenHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Mod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Id: "proj-1", Type: models.MODRINTH, Hash: sha1Hex("downloaded"), FileName: "managed.jar", DownloadUrl: "https://example.invalid/managed.jar"},
	}))

	managedPath := filepath.Join(meta.ModsFolderPath(cfg), "managed.jar")
	assert.NoError(t, afero.WriteFile(fs, managedPath, []byte("different"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	downloaded := false
	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(_ context.Context, _ string, dest string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
			downloaded = true
			useFS := fs
			if len(filesystem) > 0 {
				useFS = filesystem[0]
			}
			return afero.WriteFile(useFS, dest, []byte("downloaded"), 0644)
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)
	assert.True(t, downloaded)

	assert.Contains(t, out.String(), "cmd.install.download.hash_mismatch")
	assert.Contains(t, out.String(), "cmd.install.success")
	assert.Empty(t, errOut.String())
}

func TestRunInstallFetchesAndAppendsLockWhenMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Mod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	_, err := config.EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(_ context.Context, _ string, dest string, _ httpClient.Doer, _ httpClient.Sender, filesystem ...afero.Fs) error {
			useFS := fs
			if len(filesystem) > 0 {
				useFS = filesystem[0]
			}
			return afero.WriteFile(useFS, dest, []byte("downloaded"), 0644)
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Remote Name",
				FileName:    "remote.jar",
				ReleaseDate: "2025-01-01T00:00:00Z",
				Hash:        sha1Hex("downloaded"),
				DownloadURL: "https://example.invalid/remote.jar",
			}, nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err = runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, "Remote Name", updatedCfg.Mods[0].Name)

	updatedLock, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, updatedLock, 1)
	assert.Equal(t, "remote.jar", updatedLock[0].FileName)

	assert.Contains(t, out.String(), "cmd.install.download.missing")
	assert.Contains(t, out.String(), "cmd.install.success")
	assert.Empty(t, errOut.String())
}

func TestRunInstallReportsMissingHashWithoutHalting(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "bad", Name: "Bad Mod", Type: models.MODRINTH},
			{ID: "good", Name: "Good Mod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	_, err := config.EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		fetchMod: func(_ context.Context, _ models.Platform, id string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			if id == "bad" {
				return platform.RemoteMod{
					Name:        "Bad Remote",
					FileName:    "bad.jar",
					Hash:        "",
					ReleaseDate: "2024-01-01T00:00:00Z",
					DownloadURL: "https://example.invalid/bad.jar",
				}, nil
			}
			return platform.RemoteMod{
				Name:        "Good Remote",
				FileName:    "good.jar",
				Hash:        sha1Hex("data"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.invalid/good.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, dest string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, dest, []byte("data"), 0644)
		},
		telemetry:             func(telemetry.CommandTelemetry) {},
		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err = runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorIs(t, err, errInstallFailures)
	assert.Contains(t, errOut.String(), "cmd.install.error.missing_hash_remote")

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, lockErr)
	assert.Len(t, updatedLock, 1)
	assert.Equal(t, "good.jar", updatedLock[0].FileName)
}

func TestRunInstallReportsHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Mod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	_, err := config.EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		fetchMod: func(_ context.Context, _ models.Platform, _ string, _ platform.FetchOptions, _ platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Remote Name",
				FileName:    "remote.jar",
				Hash:        sha1Hex("expected"),
				ReleaseDate: "2024-01-01T00:00:00Z",
				DownloadURL: "https://example.invalid/remote.jar",
			}, nil
		},
		downloader: func(_ context.Context, _ string, dest string, _ httpClient.Doer, _ httpClient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, dest, []byte("actual"), 0644)
		},
		telemetry:             func(telemetry.CommandTelemetry) {},
		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err = runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorIs(t, err, errInstallFailures)
	assert.Contains(t, errOut.String(), "cmd.install.error.hash_mismatch")

	updatedLock, lockErr := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, lockErr)
	assert.Len(t, updatedLock, 0)
}

func TestRunInstallReportsMissingHashForLockEntry(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Mod", Type: models.MODRINTH},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{
			Type:        models.MODRINTH,
			Id:          "proj-1",
			Name:        "Mod",
			FileName:    "missing.jar",
			ReleasedOn:  "2024-01-01T00:00:00Z",
			Hash:        "",
			DownloadUrl: "https://example.invalid/missing.jar",
		},
	}))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when hash is missing")
			return nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			t.Fatal("fetchMod should not be called when lock entry exists")
			return platform.RemoteMod{}, nil
		},
		telemetry:             func(telemetry.CommandTelemetry) {},
		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.ErrorIs(t, err, errInstallFailures)
	assert.Contains(t, errOut.String(), "cmd.install.error.missing_hash_lock")
}

func TestRunInstallContinuesWhenFetchReturnsExpectedErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "proj-1", Name: "Mod", Type: models.MODRINTH},
			{ID: "proj-2", Name: "Mod2", Type: models.CURSEFORGE},
		},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	_, err := config.EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	call := 0
	deps := installDeps{
		fs:      fs,
		logger:  logger.New(out, errOut, false, false),
		clients: platform.Clients{},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return nil
		},
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			call++
			if call == 1 {
				return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "proj-1"}
			}
			return platform.RemoteMod{}, &platform.NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: "proj-2"}
		},
		telemetry: func(telemetry.CommandTelemetry) {},

		curseforgeFingerprint: func(string) uint32 { return 0 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpClient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpClient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpClient.Doer) (string, error) {
			return "", nil
		},
	}

	_, err = runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, deps)
	assert.NoError(t, err)

	assert.Contains(t, out.String(), "cmd.install.error.mod_not_found")
	assert.Contains(t, out.String(), "cmd.install.error.no_file")
	assert.Contains(t, out.String(), "cmd.install.success")
	assert.Empty(t, errOut.String())
}

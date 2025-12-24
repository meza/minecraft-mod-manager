package scan

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"hash"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

type fakePrompter struct {
	confirm bool
	err     error
}

func (prompter fakePrompter) ConfirmAdd() (bool, error) { return prompter.confirm, prompter.err }

func TestRunScan_PreferModrinthDoesNotCallCurseforgeWhenHit(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	curseforgeCalled := false

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth:   nil,
			Curseforge: nil,
		},
		prompter: nil,
		telemetry: func(telemetry.CommandTelemetry) {
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example Mod", nil
		},
		curseforgeFingerprint: func(string) uint32 {
			return 123
		},
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			curseforgeCalled = true
			return &curseforge.FingerprintResult{}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "CF", nil
		},
	})

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.recognized.header")
	assert.Contains(t, out.String(), "cmd.scan.recognized.entry")
	assert.False(t, curseforgeCalled)
}

func TestRunScan_FallbackOnMissUsesOtherPlatform(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth:   nil,
			Curseforge: nil,
		},
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			t.Fatal("modrinthProjectTitle should not be called on not found")
			return "", nil
		},
		curseforgeFingerprint: func(string) uint32 {
			return 999
		},
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   42,
						Fingerprint: 999,
						DownloadURL: "https://example.invalid/cf.jar",
						FileDate:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "CurseForge Mod", nil
		},
	})

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.recognized.entry")
}

func TestRunScan_Curseforge403ErrorIncludesPerFileFingerprint(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	file1 := filepath.Join(meta.ModsFolderPath(cfg), "a.jar")
	file2 := filepath.Join(meta.ModsFolderPath(cfg), "b.jar")
	assert.NoError(t, afero.WriteFile(fs, file1, []byte("a"), 0644))
	assert.NoError(t, afero.WriteFile(fs, file2, []byte("b"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth:   nil,
			Curseforge: nil,
		},
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		curseforgeFingerprint: func(path string) uint32 {
			switch filepath.Base(path) {
			case "a.jar":
				return 111
			case "b.jar":
				return 222
			default:
				return 0
			}
		},
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, &curseforge.FingerprintAPIError{
				Lookup: []int{111, 222},
				Err:    errors.New("unexpected status code: 403"),
			}
		},
	})

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "curseforge fingerprint 111")
	assert.Contains(t, out.String(), "curseforge fingerprint 222")
	assert.NotContains(t, out.String(), "Fingerprints for")
}

func TestRunScan_PreferredLookupErrorDoesNotFallbackAndIsUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	curseforgeCalled := false

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		clients: platform.Clients{
			Modrinth:   nil,
			Curseforge: nil,
		},
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionAPIError{}
		},
		curseforgeFingerprint: func(string) uint32 { return 999 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			curseforgeCalled = true
			return &curseforge.FingerprintResult{}, nil
		},
	})

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.unsure.header")
	assert.Contains(t, out.String(), "cmd.scan.unsure.entry_with_reason")
	assert.False(t, curseforgeCalled)
}

func TestRunScan_AddPersistsConfigAndLock(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
		Add:        true,
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example Mod", nil
		},
		clients: platform.Clients{},
	})

	assert.NoError(t, err)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, updatedCfg.Mods, 1)
	assert.Equal(t, "proj-1", updatedCfg.Mods[0].ID)
	assert.Equal(t, models.MODRINTH, updatedCfg.Mods[0].Type)

	lock, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, lock, 1)
	assert.Equal(t, "unmanaged.jar", lock[0].FileName)
	assert.Equal(t, models.MODRINTH, lock[0].Type)
	assert.Equal(t, "proj-1", lock[0].ID)
}

func TestRunScan_AddDoesNotPersistWhenAnyFileIsUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	matchPath := filepath.Join(meta.ModsFolderPath(cfg), "match.jar")
	unsurePath := filepath.Join(meta.ModsFolderPath(cfg), "unsure.jar")
	assert.NoError(t, afero.WriteFile(fs, matchPath, []byte("match"), 0644))
	assert.NoError(t, afero.WriteFile(fs, unsurePath, []byte("unsure"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	matchSha := sha1.Sum([]byte("match"))
	matchShaHex := hex.EncodeToString(matchSha[:])

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
		Add:        true,
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {
		},
		modrinthVersionForSha: func(_ context.Context, sha string, _ httpclient.Doer) (*modrinth.Version, error) {
			if sha == matchShaHex {
				return &modrinth.Version{
					ProjectID:     "proj-1",
					DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					Files: []modrinth.VersionFile{
						{URL: "https://example.invalid/mod.jar", Primary: true},
					},
				}, nil
			}
			return nil, &modrinth.VersionAPIError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example Mod", nil
		},
		clients: platform.Clients{},
	})

	assert.NoError(t, err)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, updatedCfg.Mods, 0)

	lock, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, lock, 0)
}

func TestRunScan_QuietSuppressesNormalOutput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
		Quiet:      true,
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, true, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example Mod", nil
		},
		clients: platform.Clients{},
	})
	assert.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestRunScan_AddBackfillsMissingLockEntry(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "proj-1", Name: "Old"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
		Add:        true,
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "New Name", nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
		clients:   platform.Clients{},
	})

	assert.NoError(t, err)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updatedCfg.Mods[0].Name)

	lock, err := config.ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Len(t, lock, 1)
	assert.Equal(t, "proj-1", lock[0].ID)
}

func TestRunScan_RespectsMmmignoreAndSkipsManagedFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{Type: models.MODRINTH, ID: "managed", FileName: "managed.jar"},
	}))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "managed.jar"), []byte("content"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "ignored.jar"), []byte("content"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("content"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("mods/ignored.jar\n"), 0644))

	called := 0
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			called++
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
		telemetry: func(telemetry.CommandTelemetry) {},
		clients:   platform.Clients{},
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, called)
}

func TestRunScan_InvalidPreferReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "unknown",
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunScan_ReturnsErrorOnListJarFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunScan_ReturnsErrorOnSha1Failure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, nil))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(baseFs, jarPath, []byte("content"), 0644))

	fs := readErrorFs{Fs: baseFs, failPath: jarPath, err: errors.New("read failed")}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunScan_ReturnsErrorOnPrompterFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		prompter:  fakePrompter{err: errors.New("confirm failed")},
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
		clients: platform.Clients{},
	})

	assert.Error(t, err)
}

func TestRunScan_PromptDeclineSkipsPersist(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		prompter:  fakePrompter{confirm: false},
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
		clients: platform.Clients{},
	})

	assert.NoError(t, err)

	updatedCfg, err := config.ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Empty(t, updatedCfg.Mods)
}

func TestRunScan_AllManagedReturnsEarly(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{
		{FileName: "managed.jar"},
	}))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "managed.jar"), []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.all_managed")
}

func TestRunScan_ReturnsErrorOnEnsureConfigFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Prefer:     "modrinth",
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, true, false),
		telemetry: func(telemetry.CommandTelemetry) {},
	})

	assert.Error(t, err)
}

func TestRunScan_AddLogsPersistFailureAndContinues(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, nil))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(fs, jarPath, []byte("content"), 0644))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
		Add:        true,
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
		curseforgeFingerprint: func(string) uint32 { return 123 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	})

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.persist_failed")
}

func TestRunScan_WriteConfigFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, nil))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(baseFs, jarPath, []byte("content"), 0644))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.ConfigPath, err: errors.New("rename failed")}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
		Add:        true,
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example Mod", nil
		},
		clients: platform.Clients{},
	})

	assert.Error(t, err)
}

func TestRunScan_WriteLockFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))

	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, nil))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	jarPath := filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")
	assert.NoError(t, afero.WriteFile(baseFs, jarPath, []byte("content"), 0644))

	fs := renameErrorFs{Fs: baseFs, failNew: meta.LockPath(), err: errors.New("rename failed")}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Prefer:     "modrinth",
		Add:        true,
	}, scanDeps{
		fs:        fs,
		logger:    logger.New(out, errOut, false, false),
		telemetry: func(telemetry.CommandTelemetry) {},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{
				ProjectID:     "proj-1",
				DatePublished: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Files: []modrinth.VersionFile{
					{URL: "https://example.invalid/mod.jar", Primary: true},
				},
			}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example Mod", nil
		},
		clients: platform.Clients{},
	})

	assert.Error(t, err)
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

type failingHasher struct{}

func (hasher *failingHasher) Write(_ []byte) (int, error) {
	return 0, errors.New("hash write failed")
}

func (hasher *failingHasher) Sum(data []byte) []byte {
	return data
}

func (hasher *failingHasher) Reset() {}

func (hasher *failingHasher) Size() int {
	return sha1.Size
}

func (hasher *failingHasher) BlockSize() int {
	return sha1.BlockSize
}

type closeErrorFile struct {
	afero.File
	closeErr error
}

func (file closeErrorFile) Close() error {
	closeErr := file.File.Close()
	if closeErr != nil && file.closeErr != nil {
		return errors.Join(closeErr, file.closeErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if file.closeErr != nil {
		return file.closeErr
	}
	return nil
}

type closeErrorFs struct {
	afero.Fs
	closeErr error
}

func (filesystem closeErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	return closeErrorFile{File: file, closeErr: filesystem.closeErr}, nil
}

func TestSha1ForFileReturnsHasherWriteError(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/example.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("content"), 0644))

	originalHasher := newSha1Hasher
	newSha1Hasher = func() hash.Hash {
		return &failingHasher{}
	}
	t.Cleanup(func() {
		newSha1Hasher = originalHasher
	})

	_, err := sha1ForFile(context.Background(), fs, path)
	assert.ErrorContains(t, err, "hash write failed")
}

func TestSha1ForFileReturnsCloseError(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/example.jar")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("content"), 0644))

	fs := closeErrorFs{Fs: base, closeErr: errors.New("close failed")}
	_, err := sha1ForFile(context.Background(), fs, path)
	assert.ErrorContains(t, err, "close failed")
}

func TestLookupModrinthReturnsUnsureOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	candidates := []scanCandidate{
		{Path: filepath.FromSlash("/mods/example.jar"), FileName: "example.jar", Sha1: "abc"},
	}

	matches, misses, unsure := lookupModrinth(ctx, candidates, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			t.Fatalf("unexpected lookup call after context cancellation")
			return nil, errors.New("unexpected lookup call after context cancellation")
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			t.Fatalf("unexpected title lookup after context cancellation")
			return "", errors.New("unexpected title lookup after context cancellation")
		},
		clients: platform.Clients{},
	})

	assert.Empty(t, matches)
	assert.Equal(t, candidates, misses)
	assert.Len(t, unsure, 1)
	assert.ErrorIs(t, unsure[candidates[0].Path], context.Canceled)
}

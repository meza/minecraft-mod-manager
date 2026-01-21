package scan

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
	"time"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type noopDoer struct{}

func (doer noopDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
	}, nil
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type errorDoer struct {
	err error
}

func (doer errorDoer) Do(*http.Request) (*http.Response, error) { return nil, doer.err }

type responseDoer struct {
	status int
	body   string
}

func (doer responseDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: doer.status,
		Body:       io.NopCloser(strings.NewReader(doer.body)),
		Header:     http.Header{},
	}, nil
}

func TestDefaultScanDepsRunInitUsesRunner(t *testing.T) {
	originalRunner := runInteractiveInit
	t.Cleanup(func() {
		runInteractiveInit = originalRunner
	})

	called := false
	runInteractiveInit = func(_ context.Context, _ *cobra.Command, _ initCmd.InteractiveInitDeps, options initCmd.InteractiveInitOptions) error {
		called = true
		assert.Equal(t, "/cfg/modlist.json", options.ConfigPath)
		return nil
	}

	cmd := &cobra.Command{}
	deps := defaultScanDeps(cmd, scanOptions{ConfigPath: "/cfg/modlist.json"})
	err := deps.runInit(context.Background(), cmd, initRequest{ConfigPath: "/cfg/modlist.json"})

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestNormalizePlatform(t *testing.T) {
	assert.Equal(t, models.MODRINTH, normalizePlatform("modrinth"))
	assert.Equal(t, models.CURSEFORGE, normalizePlatform("CURSEFORGE"))
	assert.Equal(t, models.Platform("custom"), normalizePlatform("Custom"))
}

func TestAlternatePlatform(t *testing.T) {
	assert.Equal(t, models.MODRINTH, alternatePlatform(models.CURSEFORGE))
	assert.Equal(t, models.CURSEFORGE, alternatePlatform(models.MODRINTH))
	assert.Equal(t, models.CURSEFORGE, alternatePlatform(models.Platform("custom")))
}

func TestResolvePreferredPlatformRejectsUnknownValue(t *testing.T) {
	platformValue, err := resolvePreferredPlatform("unknown-platform")
	assert.Error(t, err)
	assert.Empty(t, platformValue)
	assert.Equal(t, i18n.T("cmd.scan.error.unknown_platform", &i18n.Tvars{
		Data: &i18n.TData{"platform": "unknown-platform"},
	}), err.Error())
}

func TestModrinthDownloadDetailsErrors(t *testing.T) {
	_, err := modrinthDownloadDetails(nil)
	assert.Error(t, err)

	_, err = modrinthDownloadDetails(&modrinth.Version{})
	assert.Error(t, err)
}

func TestModrinthDownloadDetailsSelectsPrimaryFile(t *testing.T) {
	version := &modrinth.Version{
		DatePublished: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		Files: []modrinth.VersionFile{
			{URL: "https://example.invalid/secondary.jar"},
			{URL: "https://example.invalid/primary.jar", Primary: true},
		},
	}

	downloadInfo, err := modrinthDownloadDetails(version)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.invalid/primary.jar", downloadInfo.downloadURL)
	assert.Equal(t, "2024-01-02T03:04:05Z", downloadInfo.publishedAt)
}

func TestModrinthDownloadDetailsErrorsOnMissingURL(t *testing.T) {
	version := &modrinth.Version{
		DatePublished: time.Now(),
		Files:         []modrinth.VersionFile{{URL: ""}},
	}

	_, err := modrinthDownloadDetails(version)
	assert.Error(t, err)
}

func TestUniqueUint32s(t *testing.T) {
	assert.Equal(t, []uint32{3, 1, 2}, uniqueUint32s([]uint32{3, 1, 3, 2, 1}))
}

func TestLookupOnPlatformUnknownReturnsMisses(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
	}

	items, index := buildScanItems(candidates)
	state := newScanExecState(items, index, scanExecSender{})

	outcome, err := lookupPlatformWithUpdates(context.Background(), models.Platform("unknown"), candidates, scanDeps{}, state)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Equal(t, candidates, outcome.misses)
	assert.Empty(t, outcome.unsure)
}

func TestRunScanExecutionCombinesPreferredAndFallback(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/b.jar", FileName: "b.jar", Sha1: "b"},
	}

	version := &modrinth.Version{
		ProjectID:     "proj-a",
		Name:          "Modrinth Title",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files: []modrinth.VersionFile{
			{URL: "https://example.invalid/a.jar", Primary: true},
		},
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		modrinthVersionForSha: func(_ context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			if hash == "a" {
				return version, nil
			}
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup(hash, modrinth.SHA1)}
		},
		curseforgeFingerprint: func(path string) uint32 {
			if strings.Contains(path, "b.jar") {
				return 101
			}
			return 202
		},
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{
						ProjectID:   22,
						Fingerprint: 101,
						DisplayName: "Curse Name",
						DownloadURL: "https://example.invalid/b.jar",
						FileDate:    time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
					},
				},
			}, nil
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.CURSEFORGE,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.NoError(t, outcome.err)
	assert.Len(t, outcome.matches, 2)
	assert.Empty(t, outcome.unknown)
	assert.Empty(t, outcome.unsure)
	assert.ElementsMatch(t, []models.Platform{models.CURSEFORGE, models.MODRINTH}, []models.Platform{
		outcome.matches[0].Platform,
		outcome.matches[1].Platform,
	})
}

func TestRunScanExecutionRemovesUnsureWhenMatchedPathExists(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "b"},
	}

	version := &modrinth.Version{
		ProjectID:     "proj-1",
		Name:          "Example",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		modrinthVersionForSha: func(_ context.Context, sha string, _ httpclient.Doer) (*modrinth.Version, error) {
			if sha == "a" {
				return nil, httpclient.WrapTimeoutError(context.DeadlineExceeded)
			}
			return version, nil
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.NoError(t, outcome.err)
	assert.Len(t, outcome.matches, 1)
	assert.Empty(t, outcome.unknown)
	assert.Empty(t, outcome.unsure)
}

func TestRunScanExecutionSkipsUnknownWhenUnsure(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("modrinth error")
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.NoError(t, outcome.err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unknown)
	if assert.Len(t, outcome.unsure, 1) {
		assert.Equal(t, "a.jar", outcome.unsure[0].Path)
	}
}

func TestRunScanExecutionAddsUnknownWhenUnmatched(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		modrinthVersionForSha: func(_ context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup(hash, modrinth.SHA1)}
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.NoError(t, outcome.err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unsure)
	assert.Equal(t, []string{"a.jar"}, outcome.unknown)
}

func TestRunScanExecutionMergesFallbackUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		modrinthVersionForSha: func(_ context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup(hash, modrinth.SHA1)}
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("fingerprint failed")
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.NoError(t, outcome.err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unknown)
	if assert.Len(t, outcome.unsure, 1) {
		assert.Equal(t, "a.jar", outcome.unsure[0].Path)
		assert.Contains(t, outcome.unsure[0].Error.Error(), "cmd.scan.unsure.platform_error")
		assert.Contains(t, outcome.unsure[0].Error.Error(), "cmd.platform.error.reason.unknown")
	}
}

func TestRunScanExecutionSkipsUnknownWhenPathIsUnsure(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/dupe.jar", FileName: "dupe.jar", Sha1: "error"},
		{Path: "/mods/dupe.jar", FileName: "dupe.jar", Sha1: "miss"},
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		modrinthVersionForSha: func(_ context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			if hash == "error" {
				return nil, errors.New("modrinth error")
			}
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup(hash, modrinth.SHA1)}
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.NoError(t, outcome.err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unknown)
	if assert.Len(t, outcome.unsure, 1) {
		assert.Equal(t, "dupe.jar", outcome.unsure[0].Path)
	}
}

func TestListJarFilesReturnsErrorOnMissingFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	_, err := listJarFiles(fs, meta, cfg)
	assert.Error(t, err)
}

func TestListJarFilesFiltersDirectoriesAndNonJar(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	modsFolder := meta.ModsFolderPath(cfg)
	assert.NoError(t, fs.MkdirAll(filepath.Join(modsFolder, "dir"), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "good.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "notes.txt"), []byte("x"), 0644))

	files, err := listJarFiles(fs, meta, cfg)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.True(t, strings.HasSuffix(files[0], "good.jar"))
}

func TestListJarFilesReturnsErrorOnIgnoreStatFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("x"), 0644))

	fs := statErrorFs{Fs: baseFs, failPath: filepath.Join(meta.Dir(), ".mmmignore"), err: errors.New("stat failed")}

	_, err := listJarFiles(fs, meta, cfg)
	assert.ErrorContains(t, err, "stat failed")
}

func TestSha1ForFileSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/file.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	sum, err := sha1ForFile(context.Background(), fs, path)
	assert.NoError(t, err)

	expected := sha1.Sum([]byte("data"))
	assert.Equal(t, hex.EncodeToString(expected[:]), sum)
}

func TestSha1ForFileReturnsOpenError(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := sha1ForFile(context.Background(), fs, filepath.FromSlash("/missing.jar"))
	assert.Error(t, err)
}

func TestSha1ForFileReturnsContextError(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/file.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sha1ForFile(ctx, fs, path)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSha1ForFileReturnsReadError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/file.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	fs := readErrorFs{Fs: baseFs, failPath: path, err: errors.New("read failed")}
	_, err := sha1ForFile(context.Background(), fs, path)
	assert.ErrorContains(t, err, "read failed")
}

func TestSha1CandidatesReturnsError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/file.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	fs := readErrorFs{Fs: baseFs, failPath: path, err: errors.New("read failed")}
	_, err := sha1Candidates(context.Background(), fs, []string{path})
	assert.ErrorContains(t, err, "read failed")
}

func TestDefaultModrinthVersionForShaReturnsError(t *testing.T) {
	_, err := defaultModrinthVersionForSha(context.Background(), "deadbeef", errorDoer{err: errors.New("boom")})
	assert.Error(t, err)
}

func TestDefaultCurseforgeFingerprintMatchReturnsError(t *testing.T) {
	_, err := defaultCurseforgeFingerprintMatch(context.Background(), []uint32{1}, errorDoer{err: errors.New("boom")})
	assert.Error(t, err)
}

func TestLookupModrinthReturnsMatches(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/b.jar", FileName: "b.jar", Sha1: "b"},
	}

	version := &modrinth.Version{
		ProjectID:     "proj-1",
		Name:          "Example",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	deps := scanDeps{
		modrinthVersionForSha: func(_ context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
	}

	outcome, err := lookupModrinth(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Len(t, outcome.matches, 2)
	assert.Empty(t, outcome.misses)
	assert.Empty(t, outcome.unsure)
}

func TestLookupModrinthDownloadDetailsErrorAddsUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: ""}},
	}

	deps := scanDeps{
		modrinthVersionForSha: func(_ context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
	}

	outcome, err := lookupModrinth(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Len(t, outcome.misses, 1)
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.scan.unsure.platform_error")
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.platform.error.reason.unknown")
}

func TestLookupCurseforgeEmptyCandidatesReturnsMisses(t *testing.T) {
	outcome, err := lookupCurseforge(context.Background(), nil, scanDeps{})
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unsure)
	assert.Empty(t, outcome.misses)
}

func TestLookupCurseforgeMissingDownloadURLAddsUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	deps := scanDeps{
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 22, Fingerprint: 101, DownloadURL: ""},
				},
			}, nil
		},
	}

	outcome, err := lookupCurseforge(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Len(t, outcome.misses, 1)
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.scan.unsure.platform_error")
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.platform.error.reason.unknown")
}

func TestLookupCurseforgeSkipsUnknownFingerprint(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/b.jar", FileName: "b.jar", Sha1: "b"},
	}

	deps := scanDeps{
		curseforgeFingerprint: func(path string) uint32 {
			if strings.Contains(path, "a.jar") {
				return 101
			}
			return 202
		},
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 42, Fingerprint: 999, DownloadURL: "https://example.invalid/extra.jar"},
				},
			}, nil
		},
	}

	outcome, err := lookupCurseforge(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unsure)
	assert.Len(t, outcome.misses, 2)
}

func TestSummarizePlatformFailureWithNilError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	summary := summarizePlatformFailure(nil, models.CURSEFORGE)
	assert.Contains(t, summary.Reason, "cmd.platform.error.reason.unknown")
	assert.Equal(t, "", summary.DebugDetails)
}

func TestPersistScanMatchesReturnsErrorWhenSetupCoordinatorMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	input := scanExecutionInput{
		meta:             meta,
		cfg:              models.ModsJSON{ModsFolder: "mods"},
		lock:             nil,
		setupCoordinator: nil,
		deps:             scanDeps{fs: fs},
	}
	cmd := &cobra.Command{}
	_, err := persistScanMatches(context.Background(), cmd, input, []scanMatch{{FileName: "bad.jar"}})
	assert.Error(t, err)
}

func TestLogPlatformDebugReturnsError(t *testing.T) {
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true)

	err := logPlatformDebug(log, models.MODRINTH, "details")
	assert.ErrorIs(t, err, writeErr)
}

func TestLookupModrinthCandidateReturnsErrorWhenLogFails(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := scanDeps{
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", Sha1: "a"}, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestLookupCurseforgeReturnsErrorWhenLogFails(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		clients: platform.Clients{
			Curseforge: noopDoer{},
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := lookupCurseforge(context.Background(), []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestRecordCurseforgeUnsureReturnsErrorWhenLogFails(t *testing.T) {
	writeErr := errors.New("write failed")
	matchContext := curseforgeMatchContext{
		candidates: []scanCandidate{{Path: "/mods/a.jar"}},
		deps: scanDeps{
			logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		},
		unsure: map[string]error{},
	}

	err := recordCurseforgeUnsure(matchContext, []int{0}, errors.New("boom"))
	assert.ErrorIs(t, err, writeErr)
}

func TestLookupCurseforgeReturnsUnsureOnFingerprintError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	deps := scanDeps{
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, httpclient.WrapTimeoutError(context.DeadlineExceeded)
		},
	}

	outcome, err := lookupCurseforge(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.misses)
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.scan.unsure.platform_error")
}

func TestLookupCurseforgeReturnsUnsureAndMissesOnFingerprintNonConnectionError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	deps := scanDeps{
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	}

	outcome, err := lookupCurseforge(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Len(t, outcome.misses, 1)
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.scan.unsure.platform_error")
}

func TestLookupCurseforgeReturnsErrorOnUnsureLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	deps := scanDeps{
		logger:                logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 22, Fingerprint: 101, DownloadURL: ""},
				},
			}, nil
		},
	}

	_, err := lookupCurseforge(context.Background(), candidates, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestAddCurseforgeMatchesSkipsUnknownFingerprint(t *testing.T) {
	matchContext := curseforgeMatchContext{
		candidates: []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}},
		matches: []curseforge.File{
			{ProjectID: 22, Fingerprint: 999, DownloadURL: "https://example.invalid/a.jar"},
		},
		fingerprintToIndices: map[uint32][]int{},
		deps:                 scanDeps{},
		scanMatches:          &[]scanMatch{},
		unsure:               map[string]error{},
	}

	assert.NoError(t, addCurseforgeMatches(matchContext))
	assert.Empty(t, *matchContext.scanMatches)
}

func TestAddCurseforgeMatchesReturnsErrorOnUnsureLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	matchContext := curseforgeMatchContext{
		candidates: []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}},
		matches: []curseforge.File{
			{ProjectID: 22, Fingerprint: 101, DownloadURL: ""},
		},
		fingerprintToIndices: map[uint32][]int{101: {0}},
		deps: scanDeps{
			logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		},
		scanMatches: &[]scanMatch{},
		unsure:      map[string]error{},
	}

	err := addCurseforgeMatches(matchContext)
	assert.ErrorIs(t, err, writeErr)
}

func TestLookupModrinthReturnsContextCanceledOutcome(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	outcome, err := lookupModrinth(ctx, candidates, scanDeps{})
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Equal(t, candidates, outcome.misses)
	assert.ErrorIs(t, outcome.unsure["/mods/a.jar"], context.Canceled)
}

func TestLookupModrinthReturnsErrorOnLookupFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	deps := scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := lookupModrinth(context.Background(), candidates, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestRunModrinthLookupsReturnsErrorOnLookupFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	_, err := runModrinthLookups(context.Background(), []scanCandidate{{Path: "/mods/a.jar", Sha1: "a"}}, scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestLookupModrinthCandidateReturnsMissOnNotFound(t *testing.T) {
	result, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", Sha1: "a"}, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	})
	assert.NoError(t, err)
	assert.True(t, result.miss)
}

func TestLookupModrinthCandidateReturnsMatchOnSuccess(t *testing.T) {
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		Name:          "Example",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	result, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
	})
	assert.NoError(t, err)
	if assert.NotNil(t, result.match) {
		assert.Equal(t, "proj-1", result.match.ProjectID)
	}
}

func TestLookupModrinthCandidateUsesVersionNumberWhenNameEmpty(t *testing.T) {
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		Name:          "",
		VersionNumber: "1.2.3",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	result, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
	})
	assert.NoError(t, err)
	if assert.NotNil(t, result.match) {
		assert.Equal(t, "1.2.3", result.match.Name)
	}
}

func TestLookupModrinthCandidateUsesProjectIDWhenNameAndNumberEmpty(t *testing.T) {
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		Name:          "",
		VersionNumber: "",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	result, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
	})
	assert.NoError(t, err)
	if assert.NotNil(t, result.match) {
		assert.Equal(t, "proj-1", result.match.Name)
	}
}

func TestLookupModrinthCandidateReturnsUnsureOnDownloadDetailsError(t *testing.T) {
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		Name:          "Example",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: ""}},
	}

	result, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
	})
	assert.NoError(t, err)
	assert.Error(t, result.err)
}

func TestLookupModrinthCandidateReturnsErrorOnDownloadLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		Name:          "Example",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: ""}},
	}

	_, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestSplitModrinthResults(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/match.jar", FileName: "match.jar"},
		{Path: "/mods/miss.jar", FileName: "miss.jar"},
		{Path: "/mods/fallback.jar", FileName: "fallback.jar"},
		{Path: "/mods/unsure.jar", FileName: "unsure.jar"},
	}
	results := []modrinthLookupResult{
		{match: &scanMatch{FileName: "match.jar", Name: "Match", ProjectID: "match", Platform: models.MODRINTH}},
		{miss: true},
		{err: errors.New("fallback"), allowFallback: true},
		{err: errors.New("unsure"), allowFallback: false},
	}

	outcome := splitModrinthResults(candidates, results)
	assert.Len(t, outcome.matches, 1)
	assert.Len(t, outcome.misses, 2)
	assert.Len(t, outcome.unsure, 2)
	assert.Contains(t, outcome.unsure, "/mods/fallback.jar")
	assert.Contains(t, outcome.unsure, "/mods/unsure.jar")
}

func TestRunScanExecutionReturnsErrorOnLookupFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	deps := scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.ErrorIs(t, outcome.err, writeErr)
}

func TestRunScanExecutionReturnsErrorOnFallbackFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	deps := scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	}

	input := scanExecutionInput{
		candidates:     candidates,
		preferPlatform: models.MODRINTH,
		deps:           deps,
	}
	outcome := runScanExecution(context.Background(), input, scanExecSender{})
	assert.ErrorIs(t, outcome.err, writeErr)
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Stat(name)
}

type readErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem readErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return readErrorFile{File: file, err: filesystem.err}, nil
	}
	return file, nil
}

type readErrorFile struct {
	afero.File
	err error
}

func (file readErrorFile) Read([]byte) (int, error) {
	return 0, file.err
}

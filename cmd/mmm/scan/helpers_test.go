package scan

import (
	"bytes"
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

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type noopDoer struct{}

func (n noopDoer) Do(*http.Request) (*http.Response, error) { return nil, nil }

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

type errorDoer struct {
	err error
}

func (e errorDoer) Do(*http.Request) (*http.Response, error) { return nil, e.err }

type responseDoer struct {
	status int
	body   string
}

func (r responseDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: r.status,
		Body:       io.NopCloser(strings.NewReader(r.body)),
		Header:     http.Header{},
	}, nil
}

func TestReadLineReturnsLine(t *testing.T) {
	line, err := readLine(strings.NewReader("yes\n"))
	assert.NoError(t, err)
	assert.Equal(t, "yes", line)
}

func TestReadLineReturnsEOFOnEmptyInput(t *testing.T) {
	_, err := readLine(strings.NewReader(""))
	assert.ErrorIs(t, err, io.EOF)
}

func TestReadLineReturnsErrorOnReadFailure(t *testing.T) {
	_, err := readLine(errorReader{})
	assert.Error(t, err)
}

func TestTerminalPrompterConfirmAddYes(t *testing.T) {
	prompter := terminalPrompter{
		in:  strings.NewReader("Y\n"),
		out: io.Discard,
	}

	confirmed, err := prompter.ConfirmAdd()
	assert.NoError(t, err)
	assert.True(t, confirmed)
}

func TestTerminalPrompterConfirmAddNo(t *testing.T) {
	prompter := terminalPrompter{
		in:  strings.NewReader("no\n"),
		out: io.Discard,
	}

	confirmed, err := prompter.ConfirmAdd()
	assert.NoError(t, err)
	assert.False(t, confirmed)
}

func TestTerminalPrompterConfirmAddError(t *testing.T) {
	prompter := terminalPrompter{
		in:  errorReader{},
		out: io.Discard,
	}

	confirmed, err := prompter.ConfirmAdd()
	assert.Error(t, err)
	assert.False(t, confirmed)
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

func TestModrinthDownloadDetailsErrors(t *testing.T) {
	_, _, err := modrinthDownloadDetails(nil)
	assert.Error(t, err)

	_, _, err = modrinthDownloadDetails(&modrinth.Version{})
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

	url, published, err := modrinthDownloadDetails(version)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.invalid/primary.jar", url)
	assert.Equal(t, "2024-01-02T03:04:05Z", published)
}

func TestModrinthDownloadDetailsErrorsOnMissingURL(t *testing.T) {
	version := &modrinth.Version{
		DatePublished: time.Now(),
		Files:         []modrinth.VersionFile{{URL: ""}},
	}

	_, _, err := modrinthDownloadDetails(version)
	assert.Error(t, err)
}

func TestUniqueInts(t *testing.T) {
	assert.Equal(t, []int{3, 1, 2}, uniqueInts([]int{3, 1, 3, 2, 1}))
}

func TestPrintResultsLogsAllSections(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var out bytes.Buffer
	log := logger.New(&out, io.Discard, false, false)

	matches := []scanMatch{
		{Path: "/mods/a.jar", Platform: models.MODRINTH, ProjectID: "a", Name: "Alpha", FileName: "a.jar"},
	}
	unknown := []string{"/mods/b.jar"}
	unsure := []scanUnsure{{Path: "/mods/c.jar", Error: errors.New("nope")}}

	printResults(log, &out, models.MODRINTH, matches, unknown, unsure)

	output := out.String()
	assert.Contains(t, output, "cmd.scan.recognized.header")
	assert.Contains(t, output, "cmd.scan.unknown.header")
	assert.Contains(t, output, "cmd.scan.unsure.header")
}

func TestPrintResultsColorizesWhenTerminal(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	var out bytes.Buffer
	tty := fakeTTY{Buffer: &out}
	log := logger.New(&tty, io.Discard, false, false)

	printResults(log, &tty, models.MODRINTH, []scanMatch{
		{Path: "/mods/a.jar", Platform: models.MODRINTH, ProjectID: "a", Name: "Alpha", FileName: "a.jar"},
	}, nil, nil)

	assert.Contains(t, out.String(), "cmd.scan.recognized.header")
}

func TestPrintResultsUsesUnknownErrorWhenNil(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var out bytes.Buffer
	log := logger.New(&out, io.Discard, false, false)

	printResults(log, &out, models.MODRINTH, nil, nil, []scanUnsure{{Path: "/mods/a.jar"}})
	assert.Contains(t, out.String(), "unknown error")
}

func TestPrintResultsLogsNoResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var out bytes.Buffer
	log := logger.New(&out, io.Discard, false, false)

	printResults(log, &out, models.MODRINTH, nil, nil, nil)
	assert.Contains(t, out.String(), "cmd.scan.no_results")
}

func TestLookupOnPlatformUnknownReturnsMisses(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
	}

	matches, misses, unsure := lookupOnPlatform(context.Background(), candidates, models.Platform("unknown"), scanDeps{})
	assert.Empty(t, matches)
	assert.Equal(t, candidates, misses)
	assert.Empty(t, unsure)
}

func TestIdentifyCandidatesCombinesPreferredAndFallback(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/b.jar", FileName: "b.jar", Sha1: "b"},
	}

	version := &modrinth.Version{
		ProjectID:     "proj-a",
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
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Modrinth Title", nil
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 22, Fingerprint: 101, DownloadURL: "https://example.invalid/b.jar", FileDate: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Curse Name", nil
		},
	}

	matches, unknown, unsure := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.Len(t, matches, 2)
	assert.Empty(t, unknown)
	assert.Empty(t, unsure)
	assert.Equal(t, models.MODRINTH, matches[0].Platform)
	assert.Equal(t, models.CURSEFORGE, matches[1].Platform)
}

func TestIdentifyCandidatesRemovesUnsureWhenMatchedPathExists(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "b"},
	}

	version := &modrinth.Version{
		ProjectID:     "proj-1",
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
				return nil, &modrinth.VersionAPIError{}
			}
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}

	matches, unknown, unsure := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.Len(t, matches, 1)
	assert.Empty(t, unknown)
	assert.Empty(t, unsure)
}

func TestIdentifyCandidatesSkipsUnknownWhenUnsure(t *testing.T) {
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
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	matches, unknown, unsure := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.Empty(t, matches)
	assert.Empty(t, unknown)
	if assert.Len(t, unsure, 1) {
		assert.Equal(t, "/mods/a.jar", unsure[0].Path)
	}
}

func TestIdentifyCandidatesAddsUnknownWhenUnmatched(t *testing.T) {
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
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	matches, unknown, unsure := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.Empty(t, matches)
	assert.Empty(t, unsure)
	assert.Equal(t, []string{"/mods/a.jar"}, unknown)
}

func TestIdentifyCandidatesMergesFallbackUnsure(t *testing.T) {
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
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("fingerprint failed")
		},
	}

	matches, unknown, unsure := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.Empty(t, matches)
	assert.Empty(t, unknown)
	if assert.Len(t, unsure, 1) {
		assert.Equal(t, "/mods/a.jar", unsure[0].Path)
		assert.Contains(t, unsure[0].Error.Error(), "fingerprint failed")
	}
}

func TestIdentifyCandidatesSkipsUnknownWhenPathIsUnsure(t *testing.T) {
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
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	matches, unknown, unsure := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.Empty(t, matches)
	assert.Empty(t, unknown)
	if assert.Len(t, unsure, 1) {
		assert.Equal(t, "/mods/dupe.jar", unsure[0].Path)
	}
}

func TestIdentifyCandidatesSortsMatchesByNameAndFileName(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/b.jar", FileName: "b.jar", Sha1: "b"},
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/c.jar", FileName: "c.jar", Sha1: "c"},
	}

	versionFor := func(projectID string) *modrinth.Version {
		return &modrinth.Version{
			ProjectID:     projectID,
			DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Files: []modrinth.VersionFile{
				{URL: "https://example.invalid/" + projectID + ".jar", Primary: true},
			},
		}
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth:   noopDoer{},
			Curseforge: noopDoer{},
		},
		modrinthVersionForSha: func(_ context.Context, hash string, _ httpclient.Doer) (*modrinth.Version, error) {
			switch hash {
			case "a", "c":
				return versionFor("proj-alpha"), nil
			case "b":
				return versionFor("proj-beta"), nil
			default:
				return nil, &modrinth.VersionNotFoundError{}
			}
		},
		modrinthProjectTitle: func(_ context.Context, projectID string, _ httpclient.Doer) (string, error) {
			switch projectID {
			case "proj-alpha":
				return "Alpha", nil
			case "proj-beta":
				return "Beta", nil
			default:
				return "Unknown", nil
			}
		},
	}

	matches, unknown, unsure := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.Empty(t, unknown)
	assert.Empty(t, unsure)
	if assert.Len(t, matches, 3) {
		assert.Equal(t, "Alpha", matches[0].Name)
		assert.Equal(t, "a.jar", matches[0].FileName)
		assert.Equal(t, "Alpha", matches[1].Name)
		assert.Equal(t, "c.jar", matches[1].FileName)
		assert.Equal(t, "Beta", matches[2].Name)
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

func TestDefaultModrinthProjectTitle(t *testing.T) {
	title, err := defaultModrinthProjectTitle(context.Background(), "proj", responseDoer{
		status: http.StatusOK,
		body:   `{"title":"Example"}`,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Example", title)

	_, err = defaultModrinthProjectTitle(context.Background(), "proj", errorDoer{err: errors.New("boom")})
	assert.Error(t, err)
}

func TestDefaultCurseforgeFingerprintMatchReturnsError(t *testing.T) {
	_, err := defaultCurseforgeFingerprintMatch(context.Background(), []int{1}, errorDoer{err: errors.New("boom")})
	assert.Error(t, err)
}

func TestDefaultCurseforgeProjectName(t *testing.T) {
	name, err := defaultCurseforgeProjectName(context.Background(), "123", responseDoer{
		status: http.StatusOK,
		body:   `{"data":{"name":"Example"}}`,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Example", name)

	_, err = defaultCurseforgeProjectName(context.Background(), "123", errorDoer{err: errors.New("boom")})
	assert.Error(t, err)
}

func TestLookupModrinthCachesProjectTitles(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
		{Path: "/mods/b.jar", FileName: "b.jar", Sha1: "b"},
	}

	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	var titleCalls int
	deps := scanDeps{
		modrinthVersionForSha: func(_ context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			titleCalls++
			return "Example", nil
		},
	}

	matches, misses, unsure := lookupModrinth(context.Background(), candidates, deps)
	assert.Len(t, matches, 2)
	assert.Empty(t, misses)
	assert.Empty(t, unsure)
	assert.Equal(t, 1, titleCalls)
}

func TestLookupModrinthProjectTitleErrorAddsUnsure(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	deps := scanDeps{
		modrinthVersionForSha: func(_ context.Context, _ string, _ httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
	}

	matches, misses, unsure := lookupModrinth(context.Background(), candidates, deps)
	assert.Empty(t, matches)
	assert.Empty(t, misses)
	assert.Contains(t, unsure["/mods/a.jar"].Error(), "boom")
}

func TestLookupModrinthDownloadDetailsErrorAddsUnsure(t *testing.T) {
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
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}

	matches, misses, unsure := lookupModrinth(context.Background(), candidates, deps)
	assert.Empty(t, matches)
	assert.Empty(t, misses)
	assert.Contains(t, unsure["/mods/a.jar"].Error(), "missing url")
}

func TestLookupCurseforgeEmptyCandidatesReturnsMisses(t *testing.T) {
	matches, misses, unsure := lookupCurseforge(context.Background(), nil, scanDeps{})
	assert.Empty(t, matches)
	assert.Empty(t, unsure)
	assert.Empty(t, misses)
}

func TestLookupCurseforgeMissingDownloadURLAddsUnsure(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	deps := scanDeps{
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 22, Fingerprint: 101, DownloadURL: ""},
				},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Curse Name", nil
		},
	}

	matches, misses, unsure := lookupCurseforge(context.Background(), candidates, deps)
	assert.Empty(t, matches)
	assert.Empty(t, misses)
	assert.Contains(t, unsure["/mods/a.jar"].Error(), "missing download url")
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
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 42, Fingerprint: 999, DownloadURL: "https://example.invalid/extra.jar"},
				},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			t.Fatal("curseforgeProjectName should not be called")
			return "", nil
		},
	}

	matches, misses, unsure := lookupCurseforge(context.Background(), candidates, deps)
	assert.Empty(t, matches)
	assert.Empty(t, unsure)
	assert.Len(t, misses, 2)
}

func TestLookupCurseforgeProjectNameErrorAddsUnsure(t *testing.T) {
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	deps := scanDeps{
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{
					{ProjectID: 22, Fingerprint: 101, DownloadURL: "https://example.invalid/a.jar"},
				},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
	}

	matches, misses, unsure := lookupCurseforge(context.Background(), candidates, deps)
	assert.Empty(t, matches)
	assert.Empty(t, misses)
	assert.Contains(t, unsure["/mods/a.jar"].Error(), "boom")
}

func TestCurseforgeFingerprintFailureReasonNon403(t *testing.T) {
	assert.Equal(t, "unexpected status code: 500", curseforgeFingerprintFailureReason(errors.New("unexpected status code: 500")))
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (s statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(s.failPath) {
		return nil, s.err
	}
	return s.Fs.Stat(name)
}

type readErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (r readErrorFs) Open(name string) (afero.File, error) {
	file, err := r.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(r.failPath) {
		return readErrorFile{File: file, err: r.err}, nil
	}
	return file, nil
}

type readErrorFile struct {
	afero.File
	err error
}

func (r readErrorFile) Read([]byte) (int, error) {
	return 0, r.err
}

type fakeTTY struct {
	*bytes.Buffer
}

func (f fakeTTY) Fd() uintptr { return 0 }

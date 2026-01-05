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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
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

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type selectErrorWriter struct {
	err error
}

func (writer selectErrorWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "cmd.scan.confirm_init") {
		return 0, writer.err
	}
	return len(p), nil
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

func TestTerminalPrompterConfirmAddWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	prompter := terminalPrompter{
		in:  strings.NewReader("y\n"),
		out: errorWriter{err: writeErr},
	}

	confirmed, err := prompter.ConfirmAdd()
	assert.ErrorIs(t, err, writeErr)
	assert.False(t, confirmed)
}

func TestTerminalPrompterConfirmInitYes(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	prompter := terminalPrompter{
		in:  strings.NewReader("cmd.init.prompt.option.yes.short\n"),
		out: out,
	}

	confirmed, err := prompter.ConfirmInit("modlist.json")
	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.Contains(t, out.String(), "cmd.scan.config_missing")
	assert.Contains(t, out.String(), "cmd.scan.confirm_init")
}

func TestTerminalPrompterConfirmInitWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	prompter := terminalPrompter{
		in:  strings.NewReader("y\n"),
		out: errorWriter{err: writeErr},
	}

	confirmed, err := prompter.ConfirmInit("modlist.json")
	assert.ErrorIs(t, err, writeErr)
	assert.False(t, confirmed)
}

func TestTerminalPrompterConfirmInitPromptWriteError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	writeErr := errors.New("write failed")
	prompter := terminalPrompter{
		in:  strings.NewReader("cmd.init.prompt.option.yes.short\n"),
		out: selectErrorWriter{err: writeErr},
	}

	confirmed, err := prompter.ConfirmInit("modlist.json")
	assert.ErrorIs(t, err, writeErr)
	assert.False(t, confirmed)
}

func TestTerminalPrompterConfirmInitReadError(t *testing.T) {
	prompter := terminalPrompter{
		in:  errorReader{},
		out: io.Discard,
	}

	confirmed, err := prompter.ConfirmInit("modlist.json")
	assert.Error(t, err)
	assert.False(t, confirmed)
}

func TestNoopPrompterConfirmInitReturnsFalse(t *testing.T) {
	confirmed, err := noopPrompter{}.ConfirmInit("modlist.json")
	assert.NoError(t, err)
	assert.False(t, confirmed)
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

func TestPrintResultsLogsAllSections(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var out bytes.Buffer
	outWriter := output.New(&out, &out, false)

	matches := []scanMatch{
		{Path: "/mods/a.jar", Platform: models.MODRINTH, ProjectID: "a", Name: "Alpha", FileName: "a.jar"},
	}
	unknown := []string{"/mods/b.jar"}
	unsure := []scanUnsure{{Path: "/mods/c.jar", Error: errors.New("nope")}}

	err := printResults(outWriter, &out, models.MODRINTH, matches, unknown, unsure)
	assert.NoError(t, err)

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
	outWriter := output.New(&tty, &tty, false)

	err := printResults(outWriter, &tty, models.MODRINTH, []scanMatch{
		{Path: "/mods/a.jar", Platform: models.MODRINTH, ProjectID: "a", Name: "Alpha", FileName: "a.jar"},
	}, nil, nil)
	assert.NoError(t, err)

	assert.Contains(t, out.String(), "cmd.scan.recognized.header")
}

func TestPrintResultsUsesUnknownErrorWhenNil(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var out bytes.Buffer
	outWriter := output.New(&out, &out, false)

	err := printResults(outWriter, &out, models.MODRINTH, nil, nil, []scanUnsure{{Path: "/mods/a.jar"}})
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "unknown error")
}

func TestPrintResultsLogsNoResults(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	var out bytes.Buffer
	outWriter := output.New(&out, &out, false)

	err := printResults(outWriter, &out, models.MODRINTH, nil, nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.scan.no_results")
}

func TestLookupOnPlatformUnknownReturnsMisses(t *testing.T) {
	candidates := []scanCandidate{
		{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"},
	}

	outcome, err := lookupOnPlatform(context.Background(), candidates, models.Platform("unknown"), scanDeps{})
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Equal(t, candidates, outcome.misses)
	assert.Empty(t, outcome.unsure)
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
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

	identification, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.NoError(t, err)
	assert.Len(t, identification.matches, 2)
	assert.Empty(t, identification.unknown)
	assert.Empty(t, identification.unsure)
	assert.Equal(t, models.MODRINTH, identification.matches[0].Platform)
	assert.Equal(t, models.CURSEFORGE, identification.matches[1].Platform)
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
				return nil, httpclient.WrapTimeoutError(context.DeadlineExceeded)
			}
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	identification, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.NoError(t, err)
	assert.Len(t, identification.matches, 1)
	assert.Empty(t, identification.unknown)
	assert.Empty(t, identification.unsure)
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	identification, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.NoError(t, err)
	assert.Empty(t, identification.matches)
	assert.Empty(t, identification.unknown)
	if assert.Len(t, identification.unsure, 1) {
		assert.Equal(t, "/mods/a.jar", identification.unsure[0].Path)
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	identification, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.NoError(t, err)
	assert.Empty(t, identification.matches)
	assert.Empty(t, identification.unsure)
	assert.Equal(t, []string{"/mods/a.jar"}, identification.unknown)
}

func TestIdentifyCandidatesMergesFallbackUnsure(t *testing.T) {
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

	identification, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.NoError(t, err)
	assert.Empty(t, identification.matches)
	assert.Empty(t, identification.unknown)
	if assert.Len(t, identification.unsure, 1) {
		assert.Equal(t, "/mods/a.jar", identification.unsure[0].Path)
		assert.Contains(t, identification.unsure[0].Error.Error(), "cmd.scan.unsure.platform_error")
		assert.Contains(t, identification.unsure[0].Error.Error(), "cmd.platform.error.reason.unknown")
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
	}

	identification, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.NoError(t, err)
	assert.Empty(t, identification.matches)
	assert.Empty(t, identification.unknown)
	if assert.Len(t, identification.unsure, 1) {
		assert.Equal(t, "/mods/dupe.jar", identification.unsure[0].Path)
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

	identification, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.NoError(t, err)
	assert.Empty(t, identification.unknown)
	assert.Empty(t, identification.unsure)
	if assert.Len(t, identification.matches, 3) {
		assert.Equal(t, "Alpha", identification.matches[0].Name)
		assert.Equal(t, "a.jar", identification.matches[0].FileName)
		assert.Equal(t, "Alpha", identification.matches[1].Name)
		assert.Equal(t, "c.jar", identification.matches[1].FileName)
		assert.Equal(t, "Beta", identification.matches[2].Name)
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
	_, err := defaultCurseforgeFingerprintMatch(context.Background(), []uint32{1}, errorDoer{err: errors.New("boom")})
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

	outcome, err := lookupModrinth(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Len(t, outcome.matches, 2)
	assert.Empty(t, outcome.misses)
	assert.Empty(t, outcome.unsure)
	assert.Equal(t, 1, titleCalls)
}

func TestModrinthTitleCacheReusesCachedTitle(t *testing.T) {
	cache := newModrinthTitleCache()
	var titleCalls atomic.Int32

	deps := scanDeps{
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			titleCalls.Add(1)
			return "Example", nil
		},
	}

	title, err := cachedModrinthTitle(context.Background(), "proj-1", deps, cache)
	assert.NoError(t, err)
	assert.Equal(t, "Example", title)

	title, err = cachedModrinthTitle(context.Background(), "proj-1", deps, cache)
	assert.NoError(t, err)
	assert.Equal(t, "Example", title)
	assert.Equal(t, int32(1), titleCalls.Load())
}

func TestModrinthTitleCacheWaitsForInflightFetch(t *testing.T) {
	cache := newModrinthTitleCache()
	started := make(chan struct{})
	secondReady := make(chan struct{})
	finish := make(chan struct{})
	var titleCalls atomic.Int32

	deps := scanDeps{
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			titleCalls.Add(1)
			close(started)
			<-finish
			return "Example", nil
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)

	results := make([]string, 2)
	errors := make([]error, 2)

	go func() {
		defer wg.Done()
		results[0], errors[0] = cachedModrinthTitle(context.Background(), "proj-1", deps, cache)
	}()

	go func() {
		defer wg.Done()
		<-started
		close(secondReady)
		results[1], errors[1] = cachedModrinthTitle(context.Background(), "proj-1", deps, cache)
	}()

	<-secondReady
	close(finish)
	wg.Wait()

	assert.Equal(t, int32(1), titleCalls.Load())
	assert.Equal(t, "Example", results[0])
	assert.Equal(t, "Example", results[1])
	assert.NoError(t, errors[0])
	assert.NoError(t, errors[1])
}

func TestModrinthTitleCacheReturnsContextErrorWhileWaiting(t *testing.T) {
	cache := newModrinthTitleCache()
	started := make(chan struct{})
	finish := make(chan struct{})

	deps := scanDeps{
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			close(started)
			<-finish
			return "Example", nil
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	firstErr := make(chan error, 1)

	go func() {
		defer wg.Done()
		_, err := cachedModrinthTitle(context.Background(), "proj-1", deps, cache)
		firstErr <- err
	}()

	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	title, err := cachedModrinthTitle(ctx, "proj-1", deps, cache)
	assert.Empty(t, title)
	assert.ErrorIs(t, err, context.Canceled)

	close(finish)
	wg.Wait()
	assert.NoError(t, <-firstErr)
}

func TestLookupModrinthProjectTitleErrorAddsUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

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
			return "", httpclient.WrapTimeoutError(context.DeadlineExceeded)
		},
	}

	outcome, err := lookupModrinth(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.misses)
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.scan.unsure.platform_error")
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.platform.error.reason.timeout")
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
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
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
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Curse Name", nil
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
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			t.Fatal("curseforgeProjectName should not be called")
			return "", nil
		},
	}

	outcome, err := lookupCurseforge(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Empty(t, outcome.unsure)
	assert.Len(t, outcome.misses, 2)
}

func TestLookupCurseforgeProjectNameErrorAddsUnsure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	deps := scanDeps{
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
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

	outcome, err := lookupCurseforge(context.Background(), candidates, deps)
	assert.NoError(t, err)
	assert.Empty(t, outcome.matches)
	assert.Len(t, outcome.misses, 1)
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.scan.unsure.platform_error")
	assert.Contains(t, outcome.unsure["/mods/a.jar"].Error(), "cmd.platform.error.reason.unknown")
}

func TestCachedCurseforgeProjectNameCaches(t *testing.T) {
	cache := make(map[string]string)
	var mu sync.Mutex
	callCount := 0

	deps := scanDeps{
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			callCount++
			return "project-name", nil
		},
	}

	name, err := cachedCurseforgeProjectName(context.Background(), "123", deps, cache, &mu)
	assert.NoError(t, err)
	assert.Equal(t, "project-name", name)

	name, err = cachedCurseforgeProjectName(context.Background(), "123", deps, cache, &mu)
	assert.NoError(t, err)
	assert.Equal(t, "project-name", name)
	assert.Equal(t, 1, callCount)
}

func TestSummarizePlatformFailureWithNilError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	summary := summarizePlatformFailure(nil, models.CURSEFORGE)
	assert.Contains(t, summary.Reason, "cmd.platform.error.reason.unknown")
	assert.Equal(t, "", summary.DebugDetails)
}

func TestPersistScanMatchesLogsColorizedErrors(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	defer restore()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{}
	setupCoordinator := modsetup.NewSetupCoordinator(fs, nil, nil)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(fakeTTY{Buffer: out})

	deps := scanDeps{
		fs:     fs,
		logger: logger.New(out, errOut, false, false),
		output: output.New(out, errOut, false),
	}

	persisted, err := persistScanMatches(context.Background(), cmd, meta, setupCoordinator, deps, []scanMatch{
		{Platform: models.MODRINTH, ProjectID: "abc", FileName: "bad.jar"},
	}, cfg, lock)
	assert.NoError(t, err)
	assert.False(t, persisted)
	assert.Contains(t, out.String(), "bad.jar")
}

func TestReportAllManagedReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	_, err := reportAllManaged(outWriter, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestIdentifyAndPrintCandidatesReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	deps := scanDeps{
		clients: platform.Clients{
			Modrinth: noopDoer{},
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}

	_, err := identifyAndPrintCandidates(context.Background(), outWriter, io.Discard, models.MODRINTH, candidates, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintMatchResultsReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := printMatchResults(outWriter, tui.ColorDisabled, []scanMatch{
		{Path: "/mods/a.jar", Platform: models.MODRINTH, ProjectID: "a", Name: "Alpha", FileName: "a.jar"},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintUnknownResultsReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := printUnknownResults(outWriter, tui.ColorDisabled, []string{"/mods/unknown.jar"})
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintUnsureResultsReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := printUnsureResults(outWriter, tui.ColorDisabled, []scanUnsure{{Path: "/mods/unsure.jar", Error: errors.New("nope")}})
	assert.ErrorIs(t, err, writeErr)
}

func TestResolvePreferredPlatformReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	_, err := resolvePreferredPlatform("unknown", outWriter)
	assert.ErrorIs(t, err, writeErr)
}

func TestPersistScanMatchesIfRequestedReturnsErrorWhenUnsureLogFails(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	_, err := persistScanMatchesIfRequested(persistScanRequest{
		Context:        context.Background(),
		Options:        scanOptions{Add: true},
		Dependencies:   scanDeps{output: outWriter},
		PreferPlatform: models.MODRINTH,
		Unsure:         []scanUnsure{{Path: "/mods/unsure.jar", Error: errors.New("nope")}},
	})
	assert.ErrorIs(t, err, writeErr)
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

	_, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", Sha1: "a"}, deps, newModrinthTitleCache())
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
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Curse Name", nil
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
		deps: scanDeps{
			curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
				t.Fatal("curseforgeProjectName should not be called")
				return "", nil
			},
		},
		scanMatches: &[]scanMatch{},
		unsure:      map[string]error{},
	}

	assert.NoError(t, addCurseforgeMatches(context.Background(), matchContext))
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
			curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
				return "Curse Name", nil
			},
		},
		nameCache:   map[string]string{},
		nameMu:      &sync.Mutex{},
		scanMatches: &[]scanMatch{},
		unsure:      map[string]error{},
	}

	err := addCurseforgeMatches(context.Background(), matchContext)
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
	}, newModrinthTitleCache())
	assert.NoError(t, err)
	assert.True(t, result.miss)
}

func TestLookupModrinthCandidateReturnsMatchOnSuccess(t *testing.T) {
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	result, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}, newModrinthTitleCache())
	assert.NoError(t, err)
	if assert.NotNil(t, result.match) {
		assert.Equal(t, "proj-1", result.match.ProjectID)
	}
}

func TestLookupModrinthCandidateReturnsErrorOnTitleLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: "https://example.invalid/a.jar", Primary: true}},
	}

	_, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
	}, newModrinthTitleCache())
	assert.ErrorIs(t, err, writeErr)
}

func TestLookupModrinthCandidateReturnsUnsureOnDownloadDetailsError(t *testing.T) {
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: ""}},
	}

	result, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}, newModrinthTitleCache())
	assert.NoError(t, err)
	assert.Error(t, result.err)
}

func TestLookupModrinthCandidateReturnsErrorOnDownloadLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	version := &modrinth.Version{
		ProjectID:     "proj-1",
		DatePublished: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		Files:         []modrinth.VersionFile{{URL: ""}},
	}

	_, err := lookupModrinthCandidate(context.Background(), scanCandidate{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}, scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return version, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}, newModrinthTitleCache())
	assert.ErrorIs(t, err, writeErr)
}

func TestPersistScanMatchesIfRequestedReturnsOutputErrorWhenPersistedLogFails(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{}
	setupCoordinator := modsetup.NewSetupCoordinator(fs, nil, nil)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := persistScanMatchesIfRequested(persistScanRequest{
		Context:          context.Background(),
		Command:          cmd,
		Options:          scanOptions{Add: true},
		Dependencies:     scanDeps{fs: fs, output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)},
		Metadata:         meta,
		SetupCoordinator: setupCoordinator,
		Matches: []scanMatch{{
			Path:        "/mods/a.jar",
			Platform:    models.MODRINTH,
			ProjectID:   "abc",
			Name:        "Example",
			FileName:    "example.jar",
			Hash:        "hash",
			ReleaseDate: "1970-01-01T00:00:00Z",
			DownloadURL: "https://example.invalid/example.jar",
		}},
		Config:         cfg,
		Lock:           lock,
		PreferPlatform: models.MODRINTH,
		ColorMode:      tui.ColorDisabled,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestRunScanReturnsSuccessWhenAllManaged(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	telemetryPayload, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Prefer:     string(models.MODRINTH),
	}, scanDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
	})
	assert.NoError(t, err)
	assert.True(t, telemetryPayload.Success)
}

func TestRunScanReturnsErrorWhenModsFolderMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "missing"}
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Prefer:     string(models.MODRINTH),
	}, scanDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
	})
	assert.Error(t, err)
}

func TestRunScanReturnsErrorWhenSha1Fails(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))

	path := filepath.Join(meta.ModsFolderPath(cfg), "bad.jar")
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	fs := readErrorFs{Fs: baseFs, failPath: path, err: errors.New("read failed")}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Prefer:     string(models.MODRINTH),
	}, scanDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
	})
	assert.ErrorContains(t, err, "read failed")
}

func TestRunScanReturnsErrorWhenPrintResultsFails(t *testing.T) {
	writeErr := errors.New("write failed")
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, baseFs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), baseFs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), baseFs, meta, []models.ModInstall{}))

	path := filepath.Join(meta.ModsFolderPath(cfg), "unknown.jar")
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Prefer:     string(models.MODRINTH),
	}, scanDeps{
		fs:     baseFs,
		output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestRunScanReturnsErrorOnInvalidConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{invalid"), 0644))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Prefer:     string(models.MODRINTH),
	}, scanDeps{
		fs:     fs,
		output: output.New(io.Discard, io.Discard, false),
	})
	assert.Error(t, err)
}
func TestRunScanReturnsOutputErrorOnInvalidPrefer(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runScan(context.Background(), cmd, scanOptions{
		ConfigPath: meta.ConfigPath,
		Quiet:      true,
		Prefer:     "unknown",
	}, scanDeps{
		fs:     fs,
		output: output.New(io.Discard, errorWriter{err: writeErr}, false),
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestIdentifyAndPrintCandidatesReturnsErrorOnLookupFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}

	_, err := identifyAndPrintCandidates(context.Background(), output.New(io.Discard, io.Discard, false), io.Discard, models.CURSEFORGE, candidates, scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		clients: platform.Clients{
			Curseforge: noopDoer{},
		},
		curseforgeFingerprint: func(string) uint32 { return 101 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestIdentifyCandidatesReturnsErrorOnLookupFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	candidates := []scanCandidate{{Path: "/mods/a.jar", FileName: "a.jar", Sha1: "a"}}
	deps := scanDeps{
		logger: logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true),
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestIdentifyCandidatesReturnsErrorOnFallbackFailure(t *testing.T) {
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

	_, err := identifyCandidates(context.Background(), candidates, models.MODRINTH, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestPersistScanMatchesReturnsOutputErrorOnPersistFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{}
	setupCoordinator := modsetup.NewSetupCoordinator(fs, nil, nil)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := persistScanMatches(context.Background(), cmd, meta, setupCoordinator, scanDeps{
		fs:     fs,
		output: output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false),
	}, []scanMatch{{
		Platform:  models.MODRINTH,
		ProjectID: "abc",
		FileName:  "mods/bad.jar",
		Name:      "Example",
	}}, cfg, lock)
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintResultsReturnsOutputErrorOnNoResults(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := printResults(outWriter, &bytes.Buffer{}, models.MODRINTH, nil, nil, nil)
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintResultsReturnsErrorOnUnknownOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := printResults(outWriter, &bytes.Buffer{}, models.MODRINTH, nil, []string{"/mods/a.jar"}, nil)
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintResultsReturnsErrorOnMatchOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := printResults(outWriter, &bytes.Buffer{}, models.MODRINTH, []scanMatch{
		{Path: "/mods/a.jar", Platform: models.MODRINTH, ProjectID: "a", Name: "Alpha", FileName: "a.jar"},
	}, nil, nil)
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintResultsReturnsErrorOnUnsureOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false)

	err := printResults(outWriter, &bytes.Buffer{}, models.MODRINTH, nil, nil, []scanUnsure{{Path: "/mods/a.jar", Error: errors.New("nope")}})
	assert.ErrorIs(t, err, writeErr)
}
func TestPrintMatchResultsReturnsErrorOnEntryFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &errAfterWriter{remaining: 1, err: writeErr}
	outWriter := output.New(writer, writer, false)

	err := printMatchResults(outWriter, tui.ColorDisabled, []scanMatch{
		{Path: "/mods/a.jar", Platform: models.MODRINTH, ProjectID: "a", Name: "Alpha", FileName: "a.jar"},
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintUnknownResultsReturnsErrorOnEntryFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &errAfterWriter{remaining: 1, err: writeErr}
	outWriter := output.New(writer, writer, false)

	err := printUnknownResults(outWriter, tui.ColorDisabled, []string{"/mods/unknown.jar"})
	assert.ErrorIs(t, err, writeErr)
}

func TestPrintUnsureResultsReturnsErrorOnEntryFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &errAfterWriter{remaining: 1, err: writeErr}
	outWriter := output.New(writer, writer, false)

	err := printUnsureResults(outWriter, tui.ColorDisabled, []scanUnsure{{Path: "/mods/unsure.jar", Error: errors.New("nope")}})
	assert.ErrorIs(t, err, writeErr)
}

type errAfterWriter struct {
	remaining int
	err       error
}

func (writer *errAfterWriter) Write(value []byte) (int, error) {
	if writer.remaining == 0 {
		return 0, writer.err
	}
	writer.remaining--
	return len(value), nil
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

func TestPickPrompterReturnsNoopWhenPromptDisabled(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restore)

	tty := fakeTTY{Buffer: &bytes.Buffer{}}
	prompter := pickPrompter(scanOptions{Unattended: true}, tty, tty)
	_, ok := prompter.(noopPrompter)
	assert.True(t, ok)
}

func TestPickPrompterReturnsTerminalWhenPromptEnabled(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restore)

	tty := fakeTTY{Buffer: &bytes.Buffer{}}
	prompter := pickPrompter(scanOptions{}, tty, tty)
	_, ok := prompter.(terminalPrompter)
	assert.True(t, ok)
}

func TestDefaultScanDepsUnattendedUsesNoopPrompter(t *testing.T) {
	restore := tui.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restore)

	cmd := &cobra.Command{}
	cmd.SetIn(fakeTTY{Buffer: &bytes.Buffer{}})
	cmd.SetOut(fakeTTY{Buffer: &bytes.Buffer{}})

	deps := defaultScanDeps(cmd, scanOptions{Unattended: true})
	_, ok := deps.prompter.(noopPrompter)
	assert.True(t, ok)
}

func TestConfirmPersistSkipsWhenNoPrompter(t *testing.T) {
	confirmed, err := confirmPersist(scanOptions{}, scanDeps{})
	assert.NoError(t, err)
	assert.False(t, confirmed)
}

type fakeTTY struct {
	*bytes.Buffer
}

func (tty fakeTTY) Fd() uintptr { return 0 }

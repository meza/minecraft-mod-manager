package install

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
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestOptionalStringValue(t *testing.T) {
	assert.Equal(t, "", optionalStringValue(nil))
	value := "  value "
	assert.Equal(t, "value", optionalStringValue(&value))
}

func TestEffectiveAllowedReleaseTypes(t *testing.T) {
	cfg := models.ModsJSON{DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release}}
	mod := models.Mod{AllowedReleaseTypes: []models.ReleaseType{models.Beta}}
	assert.Equal(t, []models.ReleaseType{models.Beta}, effectiveAllowedReleaseTypes(mod, cfg))

	mod.AllowedReleaseTypes = nil
	assert.Equal(t, []models.ReleaseType{models.Release}, effectiveAllowedReleaseTypes(mod, cfg))
}

func TestDownloadClientPrefersCurseforge(t *testing.T) {
	clients := platform.DefaultClients(rate.NewLimiter(rate.Inf, 0))
	assert.Equal(t, clients.Curseforge, downloadClient(clients))

	clients.Curseforge = nil
	assert.Equal(t, clients.Modrinth, downloadClient(clients))
}

func TestNoopSenderSend(t *testing.T) {
	var sender noopSender
	sender.Send(nil)
}

func TestUniqueInts(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3}, uniqueInts([]int{1, 2, 2, 3, 1}))
}

func TestSortHitsPreferModrinth(t *testing.T) {
	hits := []scanHit{
		{Platform: models.CURSEFORGE, Project: "2"},
		{Platform: models.MODRINTH, Project: "1"},
	}
	sorted := sortHitsPreferModrinth(hits)
	assert.Equal(t, models.MODRINTH, sorted[0].Platform)
}

func TestSortHitsPreferModrinthAlreadyFirst(t *testing.T) {
	hits := []scanHit{
		{Platform: models.MODRINTH, Project: "1"},
		{Platform: models.CURSEFORGE, Project: "2"},
	}
	sorted := sortHitsPreferModrinth(hits)
	assert.Equal(t, models.MODRINTH, sorted[0].Platform)
}

func TestSortHitsPreferModrinthSingle(t *testing.T) {
	hits := []scanHit{{Platform: models.CURSEFORGE, Project: "1"}}
	assert.Equal(t, hits, sortHitsPreferModrinth(hits))
}

func TestSortHitsPreferModrinthSamePlatform(t *testing.T) {
	hits := []scanHit{
		{Platform: models.CURSEFORGE, Project: "b"},
		{Platform: models.CURSEFORGE, Project: "a"},
	}
	sorted := sortHitsPreferModrinth(hits)
	assert.Len(t, sorted, 2)
}

func TestCurseforgeMatchesByFingerprintEmpty(t *testing.T) {
	deps := installDeps{}
	matches, err := curseforgeMatchesByFingerprint(context.Background(), nil, deps)
	assert.NoError(t, err)
	assert.Empty(t, matches)
}

func TestCurseforgeMatchesByFingerprintMapsHits(t *testing.T) {
	deps := installDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectID: 123, Fingerprint: 42}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}

	matches, err := curseforgeMatchesByFingerprint(context.Background(), []int{42, 42}, deps)
	assert.NoError(t, err)
	assert.Equal(t, "Example", matches[42].Name)
	assert.Equal(t, models.CURSEFORGE, matches[42].Platform)
}

func TestCurseforgeMatchesByFingerprintProjectNameError(t *testing.T) {
	deps := installDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectID: 123, Fingerprint: 42}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
	}

	_, err := curseforgeMatchesByFingerprint(context.Background(), []int{42}, deps)
	assert.ErrorContains(t, err, "boom")
}

func TestSha1ForFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/example.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	sum, err := sha1ForFile(fs, path)
	assert.NoError(t, err)

	expected := sha1.Sum([]byte("data"))
	assert.Equal(t, hex.EncodeToString(expected[:]), sum)

	_, err = sha1ForFile(fs, filepath.FromSlash("/mods/missing.jar"))
	assert.Error(t, err)
}

func TestScanFilesSkipsUnknownFingerprintAndModrinthNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := []string{
		filepath.FromSlash("/mods/a.jar"),
		filepath.FromSlash("/mods/b.jar"),
	}
	for _, file := range files {
		assert.NoError(t, fs.MkdirAll(filepath.Dir(file), 0755))
		assert.NoError(t, afero.WriteFile(fs, file, []byte("data"), 0644))
	}

	deps := installDeps{
		fs:      fs,
		clients: platform.Clients{},
		curseforgeFingerprint: func(path string) uint32 {
			if strings.Contains(path, "a.jar") {
				return 1
			}
			return 2
		},
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectID: 9001, Fingerprint: 999}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Extra", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{}
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			t.Fatal("modrinthProjectTitle should not be called")
			return "", nil
		},
	}

	results, err := scanFiles(context.Background(), files, deps)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	for _, result := range results {
		assert.Empty(t, result.Hits)
	}
}

func TestListModFilesFiltersNonJarAndIgnored(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	modsFolder := meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"})
	assert.NoError(t, fs.MkdirAll(modsFolder, 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "good.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "ignored.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "notes.txt"), []byte("x"), 0644))
	assert.NoError(t, fs.MkdirAll(filepath.Join(modsFolder, "dir"), 0755))

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("**/ignored.jar\n"), 0644))

	files, err := listModFiles(fs, meta, models.ModsJSON{ModsFolder: "mods"})
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.True(t, strings.HasSuffix(files[0], "good.jar"))
}

func TestListModFilesReturnsAllWhenNoIgnorePatterns(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	modsFolder := meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"})
	assert.NoError(t, fs.MkdirAll(modsFolder, 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "good.jar"), []byte("x"), 0644))

	files, err := listModFiles(fs, meta, models.ModsJSON{ModsFolder: "mods"})
	assert.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestListModFilesReturnsErrorOnReadDirFailure(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	fs := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: meta.ModsFolderPath(cfg),
		err:      errors.New("open failed"),
	}

	_, err := listModFiles(fs, meta, cfg)
	assert.Error(t, err)
}

func TestListModFilesReturnsErrorOnIgnoreStatFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("x"), 0644))

	fs := statErrorFs{Fs: baseFs, failPath: filepath.Join(meta.Dir(), ".mmmignore"), err: errors.New("stat failed")}

	_, err := listModFiles(fs, meta, cfg)
	assert.Error(t, err)
}

func TestReportScanResultsPaths(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
			{Type: models.CURSEFORGE, ID: "def", Name: "NoLock"},
		},
	}
	lock := []models.ModInstall{
		{Type: models.MODRINTH, ID: "abc", Name: "Example", Hash: "deadbeef"},
	}
	logBuffer := &strings.Builder{}
	deps := installDeps{
		logger: logger.New(logBuffer, io.Discard, false, false),
	}

	scanned := []scannedFile{
		{Sha1: "empty"},
		{Sha1: "deadbeef", Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}}},
		{Sha1: "badhash", Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}}},
		{Sha1: "other", Hits: []scanHit{{Platform: models.MODRINTH, Project: "missing", Name: "Unknown"}}},
		{Sha1: "lockmissing", Hits: []scanHit{{Platform: models.CURSEFORGE, Project: "def", Name: "NoLock"}}},
	}

	unresolved, unmanaged, err := reportScanResults(scanned, cfg, lock, deps, false)
	assert.NoError(t, err)
	assert.True(t, unresolved)
	assert.True(t, unmanaged)
	assert.NotEmpty(t, logBuffer.String())
}

func TestReportScanResultsColorizesUnmanaged(t *testing.T) {
	cfg := models.ModsJSON{}
	lock := []models.ModInstall{}
	logBuffer := &strings.Builder{}
	deps := installDeps{
		logger: logger.New(logBuffer, io.Discard, false, false),
	}

	scanned := []scannedFile{
		{Sha1: "hash", Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Unmanaged"}}},
	}

	unresolved, unmanaged, err := reportScanResults(scanned, cfg, lock, deps, true)
	assert.NoError(t, err)
	assert.False(t, unresolved)
	assert.True(t, unmanaged)
	assert.Contains(t, logBuffer.String(), "Unmanaged")
}

func TestHandleExpectedFetchError(t *testing.T) {
	logBuffer := &strings.Builder{}
	deps := installDeps{
		logger: logger.New(logBuffer, io.Discard, false, false),
	}

	mod := models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH}
	assert.True(t, handleExpectedFetchError(&platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}, mod, deps, false))
	assert.True(t, handleExpectedFetchError(&platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "abc"}, mod, deps, false))
	assert.False(t, handleExpectedFetchError(errors.New("boom"), mod, deps, false))
}

func TestEnsureLockInstallLogsReasons(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	logBuffer := &strings.Builder{}
	deps := installDeps{
		fs:      fs,
		logger:  logger.New(logBuffer, io.Discard, false, false),
		clients: platform.Clients{},
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	}

	missingInstall := models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.com/example.jar",
	}
	mod := models.Mod{Type: models.MODRINTH, ID: "abc", Name: "Example"}

	assert.NoError(t, ensureLockInstall(context.Background(), meta, cfg, mod, missingInstall, deps))

	mismatchPath := filepath.Join(meta.ModsFolderPath(cfg), "mismatch.jar")
	assert.NoError(t, afero.WriteFile(fs, mismatchPath, []byte("bad"), 0644))
	mismatchInstall := missingInstall
	mismatchInstall.FileName = "mismatch.jar"
	mismatchInstall.Hash = sha1Hex("data")

	assert.NoError(t, ensureLockInstall(context.Background(), meta, cfg, mod, mismatchInstall, deps))
}

func TestEnsureLockInstallAlreadyPresent(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	contents := []byte("data")
	filePath := filepath.Join(meta.ModsFolderPath(cfg), "present.jar")
	assert.NoError(t, afero.WriteFile(fs, filePath, contents, 0644))

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			t.Fatal("downloader should not be called when file is already present")
			return nil
		},
	}

	installEntry := models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Example",
		FileName:    "present.jar",
		Hash:        sha1Hex(string(contents)),
		DownloadURL: "https://example.com/present.jar",
	}
	mod := models.Mod{Type: models.MODRINTH, ID: "abc", Name: "Example"}

	assert.NoError(t, ensureLockInstall(context.Background(), meta, cfg, mod, installEntry, deps))
}

func TestEnsureLockInstallReturnsErrorOnMissingDownloadURL(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
	}

	installEntry := models.ModInstall{
		Type:     models.MODRINTH,
		ID:       "abc",
		Name:     "Example",
		FileName: "present.jar",
		Hash:     sha1Hex("data"),
	}
	mod := models.Mod{Type: models.MODRINTH, ID: "abc", Name: "Example"}

	assert.Error(t, ensureLockInstall(context.Background(), meta, cfg, mod, installEntry, deps))
}

func sha1Hex(data string) string {
	sum := sha1.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

type openErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (o openErrorFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(o.failPath) {
		return nil, o.err
	}
	return o.Fs.Open(name)
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

func TestPreflightUnknownFilesNoUnmanaged(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "managed.jar"), []byte("x"), 0644))

	lock := []models.ModInstall{{FileName: "managed.jar"}}
	deps := installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	unresolved, unmanaged, err := preflightUnknownFiles(context.Background(), meta, cfg, lock, deps, false)
	assert.NoError(t, err)
	assert.False(t, unresolved)
	assert.False(t, unmanaged)
}

func TestPreflightUnknownFilesScansUnknown(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unknown.jar"), []byte("x"), 0644))

	deps := installDeps{
		fs:                    fs,
		logger:                logger.New(io.Discard, io.Discard, false, false),
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	unresolved, unmanaged, err := preflightUnknownFiles(context.Background(), meta, cfg, nil, deps, false)
	assert.NoError(t, err)
	assert.False(t, unresolved)
	assert.False(t, unmanaged)
}

func TestPreflightUnknownFilesReturnsErrorOnListFailure(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	deps := installDeps{
		fs:     openErrorFs{Fs: afero.NewMemMapFs(), failPath: meta.ModsFolderPath(cfg), err: errors.New("open failed")},
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	_, _, err := preflightUnknownFiles(context.Background(), meta, cfg, nil, deps, false)
	assert.Error(t, err)
}

func TestPreflightUnknownFilesReturnsErrorOnScanFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	path := filepath.Join(meta.ModsFolderPath(cfg), "unknown.jar")
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    readErrorFs{Fs: baseFs, failPath: path, err: errors.New("read failed")},
		logger:                logger.New(io.Discard, io.Discard, false, false),
		curseforgeFingerprint: func(string) uint32 { return 1 },
	}

	_, _, err := preflightUnknownFiles(context.Background(), meta, cfg, nil, deps, false)
	assert.Error(t, err)
}

func TestScanFilesHandlesModrinthErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/error.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := scanFiles(context.Background(), []string{path}, deps)
	assert.ErrorContains(t, err, "boom")
}

func TestScanFilesReturnsErrorOnCurseforgeFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/error.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := scanFiles(context.Background(), []string{path}, deps)
	assert.ErrorContains(t, err, "boom")
}

func TestScanFilesSkipsModrinthNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/missing.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	results, err := scanFiles(context.Background(), []string{path}, deps)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestScanFilesSuccessAddsHits(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/ok.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectID: 123, Fingerprint: 1}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Curse Project", nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{ProjectID: "proj"}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Modrinth Project", nil
		},
	}

	results, err := scanFiles(context.Background(), []string{path}, deps)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Len(t, results[0].Hits, 2)
	assert.Equal(t, models.MODRINTH, results[0].Hits[0].Platform)
}

func TestScanFilesReturnsErrorOnFingerprintMatchFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/fingerprint.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("fingerprint failed")
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	_, err := scanFiles(context.Background(), []string{path}, deps)
	assert.ErrorContains(t, err, "fingerprint failed")
}

func TestScanFilesReturnsErrorOnModrinthProjectTitleFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/modrinth.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []int, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return &modrinth.Version{ProjectID: "proj"}, nil
		},
		modrinthProjectTitle: func(context.Context, string, httpclient.Doer) (string, error) {
			return "", errors.New("project failed")
		},
	}

	_, err := scanFiles(context.Background(), []string{path}, deps)
	assert.ErrorContains(t, err, "project failed")
}

func TestSha1ForFileReturnsReadError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/bad.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	fs := readErrorFs{Fs: baseFs, failPath: path, err: errors.New("read failed")}
	_, err := sha1ForFile(fs, path)
	assert.Error(t, err)
}

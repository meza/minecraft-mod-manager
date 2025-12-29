package install

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

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

func TestOptionalStringValue(t *testing.T) {
	assert.Equal(t, "", optionalStringValue(nil))
	value := "  value "
	assert.Equal(t, "value", optionalStringValue(&value))
}

func TestEffectiveAllowedReleaseTypes(t *testing.T) {
	cfg := models.ModsJSON{DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release}}
	mod := models.Mod{AllowedReleaseTypes: []models.ReleaseType{models.Beta}}
	assert.Equal(t, []models.ReleaseType{models.Beta}, models.EffectiveAllowedReleaseTypes(mod, cfg))

	mod.AllowedReleaseTypes = nil
	assert.Equal(t, []models.ReleaseType{models.Release}, models.EffectiveAllowedReleaseTypes(mod, cfg))
}

func TestDownloadClientPrefersCurseforge(t *testing.T) {
	clients := platform.DefaultClients(rate.NewLimiter(rate.Inf, 0))
	assert.Equal(t, clients.Curseforge, platform.PreferredDownloadClient(clients))

	clients.Curseforge = nil
	assert.Equal(t, clients.Modrinth, platform.PreferredDownloadClient(clients))
}

func TestLockIndexForModReturnsMinusOne(t *testing.T) {
	mod := models.Mod{Type: models.CURSEFORGE, ID: "missing"}
	assert.Equal(t, -1, models.LockIndexForMod(mod, nil))
}

func TestUniqueUint32s(t *testing.T) {
	assert.Equal(t, []uint32{1, 2, 3}, uniqueUint32s([]uint32{1, 2, 2, 3, 1}))
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectID: 123, Fingerprint: 42}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "Example", nil
		},
	}

	matches, err := curseforgeMatchesByFingerprint(context.Background(), []uint32{42, 42}, deps)
	assert.NoError(t, err)
	assert.Equal(t, "Example", matches[42].Name)
	assert.Equal(t, models.CURSEFORGE, matches[42].Platform)
}

func TestCurseforgeMatchesByFingerprintProjectNameError(t *testing.T) {
	deps := installDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{
				Matches: []curseforge.File{{ProjectID: 123, Fingerprint: 42}},
			}, nil
		},
		curseforgeProjectName: func(context.Context, string, httpclient.Doer) (string, error) {
			return "", errors.New("boom")
		},
	}

	_, err := curseforgeMatchesByFingerprint(context.Background(), []uint32{42}, deps)
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
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

	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("ignored.jar\n"), 0644))

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

func TestListModFilesIgnoresPatternsWhenModsFolderOutsideConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: filepath.FromSlash("/data/mods")}
	modsFolder := meta.ModsFolderPath(cfg)

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, fs.MkdirAll(modsFolder, 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "ignored.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(modsFolder, "keep.jar"), []byte("x"), 0644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("ignored.jar\n"), 0644))

	files, err := listModFiles(fs, meta, cfg)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.True(t, strings.HasSuffix(files[0], "keep.jar"))
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
		output: output.New(logBuffer, io.Discard, false),
	}

	scanned := []scannedFile{
		{Sha1: "empty"},
		{Sha1: "deadbeef", Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}}},
		{Sha1: "badhash", Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}}},
		{Sha1: "other", Hits: []scanHit{{Platform: models.MODRINTH, Project: "missing", Name: "Unknown"}}},
		{Sha1: "lockmissing", Hits: []scanHit{{Platform: models.CURSEFORGE, Project: "def", Name: "NoLock"}}},
	}

	outcome, err := reportScanResults(scanReportInputs{
		scanned:  scanned,
		cfg:      cfg,
		lock:     lock,
		deps:     deps,
		colorize: false,
	})
	assert.NoError(t, err)
	assert.True(t, outcome.unresolved)
	assert.True(t, outcome.unmanagedFound)
	assert.NotEmpty(t, logBuffer.String())
}

func TestReportScanResultsColorizesUnmanaged(t *testing.T) {
	cfg := models.ModsJSON{}
	lock := []models.ModInstall{}
	logBuffer := &strings.Builder{}
	deps := installDeps{
		logger: logger.New(logBuffer, io.Discard, false, false),
		output: output.New(logBuffer, io.Discard, false),
	}

	scanned := []scannedFile{
		{Sha1: "hash", Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Unmanaged"}}},
	}

	outcome, err := reportScanResults(scanReportInputs{
		scanned:  scanned,
		cfg:      cfg,
		lock:     lock,
		deps:     deps,
		colorize: true,
	})
	assert.NoError(t, err)
	assert.False(t, outcome.unresolved)
	assert.True(t, outcome.unmanagedFound)
	assert.Contains(t, logBuffer.String(), "Unmanaged")
}

func TestHandleExpectedFetchError(t *testing.T) {
	logBuffer := &strings.Builder{}
	deps := installDeps{
		logger: logger.New(logBuffer, io.Discard, false, false),
		output: output.New(logBuffer, io.Discard, false),
	}

	mod := models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH}
	handled, err := handleExpectedFetchError(&platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}, installModInputs{mod: mod, deps: deps, colorize: false})
	assert.NoError(t, err)
	assert.True(t, handled)

	handled, err = handleExpectedFetchError(&platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "abc"}, installModInputs{mod: mod, deps: deps, colorize: false})
	assert.NoError(t, err)
	assert.True(t, handled)

	handled, err = handleExpectedFetchError(errors.New("boom"), installModInputs{mod: mod, deps: deps, colorize: false})
	assert.NoError(t, err)
	assert.False(t, handled)
}

func TestHandleExpectedFetchErrorColorizedLogs(t *testing.T) {
	logBuffer := &strings.Builder{}
	deps := installDeps{
		logger: logger.New(logBuffer, io.Discard, false, false),
		output: output.New(logBuffer, io.Discard, false),
	}

	mod := models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH}
	handled, err := handleExpectedFetchError(&platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}, installModInputs{mod: mod, deps: deps, colorize: true})
	assert.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, logBuffer.String(), "modrinth")
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
		output:  output.New(logBuffer, io.Discard, false),
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
		output:  output.New(io.Discard, io.Discard, false),
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
		output:  output.New(io.Discard, io.Discard, false),
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

func (filesystem openErrorFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Open(name)
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

type closeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem closeErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return closeErrorFile{File: file, err: filesystem.err}, nil
	}
	return file, nil
}

type closeErrorFile struct {
	afero.File
	err error
}

func (file closeErrorFile) Close() error {
	return file.err
}

type readCloseErrorFs struct {
	afero.Fs
	failPath string
	readErr  error
	closeErr error
}

func (filesystem readCloseErrorFs) Open(name string) (afero.File, error) {
	file, err := filesystem.Fs.Open(name)
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return readCloseErrorFile{File: file, readErr: filesystem.readErr, closeErr: filesystem.closeErr}, nil
	}
	return file, nil
}

type readCloseErrorFile struct {
	afero.File
	readErr  error
	closeErr error
}

func (file readCloseErrorFile) Read([]byte) (int, error) {
	return 0, file.readErr
}

func (file readCloseErrorFile) Close() error {
	return file.closeErr
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
		output: output.New(io.Discard, io.Discard, false),
	}

	outcome, err := preflightUnknownFiles(preflightInputs{
		ctx:      context.Background(),
		meta:     meta,
		cfg:      cfg,
		lock:     lock,
		deps:     deps,
		colorize: false,
	})
	assert.NoError(t, err)
	assert.False(t, outcome.unresolved)
	assert.False(t, outcome.unmanagedFound)
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
		output:                output.New(io.Discard, io.Discard, false),
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	outcome, err := preflightUnknownFiles(preflightInputs{
		ctx:      context.Background(),
		meta:     meta,
		cfg:      cfg,
		lock:     nil,
		deps:     deps,
		colorize: true,
	})
	assert.NoError(t, err)
	assert.False(t, outcome.unresolved)
	assert.False(t, outcome.unmanagedFound)
}

func TestPreflightUnknownFilesReturnsErrorOnListFailure(t *testing.T) {
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	deps := installDeps{
		fs:     openErrorFs{Fs: afero.NewMemMapFs(), failPath: meta.ModsFolderPath(cfg), err: errors.New("open failed")},
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(io.Discard, io.Discard, false),
	}

	_, err := preflightUnknownFiles(preflightInputs{
		ctx:      context.Background(),
		meta:     meta,
		cfg:      cfg,
		lock:     nil,
		deps:     deps,
		colorize: false,
	})
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
		output:                output.New(io.Discard, io.Discard, false),
		curseforgeFingerprint: func(string) uint32 { return 1 },
	}

	_, err := preflightUnknownFiles(preflightInputs{
		ctx:      context.Background(),
		meta:     meta,
		cfg:      cfg,
		lock:     nil,
		deps:     deps,
		colorize: false,
	})
	assert.Error(t, err)
}

func TestScanFilesHandlesModrinthErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/error.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return &curseforge.FingerprintResult{}, nil
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := scanFiles(context.Background(), []string{path}, deps)
	var lookupFailure *platformLookupFailure
	assert.ErrorAs(t, err, &lookupFailure)
	assert.Equal(t, models.MODRINTH, lookupFailure.Platform)
	assert.Equal(t, []string{path}, lookupFailure.Files)
	assert.Contains(t, lookupFailure.Reason, "cmd.platform.error.reason.unknown")
}

func TestScanFilesReturnsErrorOnCurseforgeFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/error.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	}

	_, err := scanFiles(context.Background(), []string{path}, deps)
	var lookupFailure *platformLookupFailure
	assert.ErrorAs(t, err, &lookupFailure)
	assert.Equal(t, models.CURSEFORGE, lookupFailure.Platform)
	assert.Equal(t, []string{path}, lookupFailure.Files)
	assert.Contains(t, lookupFailure.Reason, "cmd.platform.error.reason.unknown")
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
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
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
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
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/fingerprint.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("fingerprint failed")
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	_, err := scanFiles(context.Background(), []string{path}, deps)
	var lookupFailure *platformLookupFailure
	assert.ErrorAs(t, err, &lookupFailure)
	assert.Equal(t, models.CURSEFORGE, lookupFailure.Platform)
	assert.Equal(t, []string{path}, lookupFailure.Files)
	assert.Contains(t, lookupFailure.Reason, "cmd.platform.error.reason.unknown")
}

func TestScanFilesReturnsErrorOnModrinthProjectTitleFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/modrinth.jar")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("data"), 0644))

	deps := installDeps{
		fs:                    fs,
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 1 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
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
	var lookupFailure *platformLookupFailure
	assert.ErrorAs(t, err, &lookupFailure)
	assert.Equal(t, models.MODRINTH, lookupFailure.Platform)
	assert.Equal(t, []string{path}, lookupFailure.Files)
	assert.Contains(t, lookupFailure.Reason, "cmd.platform.error.reason.unknown")
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

func TestSha1ForFileReturnsCloseError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/close.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	closeErr := errors.New("close failed")
	fs := closeErrorFs{Fs: baseFs, failPath: path, err: closeErr}
	_, err := sha1ForFile(fs, path)
	assert.ErrorIs(t, err, closeErr)
}

func TestSha1ForFileReturnsReadAndCloseError(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/close-read.jar")
	assert.NoError(t, baseFs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("data"), 0644))

	readErr := errors.New("read failed")
	closeErr := errors.New("close failed")
	fs := readCloseErrorFs{
		Fs:       baseFs,
		failPath: path,
		readErr:  readErr,
		closeErr: closeErr,
	}
	_, err := sha1ForFile(fs, path)
	assert.ErrorIs(t, err, readErr)
	assert.ErrorIs(t, err, closeErr)
}

func TestNewPlatformLookupFailureWithNilError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	failure := newPlatformLookupFailure(models.CURSEFORGE, []string{"/mods/a.jar"}, nil)
	assert.Equal(t, models.CURSEFORGE, failure.Platform)
	assert.Equal(t, []string{"/mods/a.jar"}, failure.Files)
	assert.Contains(t, failure.Reason, "cmd.platform.error.reason.unknown")
	assert.Equal(t, "", failure.DebugDetails)
	assert.Equal(t, failure.Reason, failure.Error())
}

func TestPlatformErrorDetailsUsesResponseError(t *testing.T) {
	details := platformErrorDetails(&httpclient.ResponseError{
		Method:     http.MethodGet,
		URL:        "https://example.invalid",
		StatusCode: http.StatusForbidden,
	})

	assert.Contains(t, details, "status=403")
	assert.Contains(t, details, "https://example.invalid")
}

func TestPlatformErrorDetailsReturnsEmptyForNil(t *testing.T) {
	assert.Equal(t, "", platformErrorDetails(nil))
}

func TestPlatformErrorDetailsIncludesFingerprintLookup(t *testing.T) {
	details := platformErrorDetails(&curseforge.FingerprintAPIError{
		Lookup: []uint32{123},
		Err: &httpclient.ResponseError{
			Method:     http.MethodPost,
			URL:        "https://example.invalid",
			StatusCode: http.StatusForbidden,
		},
	})

	assert.Contains(t, details, "fingerprints=[123]")
	assert.Contains(t, details, "status=403")
}

func TestPlatformErrorDetailsHandlesFingerprintWithoutResponseError(t *testing.T) {
	details := platformErrorDetails(&curseforge.FingerprintAPIError{
		Lookup: []uint32{123, 456},
		Err:    errors.New("boom"),
	})

	assert.Contains(t, details, "fingerprints=[123 456]")
	assert.Contains(t, details, "boom")
}

func TestPlatformErrorDetailsFallsBackToErrorMessage(t *testing.T) {
	details := platformErrorDetails(errors.New("boom"))

	assert.Equal(t, "boom", details)
}

func TestLogPlatformLookupFailureOutputsMessages(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	log := logger.New(out, errOut, false, true)
	outWriter := output.New(out, errOut, false)

	failure := &platformLookupFailure{
		Platform:     models.CURSEFORGE,
		Files:        []string{"/mods/a.jar"},
		Reason:       i18n.T("cmd.platform.error.reason.unknown", nil),
		DebugDetails: "debug details",
	}

	err := logPlatformLookupFailure(outWriter, log, failure, tui.ColorDisabled)
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "cmd.install.debug.platform_error")
	assert.Contains(t, out.String(), "cmd.install.unsure.platform_error")
	assert.Contains(t, out.String(), "cmd.install.unsure.platform_error_details")
	assert.Contains(t, out.String(), "a.jar")
}

func TestLogPlatformLookupFailureHandlesNil(t *testing.T) {
	assert.NoError(t, logPlatformLookupFailure(nil, nil, nil, tui.ColorDisabled))
}

func TestLogPlatformLookupFailureErrorsWithoutLogger(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	outWriter := output.New(out, errOut, false)

	failure := &platformLookupFailure{
		Platform:     models.CURSEFORGE,
		Files:        []string{"/mods/a.jar"},
		Reason:       i18n.T("cmd.platform.error.reason.unknown", nil),
		DebugDetails: "debug details",
	}

	err := logPlatformLookupFailure(outWriter, nil, failure, tui.ColorDisabled)
	assert.ErrorContains(t, err, "missing logger for platform debug output")
}

func TestLogPlatformLookupFailureReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	outWriter := output.New(errorWriter{err: writeErr}, io.Discard, false)

	failure := &platformLookupFailure{
		Platform: models.CURSEFORGE,
		Files:    []string{"/mods/a.jar"},
		Reason:   "failure reason",
	}

	err := logPlatformLookupFailure(outWriter, logger.New(io.Discard, io.Discard, false, false), failure, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestLogPlatformLookupFailureReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	outWriter := output.New(out, errOut, false)
	log := logger.New(errorWriter{err: writeErr}, io.Discard, false, true)

	failure := &platformLookupFailure{
		Platform:     models.CURSEFORGE,
		Files:        []string{"/mods/a.jar"},
		Reason:       "failure reason",
		DebugDetails: "debug details",
	}

	err := logPlatformLookupFailure(outWriter, log, failure, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestLogPlatformLookupFailureReturnsDetailsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	limitedWriter := &limitedErrorWriter{remaining: 1, err: writeErr}
	outWriter := output.New(limitedWriter, io.Discard, false)

	failure := &platformLookupFailure{
		Platform:     models.CURSEFORGE,
		Files:        []string{"/mods/a.jar"},
		Reason:       "failure reason",
		DebugDetails: "debug details",
	}

	err := logPlatformLookupFailure(outWriter, logger.New(io.Discard, io.Discard, false, true), failure, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestPreflightUnknownFilesLogsPlatformErrors(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}

	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	badPath := filepath.Join(meta.ModsFolderPath(cfg), "bad.jar")
	assert.NoError(t, afero.WriteFile(fs, badPath, []byte("data"), 0644))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	deps := installDeps{
		fs:                    fs,
		logger:                logger.New(out, errOut, false, true),
		output:                output.New(out, errOut, false),
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 123 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, &httpclient.ResponseError{StatusCode: http.StatusForbidden}
		},
		modrinthVersionForSha: func(context.Context, string, httpclient.Doer) (*modrinth.Version, error) {
			return nil, &modrinth.VersionNotFoundError{Lookup: *modrinth.NewVersionHashLookup("missing", modrinth.SHA1)}
		},
	}

	outcome, err := preflightUnknownFiles(preflightInputs{
		ctx:      context.Background(),
		meta:     meta,
		cfg:      cfg,
		lock:     nil,
		deps:     deps,
		colorize: true,
	})

	assert.NoError(t, err)
	assert.True(t, outcome.unresolved)
	assert.Contains(t, out.String(), "cmd.install.unsure.platform_error")
	assert.Contains(t, out.String(), "cmd.platform.error.reason.auth")
	assert.Contains(t, out.String(), "cmd.install.debug.platform_error")
	assert.Contains(t, out.String(), "cmd.install.unsure.platform_error_details")
	assert.Contains(t, out.String(), "bad.jar")
}

func TestInstallFromLockReturnsOutputErrorOnInvalidFilename(t *testing.T) {
	writeErr := errors.New("write failed")
	mod := models.Mod{Name: "Example"}
	installEntry := models.ModInstall{FileName: "mods/invalid.jar"}

	_, err := installFromLock(context.Background(), config.NewMetadata("modlist.json"), models.ModsJSON{ModsFolder: "mods"}, mod, installEntry, installDeps{
		output: output.New(io.Discard, errorWriter{err: writeErr}, false),
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestInstallFromRemoteReturnsOutputErrorOnMissingLog(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{
		output:  output.New(errorWriter{err: writeErr}, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        sha1Hex("data"),
				DownloadURL: "https://example.invalid",
			}, nil
		},
	}

	_, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			ModsFolder: "mods",
		},
		mod:  models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH},
		deps: deps,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestInstallFromRemoteReturnsErrorOnNormalizeOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{
		output:  output.New(io.Discard, errorWriter{err: writeErr}, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "mods/invalid.jar",
				Hash:        sha1Hex("data"),
				DownloadURL: "https://example.invalid",
			}, nil
		},
	}

	_, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			ModsFolder: "mods",
		},
		mod:  models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH},
		deps: deps,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestInstallFromRemoteHandlesExpectedFetchError(t *testing.T) {
	deps := installDeps{
		output:  output.New(io.Discard, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
	}

	outcome, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		mod:  models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH},
		deps: deps,
	})
	assert.NoError(t, err)
	assert.False(t, outcome.failed)
}

func TestInstallFromRemoteReturnsFailedOutcomeOnMissingHash(t *testing.T) {
	deps := installDeps{
		output:  output.New(io.Discard, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{
				Name:        "Example",
				FileName:    "example.jar",
				Hash:        "",
				DownloadURL: "https://example.invalid",
			}, nil
		},
	}

	outcome, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		mod:  models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH},
		deps: deps,
	})
	assert.NoError(t, err)
	assert.True(t, outcome.failed)
}

func TestInstallFromRemoteReturnsErrorOnFetchOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{
		output:  output.New(errorWriter{err: writeErr}, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}
		},
	}

	_, err := installFromRemote(installModInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{ModsFolder: "mods"},
		mod:  models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH},
		deps: deps,
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleInstallIntegrityErrorReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	_, err := handleInstallIntegrityError(output.New(io.Discard, errorWriter{err: writeErr}, false), modinstall.MissingHashError{FileName: "mod.jar"}, "Example")
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleDownloadIntegrityErrorReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	_, err := handleDownloadIntegrityError(output.New(io.Discard, errorWriter{err: writeErr}, false), modinstall.MissingHashError{FileName: "mod.jar"}, "Example")
	assert.ErrorIs(t, err, writeErr)
}

func TestNormalizeRemoteForInstallReturnsOutputErrorOnInvalidFileName(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{output: output.New(io.Discard, errorWriter{err: writeErr}, false)}

	_, _, err := normalizeRemoteForInstall(platform.RemoteMod{
		Name:     "Example",
		FileName: "mods/invalid.jar",
		Hash:     sha1Hex("data"),
	}, models.Mod{Name: "Example"}, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestNormalizeRemoteForInstallReturnsOutputErrorOnMissingHash(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{output: output.New(io.Discard, errorWriter{err: writeErr}, false)}

	_, _, err := normalizeRemoteForInstall(platform.RemoteMod{
		Name:     "Example",
		FileName: "example.jar",
		Hash:     "",
	}, models.Mod{Name: "Example"}, deps)
	assert.ErrorIs(t, err, writeErr)
}

func TestEnsureLockInstallReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		output:  output.New(errorWriter{err: writeErr}, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	}

	installEntry := models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Example",
		FileName:    "example.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.com/example.jar",
	}
	mod := models.Mod{Type: models.MODRINTH, ID: "abc", Name: "Example"}

	assert.ErrorIs(t, ensureLockInstall(context.Background(), meta, cfg, mod, installEntry, deps), writeErr)
}

func TestEnsureLockInstallReturnsHashMismatchOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mismatch.jar"), []byte("bad"), 0644))

	deps := installDeps{
		fs:      fs,
		logger:  logger.New(io.Discard, io.Discard, false, false),
		output:  output.New(errorWriter{err: writeErr}, io.Discard, false),
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		downloader: func(_ context.Context, _ string, path string, _ httpclient.Doer, _ httpclient.Sender, _ ...afero.Fs) error {
			return afero.WriteFile(fs, path, []byte("data"), 0644)
		},
	}

	installEntry := models.ModInstall{
		Type:        models.MODRINTH,
		ID:          "abc",
		Name:        "Example",
		FileName:    "mismatch.jar",
		Hash:        sha1Hex("data"),
		DownloadURL: "https://example.com/mismatch.jar",
	}
	mod := models.Mod{Type: models.MODRINTH, ID: "abc", Name: "Example"}

	assert.ErrorIs(t, ensureLockInstall(context.Background(), meta, cfg, mod, installEntry, deps), writeErr)
}

func TestHandleExpectedFetchErrorReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{
		output: output.New(errorWriter{err: writeErr}, io.Discard, false),
	}
	mod := models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH}

	handled, err := handleExpectedFetchError(&platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}, installModInputs{mod: mod, deps: deps, colorize: false})
	assert.False(t, handled)
	assert.ErrorIs(t, err, writeErr)
}

func TestHandleExpectedFetchErrorReturnsNoFileOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{
		output: output.New(errorWriter{err: writeErr}, io.Discard, false),
	}
	mod := models.Mod{Name: "Example", ID: "abc", Type: models.MODRINTH}

	handled, err := handleExpectedFetchError(&platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "abc"}, installModInputs{mod: mod, deps: deps, colorize: false})
	assert.False(t, handled)
	assert.ErrorIs(t, err, writeErr)
}
func TestRunInstallReturnsOutputErrorOnSuccessLog(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder: "mods",
		Mods:       []models.Mod{},
	}

	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	assert.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	_, err := runInstall(context.Background(), cmd, installOptions{ConfigPath: meta.ConfigPath}, installDeps{
		fs:     fs,
		logger: logger.New(io.Discard, io.Discard, false, false),
		output: output.New(errorWriter{err: writeErr}, io.Discard, false),
	})
	assert.ErrorIs(t, err, writeErr)
}

func TestInstallConfiguredModsReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	deps := installDeps{
		logger: logger.New(errorWriter{err: writeErr}, io.Discard, false, true),
		output: output.New(io.Discard, io.Discard, false),
	}
	inputs := installConfiguredInputs{
		ctx:  context.Background(),
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			Mods: []models.Mod{{Name: "Example", ID: "abc", Type: models.MODRINTH}},
		},
		lock: nil,
		deps: deps,
	}

	_, err := installConfiguredMods(inputs)
	assert.ErrorIs(t, err, writeErr)
}

func TestPreflightInstallReturnsOutputErrorWhenUnresolved(t *testing.T) {
	writeErr := errors.New("write failed")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		ModsFolder:                 "mods",
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}

	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	unknownPath := filepath.Join(meta.ModsFolderPath(cfg), "unknown.jar")
	assert.NoError(t, afero.WriteFile(fs, unknownPath, []byte("data"), 0644))

	_, err := preflightInstall(context.Background(), meta, cfg, nil, installDeps{
		fs:                    fs,
		logger:                logger.New(io.Discard, io.Discard, false, true),
		output:                output.New(io.Discard, errorWriter{err: writeErr}, false),
		clients:               platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		curseforgeFingerprint: func(string) uint32 { return 123 },
		curseforgeFingerprintMatch: func(context.Context, []uint32, httpclient.Doer) (*curseforge.FingerprintResult, error) {
			return nil, errors.New("boom")
		},
	}, false)
	assert.ErrorIs(t, err, writeErr)
}

func TestHandlePreflightScanFailureReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	input := preflightInputs{
		deps: installDeps{
			output: output.New(errorWriter{err: writeErr}, io.Discard, false),
		},
	}
	handled, err := handlePreflightScanFailure(input, &platformLookupFailure{
		Platform: models.MODRINTH,
		Files:    []string{"mod.jar"},
		Reason:   "boom",
	})
	assert.False(t, handled)
	assert.ErrorIs(t, err, writeErr)
}

func TestLogPlatformLookupFailureReturnsErrorWhenMissingLogger(t *testing.T) {
	out := output.New(io.Discard, io.Discard, false)
	err := logPlatformLookupFailure(out, nil, &platformLookupFailure{
		Platform:     models.MODRINTH,
		Files:        []string{"mod.jar"},
		Reason:       "boom",
		DebugDetails: "details",
	}, tui.ColorDisabled)
	assert.ErrorContains(t, err, "missing logger")
}

func TestLogPlatformLookupFailureReturnsErrorOnLogFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	out := output.New(io.Discard, io.Discard, false)
	log := logger.New(errorWriter{err: writeErr}, io.Discard, false, true)

	err := logPlatformLookupFailure(out, log, &platformLookupFailure{
		Platform:     models.MODRINTH,
		Files:        []string{"mod.jar"},
		Reason:       "boom",
		DebugDetails: "details",
	}, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestLogPlatformLookupFailureReturnsErrorOnOutputFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	out := output.New(errorWriter{err: writeErr}, io.Discard, false)

	err := logPlatformLookupFailure(out, logger.New(io.Discard, io.Discard, false, false), &platformLookupFailure{
		Platform: models.MODRINTH,
		Files:    []string{"mod.jar"},
		Reason:   "boom",
	}, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestReportScanResultsReturnsError(t *testing.T) {
	writeErr := errors.New("write failed")
	input := scanReportInputs{
		scanned: []scannedFile{
			{
				Path: "mod.jar",
				Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}},
			},
		},
		cfg: models.ModsJSON{},
		deps: installDeps{
			output: output.New(errorWriter{err: writeErr}, io.Discard, false),
		},
	}

	_, err := reportScanResults(input)
	assert.ErrorIs(t, err, writeErr)
}

func TestReportScanResultReturnsOutputErrorWhenUnmanaged(t *testing.T) {
	writeErr := errors.New("write failed")
	input := scanReportInputs{
		cfg: models.ModsJSON{},
		deps: installDeps{
			output: output.New(errorWriter{err: writeErr}, io.Discard, false),
		},
	}

	_, err := reportScanResult(input, scannedFile{
		Path: "mod.jar",
		Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}},
	}, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestReportScanResultReturnsOutputErrorWhenLockMissing(t *testing.T) {
	writeErr := errors.New("write failed")
	input := scanReportInputs{
		cfg: models.ModsJSON{
			Mods: []models.Mod{{ID: "abc", Name: "Example", Type: models.MODRINTH}},
		},
		lock: []models.ModInstall{},
		deps: installDeps{
			output: output.New(errorWriter{err: writeErr}, io.Discard, false),
		},
	}

	_, err := reportScanResult(input, scannedFile{
		Path: "mod.jar",
		Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}},
	}, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

func TestReportScanResultReturnsOutputErrorWhenHashMismatch(t *testing.T) {
	writeErr := errors.New("write failed")
	input := scanReportInputs{
		cfg: models.ModsJSON{
			Mods: []models.Mod{{ID: "abc", Name: "Example", Type: models.MODRINTH}},
		},
		lock: []models.ModInstall{{ID: "abc", Type: models.MODRINTH, Hash: "abc123"}},
		deps: installDeps{
			output: output.New(errorWriter{err: writeErr}, io.Discard, false),
		},
	}

	_, err := reportScanResult(input, scannedFile{
		Path: "mod.jar",
		Sha1: "different",
		Hits: []scanHit{{Platform: models.MODRINTH, Project: "abc", Name: "Example"}},
	}, tui.ColorDisabled)
	assert.ErrorIs(t, err, writeErr)
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type limitedErrorWriter struct {
	remaining int
	err       error
}

func (writer *limitedErrorWriter) Write(p []byte) (int, error) {
	if writer.remaining <= 0 {
		return 0, writer.err
	}
	writer.remaining--
	return len(p), nil
}

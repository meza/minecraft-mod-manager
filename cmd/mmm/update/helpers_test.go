package update

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestDownloadClientPrefersCurseforge(t *testing.T) {
	curseforgeClient := noopDoer{}
	modrinthClient := noopDoer{}

	chosen := downloadClient(platform.Clients{
		Curseforge: curseforgeClient,
		Modrinth:   modrinthClient,
	})
	assert.Equal(t, curseforgeClient, chosen)
}

func TestDownloadClientFallsBackToModrinth(t *testing.T) {
	modrinthClient := noopDoer{}

	chosen := downloadClient(platform.Clients{
		Modrinth: modrinthClient,
	})
	assert.Equal(t, modrinthClient, chosen)
}

func TestEffectiveAllowedReleaseTypesUsesOverrides(t *testing.T) {
	mod := models.Mod{AllowedReleaseTypes: []models.ReleaseType{models.Beta}}
	cfg := models.ModsJSON{DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release}}

	assert.Equal(t, mod.AllowedReleaseTypes, effectiveAllowedReleaseTypes(mod, cfg))
}

func TestNoopSenderSendDoesNotPanic(t *testing.T) {
	sender := &noopSender{}
	sender.Send(nil)
}

func TestNextBackupPathSkipsExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/mods/mod.jar")
	base := target + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(fs, base, []byte("backup"), 0644))
	next, err := nextBackupPath(fs, target)
	assert.NoError(t, err)
	assert.Equal(t, base+".1", next)
}

func TestNextBackupPathReturnsErrorOnStatFailure(t *testing.T) {
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/mod.jar.mmm.bak"),
		err:      errors.New("stat failed"),
	}

	_, err := nextBackupPath(fs, filepath.FromSlash("/mods/mod.jar"))
	assert.Error(t, err)
}

func TestNextBackupPathReturnsErrorAfterExhaustion(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/mods/mod.jar")
	base := target + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(fs, base, []byte("backup"), 0644))
	for i := 1; i < 100; i++ {
		assert.NoError(t, afero.WriteFile(fs, base+"."+strconv.Itoa(i), []byte("backup"), 0644))
	}

	_, err := nextBackupPath(fs, target)
	assert.Error(t, err)
}

func TestReplaceExistingFileRenamesWhenDestinationMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	source := filepath.FromSlash("/mods/source.jar")
	destination := filepath.FromSlash("/mods/dest.jar")

	assert.NoError(t, afero.WriteFile(fs, source, []byte("source"), 0644))
	assert.NoError(t, replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination))

	exists, err := afero.Exists(fs, destination)
	assert.NoError(t, err)
	assert.True(t, exists)
	exists, err = afero.Exists(fs, source)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestReplaceExistingFileReturnsErrorOnStatFailure(t *testing.T) {
	fs := statErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/dest.jar"),
		err:      errors.New("stat failed"),
	}

	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), filepath.FromSlash("/mods/source.jar"), filepath.FromSlash("/mods/dest.jar"))
	assert.Error(t, err)
}

func TestReplaceExistingFileReturnsErrorOnBackupRenameFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := renameErrorFs{Fs: baseFs, failOld: destination, failNew: backup, err: errors.New("rename failed")}
	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination)
	assert.Error(t, err)
}

func TestReplaceExistingFileReturnsErrorOnNextBackupPathFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := statErrorFs{Fs: baseFs, failPath: backup, err: errors.New("stat failed")}
	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination)
	assert.Error(t, err)
}

func TestReplaceExistingFileRestoresBackupOnRenameFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := renameErrorFs{Fs: baseFs, failOld: source, failNew: destination, err: errors.New("rename failed")}
	err := replaceExistingFile(fs, logger.New(io.Discard, io.Discard, false, false), source, destination)
	assert.Error(t, err)

	content, readErr := afero.ReadFile(fs, destination)
	assert.NoError(t, readErr)
	assert.Equal(t, []byte("old"), content)
}

func TestReplaceExistingFileLogsWhenBackupCleanupFails(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	destination := filepath.FromSlash("/mods/dest.jar")
	source := filepath.FromSlash("/mods/source.jar")
	backup := destination + ".mmm.bak"

	assert.NoError(t, afero.WriteFile(baseFs, destination, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(baseFs, source, []byte("new"), 0644))

	fs := removeErrorFs{Fs: baseFs, failPath: backup, err: errors.New("remove failed")}
	log := logger.New(io.Discard, io.Discard, false, true)

	assert.NoError(t, replaceExistingFile(fs, log, source, destination))
}

func TestDownloadAndSwapRemovesNewFileWhenOldRemovalFails(t *testing.T) {
	fs := removeErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/old.jar"),
		err:      errors.New("remove failed"),
	}

	oldPath := filepath.FromSlash("/mods/old.jar")
	newPath := filepath.FromSlash("/mods/new.jar")
	assert.NoError(t, afero.WriteFile(fs, oldPath, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, oldPath, newPath, filepath.FromSlash("/mods"), "https://example.invalid/new.jar", sha1Hex("new")))

	exists, err := afero.Exists(fs, newPath)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestDownloadAndSwapCleansTempOnDownloaderFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/same.jar")
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, path, path, filepath.FromSlash("/mods"), "https://example.invalid/same.jar", sha1Hex("new")))

	entries, entriesErr := afero.ReadDir(fs, filepath.FromSlash("/mods"))
	assert.NoError(t, entriesErr)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), ".tmp"))
	}
}

func TestDownloadAndSwapCleansTempOnReplaceFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/same.jar")
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("old"), 0644))

	fs := renameErrorFs{Fs: baseFs, failOldContains: ".mmm.", failNew: path, err: errors.New("rename failed")}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, path, path, filepath.FromSlash("/mods"), "https://example.invalid/same.jar", sha1Hex("new")))

	entries, entriesErr := afero.ReadDir(fs, filepath.FromSlash("/mods"))
	assert.NoError(t, entriesErr)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), ".tmp"))
	}
}

func TestDownloadAndSwapReturnsMissingHashError(t *testing.T) {
	fs := afero.NewMemMapFs()
	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.FromSlash("/mods/old.jar"), filepath.FromSlash("/mods/new.jar"), filepath.FromSlash("/mods"), "https://example.invalid/new.jar", "")
	var missingHash modinstall.MissingHashError
	assert.ErrorAs(t, err, &missingHash)
}

func TestDownloadAndSwapReturnsErrorOnHashReadFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	newPath := filepath.FromSlash("/mods/new.jar")
	fs := openErrorFs{Fs: base, failContains: ".mmm.", err: errors.New("open failed")}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpclient.Doer, _ httpclient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("data"), 0644)
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	err := downloadAndSwap(context.Background(), deps, filepath.FromSlash("/mods/old.jar"), newPath, filepath.FromSlash("/mods"), "https://example.invalid/new.jar", sha1Hex("data"))
	assert.Error(t, err)
}

func TestDownloadAndSwapReturnsErrorWhenResolveWritablePathFails(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()
	modsRoot := filepath.Join(root, "mods")
	assert.NoError(t, os.MkdirAll(modsRoot, 0755))

	outside := t.TempDir()
	target := filepath.Join(outside, "target.jar")
	assert.NoError(t, os.WriteFile(target, []byte("data"), 0644))

	newPath := filepath.Join(modsRoot, "link.jar")
	if err := os.Symlink(target, newPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.Join(modsRoot, "old.jar"), newPath, modsRoot, "https://example.invalid/new.jar", sha1Hex("data"))
	assert.Error(t, err)
}

func TestDownloadAndSwapReturnsErrorOnTempFileFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	assert.NoError(t, base.MkdirAll(filepath.FromSlash("/mods"), 0755))
	fs := openFileErrorFs{Fs: base, err: errors.New("open failed")}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
	}

	err := downloadAndSwap(context.Background(), deps, filepath.FromSlash("/mods/old.jar"), filepath.FromSlash("/mods/new.jar"), filepath.FromSlash("/mods"), "https://example.invalid/new.jar", sha1Hex("data"))
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnOpenFailure(t *testing.T) {
	fs := openErrorFs{
		Fs:       afero.NewMemMapFs(),
		failPath: filepath.FromSlash("/mods/missing.jar"),
		err:      errors.New("open failed"),
	}

	_, err := sha1ForFile(fs, filepath.FromSlash("/mods/missing.jar"))
	assert.Error(t, err)
}

func TestSha1ForFileReturnsErrorOnReadFailure(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/data.jar")
	assert.NoError(t, afero.WriteFile(base, path, []byte("data"), 0644))

	fs := readErrorFs{Fs: base, failPath: path, err: errors.New("read failed")}
	_, err := sha1ForFile(fs, path)
	assert.Error(t, err)
}

func TestIntegrityErrorMessageReturnsFalseForUnknownError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message, ok := integrityErrorMessage(errors.New("unknown"), "Mod")
	assert.False(t, ok)
	assert.Empty(t, message)
}

func TestIntegrityErrorMessageHandlesMissingHash(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message, ok := integrityErrorMessage(modinstall.MissingHashError{FileName: "x.jar"}, "Mod")
	assert.True(t, ok)
	assert.Contains(t, message, "cmd.update.error.missing_hash")
}

func TestIntegrityErrorMessageHandlesHashMismatch(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	message, ok := integrityErrorMessage(modinstall.HashMismatchError{FileName: "x.jar", Expected: "a", Actual: "b"}, "Mod")
	assert.True(t, ok)
	assert.Contains(t, message, "cmd.update.error.hash_mismatch")
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

type renameErrorFs struct {
	afero.Fs
	failOld         string
	failNew         string
	failOldContains string
	err             error
}

func (r renameErrorFs) Rename(oldname, newname string) error {
	if r.failOldContains != "" && strings.Contains(filepath.Clean(oldname), r.failOldContains) && filepath.Clean(newname) == filepath.Clean(r.failNew) {
		return r.err
	}
	if r.failOld != "" && filepath.Clean(oldname) == filepath.Clean(r.failOld) && filepath.Clean(newname) == filepath.Clean(r.failNew) {
		return r.err
	}
	return r.Fs.Rename(oldname, newname)
}

type removeErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (r removeErrorFs) Remove(name string) error {
	if filepath.Clean(name) == filepath.Clean(r.failPath) {
		return r.err
	}
	return r.Fs.Remove(name)
}

func sha1Hex(data string) string {
	sum := sha1.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
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

type openErrorFs struct {
	afero.Fs
	failPath     string
	failContains string
	err          error
}

func (o openErrorFs) Open(name string) (afero.File, error) {
	if o.failPath != "" && filepath.Clean(name) == filepath.Clean(o.failPath) {
		return nil, o.err
	}
	if o.failContains != "" && strings.Contains(filepath.Clean(name), o.failContains) {
		return nil, o.err
	}
	return o.Fs.Open(name)
}

type openFileErrorFs struct {
	afero.Fs
	err error
}

func (o openFileErrorFs) OpenFile(string, int, os.FileMode) (afero.File, error) {
	return nil, o.err
}

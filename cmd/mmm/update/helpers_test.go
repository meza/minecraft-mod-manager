package update

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
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
	cfg := models.ModsJson{DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release}}

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

	exists, _ := afero.Exists(fs, destination)
	assert.True(t, exists)
	exists, _ = afero.Exists(fs, source)
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
		downloader: func(_ context.Context, _ string, destination string, _ httpClient.Doer, _ httpClient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, oldPath, newPath, "https://example.invalid/new.jar"))

	exists, _ := afero.Exists(fs, newPath)
	assert.False(t, exists)
}

func TestDownloadAndSwapCleansTempOnDownloaderFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/same.jar")
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error {
			return errors.New("download failed")
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, path, path, "https://example.invalid/same.jar"))

	exists, _ := afero.Exists(fs, path+".mmm.tmp")
	assert.False(t, exists)
}

func TestDownloadAndSwapCleansTempOnReplaceFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	path := filepath.FromSlash("/mods/same.jar")
	assert.NoError(t, afero.WriteFile(baseFs, path, []byte("old"), 0644))

	fs := renameErrorFs{Fs: baseFs, failOld: path + ".mmm.tmp", failNew: path, err: errors.New("rename failed")}

	deps := updateDeps{
		fs:      fs,
		clients: platform.Clients{Modrinth: noopDoer{}},
		downloader: func(_ context.Context, _ string, destination string, _ httpClient.Doer, _ httpClient.Sender, filesystems ...afero.Fs) error {
			return afero.WriteFile(filesystems[0], destination, []byte("new"), 0644)
		},
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	assert.Error(t, downloadAndSwap(context.Background(), deps, path, path, "https://example.invalid/same.jar"))

	exists, _ := afero.Exists(fs, path+".mmm.tmp")
	assert.False(t, exists)
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
	failOld string
	failNew string
	err     error
}

func (r renameErrorFs) Rename(oldname, newname string) error {
	if filepath.Clean(oldname) == filepath.Clean(r.failOld) && filepath.Clean(newname) == filepath.Clean(r.failNew) {
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

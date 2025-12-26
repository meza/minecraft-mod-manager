package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type renameFailFs struct {
	afero.Fs
	failures []renameFailure
}

type renameFailure struct {
	old                string
	new                string
	oldContains        string
	newContains        string
	onlyWhenDestExists bool
	err                error
}

func (filesystem renameFailFs) Rename(oldname, newname string) error {
	for _, failure := range filesystem.failures {
		if failure.old != "" && oldname != failure.old {
			continue
		}
		if failure.new != "" && newname != failure.new {
			continue
		}
		if failure.oldContains != "" && !strings.Contains(oldname, failure.oldContains) {
			continue
		}
		if failure.newContains != "" && !strings.Contains(newname, failure.newContains) {
			continue
		}
		if failure.onlyWhenDestExists {
			exists, err := afero.Exists(filesystem.Fs, newname)
			if err != nil || !exists {
				continue
			}
		}
		if failure.err != nil {
			return failure.err
		}
		return errors.New("rename failed")
	}
	return filesystem.Fs.Rename(oldname, newname)
}

type removeErrorFs struct {
	afero.Fs
	failPaths    map[string]error
	failContains string
}

func (filesystem removeErrorFs) Remove(name string) error {
	if filesystem.failContains != "" && strings.Contains(filepath.Clean(name), filesystem.failContains) {
		return errors.New("remove failed")
	}
	if err, ok := filesystem.failPaths[filepath.Clean(name)]; ok {
		if err != nil {
			return err
		}
		return errors.New("remove failed")
	}
	return filesystem.Fs.Remove(name)
}

type removeAfterFirstFs struct {
	afero.Fs
	failPath     string
	failContains string
	failErr      error
	callCount    int
}

func (filesystem *removeAfterFirstFs) Remove(name string) error {
	cleaned := filepath.Clean(name)
	if (filesystem.failPath != "" && cleaned == filepath.Clean(filesystem.failPath)) || (filesystem.failContains != "" && strings.Contains(cleaned, filesystem.failContains)) {
		filesystem.callCount++
		if filesystem.callCount == 1 {
			return nil
		}
		if filesystem.failErr != nil {
			return filesystem.failErr
		}
		return errors.New("remove failed")
	}
	return filesystem.Fs.Remove(name)
}

type openFileErrorFs struct {
	afero.Fs
	failOn string
}

func (filesystem openFileErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if strings.Contains(filepath.Clean(name), filesystem.failOn) {
		return nil, errors.New("open failed")
	}
	return filesystem.Fs.OpenFile(name, flag, perm)
}

type writeErrorFile struct {
	afero.File
	err error
}

func (file writeErrorFile) Write(_ []byte) (int, error) {
	return 0, file.err
}

type writeErrorFs struct {
	afero.Fs
	err error
}

func (filesystem writeErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, err := filesystem.Fs.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return writeErrorFile{File: file, err: filesystem.err}, nil
}

type writeCloseErrorFile struct {
	afero.File
	writeErr error
	closeErr error
}

func (file writeCloseErrorFile) Write(_ []byte) (int, error) {
	return 0, file.writeErr
}

func (file writeCloseErrorFile) Close() error {
	return file.closeErr
}

type writeCloseErrorFs struct {
	afero.Fs
	writeErr error
	closeErr error
}

func (filesystem writeCloseErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, err := filesystem.Fs.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return writeCloseErrorFile{File: file, writeErr: filesystem.writeErr, closeErr: filesystem.closeErr}, nil
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

func (filesystem closeErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, err := filesystem.Fs.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return closeErrorFile{File: file, closeErr: filesystem.closeErr}, nil
}

type chmodErrorFs struct {
	afero.Fs
	failContains string
	err          error
}

func (filesystem chmodErrorFs) Chmod(name string, mode os.FileMode) error {
	if filesystem.failContains != "" && strings.Contains(filepath.Clean(name), filesystem.failContains) {
		return filesystem.err
	}
	return filesystem.Fs.Chmod(name, mode)
}

func TestWriteFileAtomicCreatesWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, writeFileAtomic(fs, path, []byte("ok")))

	data, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("ok"), data)
}

func TestWriteFileAtomicDoesNotCorruptWhenRenameIntoMissingTargetFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	fs := renameFailFs{
		Fs: base,
		failures: []renameFailure{{
			oldContains: ".mmm.tmp",
			new:         path,
		}},
	}

	assert.Error(t, writeFileAtomic(fs, path, []byte("new")))

	exists, err := afero.Exists(base, path)
	assert.NoError(t, err)
	assert.False(t, exists, "target should not be created on failure")

	assertNoTempFiles(t, base, filepath.Dir(path))
}

func TestWriteFileAtomicReturnsJoinedErrorWhenExistsCheckFailsAndTempCleanupFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	statErr := errors.New("stat failed")
	fs := removeErrorFs{
		Fs: statErrorFs{
			Fs:       base,
			failPath: path,
			err:      statErr,
		},
		failContains: ".mmm.tmp",
	}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
	assert.ErrorIs(t, err, statErr)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestWriteFileAtomicReturnsErrorWhenTempFileCreateFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))

	fs := openFileErrorFs{Fs: base, failOn: ".mmm.tmp"}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
}

func TestWriteFileAtomicReturnsJoinedErrorWhenCleanupFailsAfterRenameToMissingTarget(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))

	renameFs := renameFailFs{
		Fs: base,
		failures: []renameFailure{{
			oldContains: ".mmm.tmp",
			new:         path,
		}},
	}
	fs := removeErrorFs{
		Fs:           renameFs,
		failContains: ".mmm.tmp",
	}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestWriteFileAtomicReturnsJoinedErrorWhenTempCleanupFailsAfterRenameToMissingTarget(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	failRename := renameFailFs{
		Fs: base,
		failures: []renameFailure{{
			oldContains: ".mmm.tmp",
			new:         path,
		}},
	}
	fs := removeErrorFs{
		Fs:           failRename,
		failContains: ".mmm.tmp",
	}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestWriteFileAtomicDoesNotCorruptWhenBackupRenameFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	fs := renameFailFs{
		Fs: base,
		failures: []renameFailure{
			{
				oldContains: ".mmm.tmp",
				new:         path,
			},
			{
				old: path,
				new: path + ".mmm.bak",
			},
		},
	}

	assert.Error(t, writeFileAtomic(fs, path, []byte("new")))

	data, err := afero.ReadFile(base, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("old"), data)

	assertNoTempFiles(t, base, filepath.Dir(path))

	backupExists, err := afero.Exists(base, path+".mmm.bak")
	assert.NoError(t, err)
	assert.False(t, backupExists, "backup should not exist on backup-rename failure")
}

func TestWriteFileAtomicReturnsJoinedErrorWhenBackupRenameCleanupFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	renameFs := renameFailFs{
		Fs: base,
		failures: []renameFailure{
			{
				oldContains: ".mmm.tmp",
				new:         path,
			},
			{
				old: path,
				new: path + ".mmm.bak",
			},
		},
	}
	fs := removeErrorFs{
		Fs:           renameFs,
		failContains: ".mmm.tmp",
	}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestWriteFileAtomicReturnsJoinedErrorWhenRollbackFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	backup := path + ".mmm.bak"

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	fs := renameFailFs{
		Fs: base,
		failures: []renameFailure{
			{
				oldContains: ".mmm.tmp",
				new:         path,
			},
			{
				old: backup,
				new: path,
			},
		},
	}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore backup")
}

func TestWriteFileAtomicReturnsJoinedErrorWhenSwapCleanupFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	backup := path + ".mmm.bak"

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	renameFs := renameFailFs{
		Fs: base,
		failures: []renameFailure{
			{
				oldContains: ".mmm.tmp",
				new:         path,
			},
			{
				old: backup,
				new: path,
			},
		},
	}
	fs := removeErrorFs{
		Fs:           renameFs,
		failContains: ".mmm.tmp",
	}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestWriteFileAtomicFallsBackToBackupSwapWhenOverwriteRenameFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	fs := renameFailFs{
		Fs: base,
		failures: []renameFailure{{
			oldContains:        ".mmm.tmp",
			new:                path,
			onlyWhenDestExists: true,
		}},
	}

	assert.NoError(t, writeFileAtomic(fs, path, []byte("new")))

	data, err := afero.ReadFile(base, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), data)

	backupExists, err := afero.Exists(base, path+".mmm.bak")
	assert.NoError(t, err)
	assert.False(t, backupExists, "backup should be cleaned up on success")
}

func TestWriteFileAtomicReturnsErrorWhenBackupCleanupFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	backup := path + ".mmm.bak"

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	failRename := renameFailFs{
		Fs: base,
		failures: []renameFailure{{
			oldContains:        ".mmm.tmp",
			new:                path,
			onlyWhenDestExists: true,
		}},
	}
	fs := removeErrorFs{
		Fs: failRename,
		failPaths: map[string]error{
			filepath.Clean(backup): errors.New("remove failed"),
		},
	}

	err := writeFileAtomic(fs, path, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove backup file")
}

func TestWriteFileAtomicRollsBackWhenSwapRenameFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(base, path, []byte("old"), 0644))

	fs := renameFailFs{
		Fs: base,
		failures: []renameFailure{{
			oldContains: ".mmm.tmp",
			new:         path,
		}},
	}

	assert.Error(t, writeFileAtomic(fs, path, []byte("new")))

	data, err := afero.ReadFile(base, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("old"), data)

	backupExists, err := afero.Exists(base, path+".mmm.bak")
	assert.NoError(t, err)
	assert.False(t, backupExists, "backup should be rolled back or cleaned up")
}

func TestWriteFileAtomicUpdatesExistingFileAndCleansBackupBestEffort(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))
	assert.NoError(t, writeFileAtomic(fs, path, []byte("new")))

	data, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), data)
}

func TestWriteFileAtomicCleansTempFilesOnSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))

	assert.NoError(t, writeFileAtomic(fs, path, []byte("new")))

	data, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), data)
	assertNoTempFiles(t, fs, filepath.Dir(path))
}

func TestNextSiblingPathReturnsErrorWhenStatFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json.mmm.tmp")
	fs := statErrorFs{Fs: base, failPath: path}

	_, err := nextSiblingPath(fs, filepath.FromSlash("/cfg/modlist.json"), ".tmp")
	assert.Error(t, err)
}

func TestNextSiblingPathReturnsErrorWhenNoSlotAvailable(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")
	base := target + ".mmm.tmp"

	assert.NoError(t, fs.MkdirAll(filepath.Dir(target), 0755))
	for i := 0; i < 100; i++ {
		name := base
		if i > 0 {
			name = base + fmt.Sprintf(".%d", i)
		}
		assert.NoError(t, afero.WriteFile(fs, name, []byte("x"), 0644))
	}

	_, err := nextSiblingPath(fs, target, ".tmp")
	assert.Error(t, err)
}

func TestWriteFileAtomicReturnsErrorWhenTargetExistenceCheckFails(t *testing.T) {
	base := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")
	fs := statErrorFs{Fs: base, failPath: target}

	assert.NoError(t, base.MkdirAll(filepath.Dir(target), 0755))
	assert.Error(t, writeFileAtomic(fs, target, []byte("new")))
}

func TestWriteFileAtomicReturnsErrorWhenTempWriteFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(target), 0755))

	err := writeFileAtomic(writeErrorFs{Fs: fs, err: errors.New("write failed")}, target, []byte("new"))
	assert.Error(t, err)
	assertNoTempFiles(t, fs, filepath.Dir(target))
}

func TestWriteFileAtomicReturnsJoinedErrorWhenTempWriteAndCloseFail(t *testing.T) {
	base := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(target), 0755))

	fs := writeCloseErrorFs{
		Fs:       base,
		writeErr: errors.New("write failed"),
		closeErr: errors.New("close failed"),
	}

	err := writeFileAtomic(fs, target, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
	assert.Contains(t, err.Error(), "close failed")
	assertNoTempFiles(t, base, filepath.Dir(target))
}

func TestWriteFileAtomicReturnsErrorWhenTempCloseFails(t *testing.T) {
	base := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(target), 0755))

	err := writeFileAtomic(closeErrorFs{Fs: base, closeErr: errors.New("close failed")}, target, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "close failed")
	assertNoTempFiles(t, base, filepath.Dir(target))
}

func TestWriteFileAtomicReturnsErrorWhenChmodFails(t *testing.T) {
	base := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(target), 0755))

	fs := chmodErrorFs{
		Fs:           base,
		failContains: ".mmm.tmp",
		err:          errors.New("chmod failed"),
	}

	err := writeFileAtomic(fs, target, []byte("new"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chmod failed")
	assertNoTempFiles(t, base, filepath.Dir(target))
}

func TestWriteFileAtomicReturnsErrorWhenCannotAllocateBackupPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(target), 0755))

	backupBase := target + ".mmm.bak"
	for i := 0; i < 100; i++ {
		name := backupBase
		if i > 0 {
			name = backupBase + fmt.Sprintf(".%d", i)
		}
		assert.NoError(t, afero.WriteFile(fs, name, []byte("x"), 0644))
	}

	assert.Error(t, writeFileAtomic(fs, target, []byte("new")))
}

func assertNoTempFiles(t *testing.T, fs afero.Fs, dir string) {
	t.Helper()

	entries, err := afero.ReadDir(fs, dir)
	assert.NoError(t, err)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".mmm.tmp") {
			t.Fatalf("unexpected temp file left behind: %s", entry.Name())
		}
	}
}

func TestWriteFileAtomicReturnsWriteError(t *testing.T) {
	base := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, base.MkdirAll(filepath.Dir(target), 0755))

	fs := afero.NewReadOnlyFs(base)
	assert.Error(t, writeFileAtomic(fs, target, []byte("new")))
}

func TestWriteConfigUsesAtomicWrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.21.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{},
	}

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	assert.NoError(t, WriteConfig(context.Background(), fs, meta, cfg))
	_, err := ReadConfig(context.Background(), fs, meta)
	assert.NoError(t, err)
}

func TestWriteLockUsesAtomicWrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	assert.NoError(t, fs.MkdirAll(meta.Dir(), 0755))

	expected := []models.ModInstall{{ID: "1", Name: "Example", Type: models.MODRINTH}}
	assert.NoError(t, WriteLock(context.Background(), fs, meta, expected))

	actual, err := ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

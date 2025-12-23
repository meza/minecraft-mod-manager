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
	onlyWhenDestExists bool
	err                error
}

func (r renameFailFs) Rename(oldname, newname string) error {
	for _, failure := range r.failures {
		if failure.old != "" && oldname != failure.old {
			continue
		}
		if failure.new != "" && newname != failure.new {
			continue
		}
		if failure.onlyWhenDestExists {
			exists, err := afero.Exists(r.Fs, newname)
			if err != nil || !exists {
				continue
			}
		}
		if failure.err != nil {
			return failure.err
		}
		return errors.New("rename failed")
	}
	return r.Fs.Rename(oldname, newname)
}

type removeErrorFs struct {
	afero.Fs
	failPaths map[string]error
}

func (r removeErrorFs) Remove(name string) error {
	if err, ok := r.failPaths[filepath.Clean(name)]; ok {
		if err != nil {
			return err
		}
		return errors.New("remove failed")
	}
	return r.Fs.Remove(name)
}

type removeAfterFirstFs struct {
	afero.Fs
	failPath  string
	failErr   error
	callCount int
}

func (r *removeAfterFirstFs) Remove(name string) error {
	if filepath.Clean(name) == filepath.Clean(r.failPath) {
		r.callCount++
		if r.callCount == 1 {
			return nil
		}
		if r.failErr != nil {
			return r.failErr
		}
		return errors.New("remove failed")
	}
	return r.Fs.Remove(name)
}

type openFileErrorFs struct {
	afero.Fs
	failOn string
}

func (o openFileErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if strings.Contains(filepath.Clean(name), o.failOn) {
		return nil, errors.New("open failed")
	}
	return o.Fs.OpenFile(name, flag, perm)
}

func TestWriteFileAtomicCreatesWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, writeFileAtomic(fs, path, []byte("ok"), 0644))

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
			old: path + ".mmm.tmp",
			new: path,
		}},
	}

	assert.Error(t, writeFileAtomic(fs, path, []byte("new"), 0644))

	exists, err := afero.Exists(base, path)
	assert.NoError(t, err)
	assert.False(t, exists, "target should not be created on failure")

	tempExists, err := afero.Exists(base, path+".mmm.tmp")
	assert.NoError(t, err)
	assert.False(t, tempExists, "temp file should be cleaned up")
}

func TestWriteFileAtomicReturnsJoinedErrorWhenExistsCheckFailsAndTempCleanupFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))
	statErr := errors.New("stat failed")
	fs := &removeAfterFirstFs{
		Fs: statErrorFs{
			Fs:       base,
			failPath: path,
			err:      statErr,
		},
		failPath: path + ".mmm.tmp",
		failErr:  errors.New("remove failed"),
	}

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
	assert.Error(t, err)
	assert.ErrorIs(t, err, statErr)
	assert.Contains(t, err.Error(), "failed to remove temp file")
}

func TestWriteFileAtomicReturnsErrorWhenTempWriteFails(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))

	fs := openFileErrorFs{Fs: base, failOn: ".mmm.tmp"}

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
	assert.Error(t, err)
}

func TestWriteFileAtomicReturnsJoinedErrorWhenCleanupFailsAfterRenameToMissingTarget(t *testing.T) {
	base := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, base.MkdirAll(filepath.Dir(path), 0755))

	renameFs := renameFailFs{
		Fs: base,
		failures: []renameFailure{{
			old: path + ".mmm.tmp",
			new: path,
		}},
	}
	fs := &removeAfterFirstFs{
		Fs:       renameFs,
		failPath: path + ".mmm.tmp",
		failErr:  errors.New("remove failed"),
	}

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
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
			old: path + ".mmm.tmp",
			new: path,
		}},
	}
	fs := removeErrorFs{
		Fs: failRename,
		failPaths: map[string]error{
			filepath.Clean(path + ".mmm.tmp"): errors.New("remove failed"),
		},
	}

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
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
				old: path + ".mmm.tmp",
				new: path,
			},
			{
				old: path,
				new: path + ".mmm.bak",
			},
		},
	}

	assert.Error(t, writeFileAtomic(fs, path, []byte("new"), 0644))

	data, err := afero.ReadFile(base, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("old"), data)

	tempExists, err := afero.Exists(base, path+".mmm.tmp")
	assert.NoError(t, err)
	assert.False(t, tempExists, "temp file should be cleaned up")

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
				old: path + ".mmm.tmp",
				new: path,
			},
			{
				old: path,
				new: path + ".mmm.bak",
			},
		},
	}
	fs := &removeAfterFirstFs{
		Fs:       renameFs,
		failPath: path + ".mmm.tmp",
		failErr:  errors.New("remove failed"),
	}

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
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
				old: path + ".mmm.tmp",
				new: path,
			},
			{
				old: backup,
				new: path,
			},
		},
	}

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
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
				old: path + ".mmm.tmp",
				new: path,
			},
			{
				old: backup,
				new: path,
			},
		},
	}
	fs := &removeAfterFirstFs{
		Fs:       renameFs,
		failPath: path + ".mmm.tmp",
		failErr:  errors.New("remove failed"),
	}

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
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
			old:                path + ".mmm.tmp",
			new:                path,
			onlyWhenDestExists: true,
		}},
	}

	assert.NoError(t, writeFileAtomic(fs, path, []byte("new"), 0644))

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
			old:                path + ".mmm.tmp",
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

	err := writeFileAtomic(fs, path, []byte("new"), 0644)
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
			old: path + ".mmm.tmp",
			new: path,
		}},
	}

	assert.Error(t, writeFileAtomic(fs, path, []byte("new"), 0644))

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
	assert.NoError(t, writeFileAtomic(fs, path, []byte("new"), 0644))

	data, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), data)
}

func TestWriteFileAtomicUsesNextTempPathWhenTempExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
	assert.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))
	assert.NoError(t, afero.WriteFile(fs, path+".mmm.tmp", []byte("collision"), 0644))

	assert.NoError(t, writeFileAtomic(fs, path, []byte("new"), 0644))

	data, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), data)
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
	assert.Error(t, writeFileAtomic(fs, target, []byte("new"), 0644))
}

func TestWriteFileAtomicReturnsErrorWhenCannotAllocateTempPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")

	assert.NoError(t, fs.MkdirAll(filepath.Dir(target), 0755))

	tempBase := target + ".mmm.tmp"
	for i := 0; i < 100; i++ {
		name := tempBase
		if i > 0 {
			name = tempBase + fmt.Sprintf(".%d", i)
		}
		assert.NoError(t, afero.WriteFile(fs, name, []byte("x"), 0644))
	}

	assert.Error(t, writeFileAtomic(fs, target, []byte("new"), 0644))
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

	assert.Error(t, writeFileAtomic(fs, target, []byte("new"), 0644))
}

func TestWriteFileAtomicReturnsWriteError(t *testing.T) {
	base := afero.NewMemMapFs()
	target := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, base.MkdirAll(filepath.Dir(target), 0755))

	fs := afero.NewReadOnlyFs(base)
	assert.Error(t, writeFileAtomic(fs, target, []byte("new"), 0644))
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

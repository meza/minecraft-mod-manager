package config

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestEnsureLockCreatesEmptyWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	lock, err := EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Empty(t, lock)
}

func TestEnsureLockReturnsErrorWhenCannotCreateLock(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	_, err := EnsureLock(context.Background(), fs, meta)
	assert.Error(t, err)
}

func TestWriteLockAndReadLockRoundTrip(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	expected := []models.ModInstall{{Id: "1", Name: "Example", Type: models.MODRINTH}}
	err := WriteLock(context.Background(), fs, meta, expected)
	assert.NoError(t, err)

	actual, err := ReadLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestEnsureLockReadsExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	expected := []models.ModInstall{{Id: "1", Name: "Example", Type: models.MODRINTH}}
	err := WriteLock(context.Background(), fs, meta, expected)
	assert.NoError(t, err)

	actual, err := EnsureLock(context.Background(), fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestReadLockReturnsErrorWhenPathIsDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))
	assert.NoError(t, fs.Mkdir(meta.LockPath(), 0755))

	_, err := ReadLock(context.Background(), fs, meta)
	assert.Error(t, err)
}

func TestReadLockReturnsReadErrorWhenPathIsDirectoryOnOsFs(t *testing.T) {
	configDir := t.TempDir()
	meta := NewMetadata(filepath.Join(configDir, "modlist.json"))
	assert.NoError(t, afero.NewOsFs().MkdirAll(meta.LockPath(), 0755))

	_, err := ReadLock(context.Background(), afero.NewOsFs(), meta)
	assert.Error(t, err)
}

func TestReadLockReturnsErrorOnMalformedJSON(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))
	assert.NoError(t, afero.WriteFile(fs, meta.LockPath(), []byte("not json"), 0644))

	_, err := ReadLock(context.Background(), fs, meta)
	assert.Error(t, err)
}

func TestEnsureLockReturnsErrorWhenExistingLockIsUnreadable(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))
	assert.NoError(t, fs.Mkdir(meta.LockPath(), 0755))

	_, err := EnsureLock(context.Background(), fs, meta)
	assert.Error(t, err)
}

func TestWriteLockReturnsErrorWhenPathIsDirectory(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	err := WriteLock(context.Background(), fs, meta, nil)
	assert.Error(t, err)
}

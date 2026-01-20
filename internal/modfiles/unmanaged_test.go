package modfiles

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs statErrorFs) Stat(name string) (os.FileInfo, error) {
	if name == fs.failPath {
		return nil, fs.err
	}
	return fs.Fs.Stat(name)
}

func TestListJarFilesReturnsReadErrorOnMissingModsFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	_, err := ListJarFiles(fs, meta, cfg)
	var readErr *ReadError
	assert.ErrorAs(t, err, &readErr)
	if assert.NotNil(t, readErr) {
		assert.Equal(t, meta.ModsFolderPath(cfg), readErr.Path)
	}
}

func TestReadErrorFormatsAndUnwraps(t *testing.T) {
	inner := errors.New("boom")
	err := &ReadError{Path: "/mods", Err: inner}

	assert.Equal(t, "/mods: boom", err.Error())
	assert.ErrorIs(t, err, inner)
}

func TestListJarFilesSkipsDirsNonJarAndIgnores(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, fs.MkdirAll(filepath.Join(meta.ModsFolderPath(cfg), "nested"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "note.txt"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "ignored.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.Dir(), ".mmmignore"), []byte("ignored.jar\n"), 0644))

	files, err := ListJarFiles(fs, meta, cfg)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), files[0])
}

func TestListJarFilesReturnsReadErrorOnIgnorePatternFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "good.jar"), []byte("data"), 0644))

	wrapped := statErrorFs{
		Fs:       fs,
		failPath: filepath.Join(meta.Dir(), ".mmmignore"),
		err:      errors.New("stat failed"),
	}

	_, err := ListJarFiles(wrapped, meta, cfg)
	var readErr *ReadError
	assert.ErrorAs(t, err, &readErr)
	if assert.NotNil(t, readErr) {
		assert.Equal(t, filepath.Join(meta.Dir(), ".mmmignore"), readErr.Path)
	}
}

func TestListUnmanagedFilesReturnsUnmanagedOnly(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "managed.jar"), []byte("data"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	lock := []models.ModInstall{{ID: "managed", FileName: "managed.jar"}}

	files, err := ListUnmanagedFiles(fs, meta, cfg, lock)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar")}, files)
}

func TestListUnmanagedFilesReturnsErrorFromJarListing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	_, err := ListUnmanagedFiles(fs, meta, cfg, nil)
	var readErr *ReadError
	assert.ErrorAs(t, err, &readErr)
}

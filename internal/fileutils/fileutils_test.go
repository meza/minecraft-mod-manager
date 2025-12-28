package fileutils

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type statErrorFs struct {
	afero.Fs
	err error
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filesystem.err != nil {
		return nil, filesystem.err
	}
	return nil, errors.New("stat failed")
}

func TestFileExists(t *testing.T) {
	mockIO := afero.NewMemMapFs()

	t.Run("file exists", func(t *testing.T) {
		err := afero.WriteFile(mockIO, "/somepath", []byte("test"), 0644)
		assert.Nil(t, err)
		exists, err := FileExists("/somepath", mockIO)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("file does not exist", func(t *testing.T) {
		exists, err := FileExists("/somepath2", mockIO)
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("stat error returns false", func(t *testing.T) {
		fs := statErrorFs{Fs: mockIO, err: errors.New("stat failed")}
		exists, err := FileExists("/somepath3", fs)
		assert.Error(t, err)
		assert.False(t, exists)
	})
}

func TestInitFilesystem(t *testing.T) {
	mockIO := afero.NewMemMapFs()

	t.Run("default filesystem", func(t *testing.T) {
		assert.Equal(t, afero.NewOsFs(), InitFilesystem())
	})

	t.Run("custom filesystem", func(t *testing.T) {
		assert.Equal(t, mockIO, InitFilesystem(mockIO))
	})
}

func TestListFilesInDir(t *testing.T) {
	t.Run("directory exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dirPath := filepath.FromSlash("/testdir")
		file1 := filepath.Join(dirPath, "file1.txt")
		file2 := filepath.Join(dirPath, "file2.txt")
		assert.NoError(t, fs.MkdirAll(dirPath, 0755))

		assert.NoError(t, afero.WriteFile(fs, file1, []byte("content1"), 0644))
		assert.NoError(t, afero.WriteFile(fs, file2, []byte("content2"), 0644))

		files, err := ListFilesInDir(dirPath, fs)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{file1, file2}, files)
	})

	t.Run("directory does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dirPath := filepath.FromSlash("/nonexistentdir")

		files, err := ListFilesInDir(dirPath, fs)
		assert.ErrorContains(t, err, "failed to list files in directory")
		assert.Nil(t, files)
	})

	t.Run("ignores subdirectories", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dirPath := filepath.FromSlash("/testdir")
		file1 := filepath.Join(dirPath, "file1.txt")
		assert.NoError(t, fs.MkdirAll(dirPath, 0755))
		assert.NoError(t, fs.MkdirAll(filepath.Join(dirPath, "subdir"), 0755))

		assert.NoError(t, afero.WriteFile(fs, file1, []byte("content1"), 0644))

		files, err := ListFilesInDir(dirPath, fs)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{file1}, files)
	})
}

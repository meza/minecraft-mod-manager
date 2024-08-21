package fileutils

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"path/filepath"
	"testing"
)

type MockFileIO struct {
	mock.Mock
}

func TestFileExists(t *testing.T) {
	mockIO := afero.NewMemMapFs()

	t.Run("file exists", func(t *testing.T) {
		err := afero.WriteFile(mockIO, "/somepath", []byte("test"), 0644)
		assert.Nil(t, err)
		assert.True(t, FileExists("/somepath", mockIO))
	})

	t.Run("file does not exist", func(t *testing.T) {
		assert.False(t, FileExists("/somepath2", mockIO))
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
		fs.MkdirAll(dirPath, 0755)

		afero.WriteFile(fs, file1, []byte("content1"), 0644)
		afero.WriteFile(fs, file2, []byte("content2"), 0644)

		files, err := ListFilesInDir(dirPath, fs)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{file1, file2}, files)
	})

	t.Run("directory does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dirPath := filepath.FromSlash("/nonexistentdir")

		files, err := ListFilesInDir(dirPath, fs)
		assert.Error(t, err)
		assert.Nil(t, files)
	})

	t.Run("ignores subdirectories", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dirPath := filepath.FromSlash("/testdir")
		file1 := filepath.Join(dirPath, "file1.txt")
		fs.MkdirAll(dirPath, 0755)
		fs.MkdirAll(filepath.Join(dirPath, "subdir"), 0755)

		afero.WriteFile(fs, file1, []byte("content1"), 0644)

		files, err := ListFilesInDir(dirPath, fs)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{file1}, files)
	})
}

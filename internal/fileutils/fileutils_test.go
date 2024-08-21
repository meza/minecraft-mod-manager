package fileutils

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

package fileutils

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"os"
	"testing"
)

type MockFileIO struct {
	mock.Mock
}

func TestFileExists(t *testing.T) {
	mockIO := new(MockFileIO)

	t.Run("file exists", func(t *testing.T) {
		mockIO.On("Stat", mock.Anything).Return(nil, nil)
		assert.True(t, FileExists("somepath"))
	})

	t.Run("file does not exist", func(t *testing.T) {
		mockIO.On("Stat", mock.Anything).Return(nil, os.ErrNotExist)
		assert.False(t, FileExists("somepath"))
	})
}

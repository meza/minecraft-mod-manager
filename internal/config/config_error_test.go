package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigErrors(t *testing.T) {
	t.Run("FileInvalidError", func(t *testing.T) {
		err := &FileInvalidError{
			Err: errors.New("sample error"),
		}
		expected := "Configuration file is invalid: sample error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("ConfigFileNotFoundException", func(t *testing.T) {
		err := &ConfigFileNotFoundException{
			Path: "/path/to/config.json",
			Err:  errors.New("file not found"),
		}
		expected := "Configuration file not found: /path/to/config.json"
		assert.Equal(t, expected, err.Error())
	})
}

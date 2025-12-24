package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		if filesystem.err != nil {
			return nil, filesystem.err
		}
		return nil, errors.New("stat failed")
	}
	return filesystem.Fs.Stat(name)
}

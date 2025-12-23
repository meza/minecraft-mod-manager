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

func (s statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(s.failPath) {
		if s.err != nil {
			return nil, s.err
		}
		return nil, errors.New("stat failed")
	}
	return s.Fs.Stat(name)
}

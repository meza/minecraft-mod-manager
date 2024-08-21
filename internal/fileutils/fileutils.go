package fileutils

import (
	"github.com/spf13/afero"
)

func FileExists(path string, filesystem ...afero.Fs) bool {
	fs := afero.NewOsFs()
	if len(filesystem) > 0 {
		fs = filesystem[0]
	}

	exists, _ := afero.Exists(fs, path)
	return exists
}

func InitFilesystem(filesystem ...afero.Fs) afero.Fs {
	if len(filesystem) > 0 {
		return filesystem[0]
	}

	return afero.NewOsFs()
}

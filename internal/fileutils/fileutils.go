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

package fileutils

import (
	"fmt"
	"github.com/spf13/afero"
	"path/filepath"
)

func FileExists(path string, filesystem ...afero.Fs) bool {
	fs := InitFilesystem(filesystem...)

	exists, err := afero.Exists(fs, path)
	if err != nil {
		return false
	}
	return exists
}

func InitFilesystem(filesystem ...afero.Fs) afero.Fs {
	if len(filesystem) > 0 {
		return filesystem[0]
	}

	return afero.NewOsFs()
}

func ListFilesInDir(path string, filesystem ...afero.Fs) ([]string, error) {
	fs := InitFilesystem(filesystem...)

	files, err := afero.ReadDir(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in directory: %w", err)
	}

	var fileNames []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileNames = append(fileNames, filepath.Join(path, file.Name()))
	}

	return fileNames, nil
}

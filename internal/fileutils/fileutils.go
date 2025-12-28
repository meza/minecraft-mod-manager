// Package fileutils contains filesystem helpers.
package fileutils

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

func FileExists(path string, filesystem ...afero.Fs) (bool, error) {
	fs := InitFilesystem(filesystem...)

	exists, err := afero.Exists(fs, path)
	if err != nil {
		return false, fmt.Errorf("failed to check if %q exists: %w", path, err)
	}
	return exists, nil
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

	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileNames = append(fileNames, filepath.Join(path, file.Name()))
	}

	return fileNames, nil
}

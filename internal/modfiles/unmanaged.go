// Package modfiles provides helpers for discovering mod files on disk.
package modfiles

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
)

// ReadError reports failures while reading mod files or ignore patterns.
type ReadError struct {
	Path string
	Err  error
}

func (err *ReadError) Error() string {
	return fmt.Sprintf("%s: %v", err.Path, err.Err)
}

func (err *ReadError) Unwrap() error {
	return err.Err
}

// ListUnmanagedFiles returns jar files in the mods folder that are not in the lock file.
// It applies .mmmignore patterns and ignores non-jar files.
func ListUnmanagedFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall) ([]string, error) {
	candidates, err := ListJarFiles(fs, meta, cfg)
	if err != nil {
		return nil, err
	}

	unmanaged := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if isManagedFile(candidate, lock) {
			continue
		}
		unmanaged = append(unmanaged, candidate)
	}
	return unmanaged, nil
}

// ListJarFiles returns all jar files in the mods folder, excluding ignored entries.
func ListJarFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON) ([]string, error) {
	modsFolder := meta.ModsFolderPath(cfg)
	allEntries, err := afero.ReadDir(fs, modsFolder)
	if err != nil {
		return nil, &ReadError{Path: modsFolder, Err: err}
	}

	candidates := make([]string, 0, len(allEntries))
	for _, entry := range allEntries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(modsFolder, entry.Name()))
	}

	patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
	if err != nil {
		return nil, &ReadError{Path: filepath.Join(meta.Dir(), ".mmmignore"), Err: err}
	}

	filtered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if mmmignore.IsIgnored(modsFolder, candidate, patterns) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered, nil
}

func isManagedFile(filePath string, lock []models.ModInstall) bool {
	fileName := filepath.Base(filePath)
	for _, install := range lock {
		if install.FileName == fileName {
			return true
		}
	}
	return false
}

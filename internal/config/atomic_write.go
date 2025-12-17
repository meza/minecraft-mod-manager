package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"
)

func writeFileAtomic(fs afero.Fs, targetPath string, data []byte, mode os.FileMode) error {
	tempPath, err := nextSiblingPath(fs, targetPath, ".tmp")
	if err != nil {
		return err
	}
	backupPath, err := nextSiblingPath(fs, targetPath, ".bak")
	if err != nil {
		return err
	}

	_ = fs.Remove(tempPath)
	if err := afero.WriteFile(fs, tempPath, data, mode); err != nil {
		return err
	}

	exists, err := afero.Exists(fs, targetPath)
	if err != nil {
		_ = fs.Remove(tempPath)
		return err
	}

	if !exists {
		if err := fs.Rename(tempPath, targetPath); err != nil {
			_ = fs.Remove(tempPath)
			return err
		}
		return nil
	}

	// Prefer overwrite-rename if the filesystem supports it (common on Unix-like systems). This avoids a window where
	// the destination is temporarily missing (though backed up) between two renames.
	if err := fs.Rename(tempPath, targetPath); err == nil {
		return nil
	}

	if err := fs.Rename(targetPath, backupPath); err != nil {
		_ = fs.Remove(tempPath)
		return err
	}

	if err := fs.Rename(tempPath, targetPath); err != nil {
		_ = fs.Remove(tempPath)
		_ = fs.Rename(backupPath, targetPath)
		return err
	}

	_ = fs.Remove(backupPath)

	return nil
}

func nextSiblingPath(fs afero.Fs, targetPath string, suffix string) (string, error) {
	base := targetPath + ".mmm" + suffix

	candidate := base
	for i := 0; i < 100; i++ {
		exists, err := afero.Exists(fs, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s.%d", base, i+1)
	}

	return "", errors.New("cannot allocate sibling path")
}

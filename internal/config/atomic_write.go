package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"
)

const defaultFileMode os.FileMode = 0o644

func writeFileAtomic(fs afero.Fs, targetPath string, data []byte) error {
	tempPath, err := nextSiblingPath(fs, targetPath, ".tmp")
	if err != nil {
		return err
	}
	backupPath, err := nextSiblingPath(fs, targetPath, ".bak")
	if err != nil {
		return err
	}

	removeErr := fs.Remove(tempPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr)
	}
	writeErr := afero.WriteFile(fs, tempPath, data, defaultFileMode)
	if writeErr != nil {
		return writeErr
	}

	exists, err := afero.Exists(fs, targetPath)
	if err != nil {
		cleanupErr := fs.Remove(tempPath)
		if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, cleanupErr))
		}
		return err
	}

	if !exists {
		renameErr := fs.Rename(tempPath, targetPath)
		if renameErr != nil {
			cleanupErr := fs.Remove(tempPath)
			if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
				return errors.Join(renameErr, fmt.Errorf("failed to remove temp file %s: %w", tempPath, cleanupErr))
			}
			return renameErr
		}
		return nil
	}

	// Prefer overwrite-rename if the filesystem supports it (common on Unix-like systems). This avoids a window where
	// the destination is temporarily missing (though backed up) between two renames.
	renameErr := fs.Rename(tempPath, targetPath)
	if renameErr == nil {
		return nil
	}

	renameErr = fs.Rename(targetPath, backupPath)
	if renameErr != nil {
		cleanupErr := fs.Remove(tempPath)
		if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			return errors.Join(renameErr, fmt.Errorf("failed to remove temp file %s: %w", tempPath, cleanupErr))
		}
		return renameErr
	}

	renameErr = fs.Rename(tempPath, targetPath)
	if renameErr != nil {
		cleanupErr := fs.Remove(tempPath)
		rollbackErr := fs.Rename(backupPath, targetPath)
		if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			renameErr = errors.Join(renameErr, fmt.Errorf("failed to remove temp file %s: %w", tempPath, cleanupErr))
		}
		if rollbackErr != nil {
			renameErr = errors.Join(renameErr, fmt.Errorf("failed to restore backup %s: %w", backupPath, rollbackErr))
		}
		return renameErr
	}

	removeBackupErr := fs.Remove(backupPath)
	if removeBackupErr != nil && !errors.Is(removeBackupErr, os.ErrNotExist) {
		return fmt.Errorf("failed to remove backup file %s: %w", backupPath, removeBackupErr)
	}

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

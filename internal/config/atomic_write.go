package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

const defaultFileMode os.FileMode = 0o644

func writeFileAtomic(fs afero.Fs, targetPath string, data []byte) error {
	backupPath, err := nextSiblingPath(fs, targetPath, ".bak")
	if err != nil {
		return err
	}

	tempPath, err := writeTempFile(fs, targetPath, data)
	if err != nil {
		return err
	}

	exists, err := afero.Exists(fs, targetPath)
	if err != nil {
		return cleanupTempOnError(fs, tempPath, err)
	}
	if !exists {
		return renameTempIntoPlace(fs, tempPath, targetPath)
	}

	return replaceExistingFile(fs, tempPath, targetPath, backupPath)
}

func writeTempFile(fs afero.Fs, targetPath string, data []byte) (string, error) {
	tempFile, err := afero.TempFile(fs, filepath.Dir(targetPath), filepath.Base(targetPath)+".mmm.tmp.*")
	if err != nil {
		return "", err
	}

	tempPath := tempFile.Name()
	if _, err := tempFile.Write(data); err != nil {
		return "", cleanupTempOnErrorWithClose(fs, tempFile, tempPath, err)
	}
	if closeErr := tempFile.Close(); closeErr != nil {
		return "", cleanupTempOnError(fs, tempPath, closeErr)
	}
	if chmodErr := fs.Chmod(tempPath, defaultFileMode); chmodErr != nil {
		return "", cleanupTempOnError(fs, tempPath, chmodErr)
	}

	return tempPath, nil
}

func cleanupTempOnErrorWithClose(fs afero.Fs, tempFile afero.File, tempPath string, err error) error {
	if closeErr := tempFile.Close(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}
	return cleanupTempOnError(fs, tempPath, err)
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

func removePathIfExists(fs afero.Fs, path string) error {
	removeErr := fs.Remove(path)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}
	return nil
}

func removePathError(kind string, path string, err error) error {
	return fmt.Errorf("failed to remove %s %s: %w", kind, path, err)
}

func cleanupTempOnError(fs afero.Fs, tempPath string, originalErr error) error {
	cleanupErr := removePathIfExists(fs, tempPath)
	if cleanupErr != nil {
		return errors.Join(originalErr, removePathError("temp file", tempPath, cleanupErr))
	}
	return originalErr
}

func renameTempIntoPlace(fs afero.Fs, tempPath string, targetPath string) error {
	renameErr := fs.Rename(tempPath, targetPath)
	if renameErr == nil {
		return nil
	}
	cleanupErr := removePathIfExists(fs, tempPath)
	if cleanupErr != nil {
		return errors.Join(renameErr, removePathError("temp file", tempPath, cleanupErr))
	}
	return renameErr
}

func replaceExistingFile(fs afero.Fs, tempPath string, targetPath string, backupPath string) error {
	// Prefer overwrite-rename if the filesystem supports it (common on Unix-like systems). This avoids a window where
	// the destination is temporarily missing (though backed up) between two renames.
	renameErr := fs.Rename(tempPath, targetPath)
	if renameErr == nil {
		return nil
	}

	renameErr = fs.Rename(targetPath, backupPath)
	if renameErr != nil {
		return cleanupTempOnError(fs, tempPath, renameErr)
	}

	renameErr = fs.Rename(tempPath, targetPath)
	if renameErr != nil {
		return restoreBackupOnFailure(fs, tempPath, targetPath, backupPath, renameErr)
	}

	removeBackupErr := removePathIfExists(fs, backupPath)
	if removeBackupErr != nil {
		return removePathError("backup file", backupPath, removeBackupErr)
	}

	return nil
}

func restoreBackupOnFailure(fs afero.Fs, tempPath string, targetPath string, backupPath string, renameErr error) error {
	cleanupErr := removePathIfExists(fs, tempPath)
	rollbackErr := fs.Rename(backupPath, targetPath)
	if cleanupErr != nil {
		renameErr = errors.Join(renameErr, removePathError("temp file", tempPath, cleanupErr))
	}
	if rollbackErr != nil {
		renameErr = errors.Join(renameErr, fmt.Errorf("failed to restore backup %s: %w", backupPath, rollbackErr))
	}
	return renameErr
}

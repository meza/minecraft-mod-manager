package update

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/afero"
)

func downloadAndSwap(ctx context.Context, deps updateDeps, oldPath string, newPath string, modsFolder string, downloadURL string, expectedHash string) error {
	if strings.TrimSpace(expectedHash) == "" {
		return modinstall.MissingHashError{FileName: filepath.Base(newPath)}
	}

	resolvedNewPath, tempPath, err := prepareDownloadPaths(deps.fs, modsFolder, newPath)
	if err != nil {
		return err
	}

	if err := downloadToTemp(ctx, deps, downloadURL, tempPath); err != nil {
		return removeTempFile(deps.fs, tempPath, err)
	}

	if err := verifyDownloadedHash(deps.fs, tempPath, expectedHash, newPath); err != nil {
		return removeTempFile(deps.fs, tempPath, err)
	}

	if err := replaceExistingFile(deps.fs, deps.logger, tempPath, resolvedNewPath); err != nil {
		return removeTempFile(deps.fs, tempPath, err)
	}

	return removeOldInstall(deps.fs, oldPath, newPath, resolvedNewPath)
}

func prepareDownloadPaths(fs afero.Fs, modsFolder string, newPath string) (resolvedNewPath string, tempPath string, err error) {
	resolvedNewPath, err = modpath.ResolveWritablePath(fs, modsFolder, newPath)
	if err != nil {
		return "", "", err
	}

	tempPath, err = createTempDownloadPath(fs, resolvedNewPath)
	if err != nil {
		return "", "", err
	}

	return resolvedNewPath, tempPath, nil
}

func downloadToTemp(ctx context.Context, deps updateDeps, downloadURL string, tempPath string) error {
	return deps.downloader(ctx, downloadURL, tempPath, platform.PreferredDownloadClient(deps.clients), nil, deps.fs)
}

func removeOldInstall(fs afero.Fs, oldPath string, newPath string, resolvedNewPath string) error {
	if filepath.Clean(oldPath) == filepath.Clean(newPath) {
		return nil
	}

	if err := fs.Remove(oldPath); err != nil {
		removeErr := fs.Remove(resolvedNewPath)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("failed to remove new file %s: %w", resolvedNewPath, removeErr))
		}
		return err
	}

	return nil
}

func createTempDownloadPath(fs afero.Fs, resolvedNewPath string) (string, error) {
	tempFile, err := afero.TempFile(fs, filepath.Dir(resolvedNewPath), filepath.Base(resolvedNewPath)+".mmm.*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return "", removeTempFile(fs, tempPath, err)
	}
	return tempPath, nil
}

func removeTempFile(fs afero.Fs, tempPath string, err error) error {
	removeErr := fs.Remove(tempPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
	}
	return err
}

func verifyDownloadedHash(fs afero.Fs, tempPath string, expectedHash string, newPath string) error {
	actualHash, err := sha1ForFile(fs, tempPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(expectedHash), actualHash) {
		return modinstall.HashMismatchError{FileName: filepath.Base(newPath), Expected: expectedHash, Actual: actualHash}
	}
	return nil
}

func replaceExistingFile(fs afero.Fs, log *logger.Logger, sourcePath string, destinationPath string) error {
	exists, err := afero.Exists(fs, destinationPath)
	if err != nil {
		return err
	}

	if !exists {
		return fs.Rename(sourcePath, destinationPath)
	}

	backupPath, err := nextBackupPath(fs, destinationPath)
	if err != nil {
		return err
	}

	if err := fs.Rename(destinationPath, backupPath); err != nil {
		return err
	}

	if err := fs.Rename(sourcePath, destinationPath); err != nil {
		rollbackErr := fs.Rename(backupPath, destinationPath)
		if rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("failed to restore backup %s: %w", backupPath, rollbackErr))
		}
		return err
	}

	if removeErr := fs.Remove(backupPath); removeErr != nil {
		if logErr := log.Debug(i18n.T("cmd.update.debug.backup_cleanup_failed", &i18n.Tvars{
			Data: &i18n.TData{
				"path": backupPath,
				"err":  removeErr.Error(),
			},
		})); logErr != nil {
			return logErr
		}
	}
	return nil
}

func nextBackupPath(fs afero.Fs, destinationPath string) (string, error) {
	base := destinationPath + ".mmm.bak"

	backup := base
	for i := 0; i < 100; i++ {
		exists, err := afero.Exists(fs, backup)
		if err != nil {
			return "", err
		}
		if !exists {
			return backup, nil
		}
		backup = fmt.Sprintf("%s.%d", base, i+1)
	}
	return "", errors.New("cannot allocate backup path")
}

func sha1ForFile(fs afero.Fs, path string) (hash string, returnErr error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

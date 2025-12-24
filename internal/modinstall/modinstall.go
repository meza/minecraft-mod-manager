// Package modinstall coordinates mod installation flows.
package modinstall

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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
)

type Downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type EnsureReason string

const (
	EnsureReasonAlreadyPresent EnsureReason = "already_present"
	EnsureReasonMissing        EnsureReason = "missing"
	EnsureReasonHashMismatch   EnsureReason = "hash_mismatch"
)

type EnsureResult struct {
	Downloaded bool
	Reason     EnsureReason
}

type MissingHashError struct {
	FileName string
}

func (err MissingHashError) Error() string {
	return "missing expected hash for " + err.FileName
}

type HashMismatchError struct {
	FileName string
	Expected string
	Actual   string
}

func (err HashMismatchError) Error() string {
	return "downloaded file hash mismatch for " + err.FileName
}

type Installer struct {
	fs         afero.Fs
	downloader Downloader
}

func NewInstaller(fs afero.Fs, downloader Downloader) *Installer {
	return &Installer{
		fs:         fs,
		downloader: downloader,
	}
}

func (installer *Installer) EnsureLockedFile(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, install models.ModInstall, downloadClient httpclient.Doer, sender httpclient.Sender) (EnsureResult, error) {
	normalizedFileName, expectedHash, err := normalizeLockData(install)
	if err != nil {
		return EnsureResult{}, err
	}
	sender = ensureSender(sender)

	modsRoot := meta.ModsFolderPath(cfg)
	mkdirErr := installer.fs.MkdirAll(modsRoot, 0755)
	if mkdirErr != nil {
		return EnsureResult{}, mkdirErr
	}

	destination := filepath.Join(modsRoot, normalizedFileName)
	resolvedDestination, err := modpath.ResolveWritablePath(installer.fs, modsRoot, destination)
	if err != nil {
		return EnsureResult{}, err
	}

	exists, err := afero.Exists(installer.fs, resolvedDestination)
	if err != nil {
		return EnsureResult{}, err
	}

	if !exists {
		if err := installer.ensureDownloader(); err != nil {
			return EnsureResult{}, err
		}
		downloadErr := installer.downloadAndVerify(ctx, install.DownloadURL, resolvedDestination, expectedHash, downloadClient, sender, normalizedFileName)
		if downloadErr != nil {
			return EnsureResult{}, downloadErr
		}
		return EnsureResult{Downloaded: true, Reason: EnsureReasonMissing}, nil
	}

	localSha, err := sha1ForFile(installer.fs, resolvedDestination)
	if err != nil {
		return EnsureResult{}, err
	}

	if !strings.EqualFold(expectedHash, localSha) {
		if err := installer.ensureDownloader(); err != nil {
			return EnsureResult{}, err
		}
		downloadErr := installer.downloadAndVerify(ctx, install.DownloadURL, resolvedDestination, expectedHash, downloadClient, sender, normalizedFileName)
		if downloadErr != nil {
			return EnsureResult{}, downloadErr
		}
		return EnsureResult{Downloaded: true, Reason: EnsureReasonHashMismatch}, nil
	}

	return EnsureResult{Downloaded: false, Reason: EnsureReasonAlreadyPresent}, nil
}

func (installer *Installer) DownloadAndVerify(ctx context.Context, url string, destination string, expectedHash string, downloadClient httpclient.Doer, sender httpclient.Sender) error {
	if strings.TrimSpace(expectedHash) == "" {
		return MissingHashError{FileName: filepath.Base(destination)}
	}
	sender = ensureSender(sender)
	if err := installer.ensureDownloader(); err != nil {
		return err
	}
	return installer.downloadAndVerify(ctx, url, destination, expectedHash, downloadClient, sender, filepath.Base(destination))
}

func (installer *Installer) downloadAndVerify(ctx context.Context, url string, destination string, expectedHash string, downloadClient httpclient.Doer, sender httpclient.Sender, displayName string) error {
	tempPath, err := installer.createTempFile(destination)
	if err != nil {
		return err
	}
	return installer.downloadTemp(ctx, url, destination, expectedHash, downloadClient, sender, displayName, tempPath)
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

func replaceExistingFile(fs afero.Fs, sourcePath string, destinationPath string) error {
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

	if err := fs.Remove(backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove backup %s: %w", backupPath, err)
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

type noopSender struct{}

func (noopSender) Send(msg tea.Msg) { _ = msg }

func normalizeLockData(install models.ModInstall) (string, string, error) {
	if strings.TrimSpace(install.FileName) == "" {
		return "", "", errors.New("missing lock fileName")
	}
	normalizedFileName, err := modfilename.Normalize(install.FileName)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(install.DownloadURL) == "" {
		return "", "", errors.New("missing lock downloadUrl")
	}
	expectedHash := strings.TrimSpace(install.Hash)
	if expectedHash == "" {
		return "", "", MissingHashError{FileName: install.FileName}
	}
	return normalizedFileName, expectedHash, nil
}

func ensureSender(sender httpclient.Sender) httpclient.Sender {
	if sender == nil {
		return noopSender{}
	}
	return sender
}

func (installer *Installer) ensureDownloader() error {
	if installer.downloader == nil {
		return errors.New("missing modinstall dependencies: downloader")
	}
	return nil
}

func (installer *Installer) createTempFile(destination string) (string, error) {
	tempFile, err := afero.TempFile(installer.fs, filepath.Dir(destination), filepath.Base(destination)+".mmm.*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	if closeErr := tempFile.Close(); closeErr != nil {
		return "", cleanupTempOnError(installer.fs, tempPath, closeErr)
	}
	return tempPath, nil
}

func (installer *Installer) downloadTemp(
	ctx context.Context,
	url string,
	destination string,
	expectedHash string,
	downloadClient httpclient.Doer,
	sender httpclient.Sender,
	displayName string,
	tempPath string,
) error {
	downloadErr := installer.downloader(ctx, url, tempPath, downloadClient, sender, installer.fs)
	if downloadErr != nil {
		return cleanupTempOnError(installer.fs, tempPath, downloadErr)
	}

	actualHash, err := sha1ForFile(installer.fs, tempPath)
	if err != nil {
		return cleanupTempOnError(installer.fs, tempPath, err)
	}

	if !strings.EqualFold(strings.TrimSpace(expectedHash), actualHash) {
		hashErr := HashMismatchError{FileName: displayName, Expected: expectedHash, Actual: actualHash}
		return cleanupTempOnError(installer.fs, tempPath, hashErr)
	}

	if err := replaceExistingFile(installer.fs, tempPath, destination); err != nil {
		return cleanupTempOnError(installer.fs, tempPath, err)
	}

	return nil
}

func cleanupTempOnError(fs afero.Fs, tempPath string, err error) error {
	removeErr := fs.Remove(tempPath)
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
	}
	return err
}

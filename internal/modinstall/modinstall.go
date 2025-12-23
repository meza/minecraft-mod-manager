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

func (s *Installer) EnsureLockedFile(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, install models.ModInstall, downloadClient httpclient.Doer, sender httpclient.Sender) (EnsureResult, error) {
	if strings.TrimSpace(install.FileName) == "" {
		return EnsureResult{}, errors.New("missing lock fileName")
	}
	normalizedFileName, err := modfilename.Normalize(install.FileName)
	if err != nil {
		return EnsureResult{}, err
	}
	if strings.TrimSpace(install.DownloadURL) == "" {
		return EnsureResult{}, errors.New("missing lock downloadUrl")
	}
	expectedHash := strings.TrimSpace(install.Hash)
	if expectedHash == "" {
		return EnsureResult{}, MissingHashError{FileName: install.FileName}
	}
	if sender == nil {
		sender = noopSender{}
	}

	modsRoot := meta.ModsFolderPath(cfg)
	if err := s.fs.MkdirAll(modsRoot, 0755); err != nil {
		return EnsureResult{}, err
	}

	destination := filepath.Join(modsRoot, normalizedFileName)
	resolvedDestination, err := modpath.ResolveWritablePath(s.fs, modsRoot, destination)
	if err != nil {
		return EnsureResult{}, err
	}

	exists, err := afero.Exists(s.fs, resolvedDestination)
	if err != nil {
		return EnsureResult{}, err
	}

	if !exists {
		if s.downloader == nil {
			return EnsureResult{}, errors.New("missing modinstall dependencies: downloader")
		}
		if err := s.downloadAndVerify(ctx, install.DownloadURL, resolvedDestination, expectedHash, downloadClient, sender, normalizedFileName); err != nil {
			return EnsureResult{}, err
		}
		return EnsureResult{Downloaded: true, Reason: EnsureReasonMissing}, nil
	}

	localSha, err := sha1ForFile(s.fs, resolvedDestination)
	if err != nil {
		return EnsureResult{}, err
	}

	if !strings.EqualFold(expectedHash, localSha) {
		if s.downloader == nil {
			return EnsureResult{}, errors.New("missing modinstall dependencies: downloader")
		}
		if err := s.downloadAndVerify(ctx, install.DownloadURL, resolvedDestination, expectedHash, downloadClient, sender, normalizedFileName); err != nil {
			return EnsureResult{}, err
		}
		return EnsureResult{Downloaded: true, Reason: EnsureReasonHashMismatch}, nil
	}

	return EnsureResult{Downloaded: false, Reason: EnsureReasonAlreadyPresent}, nil
}

func (s *Installer) DownloadAndVerify(ctx context.Context, url string, destination string, expectedHash string, downloadClient httpclient.Doer, sender httpclient.Sender) error {
	if strings.TrimSpace(expectedHash) == "" {
		return MissingHashError{FileName: filepath.Base(destination)}
	}
	if sender == nil {
		sender = noopSender{}
	}
	if s.downloader == nil {
		return errors.New("missing modinstall dependencies: downloader")
	}
	return s.downloadAndVerify(ctx, url, destination, expectedHash, downloadClient, sender, filepath.Base(destination))
}

func (s *Installer) downloadAndVerify(ctx context.Context, url string, destination string, expectedHash string, downloadClient httpclient.Doer, sender httpclient.Sender, displayName string) error {
	tempFile, err := afero.TempFile(s.fs, filepath.Dir(destination), filepath.Base(destination)+".mmm.*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		if removeErr := s.fs.Remove(tempPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
		}
		return err
	}

	if err := s.downloader(ctx, url, tempPath, downloadClient, sender, s.fs); err != nil {
		removeErr := s.fs.Remove(tempPath)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
		}
		return err
	}

	actualHash, err := sha1ForFile(s.fs, tempPath)
	if err != nil {
		removeErr := s.fs.Remove(tempPath)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
		}
		return err
	}

	if !strings.EqualFold(strings.TrimSpace(expectedHash), actualHash) {
		removeErr := s.fs.Remove(tempPath)
		hashErr := HashMismatchError{FileName: displayName, Expected: expectedHash, Actual: actualHash}
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(hashErr, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
		}
		return hashErr
	}

	if err := replaceExistingFile(s.fs, tempPath, destination); err != nil {
		removeErr := s.fs.Remove(tempPath)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return errors.Join(err, fmt.Errorf("failed to remove temp file %s: %w", tempPath, removeErr))
		}
		return err
	}

	return nil
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

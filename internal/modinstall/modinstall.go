package modinstall

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
)

type Downloader func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error

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

func (s *Installer) EnsureLockedFile(ctx context.Context, meta config.Metadata, cfg models.ModsJson, install models.ModInstall, downloadClient httpClient.Doer, sender httpClient.Sender) (EnsureResult, error) {
	if strings.TrimSpace(install.FileName) == "" {
		return EnsureResult{}, errors.New("missing lock fileName")
	}
	normalizedFileName, err := modfilename.Normalize(install.FileName)
	if err != nil {
		return EnsureResult{}, err
	}
	if strings.TrimSpace(install.DownloadUrl) == "" {
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
		if err := s.downloadAndVerify(ctx, install.DownloadUrl, resolvedDestination, expectedHash, downloadClient, sender, normalizedFileName); err != nil {
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
		if err := s.downloadAndVerify(ctx, install.DownloadUrl, resolvedDestination, expectedHash, downloadClient, sender, normalizedFileName); err != nil {
			return EnsureResult{}, err
		}
		return EnsureResult{Downloaded: true, Reason: EnsureReasonHashMismatch}, nil
	}

	return EnsureResult{Downloaded: false, Reason: EnsureReasonAlreadyPresent}, nil
}

func (s *Installer) DownloadAndVerify(ctx context.Context, url string, destination string, expectedHash string, downloadClient httpClient.Doer, sender httpClient.Sender) error {
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

func (s *Installer) downloadAndVerify(ctx context.Context, url string, destination string, expectedHash string, downloadClient httpClient.Doer, sender httpClient.Sender, displayName string) error {
	tempFile, err := afero.TempFile(s.fs, filepath.Dir(destination), filepath.Base(destination)+".mmm.*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()

	if err := s.downloader(ctx, url, tempPath, downloadClient, sender, s.fs); err != nil {
		_ = s.fs.Remove(tempPath)
		return err
	}

	actualHash, err := sha1ForFile(s.fs, tempPath)
	if err != nil {
		_ = s.fs.Remove(tempPath)
		return err
	}

	if !strings.EqualFold(strings.TrimSpace(expectedHash), actualHash) {
		_ = s.fs.Remove(tempPath)
		return HashMismatchError{FileName: displayName, Expected: expectedHash, Actual: actualHash}
	}

	if err := replaceExistingFile(s.fs, tempPath, destination); err != nil {
		_ = s.fs.Remove(tempPath)
		return err
	}

	return nil
}

func sha1ForFile(fs afero.Fs, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
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
		_ = fs.Rename(backupPath, destinationPath)
		return err
	}

	_ = fs.Remove(backupPath)

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

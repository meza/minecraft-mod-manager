package modinstall

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
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

type Service struct {
	fs         afero.Fs
	downloader Downloader
}

func NewService(fs afero.Fs, downloader Downloader) *Service {
	return &Service{
		fs:         fs,
		downloader: downloader,
	}
}

func (s *Service) EnsureLockedFile(ctx context.Context, meta config.Metadata, cfg models.ModsJson, install models.ModInstall, downloadClient httpClient.Doer, sender httpClient.Sender) (EnsureResult, error) {
	if strings.TrimSpace(install.FileName) == "" {
		return EnsureResult{}, errors.New("missing lock fileName")
	}
	if strings.TrimSpace(install.DownloadUrl) == "" {
		return EnsureResult{}, errors.New("missing lock downloadUrl")
	}
	if sender == nil {
		sender = noopSender{}
	}

	destination := filepath.Join(meta.ModsFolderPath(cfg), install.FileName)

	exists, err := afero.Exists(s.fs, destination)
	if err != nil {
		return EnsureResult{}, err
	}

	if !exists {
		if err := s.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755); err != nil {
			return EnsureResult{}, err
		}
		if s.downloader == nil {
			return EnsureResult{}, errors.New("missing modinstall dependencies: downloader")
		}
		if err := s.downloader(ctx, install.DownloadUrl, destination, downloadClient, sender, s.fs); err != nil {
			return EnsureResult{}, err
		}
		return EnsureResult{Downloaded: true, Reason: EnsureReasonMissing}, nil
	}

	localSha, err := sha1ForFile(s.fs, destination)
	if err != nil {
		return EnsureResult{}, err
	}

	if !strings.EqualFold(strings.TrimSpace(install.Hash), localSha) {
		if err := s.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755); err != nil {
			return EnsureResult{}, err
		}
		if s.downloader == nil {
			return EnsureResult{}, errors.New("missing modinstall dependencies: downloader")
		}
		if err := s.downloader(ctx, install.DownloadUrl, destination, downloadClient, sender, s.fs); err != nil {
			return EnsureResult{}, err
		}
		return EnsureResult{Downloaded: true, Reason: EnsureReasonHashMismatch}, nil
	}

	return EnsureResult{Downloaded: false, Reason: EnsureReasonAlreadyPresent}, nil
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

type noopSender struct{}

func (noopSender) Send(msg tea.Msg) { _ = msg }

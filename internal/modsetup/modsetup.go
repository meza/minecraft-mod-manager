package modsetup

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type Downloader func(context.Context, string, string, httpClient.Doer, httpClient.Sender, ...afero.Fs) error

type Service struct {
	fs              afero.Fs
	minecraftClient httpClient.Doer
	downloader      Downloader
}

func NewService(fs afero.Fs, minecraftClient httpClient.Doer, downloader Downloader) *Service {
	return &Service{
		fs:              fs,
		minecraftClient: minecraftClient,
		downloader:      downloader,
	}
}

func (s *Service) EnsureConfigAndLock(ctx context.Context, meta config.Metadata, quiet bool) (models.ModsJson, []models.ModInstall, error) {
	cfg, err := config.ReadConfig(ctx, s.fs, meta)
	if err != nil {
		var notFound *config.ConfigFileNotFoundException
		if errors.As(err, &notFound) {
			if quiet {
				return models.ModsJson{}, nil, err
			}
			if s.minecraftClient == nil {
				return models.ModsJson{}, nil, errors.New("missing modsetup dependencies: minecraftClient")
			}
			cfg, err = config.InitConfig(ctx, s.fs, meta, s.minecraftClient)
			if err != nil {
				return models.ModsJson{}, nil, err
			}
		} else {
			return models.ModsJson{}, nil, err
		}
	}

	lock, err := config.EnsureLock(ctx, s.fs, meta)
	if err != nil {
		return models.ModsJson{}, nil, err
	}

	return cfg, lock, nil
}

func (s *Service) EnsureDownloaded(ctx context.Context, meta config.Metadata, cfg models.ModsJson, remote platform.RemoteMod, downloadClient httpClient.Doer) (string, error) {
	normalizedFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(remote.DownloadURL) == "" {
		return "", errors.New("remote mod missing download url")
	}
	remote.FileName = normalizedFileName

	if err := s.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755); err != nil {
		return "", err
	}

	destination := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)
	if s.downloader == nil {
		return "", errors.New("missing modsetup dependencies: downloader")
	}

	if err := s.downloader(ctx, remote.DownloadURL, destination, downloadClient, &noopSender{}, s.fs); err != nil {
		return "", err
	}

	return destination, nil
}

type EnsurePersistOptions struct {
	Version              string
	AllowVersionFallback bool
}

type EnsureResult struct {
	ConfigAdded bool
	LockAdded   bool
}

func (s *Service) EnsurePersisted(ctx context.Context, meta config.Metadata, cfg models.ModsJson, lock []models.ModInstall, resolvedPlatform models.Platform, resolvedID string, remote platform.RemoteMod, options EnsurePersistOptions) (models.ModsJson, []models.ModInstall, EnsureResult, error) {
	if strings.TrimSpace(string(resolvedPlatform)) == "" {
		return models.ModsJson{}, nil, EnsureResult{}, errors.New("missing resolved platform")
	}
	if strings.TrimSpace(resolvedID) == "" {
		return models.ModsJson{}, nil, EnsureResult{}, errors.New("missing resolved id")
	}

	configIndex := findConfigIndex(cfg, resolvedPlatform, resolvedID)
	lockIndex := findLockIndex(lock, resolvedPlatform, resolvedID)

	result := EnsureResult{}

	if configIndex < 0 {
		cfg.Mods = append(cfg.Mods, models.Mod{
			Type:                 resolvedPlatform,
			ID:                   resolvedID,
			Name:                 remote.Name,
			AllowVersionFallback: optionalBool(options.AllowVersionFallback),
			Version:              optionalString(options.Version),
		})
		result.ConfigAdded = true
	}

	if lockIndex < 0 {
		normalizedFileName, err := validateRemoteForLock(remote)
		if err != nil {
			return models.ModsJson{}, nil, EnsureResult{}, err
		}
		remote.FileName = normalizedFileName

		lock = append(lock, models.ModInstall{
			Type:        resolvedPlatform,
			Id:          resolvedID,
			Name:        remote.Name,
			FileName:    remote.FileName,
			ReleasedOn:  remote.ReleaseDate,
			Hash:        remote.Hash,
			DownloadUrl: remote.DownloadURL,
		})
		result.LockAdded = true
	}

	if result.ConfigAdded {
		if err := config.WriteConfig(ctx, s.fs, meta, cfg); err != nil {
			return models.ModsJson{}, nil, EnsureResult{}, err
		}
	}
	if result.LockAdded {
		if err := config.WriteLock(ctx, s.fs, meta, lock); err != nil {
			return models.ModsJson{}, nil, EnsureResult{}, err
		}
	}

	return cfg, lock, result, nil
}

type UpsertResult struct {
	ConfigAdded   bool
	ConfigUpdated bool
	LockAdded     bool
	LockUpdated   bool
}

func (s *Service) UpsertConfigAndLock(cfg models.ModsJson, lock []models.ModInstall, resolvedPlatform models.Platform, resolvedID string, remote platform.RemoteMod, options EnsurePersistOptions) (models.ModsJson, []models.ModInstall, UpsertResult, error) {
	if strings.TrimSpace(string(resolvedPlatform)) == "" {
		return models.ModsJson{}, nil, UpsertResult{}, errors.New("missing resolved platform")
	}
	if strings.TrimSpace(resolvedID) == "" {
		return models.ModsJson{}, nil, UpsertResult{}, errors.New("missing resolved id")
	}

	configIndex := findConfigIndex(cfg, resolvedPlatform, resolvedID)
	lockIndex := findLockIndex(lock, resolvedPlatform, resolvedID)

	result := UpsertResult{}

	if configIndex < 0 {
		cfg.Mods = append(cfg.Mods, models.Mod{
			Type:                 resolvedPlatform,
			ID:                   resolvedID,
			Name:                 remote.Name,
			AllowVersionFallback: optionalBool(options.AllowVersionFallback),
			Version:              optionalString(options.Version),
		})
		result.ConfigAdded = true
	} else if strings.TrimSpace(remote.Name) != "" && cfg.Mods[configIndex].Name != remote.Name {
		cfg.Mods[configIndex].Name = remote.Name
		result.ConfigUpdated = true
	}

	if lockIndex < 0 {
		normalizedFileName, err := validateRemoteForLock(remote)
		if err != nil {
			return models.ModsJson{}, nil, UpsertResult{}, err
		}
		remote.FileName = normalizedFileName
		lock = append(lock, models.ModInstall{
			Type:        resolvedPlatform,
			Id:          resolvedID,
			Name:        remote.Name,
			FileName:    remote.FileName,
			ReleasedOn:  remote.ReleaseDate,
			Hash:        remote.Hash,
			DownloadUrl: remote.DownloadURL,
		})
		result.LockAdded = true
	} else {
		normalizedFileName, err := validateRemoteForLock(remote)
		if err != nil {
			return models.ModsJson{}, nil, UpsertResult{}, err
		}
		remote.FileName = normalizedFileName

		current := lock[lockIndex]
		next := models.ModInstall{
			Type:        resolvedPlatform,
			Id:          resolvedID,
			Name:        remote.Name,
			FileName:    remote.FileName,
			ReleasedOn:  remote.ReleaseDate,
			Hash:        remote.Hash,
			DownloadUrl: remote.DownloadURL,
		}
		if current.Name != next.Name ||
			current.FileName != next.FileName ||
			current.ReleasedOn != next.ReleasedOn ||
			!strings.EqualFold(current.Hash, next.Hash) ||
			current.DownloadUrl != next.DownloadUrl {
			lock[lockIndex] = next
			result.LockUpdated = true
		}
	}

	return cfg, lock, result, nil
}

func ModExists(cfg models.ModsJson, platform models.Platform, projectID string) bool {
	for _, mod := range cfg.Mods {
		if mod.ID == projectID && mod.Type == platform {
			return true
		}
	}
	return false
}

func optionalBool(value bool) *bool {
	if !value {
		return nil
	}
	return &value
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func findConfigIndex(cfg models.ModsJson, platform models.Platform, projectID string) int {
	for i := range cfg.Mods {
		if cfg.Mods[i].ID == projectID && cfg.Mods[i].Type == platform {
			return i
		}
	}
	return -1
}

func findLockIndex(lock []models.ModInstall, platform models.Platform, projectID string) int {
	for i := range lock {
		if lock[i].Type == platform && lock[i].Id == projectID {
			return i
		}
	}
	return -1
}

func validateRemoteForLock(remote platform.RemoteMod) (string, error) {
	if strings.TrimSpace(remote.Name) == "" {
		return "", errors.New("remote mod missing name")
	}
	normalizedFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(remote.Hash) == "" {
		return "", errors.New("remote mod missing hash")
	}
	if strings.TrimSpace(remote.ReleaseDate) == "" {
		return "", errors.New("remote mod missing release date")
	}
	if strings.TrimSpace(remote.DownloadURL) == "" {
		return "", errors.New("remote mod missing download url")
	}
	return normalizedFileName, nil
}

type noopSender struct{}

func (n *noopSender) Send(msg tea.Msg) { _ = msg }

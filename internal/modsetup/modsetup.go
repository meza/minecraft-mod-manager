// Package modsetup orchestrates setup flows for config and lock files.
package modsetup

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type Downloader func(context.Context, string, string, httpclient.Doer, httpclient.Sender, ...afero.Fs) error

type SetupCoordinator struct {
	fs              afero.Fs
	minecraftClient httpclient.Doer
	downloader      Downloader
}

func NewSetupCoordinator(fs afero.Fs, minecraftClient httpclient.Doer, downloader Downloader) *SetupCoordinator {
	return &SetupCoordinator{
		fs:              fs,
		minecraftClient: minecraftClient,
		downloader:      downloader,
	}
}

type EnsureConfigOptions struct {
	Quiet bool
}

func (coordinator *SetupCoordinator) EnsureConfigAndLock(ctx context.Context, meta config.Metadata, options EnsureConfigOptions) (models.ModsJSON, []models.ModInstall, error) {
	cfg, err := coordinator.ensureConfig(ctx, meta, options)
	if err != nil {
		return models.ModsJSON{}, nil, err
	}

	lock, err := config.EnsureLock(ctx, coordinator.fs, meta)
	if err != nil {
		return models.ModsJSON{}, nil, err
	}

	return cfg, lock, nil
}

func (coordinator *SetupCoordinator) EnsureDownloaded(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, remote platform.RemoteMod, downloadClient httpclient.Doer) (string, error) {
	normalizedFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(remote.DownloadURL) == "" {
		return "", errors.New("remote mod missing download url")
	}
	if strings.TrimSpace(remote.Hash) == "" {
		return "", modinstall.MissingHashError{FileName: remote.FileName}
	}
	remote.FileName = normalizedFileName

	mkdirErr := coordinator.fs.MkdirAll(meta.ModsFolderPath(cfg), 0755)
	if mkdirErr != nil {
		return "", mkdirErr
	}

	destination := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)
	resolvedDestination, err := modpath.ResolveWritablePath(coordinator.fs, meta.ModsFolderPath(cfg), destination)
	if err != nil {
		return "", err
	}
	if coordinator.downloader == nil {
		return "", errors.New("missing modsetup dependencies: downloader")
	}
	installer := modinstall.NewInstaller(coordinator.fs, modinstall.Downloader(coordinator.downloader))
	downloadErr := installer.DownloadAndVerify(ctx, remote.DownloadURL, resolvedDestination, remote.Hash, downloadClient, &noopSender{})
	if downloadErr != nil {
		return "", downloadErr
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

type EnsurePersistedOutcome struct {
	Config models.ModsJSON
	Lock   []models.ModInstall
	Result EnsureResult
}

func (coordinator *SetupCoordinator) EnsurePersisted(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, resolvedPlatform models.Platform, resolvedID string, remote platform.RemoteMod, options EnsurePersistOptions) (EnsurePersistedOutcome, error) {
	if strings.TrimSpace(string(resolvedPlatform)) == "" {
		return EnsurePersistedOutcome{}, errors.New("missing resolved platform")
	}
	if strings.TrimSpace(resolvedID) == "" {
		return EnsurePersistedOutcome{}, errors.New("missing resolved id")
	}

	configIndex := findConfigIndex(cfg, resolvedPlatform, resolvedID)
	lockIndex := findLockIndex(lock, resolvedPlatform, resolvedID)

	result := EnsureResult{}

	if configIndex < 0 {
		cfg.Mods = append(cfg.Mods, models.Mod{
			Type:                 resolvedPlatform,
			ID:                   resolvedID,
			Name:                 remote.Name,
			AllowVersionFallback: allowVersionFallbackPointer(options),
			Version:              optionalString(options.Version),
		})
		result.ConfigAdded = true
	}

	if lockIndex < 0 {
		installEntry, err := lockInstallFromRemote(remote, resolvedPlatform, resolvedID)
		if err != nil {
			return EnsurePersistedOutcome{}, err
		}

		lock = append(lock, installEntry)
		result.LockAdded = true
	}

	if result.ConfigAdded {
		if err := config.WriteConfig(ctx, coordinator.fs, meta, cfg); err != nil {
			return EnsurePersistedOutcome{}, err
		}
	}
	if result.LockAdded {
		if err := config.WriteLock(ctx, coordinator.fs, meta, lock); err != nil {
			return EnsurePersistedOutcome{}, err
		}
	}

	return EnsurePersistedOutcome{Config: cfg, Lock: lock, Result: result}, nil
}

type UpsertResult struct {
	ConfigAdded   bool
	ConfigUpdated bool
	LockAdded     bool
	LockUpdated   bool
}

type UpsertOutcome struct {
	Config models.ModsJSON
	Lock   []models.ModInstall
	Result UpsertResult
}

func (coordinator *SetupCoordinator) UpsertConfigAndLock(cfg models.ModsJSON, lock []models.ModInstall, resolvedPlatform models.Platform, resolvedID string, remote platform.RemoteMod, options EnsurePersistOptions) (UpsertOutcome, error) {
	if strings.TrimSpace(string(resolvedPlatform)) == "" {
		return UpsertOutcome{}, errors.New("missing resolved platform")
	}
	if strings.TrimSpace(resolvedID) == "" {
		return UpsertOutcome{}, errors.New("missing resolved id")
	}

	configIndex := findConfigIndex(cfg, resolvedPlatform, resolvedID)
	lockEntry, lockEntryFound := findLockEntry(lock, resolvedPlatform, resolvedID)

	result := UpsertResult{}

	if configIndex < 0 {
		cfg.Mods = append(cfg.Mods, models.Mod{
			Type:                 resolvedPlatform,
			ID:                   resolvedID,
			Name:                 remote.Name,
			AllowVersionFallback: allowVersionFallbackPointer(options),
			Version:              optionalString(options.Version),
		})
		result.ConfigAdded = true
	} else if strings.TrimSpace(remote.Name) != "" && cfg.Mods[configIndex].Name != remote.Name {
		cfg.Mods[configIndex].Name = remote.Name
		result.ConfigUpdated = true
	}

	installEntry, err := lockInstallFromRemote(remote, resolvedPlatform, resolvedID)
	if err != nil {
		return UpsertOutcome{}, err
	}

	if !lockEntryFound {
		lock = append(lock, installEntry)
		result.LockAdded = true
		return UpsertOutcome{Config: cfg, Lock: lock, Result: result}, nil
	}

	if lockEntryChanged(*lockEntry, installEntry) {
		*lockEntry = installEntry
		result.LockUpdated = true
	}

	return UpsertOutcome{Config: cfg, Lock: lock, Result: result}, nil
}

func ModExists(cfg models.ModsJSON, platform models.Platform, projectID string) bool {
	for _, mod := range cfg.Mods {
		if mod.ID == projectID && mod.Type == platform {
			return true
		}
	}
	return false
}

func allowVersionFallbackPointer(options EnsurePersistOptions) *bool {
	if !options.AllowVersionFallback {
		return nil
	}
	value := options.AllowVersionFallback
	return &value
}

func (coordinator *SetupCoordinator) ensureConfig(ctx context.Context, meta config.Metadata, options EnsureConfigOptions) (models.ModsJSON, error) {
	cfg, err := config.ReadConfig(ctx, coordinator.fs, meta)
	if err == nil {
		return cfg, nil
	}

	var notFound *config.ConfigFileNotFoundException
	if !errors.As(err, &notFound) {
		return models.ModsJSON{}, err
	}
	if options.Quiet {
		return models.ModsJSON{}, err
	}
	if coordinator.minecraftClient == nil {
		return models.ModsJSON{}, errors.New("missing modsetup dependencies: minecraftClient")
	}
	cfg, err = config.InitConfig(ctx, coordinator.fs, meta, coordinator.minecraftClient)
	if err != nil {
		return models.ModsJSON{}, err
	}
	return cfg, nil
}

func lockInstallFromRemote(remote platform.RemoteMod, resolvedPlatform models.Platform, resolvedID string) (models.ModInstall, error) {
	normalizedFileName, err := validateRemoteForLock(remote)
	if err != nil {
		return models.ModInstall{}, err
	}
	return models.ModInstall{
		Type:        resolvedPlatform,
		ID:          resolvedID,
		Name:        remote.Name,
		FileName:    normalizedFileName,
		ReleasedOn:  remote.ReleaseDate,
		Hash:        remote.Hash,
		DownloadURL: remote.DownloadURL,
	}, nil
}

func lockEntryChanged(current models.ModInstall, next models.ModInstall) bool {
	return current.Name != next.Name ||
		current.FileName != next.FileName ||
		current.ReleasedOn != next.ReleasedOn ||
		!strings.EqualFold(current.Hash, next.Hash) ||
		current.DownloadURL != next.DownloadURL
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func findConfigIndex(cfg models.ModsJSON, platform models.Platform, projectID string) int {
	for index := range cfg.Mods {
		if cfg.Mods[index].ID == projectID && cfg.Mods[index].Type == platform {
			return index
		}
	}
	return -1
}

func findLockIndex(lock []models.ModInstall, platform models.Platform, projectID string) int {
	for index := range lock {
		if lock[index].Type == platform && lock[index].ID == projectID {
			return index
		}
	}
	return -1
}

func findLockEntry(lock []models.ModInstall, platform models.Platform, projectID string) (*models.ModInstall, bool) {
	for index := range lock {
		if lock[index].Type == platform && lock[index].ID == projectID {
			return &lock[index], true
		}
	}
	return nil, false
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

func (sender *noopSender) Send(msg tea.Msg) { _ = msg }

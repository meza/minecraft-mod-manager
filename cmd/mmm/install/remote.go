package install

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func installMod(input installModInputs) (modInstallOutcome, error) {
	lockIndex := models.LockIndexForMod(input.mod, input.lock)
	if lockIndex >= 0 {
		return installFromLock(input.ctx, input.meta, input.cfg, input.mod, input.lock[lockIndex], input.deps, input.state)
	}
	return installFromRemote(input)
}

func installFromLock(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, mod models.Mod, installEntry models.ModInstall, deps installDeps, state *installExecutionState) (modInstallOutcome, error) {
	normalizedFileName, normalizeErr := modfilename.Normalize(installEntry.FileName)
	if normalizeErr != nil {
		//nolint:nilerr // surfaced as a failed outcome for this mod
		return modInstallOutcome{
			failed: true,
			failureReason: i18n.T("cmd.install.error.invalid_filename_lock", &i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"file": modfilename.Display(installEntry.FileName),
				},
			}),
		}, nil
	}
	installEntry.FileName = normalizedFileName
	var progressSender httpclient.Sender
	if state != nil {
		progressSender = installProgressSender{key: installModKey(mod), state: state}
	}
	result, err := ensureLockInstall(ctx, meta, cfg, installEntry, deps, progressSender)
	if err != nil {
		if reason, handled := installFailureReason(err, mod.Name); handled {
			return modInstallOutcome{failed: true, failureReason: reason}, nil
		}
		return modInstallOutcome{}, err
	}
	if result.Downloaded {
		return modInstallOutcome{}, nil
	}
	return modInstallOutcome{}, nil
}

func installFromRemote(input installModInputs) (modInstallOutcome, error) {
	remote, fetchErr := fetchRemoteModForInstall(input.ctx, input.mod, input.cfg, input.deps)
	if fetchErr != nil {
		outcome, handled := handleExpectedFetchError(fetchErr, input)
		if handled {
			return outcome, nil
		}
		return modInstallOutcome{}, fetchErr
	}

	normalizedRemote, outcome := normalizeRemoteForInstall(remote, input.mod)
	if outcome.failed {
		return outcome, nil
	}

	resolvedDestination, outcome, err := resolveRemoteDestination(input.meta, input.cfg, normalizedRemote, input.mod, input.deps)
	if err != nil || outcome.failed {
		return outcome, err
	}

	var progressSender httpclient.Sender
	if input.state != nil {
		progressSender = installProgressSender{key: installModKey(input.mod), state: input.state}
	}
	if err := downloadRemoteMod(input.ctx, normalizedRemote, resolvedDestination, input.deps, progressSender); err != nil {
		if reason, handled := installFailureReason(err, input.mod.Name); handled {
			return modInstallOutcome{failed: true, failureReason: reason}, nil
		}
		return modInstallOutcome{}, err
	}

	lockEntry := buildLockEntry(input.mod, normalizedRemote)
	return modInstallOutcome{newName: normalizedRemote.Name, lockEntry: &lockEntry}, nil
}

func fetchRemoteModForInstall(ctx context.Context, mod models.Mod, cfg models.ModsJSON, deps installDeps) (platform.RemoteMod, error) {
	return deps.fetchMod(ctx, mod.Type, mod.ID, platform.FetchOptions{
		AllowedReleaseTypes: models.EffectiveAllowedReleaseTypes(mod, cfg),
		GameVersion:         cfg.GameVersion,
		Loader:              cfg.Loader,
		AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
		FixedVersion:        optionalStringValue(mod.Version),
	}, deps.clients)
}

func downloadRemoteMod(ctx context.Context, remote platform.RemoteMod, resolvedDestination string, deps installDeps, sender httpclient.Sender) error {
	installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	if err := installer.DownloadAndVerify(ctx, remote.DownloadURL, resolvedDestination, remote.Hash, resolveInstallDownloadClient(deps), sender); err != nil {
		return err
	}
	return nil
}

func buildLockEntry(mod models.Mod, remote platform.RemoteMod) models.ModInstall {
	return models.ModInstall{
		Type:        mod.Type,
		ID:          mod.ID,
		Name:        remote.Name,
		FileName:    remote.FileName,
		ReleasedOn:  remote.ReleaseDate,
		Hash:        remote.Hash,
		DownloadURL: remote.DownloadURL,
	}
}

func normalizeRemoteForInstall(remote platform.RemoteMod, mod models.Mod) (platform.RemoteMod, modInstallOutcome) {
	normalizedFileName, normalizeErr := modfilename.Normalize(remote.FileName)
	if normalizeErr != nil {
		return platform.RemoteMod{}, modInstallOutcome{
			failed: true,
			failureReason: i18n.T("cmd.install.error.invalid_filename_remote", &i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"file": modfilename.Display(remote.FileName),
				},
			}),
		}
	}
	remote.FileName = normalizedFileName
	if strings.TrimSpace(remote.Hash) == "" {
		return platform.RemoteMod{}, modInstallOutcome{
			failed: true,
			failureReason: i18n.T("cmd.install.error.missing_hash_remote", &i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
				},
			}),
		}
	}
	return remote, modInstallOutcome{}
}

func resolveRemoteDestination(meta config.Metadata, cfg models.ModsJSON, remote platform.RemoteMod, mod models.Mod, deps installDeps) (string, modInstallOutcome, error) {
	destination := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)
	resolvedDestination, err := modpath.ResolveWritablePath(deps.fs, meta.ModsFolderPath(cfg), destination)
	if err != nil {
		if reason, handled := installFailureReason(err, mod.Name); handled {
			return "", modInstallOutcome{failed: true, failureReason: reason}, nil
		}
		return "", modInstallOutcome{}, err
	}
	return resolvedDestination, modInstallOutcome{}, nil
}

func ensureLockInstall(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, install models.ModInstall, deps installDeps, sender httpclient.Sender) (modinstall.EnsureResult, error) {
	installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	return installer.EnsureLockedFile(ctx, meta, cfg, install, resolveInstallDownloadClient(deps), sender)
}

func handleExpectedFetchError(err error, input installModInputs) (modInstallOutcome, bool) {
	var notFound *platform.ModNotFoundError
	if errors.As(err, &notFound) {
		return modInstallOutcome{
			failed: true,
			failureReason: i18n.T("cmd.install.error.mod_not_found", &i18n.Tvars{
				Data: &i18n.TData{
					"name":     input.mod.Name,
					"id":       input.mod.ID,
					"platform": string(input.mod.Type),
				},
			}),
		}, true
	}
	var noFile *platform.NoCompatibleFileError
	if errors.As(err, &noFile) {
		return modInstallOutcome{
			failed: true,
			failureReason: i18n.T("cmd.install.error.no_file", &i18n.Tvars{
				Data: &i18n.TData{
					"name":     input.mod.Name,
					"id":       input.mod.ID,
					"platform": string(input.mod.Type),
				},
			}),
		}, true
	}
	return modInstallOutcome{}, false
}

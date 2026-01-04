package install

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

func installMod(input installModInputs) (modInstallOutcome, error) {
	lockIndex := models.LockIndexForMod(input.mod, input.lock)
	if lockIndex >= 0 {
		return installFromLock(input.ctx, input.meta, input.cfg, input.mod, input.lock[lockIndex], input.deps)
	}
	return installFromRemote(input)
}

func installFromLock(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, mod models.Mod, installEntry models.ModInstall, deps installDeps) (modInstallOutcome, error) {
	normalizedFileName, normalizeErr := modfilename.Normalize(installEntry.FileName)
	if normalizeErr != nil {
		if err := deps.output.Error(i18n.T("cmd.install.error.invalid_filename_lock", &i18n.Tvars{
			Data: &i18n.TData{
				"name": mod.Name,
				"file": modfilename.Display(installEntry.FileName),
			},
		})); err != nil {
			return modInstallOutcome{}, err
		}
		return modInstallOutcome{failed: true}, nil
	}
	installEntry.FileName = normalizedFileName
	if err := ensureLockInstall(ctx, meta, cfg, mod, installEntry, deps); err != nil {
		return handleInstallIntegrityError(deps.output, err, mod.Name)
	}
	return modInstallOutcome{}, nil
}

func installFromRemote(input installModInputs) (modInstallOutcome, error) {
	remote, fetchErr := fetchRemoteModForInstall(input.ctx, input.mod, input.cfg, input.deps)
	if fetchErr != nil {
		handled, err := handleExpectedFetchError(fetchErr, input)
		if err != nil {
			return modInstallOutcome{}, err
		}
		if handled {
			return modInstallOutcome{}, nil
		}
		return modInstallOutcome{}, fetchErr
	}

	normalizedRemote, outcome, err := normalizeRemoteForInstall(remote, input.mod, input.deps)
	if err != nil {
		return modInstallOutcome{}, err
	}
	if outcome.failed {
		return outcome, nil
	}

	if outputErr := input.deps.output.Log(i18n.T("cmd.install.download.missing", &i18n.Tvars{
		Data: &i18n.TData{
			"name":     input.mod.Name,
			"platform": string(input.mod.Type),
		},
	}), output.LogForce); outputErr != nil {
		return modInstallOutcome{}, outputErr
	}

	resolvedDestination, outcome, err := resolveRemoteDestination(input.meta, input.cfg, normalizedRemote, input.mod, input.deps)
	if err != nil || outcome.failed {
		return outcome, err
	}

	if handled, err := downloadRemoteMod(input.ctx, normalizedRemote, resolvedDestination, input.mod, input.deps); err != nil {
		return modInstallOutcome{}, err
	} else if handled {
		return modInstallOutcome{failed: true}, nil
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

func downloadRemoteMod(ctx context.Context, remote platform.RemoteMod, resolvedDestination string, mod models.Mod, deps installDeps) (bool, error) {
	installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	if err := installer.DownloadAndVerify(ctx, remote.DownloadURL, resolvedDestination, remote.Hash, platform.PreferredDownloadClient(deps.clients), nil); err != nil {
		return handleDownloadIntegrityError(deps.output, err, mod.Name)
	}
	return false, nil
}

func handleInstallIntegrityError(out *output.Output, err error, modName string) (modInstallOutcome, error) {
	message, handled := integrityErrorMessage(err, modName)
	if !handled {
		return modInstallOutcome{}, err
	}
	if outputErr := out.Error(message); outputErr != nil {
		return modInstallOutcome{}, outputErr
	}
	return modInstallOutcome{failed: true}, nil
}

func handleDownloadIntegrityError(out *output.Output, err error, modName string) (bool, error) {
	message, handled := integrityErrorMessage(err, modName)
	if !handled {
		return false, err
	}
	if outputErr := out.Error(message); outputErr != nil {
		return false, outputErr
	}
	return true, nil
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

func normalizeRemoteForInstall(remote platform.RemoteMod, mod models.Mod, deps installDeps) (platform.RemoteMod, modInstallOutcome, error) {
	normalizedFileName, normalizeErr := modfilename.Normalize(remote.FileName)
	if normalizeErr != nil {
		if err := deps.output.Error(i18n.T("cmd.install.error.invalid_filename_remote", &i18n.Tvars{
			Data: &i18n.TData{
				"name": mod.Name,
				"file": modfilename.Display(remote.FileName),
			},
		})); err != nil {
			return platform.RemoteMod{}, modInstallOutcome{}, err
		}
		return platform.RemoteMod{}, modInstallOutcome{failed: true}, nil
	}
	remote.FileName = normalizedFileName
	if strings.TrimSpace(remote.Hash) == "" {
		if err := deps.output.Error(i18n.T("cmd.install.error.missing_hash_remote", &i18n.Tvars{
			Data: &i18n.TData{
				"name": mod.Name,
			},
		})); err != nil {
			return platform.RemoteMod{}, modInstallOutcome{}, err
		}
		return platform.RemoteMod{}, modInstallOutcome{failed: true}, nil
	}
	return remote, modInstallOutcome{}, nil
}

func resolveRemoteDestination(meta config.Metadata, cfg models.ModsJSON, remote platform.RemoteMod, mod models.Mod, deps installDeps) (string, modInstallOutcome, error) {
	destination := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)
	resolvedDestination, err := modpath.ResolveWritablePath(deps.fs, meta.ModsFolderPath(cfg), destination)
	if err != nil {
		outcome, outcomeErr := handleInstallIntegrityError(deps.output, err, mod.Name)
		if outcomeErr != nil {
			return "", modInstallOutcome{}, outcomeErr
		}
		return "", outcome, nil
	}
	return resolvedDestination, modInstallOutcome{}, nil
}

func ensureLockInstall(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, mod models.Mod, install models.ModInstall, deps installDeps) error {
	installer := modinstall.NewInstaller(deps.fs, modinstall.Downloader(deps.downloader))
	result, err := installer.EnsureLockedFile(ctx, meta, cfg, install, platform.PreferredDownloadClient(deps.clients), nil)
	if err != nil {
		return err
	}

	switch result.Reason {
	case modinstall.EnsureReasonMissing:
		if err := deps.output.Log(i18n.T("cmd.install.download.missing", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": string(install.Type),
			},
		}), output.LogForce); err != nil {
			return err
		}
	case modinstall.EnsureReasonHashMismatch:
		if err := deps.output.Log(i18n.T("cmd.install.download.hash_mismatch", &i18n.Tvars{
			Data: &i18n.TData{"name": mod.Name},
		}), output.LogForce); err != nil {
			return err
		}
	}
	return nil
}

func integrityErrorMessage(err error, modName string) (string, bool) {
	var missingHash modinstall.MissingHashError
	if errors.As(err, &missingHash) {
		return i18n.T("cmd.install.error.missing_hash_lock", &i18n.Tvars{
			Data: &i18n.TData{"name": modName},
		}), true
	}

	var hashMismatch modinstall.HashMismatchError
	if errors.As(err, &hashMismatch) {
		return i18n.T("cmd.install.error.hash_mismatch", &i18n.Tvars{
			Data: &i18n.TData{
				"name": modName,
			},
		}), true
	}

	var outsideRoot modpath.OutsideRootError
	if errors.As(err, &outsideRoot) {
		return i18n.T("cmd.install.error.symlink_outside_mods", &i18n.Tvars{
			Data: &i18n.TData{
				"name": modName,
				"path": outsideRoot.ResolvedPath,
				"root": outsideRoot.Root,
			},
		}), true
	}
	return "", false
}

func handleExpectedFetchError(err error, input installModInputs) (bool, error) {
	colorMode := tui.ColorDisabled
	if input.colorize {
		colorMode = tui.ColorEnabled
	}
	var notFound *platform.ModNotFoundError
	if errors.As(err, &notFound) {
		if outputErr := input.deps.output.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.install.error.mod_not_found", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     input.mod.Name,
				"id":       input.mod.ID,
				"platform": string(input.mod.Type),
			},
		})), output.LogForce); outputErr != nil {
			return false, outputErr
		}
		return true, nil
	}
	var noFile *platform.NoCompatibleFileError
	if errors.As(err, &noFile) {
		if outputErr := input.deps.output.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.install.error.no_file", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     input.mod.Name,
				"id":       input.mod.ID,
				"platform": string(input.mod.Type),
			},
		})), output.LogForce); outputErr != nil {
			return false, outputErr
		}
		return true, nil
	}
	return false, nil
}

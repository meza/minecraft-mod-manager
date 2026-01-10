package update

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/afero"
)

func processMod(
	ctx context.Context,
	meta config.Metadata,
	cfg models.ModsJSON,
	lock []models.ModInstall,
	candidate modUpdateCandidate,
	deps updateDeps,
	sender updateExecSender,
) modUpdateOutcome {
	mod := candidate.Mod

	outcome, lockIndex, shouldContinue := initializeUpdateOutcome(candidate, lock)
	if !shouldContinue {
		return outcome
	}

	remote, ok := fetchRemoteForUpdate(ctx, cfg, mod, deps, &outcome)
	if !ok {
		return outcome
	}

	remote, ok = normalizeRemoteFileNameForUpdate(remote, &outcome)
	if !ok {
		return outcome
	}

	outcome.NewName = remote.Name

	installed := lock[lockIndex]
	oldPath, ok := resolveInstalledUpdatePath(meta, cfg, installed, deps, &outcome)
	if !ok {
		return outcome
	}

	shouldUpdate, ok := shouldUpdateMod(installed, remote, &outcome)
	if !ok || !shouldUpdate {
		return outcome
	}

	sendUpdateDownloadStart(candidate, sender)

	newPath := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)

	if err := downloadAndSwap(ctx, deps, oldPath, newPath, meta.ModsFolderPath(cfg), remote.DownloadURL, remote.Hash, updateProgressSender{
		index:  candidate.ConfigIndex,
		sender: sender,
	}); err != nil {
		appendUpdateFailure(&outcome, err)
		return outcome
	}

	outcome.NewInstall = buildUpdatedInstall(mod, remote)
	outcome.Result = updateOutcomeUpdated
	return outcome
}

func initializeUpdateOutcome(candidate modUpdateCandidate, lock []models.ModInstall) (modUpdateOutcome, int, bool) {
	mod := candidate.Mod
	outcome := modUpdateOutcome{
		ConfigIndex: candidate.ConfigIndex,
		Result:      updateOutcomeFailed,
	}

	lockIndex := models.LockIndexForMod(mod, lock)
	if lockIndex < 0 {
		outcome.FailReason = i18n.T("cmd.update.error.missing_lock_entry", nil)
		return outcome, -1, false
	}
	outcome.LockIndex = lockIndex

	if isPinned(mod) {
		outcome.NewName = lock[lockIndex].Name
		outcome.Result = updateOutcomeSkipped
		return outcome, lockIndex, false
	}

	outcome.Result = updateOutcomeUpToDate
	return outcome, lockIndex, true
}

func fetchRemoteForUpdate(
	ctx context.Context,
	cfg models.ModsJSON,
	mod models.Mod,
	deps updateDeps,
	outcome *modUpdateOutcome,
) (platform.RemoteMod, bool) {
	remote, err := deps.fetchMod(ctx, mod.Type, mod.ID, platform.FetchOptions{
		AllowedReleaseTypes: models.EffectiveAllowedReleaseTypes(mod, cfg),
		GameVersion:         cfg.GameVersion,
		Loader:              cfg.Loader,
		AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
		FixedVersion:        "",
	}, deps.clients)
	if err != nil {
		outcome.FailReason = fetchErrorReason(err, mod, deps.logger)
		outcome.Result = updateOutcomeFailed
		return platform.RemoteMod{}, false
	}
	return remote, true
}

func normalizeRemoteFileNameForUpdate(remote platform.RemoteMod, outcome *modUpdateOutcome) (platform.RemoteMod, bool) {
	normalizedRemoteFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		outcome.FailReason = i18n.T("cmd.update.error.invalid_filename_remote", &i18n.Tvars{
			Data: &i18n.TData{
				"file": modfilename.Display(remote.FileName),
			},
		})
		outcome.Result = updateOutcomeFailed
		return platform.RemoteMod{}, false
	}
	remote.FileName = normalizedRemoteFileName
	return remote, true
}

func resolveInstalledUpdatePath(
	meta config.Metadata,
	cfg models.ModsJSON,
	installed models.ModInstall,
	deps updateDeps,
	outcome *modUpdateOutcome,
) (string, bool) {
	normalizedInstalledFileName, err := modfilename.Normalize(installed.FileName)
	if err != nil {
		outcome.FailReason = i18n.T("cmd.update.error.invalid_filename_lock", &i18n.Tvars{
			Data: &i18n.TData{
				"file": modfilename.Display(installed.FileName),
			},
		})
		outcome.Result = updateOutcomeFailed
		return "", false
	}

	oldPath := filepath.Join(meta.ModsFolderPath(cfg), normalizedInstalledFileName)
	exists, err := afero.Exists(deps.fs, oldPath)
	if err != nil {
		outcome.FailReason = err.Error()
		outcome.Result = updateOutcomeFailed
		return "", false
	}
	if !exists {
		outcome.FailReason = i18n.T("cmd.update.error.locked_file_missing", &i18n.Tvars{
			Data: &i18n.TData{
				"path": oldPath,
			},
		})
		outcome.Result = updateOutcomeFailed
		return "", false
	}

	return oldPath, true
}

func shouldUpdateMod(installed models.ModInstall, remote platform.RemoteMod, outcome *modUpdateOutcome) (shouldUpdate bool, shouldCheck bool) {
	installedDate, err := parseRFC3339(installed.ReleasedOn)
	if err != nil {
		outcome.FailReason = err.Error()
		outcome.Result = updateOutcomeFailed
		return false, false
	}
	remoteDate, err := parseRFC3339(remote.ReleaseDate)
	if err != nil {
		outcome.FailReason = err.Error()
		outcome.Result = updateOutcomeFailed
		return false, false
	}
	if !remoteDate.After(installedDate) {
		return false, true
	}
	if strings.TrimSpace(installed.Hash) == "" {
		outcome.FailReason = i18n.T("cmd.update.error.missing_hash_lock", nil)
		outcome.Result = updateOutcomeFailed
		return false, false
	}
	if strings.TrimSpace(remote.Hash) == "" {
		outcome.FailReason = i18n.T("cmd.update.error.missing_hash_remote", nil)
		outcome.Result = updateOutcomeFailed
		return false, false
	}
	if strings.EqualFold(strings.TrimSpace(remote.Hash), strings.TrimSpace(installed.Hash)) {
		return false, true
	}
	return true, true
}

func appendUpdateFailure(outcome *modUpdateOutcome, err error) {
	if message, handled := integrityErrorMessage(err); handled {
		outcome.FailReason = message
	} else {
		outcome.FailReason = err.Error()
	}
	outcome.Result = updateOutcomeFailed
}

func buildUpdatedInstall(mod models.Mod, remote platform.RemoteMod) models.ModInstall {
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

func expectedFetchErrorReason(err error, mod models.Mod) (string, bool) {
	var notFound *platform.ModNotFoundError
	if errors.As(err, &notFound) {
		return i18n.T("cmd.update.error.mod_not_found", &i18n.Tvars{
			Data: &i18n.TData{
				"platform": string(mod.Type),
			},
		}), true
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(err, &noFile) {
		return i18n.T("cmd.update.error.no_file", &i18n.Tvars{
			Data: &i18n.TData{
				"platform": string(mod.Type),
			},
		}), true
	}

	return "", false
}

func fetchErrorReason(fetchErr error, mod models.Mod, log *logger.Logger) string {
	if fetchErr == nil {
		return ""
	}

	if reason, handled := expectedFetchErrorReason(fetchErr, mod); handled {
		return reason
	}

	summary, _ := clierrors.SummarizePlatformError(fetchErr, mod.Type)
	reason := summary.Reason
	debugDetails := summary.DebugDetails
	if strings.TrimSpace(debugDetails) == "" {
		debugDetails = fetchErr.Error()
	}

	if log != nil && strings.TrimSpace(debugDetails) != "" {
		debugMessage := fmt.Sprintf("Debug: %s on %s failed: %s", mod.Name, mod.Type, debugDetails)
		if err := log.Debug(debugMessage); err != nil {
			// Debug logging must never block update outcomes.
			_ = err
		}
	}
	return i18n.T("cmd.update.error.platform", &i18n.Tvars{
		Data: &i18n.TData{
			"platform": string(mod.Type),
			"reason":   reason,
		},
	})
}

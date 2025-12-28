package update

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
)

func processMod(
	ctx context.Context,
	meta config.Metadata,
	cfg models.ModsJSON,
	lock []models.ModInstall,
	candidate modUpdateCandidate,
	deps updateDeps,
	colorMode tui.ColorMode,
) modUpdateOutcome {
	mod := candidate.Mod

	outcome, lockIndex, shouldContinue := initializeUpdateOutcome(candidate, lock)
	if !shouldContinue {
		return outcome
	}

	appendUpdateCheckEvent(&outcome, mod)

	remote, ok := fetchRemoteForUpdate(ctx, cfg, mod, deps, colorMode, &outcome)
	if !ok {
		return outcome
	}

	remote, ok = normalizeRemoteFileNameForUpdate(remote, mod, &outcome)
	if !ok {
		return outcome
	}

	outcome.NewName = remote.Name

	installed := lock[lockIndex]
	oldPath, ok := resolveInstalledUpdatePath(meta, cfg, mod, installed, deps, &outcome)
	if !ok {
		return outcome
	}

	shouldUpdate, ok := shouldUpdateMod(installed, remote, mod, &outcome)
	if !ok || !shouldUpdate {
		return outcome
	}

	appendUpdateAvailableEvent(&outcome, mod)

	newPath := filepath.Join(meta.ModsFolderPath(cfg), remote.FileName)

	if err := downloadAndSwap(ctx, deps, oldPath, newPath, meta.ModsFolderPath(cfg), remote.DownloadURL, remote.Hash); err != nil {
		appendUpdateFailure(&outcome, err, mod.Name)
		return outcome
	}

	outcome.NewInstall = buildUpdatedInstall(mod, remote)
	outcome.Updated = true
	return outcome
}

func initializeUpdateOutcome(candidate modUpdateCandidate, lock []models.ModInstall) (modUpdateOutcome, int, bool) {
	mod := candidate.Mod
	outcome := modUpdateOutcome{
		ConfigIndex: candidate.ConfigIndex,
	}

	lockIndex := models.LockIndexForMod(mod, lock)
	if lockIndex < 0 {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.missing_lock_entry", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"id":   mod.ID,
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return outcome, -1, false
	}
	outcome.LockIndex = lockIndex

	if isPinned(mod) {
		outcome.NewName = lock[lockIndex].Name
		return outcome, lockIndex, false
	}

	return outcome, lockIndex, true
}

func appendUpdateCheckEvent(outcome *modUpdateOutcome, mod models.Mod) {
	outcome.LogEvents = append(outcome.LogEvents, logEvent{
		Kind: logEventKindDebug,
		Message: i18n.T("cmd.update.debug.checking", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
			},
		}),
	})
}

func fetchRemoteForUpdate(
	ctx context.Context,
	cfg models.ModsJSON,
	mod models.Mod,
	deps updateDeps,
	colorMode tui.ColorMode,
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
		outcome.LogEvents = append(outcome.LogEvents, fetchErrorEvents(err, mod, colorMode)...)
		outcome.Error = errUpdateFailures
		return platform.RemoteMod{}, false
	}
	return remote, true
}

func normalizeRemoteFileNameForUpdate(remote platform.RemoteMod, mod models.Mod, outcome *modUpdateOutcome) (platform.RemoteMod, bool) {
	normalizedRemoteFileName, err := modfilename.Normalize(remote.FileName)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.invalid_filename_remote", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"file": modfilename.Display(remote.FileName),
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return platform.RemoteMod{}, false
	}
	remote.FileName = normalizedRemoteFileName
	return remote, true
}

func resolveInstalledUpdatePath(
	meta config.Metadata,
	cfg models.ModsJSON,
	mod models.Mod,
	installed models.ModInstall,
	deps updateDeps,
	outcome *modUpdateOutcome,
) (string, bool) {
	normalizedInstalledFileName, err := modfilename.Normalize(installed.FileName)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.invalid_filename_lock", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"file": modfilename.Display(installed.FileName),
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return "", false
	}

	oldPath := filepath.Join(meta.ModsFolderPath(cfg), normalizedInstalledFileName)
	exists, err := afero.Exists(deps.fs, oldPath)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
		outcome.Error = errUpdateFailures
		return "", false
	}
	if !exists {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.locked_file_missing", i18n.Tvars{
				Data: &i18n.TData{
					"name": mod.Name,
					"id":   mod.ID,
					"path": oldPath,
				},
			}),
		})
		outcome.Error = errUpdateFailures
		return "", false
	}

	return oldPath, true
}

func shouldUpdateMod(installed models.ModInstall, remote platform.RemoteMod, mod models.Mod, outcome *modUpdateOutcome) (shouldUpdate bool, shouldCheck bool) {
	installedDate, err := parseRFC3339(installed.ReleasedOn)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
		outcome.Error = errUpdateFailures
		return false, false
	}
	remoteDate, err := parseRFC3339(remote.ReleaseDate)
	if err != nil {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
		outcome.Error = errUpdateFailures
		return false, false
	}
	if !remoteDate.After(installedDate) {
		return false, true
	}
	if strings.TrimSpace(installed.Hash) == "" {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.missing_hash_lock", i18n.Tvars{
				Data: &i18n.TData{"name": mod.Name},
			}),
		})
		outcome.Error = errUpdateFailures
		return false, false
	}
	if strings.TrimSpace(remote.Hash) == "" {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{
			Kind: logEventKindError,
			Message: i18n.T("cmd.update.error.missing_hash_remote", i18n.Tvars{
				Data: &i18n.TData{"name": mod.Name},
			}),
		})
		outcome.Error = errUpdateFailures
		return false, false
	}
	if strings.EqualFold(strings.TrimSpace(remote.Hash), strings.TrimSpace(installed.Hash)) {
		return false, true
	}
	return true, true
}

func appendUpdateAvailableEvent(outcome *modUpdateOutcome, mod models.Mod) {
	outcome.LogEvents = append(outcome.LogEvents, logEvent{
		Kind:      logEventKindLog,
		ForceShow: true,
		Message: i18n.T("cmd.update.has_update", i18n.Tvars{
			Data: &i18n.TData{"name": mod.Name},
		}),
	})
}

func appendUpdateFailure(outcome *modUpdateOutcome, err error, modName string) {
	if message, handled := integrityErrorMessage(err, modName); handled {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: message})
	} else {
		outcome.LogEvents = append(outcome.LogEvents, logEvent{Kind: logEventKindError, Message: err.Error()})
	}
	outcome.Error = errUpdateFailures
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

func expectedFetchErrorEvent(err error, mod models.Mod, colorMode tui.ColorMode) (logEvent, bool) {
	var notFound *platform.ModNotFoundError
	if errors.As(err, &notFound) {
		return logEvent{
			Kind:      logEventKindLog,
			ForceShow: true,
			Message: messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.update.error.mod_not_found", i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": mod.Type,
				},
			})),
		}, true
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(err, &noFile) {
		return logEvent{
			Kind:      logEventKindLog,
			ForceShow: true,
			Message: messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.update.error.no_file", i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": mod.Type,
				},
			})),
		}, true
	}

	return logEvent{}, false
}

func fetchErrorEvents(fetchErr error, mod models.Mod, colorMode tui.ColorMode) []logEvent {
	if fetchErr == nil {
		return nil
	}

	if event, handled := expectedFetchErrorEvent(fetchErr, mod, colorMode); handled {
		return []logEvent{event}
	}

	summary, _ := clierrors.SummarizePlatformError(fetchErr, mod.Type)
	reason := summary.Reason
	debugDetails := summary.DebugDetails
	if strings.TrimSpace(debugDetails) == "" {
		debugDetails = fetchErr.Error()
	}

	events := []logEvent{{
		Kind: logEventKindError,
		Message: i18n.T("cmd.update.error.platform", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
				"reason":   reason,
			},
		}),
	}}

	if strings.TrimSpace(debugDetails) == "" {
		return events
	}

	events = append(events, logEvent{
		Kind: logEventKindDebug,
		Message: i18n.T("cmd.update.debug.platform_error", i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": mod.Type,
				"details":  debugDetails,
			},
		}),
	})
	return events
}

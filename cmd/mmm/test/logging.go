package test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type logEventKind int

const (
	logEventKindError logEventKind = iota
	logEventKindDebug
)

type logEvent struct {
	Kind      logEventKind
	Message   string
	ForceShow bool
}

func fetchFailureDetails(mod models.Mod, cfg models.ModsJSON, targetVersion string, opts platform.FetchOptions) string {
	fixedVersion := strings.TrimSpace(opts.FixedVersion)
	if fixedVersion == "" {
		fixedVersion = "none"
	}
	return fmt.Sprintf("id=%s version=%s loader=%s releases=%s fixedVersion=%s allowFallback=%t",
		mod.ID,
		targetVersion,
		cfg.Loader,
		formatReleaseTypes(opts.AllowedReleaseTypes),
		fixedVersion,
		opts.AllowFallback,
	)
}

func formatReleaseTypes(releaseTypes []models.ReleaseType) string {
	if len(releaseTypes) == 0 {
		return "none"
	}
	entries := make([]string, 0, len(releaseTypes))
	for _, releaseType := range releaseTypes {
		entries = append(entries, string(releaseType))
	}
	return strings.Join(entries, ",")
}

func fetchFailureUserEvent(fetchErr error, mod models.Mod) logEvent {
	var notFound *platform.ModNotFoundError
	if errors.As(fetchErr, &notFound) {
		return logEvent{
			Kind: logEventKindDebug,
			Message: i18n.T("cmd.test.error.mod_not_found", &i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": string(mod.Type),
				},
			}),
		}
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(fetchErr, &noFile) {
		return logEvent{
			Kind: logEventKindDebug,
			Message: i18n.T("cmd.test.error.no_file", &i18n.Tvars{
				Data: &i18n.TData{
					"name":     mod.Name,
					"id":       mod.ID,
					"platform": string(mod.Type),
				},
			}),
		}
	}

	summary, ok := clierrors.SummarizePlatformError(fetchErr, mod.Type)
	reason := i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(mod.Type)},
	})
	if ok && strings.TrimSpace(summary.Reason) != "" {
		reason = summary.Reason
	}

	return logEvent{
		Kind: logEventKindError,
		Message: i18n.T("cmd.test.error.platform", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": string(mod.Type),
				"reason":   reason,
			},
		}),
	}
}

func fetchFailureDetailEvent(fetchErr error, mod models.Mod) (logEvent, bool) {
	if fetchErr == nil {
		return logEvent{}, false
	}

	var notFound *platform.ModNotFoundError
	if errors.As(fetchErr, &notFound) {
		return logEvent{}, false
	}

	var noFile *platform.NoCompatibleFileError
	if errors.As(fetchErr, &noFile) {
		return logEvent{}, false
	}

	summary, _ := clierrors.SummarizePlatformError(fetchErr, mod.Type)

	details := summary.DebugDetails
	if strings.TrimSpace(details) == "" {
		details = fetchErr.Error()
	}
	if strings.TrimSpace(details) == "" {
		return logEvent{}, false
	}

	return logEvent{
		Kind: logEventKindError,
		Message: i18n.T("cmd.test.error.platform_details", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": string(mod.Type),
				"details":  details,
			},
		}),
	}, true
}

func fetchFailureDebugEvent(fetchErr error, mod models.Mod, cfg models.ModsJSON, targetVersion string, opts platform.FetchOptions) (logEvent, bool) {
	details := fetchFailureDetails(mod, cfg, targetVersion, opts)
	debugError := fetchErr.Error()
	if summary, ok := clierrors.SummarizePlatformError(fetchErr, mod.Type); ok && strings.TrimSpace(summary.DebugDetails) != "" {
		debugError = summary.DebugDetails
	}
	return logEvent{
		Kind: logEventKindDebug,
		Message: i18n.T("cmd.test.debug.platform_error", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": string(mod.Type),
				"error":    debugError,
				"details":  details,
			},
		}),
	}, true
}

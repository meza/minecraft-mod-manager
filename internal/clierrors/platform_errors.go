package clierrors

import (
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"net/http"
)

type PlatformErrorSummary struct {
	Reason       string
	DebugDetails string
}

func SummarizePlatformError(err error, platform models.Platform) (PlatformErrorSummary, bool) {
	if err == nil {
		return PlatformErrorSummary{}, false
	}

	if responseErr, ok := httpclient.ExtractResponseError(err); ok {
		return PlatformErrorSummary{
			Reason:       reasonForStatusCode(responseErr.StatusCode, platform),
			DebugDetails: responseErr.Error(),
		}, true
	}

	if httpclient.IsTimeoutError(err) {
		return PlatformErrorSummary{
			Reason: i18n.T("cmd.platform.error.reason.timeout", &i18n.Tvars{
				Data: &i18n.TData{"platform": platform},
			}),
			DebugDetails: err.Error(),
		}, true
	}

	if httpclient.IsConnectionError(err) {
		return PlatformErrorSummary{
			Reason: i18n.T("cmd.platform.error.reason.connection", &i18n.Tvars{
				Data: &i18n.TData{"platform": platform},
			}),
			DebugDetails: err.Error(),
		}, true
	}

	return PlatformErrorSummary{
		Reason: i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
			Data: &i18n.TData{"platform": platform},
		}),
		DebugDetails: err.Error(),
	}, true
}

func reasonForStatusCode(statusCode int, platform models.Platform) string {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return i18n.T("cmd.platform.error.reason.auth", &i18n.Tvars{
			Data: &i18n.TData{
				"token":    apiTokenName(platform),
				"platform": platform,
			},
		})
	case statusCode == http.StatusTooManyRequests:
		return i18n.T("cmd.platform.error.reason.rate_limited", &i18n.Tvars{
			Data: &i18n.TData{"platform": platform},
		})
	case statusCode == http.StatusNotFound:
		return i18n.T("cmd.platform.error.reason.not_found", &i18n.Tvars{
			Data: &i18n.TData{"platform": platform},
		})
	case statusCode >= http.StatusInternalServerError:
		return i18n.T("cmd.platform.error.reason.server_error", &i18n.Tvars{
			Data: &i18n.TData{"platform": platform},
		})
	case statusCode >= http.StatusBadRequest:
		return i18n.T("cmd.platform.error.reason.bad_request", &i18n.Tvars{
			Data: &i18n.TData{"platform": platform},
		})
	default:
		return i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
			Data: &i18n.TData{"platform": platform},
		})
	}
}

func apiTokenName(platform models.Platform) string {
	switch platform {
	case models.CURSEFORGE:
		return "CURSEFORGE_API_KEY"
	case models.MODRINTH:
		return "MODRINTH_API_KEY"
	default:
		return "API_KEY"
	}
}

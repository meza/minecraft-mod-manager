package scan

import (
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func summarizePlatformFailure(err error, platform models.Platform) clierrors.PlatformErrorSummary {
	if err == nil {
		return clierrors.PlatformErrorSummary{
			Reason: i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
				Data: &i18n.TData{"platform": string(platform)},
			}),
		}
	}

	summary, _ := clierrors.SummarizePlatformError(err, platform)
	return summary
}

func platformUnsureReason(platform models.Platform, reason string) string {
	return i18n.T("cmd.scan.unsure.platform_error", &i18n.Tvars{
		Data: &i18n.TData{
			"platform": string(platform),
			"reason":   reason,
		},
	})
}

func logPlatformDebug(log *logger.Logger, platform models.Platform, details string) error {
	if log == nil || strings.TrimSpace(details) == "" {
		return nil
	}
	if err := log.Debug(i18n.T("cmd.platform.debug.lookup_failed", &i18n.Tvars{
		Data: &i18n.TData{
			"platform": string(platform),
			"details":  details,
		},
	})); err != nil {
		return err
	}
	return nil
}

func allowPlatformFallback(err error) bool {
	if err == nil {
		return true
	}
	if httpclient.IsTimeoutError(err) {
		return false
	}
	if httpclient.IsConnectionError(err) {
		return false
	}
	return true
}

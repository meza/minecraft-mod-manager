package scan

import (
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func summarizePlatformFailure(err error, platform models.Platform) clierrors.PlatformErrorSummary {
	if err == nil {
		return clierrors.PlatformErrorSummary{
			Reason: i18n.T("cmd.platform.error.reason.unknown"),
		}
	}

	summary, _ := clierrors.SummarizePlatformError(err, platform)
	return summary
}

func platformUnsureReason(platform models.Platform, reason string) string {
	return i18n.T("cmd.scan.unsure.platform_error", i18n.Tvars{
		Data: &i18n.TData{
			"platform": platform,
			"reason":   reason,
		},
	})
}

func logPlatformDebug(log *logger.Logger, platform models.Platform, details string) {
	if log == nil || strings.TrimSpace(details) == "" {
		return
	}
	log.Debug(i18n.T("cmd.scan.debug.platform_error", i18n.Tvars{
		Data: &i18n.TData{
			"platform": platform,
			"details":  details,
		},
	}))
}

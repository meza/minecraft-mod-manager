package add

import (
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modinstall"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func recordAddTelemetry(telemetryPayload telemetry.CommandTelemetry, err error) {
	telemetryPayload.Success = err == nil
	if telemetryPayload.Success {
		telemetryPayload.Error = nil
		telemetryPayload.ExitCode = 0
	} else {
		telemetryPayload.ExitCode = 1
	}
	telemetry.RecordCommand(telemetryPayload)
}

func addFailureTelemetry(platformValue models.Platform, projectID string, opts addOptions, interactive bool, err error) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     false,
		Error:       err,
		ExitCode:    1,
		Interactive: interactive,
		Arguments:   addTelemetryArgs(platformValue, projectID, opts),
	}
}

func addFailureTelemetryWithoutArgs(interactive bool, err error) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     false,
		Error:       err,
		ExitCode:    1,
		Interactive: interactive,
	}
}

func addExistingInstallTelemetry(platformValue models.Platform, projectID string, opts addOptions, interactive bool, reason modinstall.EnsureReason) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     true,
		ExitCode:    0,
		Interactive: interactive,
		Arguments:   addTelemetryArgs(platformValue, projectID, opts),
		Extra: map[string]interface{}{
			"flag":               "already-exists",
			"ensure_file_reason": string(reason),
		},
	}
}

func addSuccessTelemetry(platformValue models.Platform, projectID string, opts addOptions, interactive bool) telemetry.CommandTelemetry {
	return telemetry.CommandTelemetry{
		Command:     "add",
		Success:     true,
		ExitCode:    0,
		Interactive: interactive,
		Arguments:   addTelemetryArgs(platformValue, projectID, opts),
	}
}

func addTelemetryArgs(platformValue models.Platform, projectID string, opts addOptions) map[string]interface{} {
	return map[string]interface{}{
		"platform": platformValue,
		"id":       projectID,
		"version":  opts.Version,
		"fallback": opts.AllowVersionFallback,
	}
}

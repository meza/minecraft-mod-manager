package update

import (
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func recordUpdateTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), updated int, failed int, mode interaction.ExecutionMode, err error) {
	payload := telemetry.CommandTelemetry{
		Command:       "update",
		Success:       err == nil,
		Error:         err,
		ExitCode:      0,
		Interactive:   mode.IsInteractive(),
		ExecutionMode: mode.String(),
		Extra: map[string]interface{}{
			"updatedMods": updated,
			"failedMods":  failed,
		},
	}
	if err != nil {
		payload.ExitCode = 1
	}
	telemetryRecorder(payload)
}

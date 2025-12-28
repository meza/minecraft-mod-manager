package install

import "github.com/meza/minecraft-mod-manager/internal/telemetry"

func recordInstallTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), result Result, err error) {
	payload := telemetry.CommandTelemetry{
		Command:     "install",
		Success:     err == nil,
		Error:       err,
		ExitCode:    0,
		Interactive: false,
		Extra: map[string]interface{}{
			"numberOfMods": result.InstalledCount,
		},
	}
	if err != nil {
		payload.ExitCode = 1
	}
	telemetryRecorder(payload)
}

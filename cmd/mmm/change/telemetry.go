package change

import (
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func recordChangeTelemetry(recorder func(telemetry.CommandTelemetry), opts changeOptions, result changeResult, err error) {
	payload := telemetry.CommandTelemetry{
		Command:     "change",
		Success:     err == nil && result.ExitCode == 0,
		Error:       err,
		ExitCode:    result.ExitCode,
		Interactive: result.Interactive,
		Arguments: map[string]interface{}{
			"force":       opts.Force,
			"gameVersion": opts.GameVersion,
			"unattended":  opts.Unattended,
		},
		Extra: map[string]interface{}{
			"targetVersion":  result.TargetVersion,
			"totalMods":      result.TotalMods,
			"skippedMods":    result.SkippedMods,
			"downloadedMods": result.DownloadedMods,
		},
	}

	recorder(payload)
}

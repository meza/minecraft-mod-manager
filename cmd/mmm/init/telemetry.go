package init

import (
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/privacy"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func releaseTypesToStrings(releaseTypes []models.ReleaseType) []string {
	out := make([]string, 0, len(releaseTypes))
	for _, releaseType := range releaseTypes {
		out = append(out, string(releaseType))
	}
	return out
}

func buildTelemetryPayload(options initOptions, didUseTUI bool, err error) telemetry.CommandTelemetry {
	telemetryError := redactModsFolderErrorForTelemetry(err)
	payload := telemetry.CommandTelemetry{
		Command:     "init",
		Success:     telemetryError == nil,
		Error:       telemetryError,
		ExitCode:    0,
		Interactive: didUseTUI,
		Arguments: map[string]interface{}{
			"loader":       options.Loader,
			"gameVersion":  options.GameVersion,
			"releaseTypes": releaseTypesToStrings(options.ReleaseTypes),
			"modsFolder":   privacy.RedactPathUsernames(options.ModsFolder),
		},
	}
	if err != nil {
		payload.ExitCode = 1
	}
	return payload
}

type telemetryRedactedError struct {
	original error
	message  string
}

func (err telemetryRedactedError) Error() string {
	return err.message
}

func (err telemetryRedactedError) Unwrap() error {
	return err.original
}

func redactModsFolderErrorForTelemetry(err error) error {
	if err == nil {
		return nil
	}

	message := err.Error()
	redactedMessage := privacy.RedactPathUsernames(message)
	if redactedMessage == message {
		return err
	}

	return telemetryRedactedError{original: err, message: redactedMessage}
}

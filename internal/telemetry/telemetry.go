package telemetry

import (
	"github.com/denisbrodbeck/machineid"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/posthog/posthog-go"
	"io"
	"os"
)

type Client interface {
	io.Closer
	Enqueue(posthog.Message) error
}

var singleClient Client
var machineId string

type CommandTelemetry struct {
	Command string                 `json:"command"`
	Success bool                   `json:"success"`
	Config  *models.ModsJson       `json:"config,omitempty"`
	Error   error                  `json:"error,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

func getMachineId() string {
	envMachineId, hasEnvId := os.LookupEnv("MACHINE_ID")

	if hasEnvId {
		return envMachineId
	}

	machineId, _ = machineid.ID()
	return machineId
}

func initClient() Client {
	if singleClient != nil {
		return singleClient
	}
	machineId = getMachineId()

	pc, _ := posthog.NewWithConfig(
		environment.PosthogApiKey(),
		posthog.Config{
			Endpoint: "https://eu.i.posthog.com",
		},
	)
	singleClient = pc
	return singleClient
}

func Capture(event string, properties map[string]interface{}) {
	client := initClient()
	_ = client.Enqueue(posthog.Capture{
		Event:      event,
		DistinctId: machineId,
		Properties: properties,
	})
	_ = client.Close()
}

func CaptureCommand(command CommandTelemetry) {
	properties := map[string]interface{}{
		"type":    "command",
		"success": command.Success,
	}

	if command.Success == false {
		properties["config"] = command.Config
	}

	if command.Error != nil {
		properties["error"] = command.Error.Error()
	}

	if command.Extra != nil {
		properties["extra"] = command.Extra
	}

	Capture(command.Command, properties)
}

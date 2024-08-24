package telemetry

import (
	"github.com/denisbrodbeck/machineid"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/posthog/posthog-go"
)

var singleClient *Client
var machineId string

type CommandTelemetry struct {
	Command string                 `json:"command"`
	Success bool                   `json:"success"`
	Config  *models.ModsJson       `json:"config,omitempty"`
	Error   error                  `json:"error,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

type Client struct {
	PosthogClient posthog.Client
}

func (c *Client) Close() {
	_ = c.PosthogClient.Close()
}

func initClient() *Client {
	if singleClient != nil {
		return singleClient
	}
	machineId, _ = machineid.ID()

	pc, _ := posthog.NewWithConfig(
		environment.PosthogApiKey(),
		posthog.Config{
			Endpoint: "https://eu.i.posthog.com",
		},
	)
	singleClient = &Client{
		PosthogClient: pc,
	}
	return singleClient
}

func Capture(event string, properties map[string]interface{}) {
	client := initClient()
	_ = client.PosthogClient.Enqueue(posthog.Capture{
		Event:      event,
		DistinctId: machineId,
		Properties: properties,
	})
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

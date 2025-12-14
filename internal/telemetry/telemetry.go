package telemetry

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/posthog/posthog-go"
)

const (
	disableEnvVar      = "MMM_DISABLE_TELEMETRY"
	machineIDEnvVar    = "MACHINE_ID"
	unknownMachineID   = "unknown-machine"
	defaultPosthogHost = "https://eu.i.posthog.com"
)

var (
	defaultFlushTimeout = 2 * time.Second

	machineIDProvider        = machineid.ID
	clientBuilder            = defaultClientFactory
	baseLogger        Logger = noopLogger{}
	baseFlushTimeout         = defaultFlushTimeout
)

// Logger is the minimal logging interface consumed by the telemetry package.
type Logger interface {
	Debugf(format string, args ...interface{})
}

// MachineIDFetcher resolves an identifier for the current host.
type MachineIDFetcher func() (string, error)

// ClientFactory creates a PostHog client for the supplied API key and endpoint.
type ClientFactory func(apiKey, endpoint string) (Client, error)

// Client mirrors the subset of the posthog-go client used by the tracker.
type Client interface {
	io.Closer
	Enqueue(posthog.Message) error
}

type telemetryState struct {
	initOnce     sync.Once
	shutdownOnce sync.Once
	mu           sync.RWMutex
	client       Client
	machineID    string
	logger       Logger
	flushTimeout time.Duration
	enabled      bool
}

var state = newTelemetryState()

func newTelemetryState() *telemetryState {
	return &telemetryState{
		logger:       noopLogger{},
		flushTimeout: defaultFlushTimeout,
	}
}

// Reset clears the telemetry state. This should only be used from tests.
func Reset() {
	state = newTelemetryState()
	machineIDProvider = machineid.ID
	clientBuilder = defaultClientFactory
	baseLogger = noopLogger{}
	baseFlushTimeout = defaultFlushTimeout
}

// Init configures the telemetry client. It is safe to call multiple times but
// only the first invocation performs work.
func Init() {
	state.initOnce.Do(func() {
		logger := baseLogger
		if logger == nil {
			logger = noopLogger{}
		}

		timeout := baseFlushTimeout
		if timeout <= 0 {
			timeout = defaultFlushTimeout
		}

		if disabledByEnv() {
			state.disable(logger, timeout)
			return
		}

		apiKey := environment.PosthogApiKey()
		if apiKey == "" {
			state.disable(logger, timeout)
			return
		}

		client, err := clientBuilder(apiKey, defaultPosthogHost)
		if err != nil {
			logger.Debugf("telemetry: failed to create client: %v", err)
			state.disable(logger, timeout)
			return
		}

		state.mu.Lock()
		state.client = client
		state.machineID = resolveMachineID(machineIDProvider)
		state.logger = logger
		state.flushTimeout = timeout
		state.enabled = true
		state.mu.Unlock()
	})
}

// Capture sends an arbitrary event to PostHog.
func Capture(event string, properties map[string]interface{}) {
	if event == "" {
		return
	}

	snapshot := state.snapshot()
	if !snapshot.enabled || snapshot.client == nil {
		return
	}

	payload := mergeProperties(properties)
	payload["version"] = environment.AppVersion()

	err := snapshot.client.Enqueue(posthog.Capture{
		Event:      event,
		DistinctId: snapshot.machineID,
		Properties: payload,
	})
	if err != nil {
		snapshot.logger.Debugf("telemetry: failed to enqueue %q: %v", event, err)
	}
}

// CommandTelemetry captures high-level command execution metadata.
type CommandTelemetry struct {
	Command   string                 `json:"command"`
	Success   bool                   `json:"success"`
	Config    *models.ModsJson       `json:"config,omitempty"`
	Error     error                  `json:"error,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

// CaptureCommand emits a structured command telemetry event.
func CaptureCommand(command CommandTelemetry) {
	if command.Command == "" {
		return
	}

	properties := map[string]interface{}{
		"type":    "command",
		"success": command.Success,
	}

	if !command.Success && command.Config != nil {
		properties["config"] = command.Config
	}

	if command.Error != nil {
		properties["error"] = command.Error.Error()
	}

	if len(command.Extra) > 0 {
		properties["extra"] = command.Extra
	}

	if len(command.Arguments) > 0 {
		properties["arguments"] = command.Arguments
	}

	if command.Duration > 0 {
		properties["duration_ms"] = command.Duration.Milliseconds()
	}

	Capture(command.Command, properties)
}

// Shutdown flushes and closes the telemetry client. Additional calls are ignored.
func Shutdown(ctx context.Context) {
	state.shutdownOnce.Do(func() {
		snapshot := state.shutdownSnapshot()
		if !snapshot.enabled || snapshot.client == nil {
			return
		}

		if ctx == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), snapshot.flushTimeout)
			defer cancel()
		}

		done := make(chan struct{})
		go func() {
			err := snapshot.client.Close()
			if err != nil {
				snapshot.logger.Debugf("telemetry: shutdown failed: %v", err)
			}
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			snapshot.logger.Debugf("telemetry: shutdown timed out: %v", ctx.Err())
		}
	})
}

func (s *telemetryState) disable(logger Logger, timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = logger
	s.flushTimeout = timeout
	s.enabled = false
	s.client = nil
}

func (s *telemetryState) snapshot() telemetrySnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return telemetrySnapshot{
		client:       s.client,
		machineID:    s.machineID,
		logger:       s.logger,
		flushTimeout: s.flushTimeout,
		enabled:      s.enabled,
	}
}

func (s *telemetryState) shutdownSnapshot() telemetrySnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := telemetrySnapshot{
		client:       s.client,
		machineID:    s.machineID,
		logger:       s.logger,
		flushTimeout: s.flushTimeout,
		enabled:      s.enabled,
	}

	s.client = nil
	s.enabled = false

	return snap
}

type telemetrySnapshot struct {
	client       Client
	machineID    string
	logger       Logger
	flushTimeout time.Duration
	enabled      bool
}

func mergeProperties(properties map[string]interface{}) map[string]interface{} {
	if properties == nil {
		return map[string]interface{}{}
	}

	cloned := make(map[string]interface{}, len(properties))
	for k, v := range properties {
		cloned[k] = v
	}
	return cloned
}

func resolveMachineID(fetcher MachineIDFetcher) string {
	if envID, ok := os.LookupEnv(machineIDEnvVar); ok && envID != "" {
		return envID
	}

	if fetcher != nil {
		if id, err := fetcher(); err == nil && id != "" {
			return id
		}
	}

	return unknownMachineID
}

func disabledByEnv() bool {
	raw, ok := os.LookupEnv(disableEnvVar)
	if !ok {
		return false
	}

	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "1", "true", "yes", "on":
		return true
	}

	return false
}

func defaultClientFactory(apiKey, endpoint string) (Client, error) {
	return posthog.NewWithConfig(apiKey, posthog.Config{Endpoint: endpoint})
}

// noopLogger is used when no logger was provided.
type noopLogger struct{}

func (noopLogger) Debugf(string, ...interface{}) {}

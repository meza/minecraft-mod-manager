// Package telemetry handles usage telemetry and perf export.
package telemetry

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
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

	sessionNameHint string
	perfBaseDir     string
	commands        []recordedCommand
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

		apiKey := environment.PosthogAPIKey()
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

// SetSessionNameHint configures the session-level event name used during
// Shutdown when multiple commands were executed (tui) or when no command has
// been recorded yet.
func SetSessionNameHint(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	state.mu.Lock()
	state.sessionNameHint = name
	state.mu.Unlock()
}

// SetPerfBaseDir configures the base directory used when exporting perf spans
// for telemetry attachments, matching the mmm-perf.json normalization rules.
func SetPerfBaseDir(baseDir string) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return
	}
	state.mu.Lock()
	state.perfBaseDir = baseDir
	state.mu.Unlock()
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
	Command     string                 `json:"command"`
	Success     bool                   `json:"success"`
	Config      *models.ModsJSON       `json:"config,omitempty"`
	Error       error                  `json:"error,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	ExitCode    int                    `json:"exit_code,omitempty"`
	Interactive bool                   `json:"interactive,omitempty"`
}

type recordedCommand struct {
	Name        string
	Success     bool
	ExitCode    int
	Interactive bool

	ErrorCategory string
	ErrorMessage  string

	Extra     map[string]interface{}
	Arguments map[string]interface{}
}

// RecordCommand stores structured command telemetry to be sent once per session
// during Shutdown. If telemetry is disabled, this is a no-op.
func RecordCommand(command CommandTelemetry) {
	if command.Command == "" {
		return
	}

	snapshot := state.snapshot()
	if !snapshot.enabled || snapshot.client == nil {
		return
	}

	record := recordedCommand{
		Name:        command.Command,
		Success:     command.Success,
		ExitCode:    commandExitCode(command),
		Interactive: command.Interactive,
	}

	if command.Error != nil {
		record.ErrorCategory = errorCategory(command.Error)
		record.ErrorMessage = command.Error.Error()
	}

	if len(command.Extra) > 0 {
		record.Extra = command.Extra
	}

	if len(command.Arguments) > 0 {
		record.Arguments = command.Arguments
	}

	state.mu.Lock()
	state.commands = append(state.commands, record)
	state.mu.Unlock()
}

func commandDurationFromPerf(commandName string, performance []*perf.ExportSpan) (time.Duration, bool) {
	if commandName == "" || len(performance) == 0 {
		return 0, false
	}

	targetSpanName := "app.command." + commandName

	var best *perf.ExportSpan
	var bestDurationNS int64

	var walk func(span *perf.ExportSpan)
	walk = func(span *perf.ExportSpan) {
		if span == nil {
			return
		}

		if span.Name == targetSpanName {
			if best == nil || span.EndTime.After(best.EndTime) {
				best = span
				bestDurationNS = span.DurationNS
			}
		}

		for _, child := range span.Children {
			walk(child)
		}
	}

	for _, root := range performance {
		walk(root)
	}

	if best == nil {
		return 0, false
	}

	return time.Duration(bestDurationNS), true
}

func commandNameFromPerfSpan(spanName string) (string, bool) {
	const prefix = "app.command."
	if !strings.HasPrefix(spanName, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(spanName, prefix)
	if name == "" || strings.Contains(name, ".") {
		return "", false
	}
	return name, true
}

func topCommandNameFromPerformance(performance []*perf.ExportSpan) (string, bool) {
	if len(performance) == 0 {
		return "", false
	}

	var (
		bestName      string
		bestDepth     = int(^uint(0) >> 1)
		bestStartTime time.Time
	)

	var walk func(span *perf.ExportSpan, depth int)
	walk = func(span *perf.ExportSpan, depth int) {
		if span == nil {
			return
		}

		if name, ok := commandNameFromPerfSpan(span.Name); ok {
			switch {
			case bestName == "":
				bestName = name
				bestDepth = depth
				bestStartTime = span.StartTime
			case depth < bestDepth:
				bestName = name
				bestDepth = depth
				bestStartTime = span.StartTime
			case depth == bestDepth && span.StartTime.Before(bestStartTime):
				bestName = name
				bestDepth = depth
				bestStartTime = span.StartTime
			}
		}

		for _, child := range span.Children {
			walk(child, depth+1)
		}
	}

	for _, root := range performance {
		walk(root, 0)
	}

	if bestName == "" {
		return "", false
	}

	return bestName, true
}

// Shutdown flushes and closes the telemetry client. Additional calls are ignored.
func Shutdown(ctx context.Context) {
	state.shutdownOnce.Do(func() {
		snapshot := state.snapshot()
		if !snapshot.enabled || snapshot.client == nil {
			return
		}

		state.mu.RLock()
		commands := append([]recordedCommand(nil), state.commands...)
		sessionNameHint := state.sessionNameHint
		perfBaseDir := state.perfBaseDir
		state.mu.RUnlock()

		performance := []*perf.ExportSpan{}
		if spans, err := perf.GetExportTree(perfBaseDir); err == nil && spans != nil {
			performance = spans
		}

		canonicalCommand, _ := topCommandNameFromPerformance(performance)
		if canonicalCommand != "" && len(commands) == 1 {
			if strings.TrimSpace(commands[0].Name) == "" || commands[0].Name != canonicalCommand {
				commands[0].Name = canonicalCommand
			}
		}

		properties := map[string]interface{}{
			"type":        "session",
			"performance": performance,
			"commands":    buildCommandSummaries(commands, performance),
		}

		if d, err := perf.GetSessionDurations(); err == nil {
			properties["total_time_ms"] = d.Total.Milliseconds()
			properties["work_time_ms"] = d.Work.Milliseconds()
		}

		sessionName := resolveSessionName(sessionNameHint, canonicalCommand, commands)
		captureWithSnapshot(snapshot, sessionName, properties)

		var cancel context.CancelFunc
		if ctx == nil {
			ctx = context.Background()
		}
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			ctx, cancel = context.WithTimeout(ctx, snapshot.flushTimeout)
		}
		if cancel != nil {
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

		_ = state.shutdownSnapshot()
	})
}

func captureWithSnapshot(snapshot telemetrySnapshot, event string, properties map[string]interface{}) {
	if event == "" || !snapshot.enabled || snapshot.client == nil {
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

func commandExitCode(command CommandTelemetry) int {
	if command.ExitCode != 0 {
		return command.ExitCode
	}
	if command.Success {
		return 0
	}
	return 1
}

func resolveSessionName(sessionNameHint string, canonicalCommand string, commands []recordedCommand) string {
	if len(commands) > 1 {
		return "tui"
	}
	if len(commands) == 1 {
		name := strings.TrimSpace(commands[0].Name)
		if name != "" {
			return name
		}
	}
	if canonicalCommand != "" {
		return canonicalCommand
	}
	if sessionNameHint != "" {
		return sessionNameHint
	}
	return "unknown"
}

func buildCommandSummaries(commands []recordedCommand, performance []*perf.ExportSpan) []map[string]interface{} {
	if len(commands) == 0 {
		return []map[string]interface{}{}
	}

	out := make([]map[string]interface{}, 0, len(commands))
	for _, cmd := range commands {
		summary := map[string]interface{}{
			"name":        cmd.Name,
			"success":     cmd.Success,
			"exit_code":   cmd.ExitCode,
			"interactive": cmd.Interactive,
		}

		if cmd.ErrorCategory != "" {
			summary["error_category"] = cmd.ErrorCategory
		}
		if cmd.ErrorMessage != "" {
			summary["error"] = cmd.ErrorMessage
		}

		if len(cmd.Extra) > 0 {
			summary["extra"] = cmd.Extra
		}
		if len(cmd.Arguments) > 0 {
			summary["arguments"] = cmd.Arguments
		}

		duration, ok := commandDurationFromPerf(cmd.Name, performance)
		if ok {
			summary["duration_ms"] = duration.Milliseconds()
		}

		out = append(out, summary)
	}

	return out
}

func errorCategory(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	var notFound *globalerrors.ProjectNotFoundError
	if errors.As(err, &notFound) {
		return "project_not_found"
	}

	var apiErr *globalerrors.ProjectAPIError
	if errors.As(err, &apiErr) {
		return "project_api_error"
	}

	return "unknown"
}

func (state *telemetryState) disable(logger Logger, timeout time.Duration) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.logger = logger
	state.flushTimeout = timeout
	state.enabled = false
	state.client = nil
}

func (state *telemetryState) snapshot() telemetrySnapshot {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return telemetrySnapshot{
		client:       state.client,
		machineID:    state.machineID,
		logger:       state.logger,
		flushTimeout: state.flushTimeout,
		enabled:      state.enabled,
	}
}

func (state *telemetryState) shutdownSnapshot() telemetrySnapshot {
	state.mu.Lock()
	defer state.mu.Unlock()

	snap := telemetrySnapshot{
		client:       state.client,
		machineID:    state.machineID,
		logger:       state.logger,
		flushTimeout: state.flushTimeout,
		enabled:      state.enabled,
	}

	state.client = nil
	state.enabled = false

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

func (noopLogger) Debugf(format string, args ...interface{}) {
	_ = format
	_ = args
}

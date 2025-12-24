package telemetry

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/posthog/posthog-go"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

type stubClient struct {
	enqueued   []posthog.Message
	enqueueErr error
	closeErr   error
	closeDelay time.Duration
	closeCount int
}

func (client *stubClient) Enqueue(msg posthog.Message) error {
	client.enqueued = append(client.enqueued, msg)
	return client.enqueueErr
}

func (client *stubClient) Close() error {
	client.closeCount++
	if client.closeDelay > 0 {
		time.Sleep(client.closeDelay)
	}
	return client.closeErr
}

type recordingLogger struct {
	messages []string
}

func (logger *recordingLogger) Debugf(format string, args ...interface{}) {
	logger.messages = append(logger.messages, fmt.Sprintf(format, args...))
}

func resetTelemetryState(tb testing.TB) {
	tb.Helper()
	Reset()
	tb.Cleanup(func() {
		Reset()
	})
}

func initWithClient(t *testing.T, client Client, machineID string) {
	t.Helper()
	t.Setenv(disableEnvVar, "")
	t.Setenv("POSTHOG_API_KEY", "test-key")
	machineIDProvider = func() (string, error) {
		return machineID, nil
	}
	clientBuilder = func(apiKey, endpoint string) (Client, error) {
		return client, nil
	}
	Init()
}

func TestCaptureWithoutInitIsNoop(t *testing.T) {
	resetTelemetryState(t)
	assert.NotPanics(t, func() {
		Capture("noop", nil)
	})
}

func TestCaptureSkipsEmptyEvent(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "machine")

	Capture("", map[string]interface{}{"foo": "bar"})
	assert.Empty(t, client.enqueued)
}

func TestInitAndCaptureSendsEvent(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "machine-test")

	Capture("test-event", map[string]interface{}{"foo": "bar"})

	if assert.Len(t, client.enqueued, 1) {
		capture, ok := client.enqueued[0].(posthog.Capture)
		assert.True(t, ok)
		assert.Equal(t, "machine-test", capture.DistinctId)
		assert.Equal(t, "bar", capture.Properties["foo"])
		assert.Equal(t, environment.AppVersion(), capture.Properties["version"])
	}
}

func TestRecordCommandDoesNotEnqueueImmediately(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	err := errors.New("boom")
	config := &models.ModsJSON{}
	RecordCommand(CommandTelemetry{
		Command:   "list",
		Success:   false,
		Config:    config,
		Error:     err,
		Extra:     map[string]interface{}{"total": 2},
		Arguments: map[string]interface{}{"quiet": true},
		Duration:  150 * time.Millisecond,
		ExitCode:  1,
	})

	assert.Empty(t, client.enqueued)
}

func TestShutdownEmitsSingleSessionEvent(t *testing.T) {
	resetTelemetryState(t)

	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	rootCtx, rootSpan := perf.StartSpan(context.Background(), "app.lifecycle")
	time.Sleep(5 * time.Millisecond)
	_, cmdSpan := perf.StartSpan(rootCtx, "app.command.list")
	time.Sleep(5 * time.Millisecond)
	cmdSpan.End()
	_, waitSpan := perf.StartSpan(rootCtx, "tui.list.wait.view")
	time.Sleep(10 * time.Millisecond)
	waitSpan.End()
	rootSpan.End()

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	SetSessionNameHint("list")
	RecordCommand(CommandTelemetry{
		Command:     "list",
		Success:     true,
		ExitCode:    0,
		Interactive: true,
		Extra:       map[string]interface{}{"numberOfMods": 2},
	})

	assert.Empty(t, client.enqueued)
	Shutdown(context.Background())

	if assert.Len(t, client.enqueued, 1) {
		capture := client.enqueued[0].(posthog.Capture)
		props := capture.Properties
		assert.Equal(t, "session", props["type"])
		assert.NotEmpty(t, props["performance"])

		commands := props["commands"].([]map[string]interface{})
		if assert.Len(t, commands, 1) {
			assert.Equal(t, "list", commands[0]["name"])
			assert.Equal(t, true, commands[0]["success"])
			assert.Equal(t, 0, commands[0]["exit_code"])
			assert.Equal(t, true, commands[0]["interactive"])
			assert.Equal(t, map[string]interface{}{"numberOfMods": 2}, commands[0]["extra"])
		}

		total, hasTotal := props["total_time_ms"].(int64)
		work, hasWork := props["work_time_ms"].(int64)
		assert.True(t, hasTotal)
		assert.True(t, hasWork)
		assert.Greater(t, total, int64(0))
		assert.GreaterOrEqual(t, work, int64(0))
		assert.LessOrEqual(t, work, total)
	}
}

func TestShutdown_UsesPerfSpanCommandNameWhenNoCommandRecorded(t *testing.T) {
	resetTelemetryState(t)

	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	ctx, root := perf.StartSpan(context.Background(), "app.lifecycle")
	_, cmdSpan := perf.StartSpan(ctx, "app.command.install")
	cmdSpan.End()
	root.End()

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	SetSessionNameHint("i")
	Shutdown(context.Background())

	if assert.Len(t, client.enqueued, 1) {
		capture := client.enqueued[0].(posthog.Capture)
		assert.Equal(t, "install", capture.Event)

		props := capture.Properties
		commands := props["commands"].([]map[string]interface{})
		assert.Empty(t, commands)
	}
}

func TestShutdown_CanonicalizesSingleRecordedCommandNameFromPerf(t *testing.T) {
	resetTelemetryState(t)

	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	ctx, root := perf.StartSpan(context.Background(), "app.lifecycle")
	_, cmdSpan := perf.StartSpan(ctx, "app.command.install")
	cmdSpan.End()
	root.End()

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	RecordCommand(CommandTelemetry{Command: "i", Success: true})
	Shutdown(context.Background())

	if assert.Len(t, client.enqueued, 1) {
		capture := client.enqueued[0].(posthog.Capture)
		assert.Equal(t, "install", capture.Event)

		props := capture.Properties
		commands := props["commands"].([]map[string]interface{})
		if assert.Len(t, commands, 1) {
			assert.Equal(t, "install", commands[0]["name"])
		}
	}
}

func TestShutdownUsesTUIEventNameWhenMultipleCommandsRecorded(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	SetSessionNameHint("tui")
	RecordCommand(CommandTelemetry{Command: "add", Success: true})
	RecordCommand(CommandTelemetry{Command: "list", Success: true})
	Shutdown(context.Background())

	if assert.Len(t, client.enqueued, 1) {
		capture := client.enqueued[0].(posthog.Capture)
		assert.Equal(t, "tui", capture.Event)
	}
}

func TestSetSessionNameHint_IgnoresEmpty(t *testing.T) {
	resetTelemetryState(t)
	assert.NotPanics(t, func() {
		SetSessionNameHint(" ")
	})
}

func TestSetPerfBaseDir_IgnoresEmpty(t *testing.T) {
	resetTelemetryState(t)
	assert.NotPanics(t, func() {
		SetPerfBaseDir(" ")
	})
}

func TestSetPerfBaseDir_NormalizesPerfPathsForShutdown(t *testing.T) {
	resetTelemetryState(t)

	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	baseDir := t.TempDir()
	absPath := filepath.Join(baseDir, "example.jar")

	ctx, root := perf.StartSpan(context.Background(), "app.lifecycle")
	_, child := perf.StartSpan(ctx, "app.command.list", perf.WithAttributes(attribute.String("path", absPath)))
	child.End()
	root.End()

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	SetPerfBaseDir(baseDir)
	RecordCommand(CommandTelemetry{Command: "list", Success: true})
	Shutdown(context.Background())

	capture := client.enqueued[0].(posthog.Capture)
	tree := capture.Properties["performance"].([]*perf.ExportSpan)
	if assert.NotEmpty(t, tree) {
		assert.Equal(t, "app.lifecycle", tree[0].Name)
		if assert.NotEmpty(t, tree[0].Children) {
			path, ok := tree[0].Children[0].Attributes["path"].(string)
			assert.True(t, ok)
			assert.Equal(t, "example.jar", path)
		}
	}
}

func TestResolveSessionName_UsesHintWhenNoCommandsRecorded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		sessionNameHint  string
		canonicalCommand string
		commands         []recordedCommand
		expected         string
	}{
		{
			name:             "multiple commands uses tui",
			sessionNameHint:  "list",
			canonicalCommand: "install",
			commands: []recordedCommand{
				{Name: "add"},
				{Name: "list"},
			},
			expected: "tui",
		},
		{
			name:             "single command prefers recorded name",
			sessionNameHint:  "i",
			canonicalCommand: "install",
			commands:         []recordedCommand{{Name: "list"}},
			expected:         "list",
		},
		{
			name:             "single command uses canonical when recorded empty",
			sessionNameHint:  "i",
			canonicalCommand: "install",
			commands:         []recordedCommand{{Name: " "}},
			expected:         "install",
		},
		{
			name:             "no commands uses canonical before hint",
			sessionNameHint:  "i",
			canonicalCommand: "install",
			commands:         nil,
			expected:         "install",
		},
		{
			name:            "no commands uses hint when canonical empty",
			sessionNameHint: "list",
			commands:        nil,
			expected:        "list",
		},
		{
			name:     "no commands falls back to unknown",
			commands: nil,
			expected: "unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, resolveSessionName(tc.sessionNameHint, tc.canonicalCommand, tc.commands))
		})
	}
}

func TestBuildCommandSummaries_ReturnsEmptySliceWhenNoCommands(t *testing.T) {
	assert.Equal(t, []map[string]interface{}{}, buildCommandSummaries(nil, nil))
}

func TestCommandNameFromPerfSpan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		spanName string
		expected string
		ok       bool
	}{
		{name: "not a command span", spanName: "app.lifecycle", ok: false},
		{name: "empty command suffix", spanName: "app.command.", ok: false},
		{name: "stage span", spanName: "app.command.add.stage.prepare", ok: false},
		{name: "valid command span", spanName: "app.command.install", expected: "install", ok: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := commandNameFromPerfSpan(tc.spanName)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestTopCommandNameFromPerformance(t *testing.T) {
	t.Parallel()

	t.Run("returns false on empty performance", func(t *testing.T) {
		name, ok := topCommandNameFromPerformance(nil)
		assert.False(t, ok)
		assert.Empty(t, name)
	})

	t.Run("ignores stage spans", func(t *testing.T) {
		performance := []*perf.ExportSpan{
			{Name: "app.lifecycle", Children: []*perf.ExportSpan{
				{Name: "app.command.install.stage.prepare"},
			}},
		}
		name, ok := topCommandNameFromPerformance(performance)
		assert.False(t, ok)
		assert.Empty(t, name)
	})

	t.Run("chooses shallower over deeper", func(t *testing.T) {
		base := time.Now()

		performance := []*perf.ExportSpan{
			nil,
			{
				Name: "root",
				Children: []*perf.ExportSpan{
					{Name: "not-command", Children: []*perf.ExportSpan{
						{Name: "app.command.deep", StartTime: base.Add(2 * time.Second)},
					}},
				},
			},
			{
				Name:      "app.command.shallow",
				StartTime: base.Add(3 * time.Second),
			},
		}

		name, ok := topCommandNameFromPerformance(performance)
		assert.True(t, ok)
		assert.Equal(t, "shallow", name)
	})

	t.Run("chooses shallowest then earliest", func(t *testing.T) {
		base := time.Now()

		performance := []*perf.ExportSpan{
			{
				Name:      "root",
				StartTime: base.Add(1 * time.Second),
				Children: []*perf.ExportSpan{
					{Name: "app.command.list", StartTime: base.Add(2 * time.Second)},
					{Name: "app.command.add", StartTime: base.Add(3 * time.Second)},
				},
			},
			{
				Name:      "root-2",
				StartTime: base,
				Children: []*perf.ExportSpan{
					{Name: "app.command.install", StartTime: base.Add(500 * time.Millisecond)},
				},
			},
		}

		name, ok := topCommandNameFromPerformance(performance)
		assert.True(t, ok)
		assert.Equal(t, "install", name)
	})
}

func TestBuildCommandSummaries_IncludesErrorsAndSkipsMissingDuration(t *testing.T) {
	commands := []recordedCommand{
		{
			Name:          "list",
			Success:       false,
			ExitCode:      1,
			Interactive:   false,
			ErrorCategory: "unknown",
			ErrorMessage:  "boom",
			Arguments:     map[string]interface{}{"quiet": true},
		},
	}

	summaries := buildCommandSummaries(commands, nil)
	if assert.Len(t, summaries, 1) {
		assert.Equal(t, "unknown", summaries[0]["error_category"])
		assert.Equal(t, "boom", summaries[0]["error"])
		assert.Equal(t, map[string]interface{}{"quiet": true}, summaries[0]["arguments"])
		_, hasDuration := summaries[0]["duration_ms"]
		assert.False(t, hasDuration)
	}
}

func TestCommandDurationFromPerf_ReturnsFalseOnEmptyInputs(t *testing.T) {
	duration, ok := commandDurationFromPerf("", nil)
	assert.False(t, ok)
	assert.Equal(t, time.Duration(0), duration)

	duration, ok = commandDurationFromPerf("list", nil)
	assert.False(t, ok)
	assert.Equal(t, time.Duration(0), duration)
}

func TestCommandDurationFromPerf_ReturnsFalseWhenSpanMissing(t *testing.T) {
	performance := []*perf.ExportSpan{
		{Name: "unrelated", DurationNS: int64(5 * time.Millisecond)},
	}

	duration, ok := commandDurationFromPerf("list", performance)
	assert.False(t, ok)
	assert.Equal(t, time.Duration(0), duration)
}

func TestCommandDurationFromPerf_SelectsLatestMatchingSpan(t *testing.T) {
	now := time.Now()
	earlier := &perf.ExportSpan{
		Name:       "app.command.list",
		DurationNS: int64(1 * time.Millisecond),
		EndTime:    now,
	}
	later := &perf.ExportSpan{
		Name:       "app.command.list",
		DurationNS: int64(2 * time.Millisecond),
		EndTime:    now.Add(time.Second),
		Children:   []*perf.ExportSpan{nil},
	}

	duration, ok := commandDurationFromPerf("list", []*perf.ExportSpan{earlier, later})
	assert.True(t, ok)
	assert.Equal(t, 2*time.Millisecond, duration)
}

func TestNoopLogger_Debugf_NoPanic(t *testing.T) {
	var logger noopLogger
	assert.NotPanics(t, func() {
		logger.Debugf("hello %s", "world")
	})
}

func TestRecordCommandWithEmptyNameDoesNothing(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	RecordCommand(CommandTelemetry{})

	assert.Empty(t, client.enqueued)
}

func TestRecordCommandWithoutInitIsNoop(t *testing.T) {
	resetTelemetryState(t)
	assert.NotPanics(t, func() {
		RecordCommand(CommandTelemetry{Command: "list", Success: true})
	})
}

func TestCaptureWithSnapshot_LogsEnqueueError(t *testing.T) {
	logger := &recordingLogger{}
	client := &stubClient{enqueueErr: errors.New("enqueue failed")}

	snap := telemetrySnapshot{
		client:       client,
		machineID:    "machine",
		logger:       logger,
		flushTimeout: time.Second,
		enabled:      true,
	}

	captureWithSnapshot(snap, "", nil)
	assert.Empty(t, client.enqueued)

	captureWithSnapshot(snap, "event", map[string]interface{}{"foo": "bar"})
	assert.NotEmpty(t, logger.messages)
	assert.Contains(t, strings.Join(logger.messages, "\n"), "enqueue failed")
}

func TestCommandExitCode_UsesExplicitValue(t *testing.T) {
	assert.Equal(t, 2, commandExitCode(CommandTelemetry{Success: true, ExitCode: 2}))
	assert.Equal(t, 0, commandExitCode(CommandTelemetry{Success: true}))
	assert.Equal(t, 1, commandExitCode(CommandTelemetry{Success: false}))
}

func TestErrorCategory_RecognizesKnownErrors(t *testing.T) {
	assert.Equal(t, "", errorCategory(nil))
	assert.Equal(t, "canceled", errorCategory(context.Canceled))
	assert.Equal(t, "project_not_found", errorCategory(&globalerrors.ProjectNotFoundError{ProjectID: "x", Platform: models.MODRINTH}))
	assert.Equal(t, "project_api_error", errorCategory(&globalerrors.ProjectAPIError{Err: errors.New("boom"), ProjectID: "x", Platform: models.MODRINTH}))
	assert.Equal(t, "unknown", errorCategory(errors.New("boom")))
}

func TestInitHonorsDisableEnv(t *testing.T) {
	resetTelemetryState(t)
	t.Setenv(disableEnvVar, "true")
	called := false
	clientBuilder = func(apiKey, endpoint string) (Client, error) {
		called = true
		return &stubClient{}, nil
	}
	t.Setenv("POSTHOG_API_KEY", "test-key")
	Init()
	Capture("event", nil)

	assert.False(t, called)
}

func TestInitDisablesWhenAPIKeyEmpty(t *testing.T) {
	resetTelemetryState(t)
	t.Setenv(disableEnvVar, "")

	called := false
	clientBuilder = func(apiKey, endpoint string) (Client, error) {
		called = true
		return &stubClient{}, nil
	}
	t.Setenv("POSTHOG_API_KEY", "")

	Init()

	assert.False(t, called)
	assert.False(t, state.snapshot().enabled)
}

func TestInitUsesCustomClientFactory(t *testing.T) {
	resetTelemetryState(t)
	t.Setenv(disableEnvVar, "")

	created := false
	client := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) {
		created = true
		return client, nil
	}

	machineIDProvider = func() (string, error) { return "factory-test", nil }
	t.Setenv("POSTHOG_API_KEY", "example")

	Init()

	assert.True(t, created)
	Capture("event", nil)
	assert.Len(t, client.enqueued, 1)
}

func TestInitHandlesClientFactoryError(t *testing.T) {
	resetTelemetryState(t)
	t.Setenv(disableEnvVar, "")

	clientBuilder = func(apiKey, endpoint string) (Client, error) {
		return nil, errors.New("fail")
	}
	t.Setenv("POSTHOG_API_KEY", "example")
	machineIDProvider = func() (string, error) { return "machine", nil }

	Init()

	Capture("event", nil)
	snap := state.snapshot()
	assert.False(t, snap.enabled)
}

func TestInitNormalizesFlushTimeout(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	baseFlushTimeout = 0
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()
	assert.Equal(t, defaultFlushTimeout, state.snapshot().flushTimeout)
}

func TestInitUsesDefaultLoggerWhenNil(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	baseLogger = nil
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()
	Capture("event", nil)

	assert.Len(t, client.enqueued, 1)
}

func TestMachineIDEnvOverridesFetcher(t *testing.T) {
	resetTelemetryState(t)
	t.Setenv(machineIDEnvVar, "env-id")

	client := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) {
		return "fetcher-id", nil
	}
	t.Setenv("POSTHOG_API_KEY", "key")

	Init()

	Capture("event", nil)
	capture := client.enqueued[0].(posthog.Capture)
	assert.Equal(t, "env-id", capture.DistinctId)
}

func TestMachineIDFetcherUsage(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) {
		return "fetcher-id", nil
	}
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	Capture("event", nil)
	capture := client.enqueued[0].(posthog.Capture)
	assert.Equal(t, "fetcher-id", capture.DistinctId)
}

func TestMachineIDFallback(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) {
		return "", errors.New("fail")
	}
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	Capture("event", nil)
	capture := client.enqueued[0].(posthog.Capture)
	assert.Equal(t, unknownMachineID, capture.DistinctId)
}

func TestCaptureHandlesEnqueueError(t *testing.T) {
	resetTelemetryState(t)

	logger := &recordingLogger{}
	client := &stubClient{enqueueErr: errors.New("enqueue failed")}
	baseLogger = logger
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	Capture("event", nil)

	assert.NotEmpty(t, logger.messages)
	assert.Contains(t, strings.Join(logger.messages, "\n"), "enqueue failed")
}

func TestShutdownClosesClientOnce(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	Shutdown(ctx)
	Shutdown(ctx)

	assert.Equal(t, 1, client.closeCount)
}

func TestShutdownTimeoutLogs(t *testing.T) {
	resetTelemetryState(t)

	logger := &recordingLogger{}
	client := &stubClient{closeDelay: 250 * time.Millisecond}
	baseLogger = logger
	baseFlushTimeout = 5 * time.Millisecond
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	start := time.Now()
	Shutdown(nil)
	duration := time.Since(start)

	assert.Less(t, duration, 10*baseFlushTimeout)
	assert.Less(t, duration, client.closeDelay)
	joined := strings.Join(logger.messages, "\n")
	assert.Contains(t, joined, "timed out")
}

func TestShutdownTimeoutLogsWithContextWithoutDeadline(t *testing.T) {
	resetTelemetryState(t)

	logger := &recordingLogger{}
	client := &stubClient{closeDelay: 250 * time.Millisecond}
	baseLogger = logger
	baseFlushTimeout = 5 * time.Millisecond
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	start := time.Now()
	Shutdown(context.Background())
	duration := time.Since(start)

	assert.Less(t, duration, 10*baseFlushTimeout)
	assert.Less(t, duration, client.closeDelay)
	joined := strings.Join(logger.messages, "\n")
	assert.Contains(t, joined, "timed out")
}

func TestShutdownLogsCloseError(t *testing.T) {
	resetTelemetryState(t)

	logger := &recordingLogger{}
	client := &stubClient{closeErr: errors.New("close failed")}
	baseLogger = logger
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	Shutdown(context.Background())

	assert.Contains(t, strings.Join(logger.messages, "\n"), "close failed")
}

func TestShutdownWithoutInit(t *testing.T) {
	resetTelemetryState(t)
	Shutdown(nil)
}

func TestResetAllowsReinit(t *testing.T) {
	resetTelemetryState(t)

	first := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return first, nil }
	machineIDProvider = func() (string, error) { return "first", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()
	Capture("first", nil)
	Shutdown(nil)

	Reset()

	second := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return second, nil }
	machineIDProvider = func() (string, error) { return "second", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()
	Capture("second", nil)

	assert.Len(t, first.enqueued, 2)
	assert.Len(t, second.enqueued, 1)
}

func TestDefaultClientFactory(t *testing.T) {
	client, err := defaultClientFactory("key", defaultPosthogHost)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NoError(t, client.Close())
}

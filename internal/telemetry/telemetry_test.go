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
	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
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

func (s *stubClient) Enqueue(msg posthog.Message) error {
	s.enqueued = append(s.enqueued, msg)
	return s.enqueueErr
}

func (s *stubClient) Close() error {
	s.closeCount++
	if s.closeDelay > 0 {
		time.Sleep(s.closeDelay)
	}
	return s.closeErr
}

type recordingLogger struct {
	messages []string
}

func (l *recordingLogger) Debugf(format string, args ...interface{}) {
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
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
	config := &models.ModsJson{}
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

func TestCaptureCommand_AliasesRecordCommand(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	CaptureCommand(CommandTelemetry{Command: "list", Success: true, ExitCode: 0})
	Shutdown(context.Background())

	capture := client.enqueued[0].(posthog.Capture)
	props := capture.Properties
	commands := props["commands"].([]map[string]interface{})
	assert.Len(t, commands, 1)
	assert.Equal(t, "list", commands[0]["name"])
}

func TestResolveSessionName_UsesHintWhenNoCommandsRecorded(t *testing.T) {
	assert.Equal(t, "list", resolveSessionName("list", nil))
	assert.Equal(t, "unknown", resolveSessionName("", nil))
}

func TestBuildCommandSummaries_ReturnsEmptySliceWhenNoCommands(t *testing.T) {
	assert.Equal(t, []map[string]interface{}{}, buildCommandSummaries(nil, nil))
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
	assert.Equal(t, "project_not_found", errorCategory(&globalErrors.ProjectNotFoundError{ProjectID: "x", Platform: models.MODRINTH}))
	assert.Equal(t, "project_api_error", errorCategory(&globalErrors.ProjectApiError{Err: errors.New("boom"), ProjectID: "x", Platform: models.MODRINTH}))
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
	client := &stubClient{closeDelay: 20 * time.Millisecond}
	baseLogger = logger
	baseFlushTimeout = 5 * time.Millisecond
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	start := time.Now()
	Shutdown(nil)
	duration := time.Since(start)

	assert.Less(t, duration, 20*time.Millisecond)
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
	_ = client.Close()
}

package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/posthog/posthog-go"
	"github.com/stretchr/testify/assert"
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

func TestCaptureCommandProperties(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	err := errors.New("boom")
	config := &models.ModsJson{}
	CaptureCommand(CommandTelemetry{
		Command:   "list",
		Success:   false,
		Config:    config,
		Error:     err,
		Extra:     map[string]interface{}{"total": 2},
		Arguments: map[string]interface{}{"quiet": true},
		Duration:  150 * time.Millisecond,
	})

	if assert.Len(t, client.enqueued, 1) {
		capture := client.enqueued[0].(posthog.Capture)
		props := capture.Properties
		assert.Equal(t, "command", props["type"])
		assert.Equal(t, false, props["success"])
		assert.Equal(t, "boom", props["error"])
		assert.Equal(t, config, props["config"])
		assert.Equal(t, map[string]interface{}{"total": 2}, props["extra"])
		assert.Equal(t, map[string]interface{}{"quiet": true}, props["arguments"])
		assert.Equal(t, int64(150), props["duration_ms"])
	}
}

func TestCaptureCommandSkipsSuccessConfig(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	CaptureCommand(CommandTelemetry{Command: "list", Success: true, Config: &models.ModsJson{}})

	capture := client.enqueued[0].(posthog.Capture)
	_, hasConfig := capture.Properties["config"]
	assert.False(t, hasConfig)
}

func TestCaptureCommandWithEmptyNameDoesNothing(t *testing.T) {
	resetTelemetryState(t)

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	CaptureCommand(CommandTelemetry{})

	assert.Empty(t, client.enqueued)
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

	assert.Len(t, first.enqueued, 1)
	assert.Len(t, second.enqueued, 1)
}

func TestDefaultClientFactory(t *testing.T) {
	client, err := defaultClientFactory("key", defaultPosthogHost)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	_ = client.Close()
}

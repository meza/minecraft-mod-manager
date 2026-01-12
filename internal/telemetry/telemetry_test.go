package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/posthog/posthog-go"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type stubClient struct {
	enqueued    []posthog.Message
	enqueueErr  error
	closeErr    error
	closeCount  int
	closeWaitCh <-chan struct{}
}

func (client *stubClient) Enqueue(msg posthog.Message) error {
	client.enqueued = append(client.enqueued, msg)
	return client.enqueueErr
}

func (client *stubClient) Close() error {
	client.closeCount++
	if client.closeWaitCh != nil {
		<-client.closeWaitCh
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
	_, cmdSpan := perf.StartSpan(rootCtx, "app.command.list")
	cmdSpan.End()
	_, waitSpan := perf.StartSpan(rootCtx, "interaction.list.wait.view")
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
		summary, ok := props["performance"].(map[string]interface{})
		assert.True(t, ok)
		_, hasLegacySummary := props["perf_summary_v1"]
		assert.False(t, hasLegacySummary)

		commands := summary["commands"].([]map[string]interface{})
		if assert.Len(t, commands, 1) {
			assert.Equal(t, "list", commands[0]["name"])
			assert.Equal(t, true, commands[0]["success"])
			assert.Equal(t, 0, commands[0]["exit_code"])
			assert.Equal(t, "interactive", commands[0]["execution_mode"])
			assert.Equal(t, map[string]interface{}{"numberOfMods": 2}, commands[0]["extra"])
		}

		total, hasTotal := props["total_time_ms"].(int64)
		work, hasWork := props["work_time_ms"].(int64)
		assert.True(t, hasTotal)
		assert.True(t, hasWork)
		assert.GreaterOrEqual(t, total, int64(0))
		assert.GreaterOrEqual(t, work, int64(0))
		assert.LessOrEqual(t, work, total)
	}
}

func TestShutdownDoesNotLeakAPIKeys(t *testing.T) {
	resetTelemetryState(t)

	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	t.Setenv("MODRINTH_API_KEY", "test-secret")
	t.Setenv("CURSEFORGE_API_KEY", "other-secret")

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	RecordCommand(CommandTelemetry{
		Command:  "list",
		Success:  true,
		ExitCode: 0,
	})

	Shutdown(context.Background())

	if assert.Len(t, client.enqueued, 1) {
		capture := client.enqueued[0].(posthog.Capture)
		raw, err := json.Marshal(capture)
		assert.NoError(t, err)
		payload := string(raw)
		assert.False(t, strings.Contains(payload, "test-secret"))
		assert.False(t, strings.Contains(payload, "other-secret"))
		assert.False(t, strings.Contains(payload, "test-key"))
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
		summary := props["performance"].(map[string]interface{})
		commands := summary["commands"].([]map[string]interface{})
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
		summary := props["performance"].(map[string]interface{})
		commands := summary["commands"].([]map[string]interface{})
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

func TestSetConfigPathAddsModlistSummary(t *testing.T) {
	resetTelemetryState(t)

	fs := afero.NewOsFs()
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "modlist.json")

	configMeta := config.NewMetadata(configPath)
	configPayload := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.21.1",
		ModsFolder:  "mods",
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "AANobbMI", Name: "Example Mod"},
		},
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, configMeta, configPayload))

	client := &stubClient{}
	initWithClient(t, client, "cmd-test")

	SetConfigPath(configPath)
	RecordCommand(CommandTelemetry{Command: "list", Success: true})
	Shutdown(context.Background())

	capture := client.enqueued[0].(posthog.Capture)
	summary := capture.Properties["performance"].(map[string]interface{})
	modlist := summary["modlist"].(map[string]interface{})
	assert.Equal(t, "1.21.1", modlist["game_version"])
	assert.Equal(t, "fabric", modlist["loader"])
	assert.Equal(t, 1, modlist["mod_count"])
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

func TestResolveExecutionModeSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		commands []recordedCommand
		expected string
	}{
		{name: "empty", expected: "unknown"},
		{name: "interactive wins", commands: []recordedCommand{{ExecutionMode: "unattended"}, {ExecutionMode: "interactive"}}, expected: "interactive"},
		{name: "non-tty before unattended", commands: []recordedCommand{{ExecutionMode: "unattended"}, {ExecutionMode: "non_tty"}}, expected: "non_tty"},
		{name: "unattended fallback", commands: []recordedCommand{{ExecutionMode: "unattended"}}, expected: "unattended"},
		{name: "interactive inferred", commands: []recordedCommand{{Interactive: true}}, expected: "interactive"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, resolveExecutionModeSummary(testCase.commands))
		})
	}
}

func TestResolveCommandExecutionMode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "interactive", resolveCommandExecutionMode(recordedCommand{Interactive: true}))
	assert.Equal(t, "unattended", resolveCommandExecutionMode(recordedCommand{}))
	assert.Equal(t, "non_tty", resolveCommandExecutionMode(recordedCommand{ExecutionMode: "non_tty"}))
}

func TestBuildCommandSummariesV1_UsesTimingsAndStagesInOrder(t *testing.T) {
	base := time.Now()

	performance := []*perf.ExportSpan{
		{
			Name:       "app.command.add",
			StartTime:  base,
			EndTime:    base.Add(2 * time.Second),
			DurationNS: int64(2 * time.Second),
			Children: []*perf.ExportSpan{
				{Name: "app.command.add.stage.download", DurationNS: int64(500 * time.Millisecond)},
				{Name: "app.command.add.stage.persist", DurationNS: int64(300 * time.Millisecond)},
			},
		},
		{
			Name:       "app.command.add",
			StartTime:  base.Add(3 * time.Second),
			EndTime:    base.Add(4 * time.Second),
			DurationNS: int64(1 * time.Second),
		},
		{
			Name:       "app.command.list",
			StartTime:  base.Add(5 * time.Second),
			EndTime:    base.Add(5500 * time.Millisecond),
			DurationNS: int64(500 * time.Millisecond),
		},
	}

	commands := []recordedCommand{
		{Name: "add", Success: true},
		{Name: "add", Success: true},
		{Name: "list", Success: true},
	}

	summaries := buildCommandSummariesV1(commands, performance)
	if assert.Len(t, summaries, 3) {
		assert.Equal(t, int64(2000), summaries[0]["duration_ms"])
		stageDurations := summaries[0]["stage_durations_ms"].(map[string]int64)
		assert.Equal(t, int64(500), stageDurations["download"])
		assert.Equal(t, int64(300), stageDurations["persist"])

		assert.Equal(t, int64(1000), summaries[1]["duration_ms"])
		assert.Equal(t, int64(500), summaries[2]["duration_ms"])
	}
}

func TestBuildStageDurationSummarySkipsZeroDurations(t *testing.T) {
	summary := buildStageDurationSummary(map[string]time.Duration{"download": 0})
	assert.Nil(t, summary)
}

func TestBuildPerformanceCounts(t *testing.T) {
	performance := []*perf.ExportSpan{
		{
			Name: "root",
			Children: []*perf.ExportSpan{
				{Name: "net.http.request"},
				{Name: "io.download.file", Attributes: map[string]interface{}{"bytes": int64(120)}},
				{Name: "io.download.file", Attributes: map[string]interface{}{"bytes": 80}},
				{Name: "io.download.file", Attributes: map[string]interface{}{"bytes": 50.0}},
				{Name: "io.download.file", Attributes: map[string]interface{}{"bytes": "unknown"}},
			},
		},
		{Name: "net.http.request"},
	}

	counts := buildPerformanceCounts(performance)
	assert.Equal(t, 2, counts["http_requests"])
	assert.Equal(t, 4, counts["downloads"])
	assert.Equal(t, int64(250), counts["download_bytes"])
}

func TestBuildPerfSummaryV1IncludesModlist(t *testing.T) {
	commands := []recordedCommand{{Name: "list", Success: true, ExecutionMode: "non_tty"}}
	performance := []*perf.ExportSpan{{Name: "net.http.request"}}
	modlist := &modlistSummary{GameVersion: "1.21.1", Loader: "fabric", ModCount: 2}

	summary := buildPerfSummaryV1(commands, performance, modlist)
	assert.Equal(t, 1, summary["schema_version"])
	assert.Equal(t, environment.AppVersion(), summary["app_version"])
	assert.Equal(t, "non_tty", summary["execution_mode"])

	modlistEntry := summary["modlist"].(map[string]interface{})
	assert.Equal(t, "1.21.1", modlistEntry["game_version"])
	assert.Equal(t, "fabric", modlistEntry["loader"])
	assert.Equal(t, 2, modlistEntry["mod_count"])
}

func TestLoadModlistSummaryEmptyPathReturnsNil(t *testing.T) {
	summary := loadModlistSummary("", nil)
	assert.Nil(t, summary)
}

func TestLoadModlistSummaryMissingFileReturnsNil(t *testing.T) {
	logger := &recordingLogger{}
	summary := loadModlistSummary(filepath.Join(t.TempDir(), "missing.json"), logger)
	assert.Nil(t, summary)
	assert.NotEmpty(t, logger.messages)
}

func TestLoadModlistSummaryWithNilLoggerUsesConfig(t *testing.T) {
	fs := afero.NewOsFs()
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "modlist.json")

	meta := config.NewMetadata(configPath)
	payload := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
	}
	assert.NoError(t, config.WriteConfig(context.Background(), fs, meta, payload))

	summary := loadModlistSummary(configPath, nil)
	if assert.NotNil(t, summary) {
		assert.Equal(t, "1.20.1", summary.GameVersion)
		assert.Equal(t, "fabric", summary.Loader)
		assert.Equal(t, 0, summary.ModCount)
	}
}

func TestResolveExecutionModeSummaryUnknownMode(t *testing.T) {
	commands := []recordedCommand{{ExecutionMode: "unexpected"}}
	assert.Equal(t, "unknown", resolveExecutionModeSummary(commands))
}

func TestNextCommandTimingReturnsFalseWhenMissingOrExhausted(t *testing.T) {
	timing, ok := nextCommandTiming("list", map[string][]commandTiming{}, map[string]int{})
	assert.False(t, ok)
	assert.Equal(t, commandTiming{}, timing)

	timingIndex := map[string][]commandTiming{"add": {{Duration: time.Second}}}
	selectionIndex := map[string]int{"add": 2}
	timing, ok = nextCommandTiming("add", timingIndex, selectionIndex)
	assert.False(t, ok)
	assert.Equal(t, commandTiming{}, timing)
}

func TestRecordCommandTimingNilSpanDoesNothing(t *testing.T) {
	index := map[string][]commandTiming{}
	recordCommandTiming(nil, index)
	assert.Empty(t, index)
}

func TestBuildStageDurationsEmptyInputsReturnNil(t *testing.T) {
	assert.Nil(t, buildStageDurations(nil, "add"))

	emptySpan := &perf.ExportSpan{Name: "app.command.add"}
	assert.Nil(t, buildStageDurations(emptySpan, "add"))

	stageNameEmpty := &perf.ExportSpan{
		Name: "app.command.add",
		Children: []*perf.ExportSpan{
			{Name: "app.command.add.stage."},
		},
	}
	assert.Nil(t, buildStageDurations(stageNameEmpty, "add"))
	assert.Nil(t, buildStageDurations(stageNameEmpty, ""))
}

func TestAddStageDurationsSkipsEmptyStageName(t *testing.T) {
	stageDurations := map[string]time.Duration{}
	addStageDurations(stageDurations, "add", &perf.ExportSpan{Name: "app.command.add.stage."})
	assert.Empty(t, stageDurations)
}

func TestAddStageDurationsNilSpanIsNoop(t *testing.T) {
	stageDurations := map[string]time.Duration{}
	addStageDurations(stageDurations, "add", nil)
	assert.Empty(t, stageDurations)
}

func TestAddStageDurationsWalksChildren(t *testing.T) {
	stageDurations := map[string]time.Duration{}
	parent := &perf.ExportSpan{
		Name: "root",
		Children: []*perf.ExportSpan{
			{Name: "app.command.add.stage.fetch", DurationNS: int64(2 * time.Second)},
		},
	}
	addStageDurations(stageDurations, "add", parent)
	assert.Equal(t, int64(2000), stageDurations["fetch"].Milliseconds())
}

func TestAccumulateCountsIgnoresNilSpan(t *testing.T) {
	httpCount := 0
	downloadCount := 0
	var downloadBytes int64
	accumulateCounts(nil, &httpCount, &downloadCount, &downloadBytes)
	assert.Equal(t, 0, httpCount)
	assert.Equal(t, 0, downloadCount)
	assert.Equal(t, int64(0), downloadBytes)
}

func TestSpanInt64AttributeMissingReturnsZero(t *testing.T) {
	assert.Equal(t, int64(0), spanInt64Attribute(nil))
	assert.Equal(t, int64(0), spanInt64Attribute(map[string]interface{}{}))
	assert.Equal(t, int64(0), spanInt64Attribute(map[string]interface{}{"other": 123}))
	assert.Equal(t, int64(0), spanInt64Attribute(map[string]interface{}{"bytes": "n/a"}))
}

func TestSetPerfBaseDirStoresValue(t *testing.T) {
	resetTelemetryState(t)

	SetPerfBaseDir("/perf")
	snapshot := snapshotState()
	assert.Equal(t, "/perf", snapshot.perfBaseDir)
}

func TestSetConfigPathIgnoresEmptyValue(t *testing.T) {
	resetTelemetryState(t)

	SetConfigPath(" ")
	snapshot := snapshotState()
	assert.Equal(t, "", snapshot.configPath)
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

func TestEnsureShutdownContext_UsesBackgroundForNil(t *testing.T) {
	//nolint:staticcheck // Validates nil context handling.
	ctx, cancel := ensureShutdownContext(nil, time.Millisecond)
	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)
	_, hasDeadline := ctx.Deadline()
	assert.True(t, hasDeadline)
	cancel()
}

func TestEnsureShutdownContext_ReturnsExistingDeadline(t *testing.T) {
	ctxWithDeadline, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ctx, cancelFunc := ensureShutdownContext(ctxWithDeadline, time.Millisecond)
	assert.Equal(t, ctxWithDeadline, ctx)
	assert.Nil(t, cancelFunc)
}

func TestEnsureShutdownContext_AddsTimeoutWhenMissing(t *testing.T) {
	ctx, cancel := ensureShutdownContext(context.Background(), time.Millisecond)
	assert.NotNil(t, cancel)
	_, hasDeadline := ctx.Deadline()
	assert.True(t, hasDeadline)
	cancel()
}

func TestShutdownTimeoutLogs(t *testing.T) {
	resetTelemetryState(t)

	logger := &recordingLogger{}
	closeWaitCh := make(chan struct{})
	client := &stubClient{closeWaitCh: closeWaitCh}
	baseLogger = logger
	baseFlushTimeout = 5 * time.Millisecond
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	start := time.Now()
	Shutdown(context.TODO())
	duration := time.Since(start)
	close(closeWaitCh)

	assert.Less(t, duration, 10*baseFlushTimeout)
	joined := strings.Join(logger.messages, "\n")
	assert.Contains(t, joined, "timed out")
}

func TestShutdownTimeoutLogsWithContextWithoutDeadline(t *testing.T) {
	resetTelemetryState(t)

	logger := &recordingLogger{}
	closeWaitCh := make(chan struct{})
	client := &stubClient{closeWaitCh: closeWaitCh}
	baseLogger = logger
	baseFlushTimeout = 5 * time.Millisecond
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return client, nil }
	machineIDProvider = func() (string, error) { return "machine", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()

	start := time.Now()
	Shutdown(context.Background())
	duration := time.Since(start)
	close(closeWaitCh)

	assert.Less(t, duration, 10*baseFlushTimeout)
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
	Shutdown(context.TODO())
}

func TestResetAllowsReinit(t *testing.T) {
	resetTelemetryState(t)

	first := &stubClient{}
	clientBuilder = func(apiKey, endpoint string) (Client, error) { return first, nil }
	machineIDProvider = func() (string, error) { return "first", nil }
	t.Setenv("POSTHOG_API_KEY", "key")
	Init()
	Capture("first", nil)
	Shutdown(context.TODO())

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

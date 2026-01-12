package telemetry

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/afero"
)

type modlistSummary struct {
	GameVersion string
	Loader      string
	ModCount    int
}

type commandTiming struct {
	Duration       time.Duration
	StageDurations map[string]time.Duration
}

func loadModlistSummary(configPath string, logger Logger) *modlistSummary {
	if strings.TrimSpace(configPath) == "" {
		return nil
	}
	if logger == nil {
		logger = noopLogger{}
	}

	metadata := config.NewMetadata(configPath)
	configuration, err := config.ReadConfig(context.Background(), afero.NewOsFs(), metadata)
	if err != nil {
		logger.Debugf("telemetry: failed to read config for summary: %v", err)
		return nil
	}

	return &modlistSummary{
		GameVersion: configuration.GameVersion,
		Loader:      configuration.Loader.String(),
		ModCount:    len(configuration.Mods),
	}
}

func buildPerfSummaryV1(commands []recordedCommand, performance []*perf.ExportSpan, modlist *modlistSummary) map[string]interface{} {
	summary := map[string]interface{}{
		"schema_version": 1,
		"app_version":    environment.AppVersion(),
		"os": map[string]interface{}{
			"goos":   runtime.GOOS,
			"goarch": runtime.GOARCH,
		},
		"execution_mode": resolveExecutionModeSummary(commands),
		"commands":       buildCommandSummariesV1(commands, performance),
		"counts":         buildPerformanceCounts(performance),
	}

	if modlist != nil {
		summary["modlist"] = map[string]interface{}{
			"game_version": modlist.GameVersion,
			"loader":       modlist.Loader,
			"mod_count":    modlist.ModCount,
		}
	}

	return summary
}

func resolveExecutionModeSummary(commands []recordedCommand) string {
	if len(commands) == 0 {
		return "unknown"
	}

	seenNonTTY := false
	seenUnattended := false
	for _, command := range commands {
		mode := resolveCommandExecutionMode(command)
		switch mode {
		case "interactive":
			return "interactive"
		case "non_tty":
			seenNonTTY = true
		case "unattended":
			seenUnattended = true
		}
	}

	if seenNonTTY {
		return "non_tty"
	}
	if seenUnattended {
		return "unattended"
	}
	return "unknown"
}

func resolveCommandExecutionMode(command recordedCommand) string {
	if strings.TrimSpace(command.ExecutionMode) != "" {
		return command.ExecutionMode
	}
	if command.Interactive {
		return "interactive"
	}
	return "unattended"
}

func buildCommandSummariesV1(commands []recordedCommand, performance []*perf.ExportSpan) []map[string]interface{} {
	if len(commands) == 0 {
		return []map[string]interface{}{}
	}

	timingIndex := buildCommandTimingIndex(performance)
	selectionIndex := map[string]int{}

	summaries := make([]map[string]interface{}, 0, len(commands))
	for _, command := range commands {
		summary := map[string]interface{}{
			"name":           command.Name,
			"success":        command.Success,
			"exit_code":      command.ExitCode,
			"execution_mode": resolveCommandExecutionMode(command),
		}

		if command.ErrorCategory != "" {
			summary["error_category"] = command.ErrorCategory
		}
		if command.ErrorMessage != "" {
			summary["error"] = command.ErrorMessage
		}

		if len(command.Extra) > 0 {
			summary["extra"] = command.Extra
		}
		if len(command.Arguments) > 0 {
			summary["arguments"] = command.Arguments
		}

		timing, ok := nextCommandTiming(command.Name, timingIndex, selectionIndex)
		if ok {
			if timing.Duration > 0 {
				summary["duration_ms"] = timing.Duration.Milliseconds()
			}
			stageDurations := buildStageDurationSummary(timing.StageDurations)
			if len(stageDurations) > 0 {
				summary["stage_durations_ms"] = stageDurations
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

func buildStageDurationSummary(stageDurations map[string]time.Duration) map[string]int64 {
	if len(stageDurations) == 0 {
		return nil
	}

	summary := make(map[string]int64, len(stageDurations))
	for stageName, duration := range stageDurations {
		if duration <= 0 {
			continue
		}
		summary[stageName] = duration.Milliseconds()
	}
	if len(summary) == 0 {
		return nil
	}
	return summary
}

func nextCommandTiming(
	commandName string,
	timingIndex map[string][]commandTiming,
	selectionIndex map[string]int,
) (commandTiming, bool) {
	timings := timingIndex[commandName]
	if len(timings) == 0 {
		return commandTiming{}, false
	}

	index := selectionIndex[commandName]
	if index >= len(timings) {
		return commandTiming{}, false
	}

	selectionIndex[commandName] = index + 1
	return timings[index], true
}

func buildCommandTimingIndex(performance []*perf.ExportSpan) map[string][]commandTiming {
	index := map[string][]commandTiming{}
	for _, span := range performance {
		recordCommandTiming(span, index)
	}
	return index
}

func recordCommandTiming(span *perf.ExportSpan, index map[string][]commandTiming) {
	if span == nil {
		return
	}

	commandName, isCommand := commandNameFromPerfSpan(span.Name)
	if isCommand {
		index[commandName] = append(index[commandName], commandTiming{
			Duration:       time.Duration(span.DurationNS),
			StageDurations: buildStageDurations(span, commandName),
		})
	}

	for _, child := range span.Children {
		recordCommandTiming(child, index)
	}
}

func buildStageDurations(commandSpan *perf.ExportSpan, commandName string) map[string]time.Duration {
	if commandSpan == nil || len(commandSpan.Children) == 0 || commandName == "" {
		return nil
	}

	stageDurations := map[string]time.Duration{}
	for _, child := range commandSpan.Children {
		addStageDurations(stageDurations, commandName, child)
	}
	if len(stageDurations) == 0 {
		return nil
	}
	return stageDurations
}

func addStageDurations(stageDurations map[string]time.Duration, commandName string, span *perf.ExportSpan) {
	if span == nil {
		return
	}

	stagePrefix := "app.command." + commandName + ".stage."
	if strings.HasPrefix(span.Name, stagePrefix) {
		stageName := strings.TrimPrefix(span.Name, stagePrefix)
		if stageName != "" {
			stageDurations[stageName] += time.Duration(span.DurationNS)
		}
	}

	for _, child := range span.Children {
		addStageDurations(stageDurations, commandName, child)
	}
}

func buildPerformanceCounts(performance []*perf.ExportSpan) map[string]interface{} {
	httpRequestCount := 0
	downloadCount := 0
	var downloadBytes int64

	for _, span := range performance {
		accumulateCounts(span, &httpRequestCount, &downloadCount, &downloadBytes)
	}

	counts := map[string]interface{}{
		"http_requests": httpRequestCount,
		"downloads":     downloadCount,
	}
	if downloadBytes > 0 {
		counts["download_bytes"] = downloadBytes
	}

	return counts
}

func accumulateCounts(span *perf.ExportSpan, httpRequestCount *int, downloadCount *int, downloadBytes *int64) {
	if span == nil {
		return
	}

	switch span.Name {
	case "net.http.request":
		*httpRequestCount++
	case "io.download.file":
		*downloadCount++
		*downloadBytes += spanInt64Attribute(span.Attributes)
	}

	for _, child := range span.Children {
		accumulateCounts(child, httpRequestCount, downloadCount, downloadBytes)
	}
}

func spanInt64Attribute(attributes map[string]interface{}) int64 {
	if len(attributes) == 0 {
		return 0
	}

	value, ok := attributes["bytes"]
	if !ok {
		return 0
	}

	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

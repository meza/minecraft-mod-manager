package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/stretchr/testify/assert"
)

func TestRunWithDeps_RecordsLifecycleRegions(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	var calls []string
	deps := runDeps{
		execute: func(context.Context) error {
			calls = append(calls, "execute")
			return nil
		},
		telemetryInit: func() {
			calls = append(calls, "telemetryInit")
		},
		telemetryShutdown: func(context.Context) {
			calls = append(calls, "telemetryShutdown")
		},
		register: func(handler lifecycle.Handler) lifecycle.HandlerID {
			assert.NotNil(t, handler)
			calls = append(calls, "register")
			return 42
		},
		unregister: func(id lifecycle.HandlerID) {
			calls = append(calls, "unregister")
			assert.Equal(t, lifecycle.HandlerID(42), id)
		},
		args: []string{"--perf"},
	}

	exitCode := runWithDeps(deps)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, []string{"telemetryInit", "register", "execute", "telemetryShutdown", "unregister"}, calls)

	spans, err := perf.GetSpans()
	assert.NoError(t, err)
	assertSpanExists(t, spans, perfLifecycleStartup)
	assertSpanExists(t, spans, perfLifecycleExecute)
	assertSpanExists(t, spans, perfLifecycleShutdown)
}

func TestRunWithDeps_SignalShutdownIsRecordedOnce(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	var calls []string
	var registeredHandler lifecycle.Handler

	deps := runDeps{
		execute: func(context.Context) error {
			calls = append(calls, "execute-start")
			assert.NotNil(t, registeredHandler)
			registeredHandler(os.Interrupt)
			calls = append(calls, "execute-end")
			return nil
		},
		telemetryInit: func() {
			calls = append(calls, "telemetryInit")
		},
		telemetryShutdown: func(context.Context) {
			calls = append(calls, "telemetryShutdown")
		},
		register: func(handler lifecycle.Handler) lifecycle.HandlerID {
			calls = append(calls, "register")
			registeredHandler = handler
			return 7
		},
		unregister: func(id lifecycle.HandlerID) {
			calls = append(calls, "unregister")
			assert.Equal(t, lifecycle.HandlerID(7), id)
		},
		args: []string{"--perf"},
	}

	exitCode := runWithDeps(deps)
	assert.Equal(t, 0, exitCode)

	var shutdownCalls int
	for _, call := range calls {
		if call == "telemetryShutdown" {
			shutdownCalls++
		}
	}
	assert.Equal(t, 1, shutdownCalls)

	spans, err := perf.GetSpans()
	assert.NoError(t, err)
	assertSpanExists(t, spans, perfLifecycleStartup)
	assertSpanExists(t, spans, perfLifecycleExecute)
	assertSpanExists(t, spans, perfLifecycleShutdown)

	shutdownSpan, ok := perf.FindSpanByName(spans, perfLifecycleShutdown)
	assert.True(t, ok)
	assert.Equal(t, string(shutdownTriggerSignal), shutdownSpan.Attributes["trigger"])
	assert.Equal(t, os.Interrupt.String(), shutdownSpan.Attributes["signal"])
}

func TestPerfExportConfigFromArgs_DefaultsToConfigDir(t *testing.T) {
	cwd := filepath.FromSlash("/workdir")
	cfg := perfExportConfigFromArgs([]string{"--perf", "--config", "configs/modlist.json"}, cwd)
	assert.True(t, cfg.enabled)

	expectedConfig, err := filepath.Abs(filepath.Join(cwd, filepath.FromSlash("configs/modlist.json")))
	assert.NoError(t, err)
	expectedDir := filepath.Dir(expectedConfig)
	assert.Equal(t, expectedDir, cfg.baseDir)
	assert.Equal(t, expectedDir, cfg.outDir)
}

func TestPerfExportConfigFromArgs_PerfOutDirRelativeToConfigDir(t *testing.T) {
	cwd := filepath.FromSlash("/workdir")
	cfg := perfExportConfigFromArgs([]string{"--perf", "--config=cfg/modlist.json", "--perf-out-dir", "perf"}, cwd)

	expectedConfig, err := filepath.Abs(filepath.Join(cwd, filepath.FromSlash("cfg/modlist.json")))
	assert.NoError(t, err)
	expectedDir := filepath.Dir(expectedConfig)
	assert.Equal(t, expectedDir, cfg.baseDir)
	assert.Equal(t, filepath.Join(expectedDir, "perf"), cfg.outDir)
}

func TestPerfExportConfigFromArgs_CapturesDebugFlag(t *testing.T) {
	cfg := perfExportConfigFromArgs([]string{"--perf", "--debug"}, "/workdir")
	assert.True(t, cfg.debug)
}

func assertSpanExists(t *testing.T, spans []perf.SpanSnapshot, name string) {
	t.Helper()
	_, ok := perf.FindSpanByName(spans, name)
	assert.True(t, ok, "expected span %q to exist", name)
}

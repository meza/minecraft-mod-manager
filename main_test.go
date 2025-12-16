package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/stretchr/testify/assert"
)

func TestRunWithDeps_RecordsLifecycleRegions(t *testing.T) {
	perf.ClearPerformanceLog()

	var calls []string
	deps := runDeps{
		execute: func() error {
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
	}

	exitCode := runWithDeps(deps)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, []string{"telemetryInit", "register", "execute", "telemetryShutdown", "unregister"}, calls)

	log := perf.GetPerformanceLog()
	assertLifecycleRegionRecorded(t, log, perfLifecycleStartup)
	assertLifecycleRegionRecorded(t, log, perfLifecycleExecute)
	assertLifecycleRegionRecorded(t, log, perfLifecycleShutdown)
}

func TestRunWithDeps_SignalShutdownIsRecordedOnce(t *testing.T) {
	perf.ClearPerformanceLog()

	var calls []string
	var registeredHandler lifecycle.Handler

	deps := runDeps{
		execute: func() error {
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

	log := perf.GetPerformanceLog()
	assertLifecycleRegionRecorded(t, log, perfLifecycleStartup)
	assertLifecycleRegionRecorded(t, log, perfLifecycleExecute)

	shutdownMarkers := countEntriesByName(log, perfLifecycleShutdown, perf.MarkType)
	assert.Equal(t, 1, shutdownMarkers)

	assertLifecycleRegionRecorded(t, log, perfLifecycleShutdown)
	shutdownEntry := findEntryByName(log, perfLifecycleShutdown, perf.MarkType)
	assert.NotNil(t, shutdownEntry)
	assert.NotNil(t, shutdownEntry.Details)
	assert.Equal(t, string(shutdownTriggerSignal), (*shutdownEntry.Details)["trigger"])
	assert.Equal(t, os.Interrupt.String(), (*shutdownEntry.Details)["signal"])
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

func assertLifecycleRegionRecorded(t *testing.T, log perf.PerformanceLog, name string) {
	t.Helper()

	assert.NotNil(t, findEntryByName(log, name, perf.MarkType))
	assert.NotNil(t, findEntryByName(log, name+"-end", perf.MarkType))
	measurement := findEntryByName(log, name+"-duration", perf.MeasureType)
	assert.NotNil(t, measurement)
	assert.GreaterOrEqual(t, measurement.Duration, time.Duration(0))
}

func findEntryByName(log perf.PerformanceLog, name string, entryType perf.EntryType) *perf.Entry {
	for i := range log {
		if log[i].Name == name && log[i].Type == entryType {
			return &log[i]
		}
	}
	return nil
}

func countEntriesByName(log perf.PerformanceLog, name string, entryType perf.EntryType) int {
	var count int
	for _, entry := range log {
		if entry.Name == name && entry.Type == entryType {
			count++
		}
	}
	return count
}

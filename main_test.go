package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/stretchr/testify/assert"
)

var mainDepsTestMu sync.Mutex

func withMainDeps(t *testing.T, runFunc func() int, exitFunc func(int)) {
	t.Helper()

	mainDepsTestMu.Lock()
	original := mainDepsValue.Load().(mainDeps)
	mainDepsValue.Store(mainDeps{run: runFunc, exit: exitFunc})

	t.Cleanup(func() {
		mainDepsValue.Store(original)
		mainDepsTestMu.Unlock()
	})
}

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

func TestRunWithDeps_SignalBeforeExecuteSpanDoesNotPanic(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	deps := runDeps{
		execute:           func(context.Context) error { return nil },
		telemetryInit:     func() {},
		telemetryShutdown: func(context.Context) {},
		register: func(handler lifecycle.Handler) lifecycle.HandlerID {
			handler(os.Interrupt)
			return 1
		},
		unregister: func(lifecycle.HandlerID) {},
		args:       []string{"--perf"},
	}

	assert.Equal(t, 0, runWithDeps(deps))
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

func TestRunWithDeps_ExitCodeErrorReturnsCustomExitCode(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	deps := runDeps{
		execute: func(context.Context) error {
			return &exitCodeError{code: 2, message: "version matches current"}
		},
		telemetryInit:     func() {},
		telemetryShutdown: func(context.Context) {},
		register: func(handler lifecycle.Handler) lifecycle.HandlerID {
			return 1
		},
		unregister: func(id lifecycle.HandlerID) {},
		args:       []string{},
	}

	result := runWithDeps(deps)
	assert.Equal(t, 2, result, "expected exit code 2 from exitCodeError")
}

func TestRunWithDeps_RegularErrorReturns1(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	deps := runDeps{
		execute: func(context.Context) error {
			return assert.AnError
		},
		telemetryInit:     func() {},
		telemetryShutdown: func(context.Context) {},
		register: func(handler lifecycle.Handler) lifecycle.HandlerID {
			return 1
		},
		unregister: func(id lifecycle.HandlerID) {},
		args:       []string{},
	}

	result := runWithDeps(deps)
	assert.Equal(t, 1, result, "expected exit code 1 for regular error")
}

func TestMainWithDeps_DoesNotExitOnZero(t *testing.T) {
	called := false
	mainWithDeps(func() int { return 0 }, func(int) {
		called = true
	})

	assert.False(t, called)
}

func TestMainWithDeps_ExitsOnNonZero(t *testing.T) {
	var code int
	mainWithDeps(func() int { return 7 }, func(exitCode int) {
		code = exitCode
	})

	assert.Equal(t, 7, code)
}

func TestMainUsesInjectedDependencies(t *testing.T) {
	var gotExitCode int
	withMainDeps(t, func() int { return 5 }, func(exitCode int) { gotExitCode = exitCode })

	main()

	assert.Equal(t, 5, gotExitCode)
}

func TestRun_ReturnsZeroForHelp(t *testing.T) {
	t.Setenv("MMM_DISABLE_TELEMETRY", "true")
	t.Setenv("MMM_TEST", "true")

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"mmm", "--help"}

	assert.Equal(t, 0, run())
}

func TestGetExitCode_ReturnsZeroOnNilError(t *testing.T) {
	assert.Equal(t, 0, getExitCode(nil, 5))
}

func TestGetExitCode_ReturnsDefaultOnNonExitCoder(t *testing.T) {
	assert.Equal(t, 5, getExitCode(assert.AnError, 5))
}

func TestGetExitCode_ReturnsExitCoderValue(t *testing.T) {
	err := &exitCodeError{code: 9, message: "boom"}
	assert.Equal(t, 9, getExitCode(err, 5))
}

func TestExitCodeErrorUsesDefaultMessageWhenMissing(t *testing.T) {
	err := &exitCodeError{code: 4}
	assert.Equal(t, "exit code 4", err.Error())
}

func TestExitCodeErrorUsesProvidedMessage(t *testing.T) {
	err := &exitCodeError{code: 4, message: "custom"}
	assert.Equal(t, "custom", err.Error())
}

func TestPerfExportConfigFromArgs_PerfOutDirAbsolute(t *testing.T) {
	cwd := filepath.FromSlash("/workdir")
	absOut, err := filepath.Abs(filepath.Join(os.TempDir(), "perf"))
	assert.NoError(t, err)
	cfg := perfExportConfigFromArgs([]string{"--perf", "--config", "cfg/modlist.json", "--perf-out-dir", absOut}, cwd)

	assert.Equal(t, absOut, cfg.outDir)
}

func TestPerfExportConfigFromArgs_PerfOutDirWithEquals(t *testing.T) {
	cwd := filepath.FromSlash("/workdir")
	absOut, err := filepath.Abs(filepath.Join(os.TempDir(), "perf"))
	assert.NoError(t, err)
	cfg := perfExportConfigFromArgs([]string{"--perf", "--perf-out-dir=" + absOut}, cwd)

	assert.Equal(t, absOut, cfg.outDir)
}

func TestSessionNameHintFromArgs_DefaultsToTui(t *testing.T) {
	assert.Equal(t, "tui", sessionNameHintFromArgs(nil))
}

func TestSessionNameHintFromArgs_UsesFirstCommand(t *testing.T) {
	args := []string{"--config", "cfg/modlist.json", "--debug", "install"}
	assert.Equal(t, "install", sessionNameHintFromArgs(args))
}

func TestFirstCommandFromArgs_ReturnsNoneWhenOnlyFlags(t *testing.T) {
	command, ok := firstCommandFromArgs([]string{"--config", "cfg/modlist.json", "--perf-out-dir=perf", "-d"})
	assert.False(t, ok)
	assert.Equal(t, "", command)
}

func TestFirstCommandFromArgs_SkipsFlagsAndFindsCommand(t *testing.T) {
	command, ok := firstCommandFromArgs([]string{"--config=cfg/modlist.json", "--perf-out-dir", "perf", "list"})
	assert.True(t, ok)
	assert.Equal(t, "list", command)
}

func TestPerfExportConfigFromArgsFallsBackOnAbsError(t *testing.T) {
	cfg := perfExportConfigFromArgsWithAbs([]string{"--perf", "--config", "modlist.json"}, "", func(string) (string, error) {
		return "", errors.New("abs failed")
	})
	assert.Equal(t, ".", cfg.baseDir)
}

func TestRunWithDepsLogsPerfFailuresWhenDebug(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	var logOutput bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logOutput)
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	deps := runDeps{
		execute:           func(context.Context) error { return nil },
		telemetryInit:     func() {},
		telemetryShutdown: func(context.Context) {},
		register:          func(lifecycle.Handler) lifecycle.HandlerID { return 1 },
		unregister:        func(lifecycle.HandlerID) {},
		args:              []string{"--perf", "--debug"},
		perfExport:        func(perfExportConfig) error { return errors.New("export failed") },
		perfInit:          func(perf.Config) error { return errors.New("init failed") },
		perfShutdown:      func(context.Context) error { return errors.New("shutdown failed") },
	}

	assert.Equal(t, 0, runWithDeps(deps))
	logs := logOutput.String()
	assert.Contains(t, logs, "perf init failed")
	assert.Contains(t, logs, "perf export failed")
	assert.Contains(t, logs, "perf shutdown failed")
}

func TestRunWithDepsUsesProvidedGetwd(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	var called bool
	deps := runDeps{
		execute:           func(context.Context) error { return nil },
		telemetryInit:     func() {},
		telemetryShutdown: func(context.Context) {},
		register:          func(lifecycle.Handler) lifecycle.HandlerID { return 1 },
		unregister:        func(lifecycle.HandlerID) {},
		getwd: func() (string, error) {
			called = true
			return filepath.FromSlash("/tmp"), nil
		},
	}

	assert.Equal(t, 0, runWithDeps(deps))
	assert.True(t, called)
}

func TestRunWritesPerfExportWhenEnabled(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)

	t.Setenv("MMM_DISABLE_TELEMETRY", "true")
	t.Setenv("MMM_TEST", "true")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")

	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"mmm", "--help", "--perf", "--perf-out-dir", tempDir, "--config", configPath}

	assert.Equal(t, 0, run())

	_, err := os.Stat(filepath.Join(tempDir, "mmm-perf.json"))
	assert.NoError(t, err)
}

func assertSpanExists(t *testing.T, spans []perf.SpanSnapshot, name string) {
	t.Helper()
	_, ok := perf.FindSpanByName(spans, name)
	assert.True(t, ok, "expected span %q to exist", name)
}

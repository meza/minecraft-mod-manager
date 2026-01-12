// Package main is the CLI entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/meza/minecraft-mod-manager/cmd/mmm"
	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type mainDeps struct {
	run  func() int
	exit func(int)
}

var mainDepsValue atomic.Value

func init() {
	mainDepsValue.Store(mainDeps{run: run, exit: os.Exit})
}

func main() {
	deps := mainDepsValue.Load().(mainDeps) //nolint:errcheck // atomic.Value enforces consistent type after first Store.
	mainWithDeps(deps.run, deps.exit)
}

func mainWithDeps(runFunc func() int, exitFunc func(int)) {
	exitCode := runFunc()
	if exitCode != 0 {
		exitFunc(exitCode)
	}
}

const (
	perfLifecycleStartup  = "app.lifecycle.startup"
	perfLifecycleExecute  = "app.lifecycle.execute"
	perfLifecycleShutdown = "app.lifecycle.shutdown"
)

const perfShutdownTimeout = 10 * time.Second

// exitCodeError is a private error type that carries a specific exit code.
// Commands can return this error to signal non-standard exit codes
// (e.g., exit code 2 for the test command when version matches current).
type exitCodeError struct {
	code    int
	message string
}

func (exitError *exitCodeError) Error() string {
	if exitError.message != "" {
		return exitError.message
	}
	return fmt.Sprintf("exit code %d", exitError.code)
}

func (exitError *exitCodeError) ExitCode() int {
	return exitError.code
}

// exitCoder is an interface for errors that carry a specific exit code.
// Any error type implementing this interface can signal a non-standard exit code
// to the main function.
type exitCoder interface {
	ExitCode() int
}

// getExitCode extracts the exit code from an error if it implements exitCoder,
// otherwise returns the default code provided.
func getExitCode(err error, defaultCode int) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(exitCoder); ok {
		return exitErr.ExitCode()
	}
	return defaultCode
}

type shutdownTrigger string

const (
	shutdownTriggerGraceful shutdownTrigger = "graceful"
	shutdownTriggerSignal   shutdownTrigger = "signal"
)

func run() int {
	return runWithDeps(runDeps{
		execute: func(ctx context.Context) error {
			return mmm.Command().ExecuteContext(ctx)
		},
		telemetryInit:     telemetry.Init,
		telemetryShutdown: telemetry.Shutdown,
		register:          lifecycle.Register,
		unregister:        lifecycle.Unregister,
		args:              os.Args[1:],
		getwd:             os.Getwd,
		perfExport:        func(cfg perfExportConfig) error { _, err := perf.ExportToFile(cfg.outDir, cfg.baseDir); return err },
	})
}

type runDeps struct {
	execute           func(context.Context) error
	telemetryInit     func()
	telemetryShutdown func(context.Context)
	register          func(lifecycle.Handler) lifecycle.HandlerID
	unregister        func(lifecycle.HandlerID)
	args              []string
	getwd             func() (string, error)
	perfExport        func(perfExportConfig) error
	perfInit          func(perf.Config) error
	perfShutdown      func(context.Context) error
}

func runWithDeps(deps runDeps) int {
	parsedArgs := parsePerfExportArgs(deps.args)
	cwd, cwdErr := resolveWorkingDir(deps.getwd)
	if cwdErr != nil && (parsedArgs.perfEnabled || parsedArgs.debug) {
		log.Printf("working directory unavailable: %v", cwdErr)
	}
	perfCfg := perfExportConfigFromParsedArgs(parsedArgs, cwd)
	configureTelemetry(perfCfg, deps.args)

	perfInit, perfShutdown := resolvePerfHooks(deps)
	initPerf(perfInit, perfCfg)

	rootCtx, rootSpan := perf.StartSpan(context.Background(), "app.lifecycle")
	_, startupSpan := perf.StartSpan(rootCtx, perfLifecycleStartup)
	deps.telemetryInit()

	state := newLifecycleState(rootCtx, rootSpan, perfCfg, deps, perfShutdown)

	handlerID := deps.register(func(sig os.Signal) {
		state.shutdown(shutdownTriggerSignal, sig)
	})
	defer deps.unregister(handlerID)
	defer state.shutdown(shutdownTriggerGraceful, nil)

	startupSpan.End()

	executeCtx, executeSpan := perf.StartSpan(rootCtx, perfLifecycleExecute)
	state.setExecuteSpan(executeSpan)
	err := deps.execute(executeCtx)
	state.endExecute(err == nil)

	if err != nil {
		return getExitCode(err, 1)
	}

	return 0
}

type lifecycleState struct {
	rootCtx           context.Context
	rootSpan          *perf.Span
	perfCfg           perfExportConfig
	perfExport        func(perfExportConfig) error
	telemetryShutdown func(context.Context)
	perfShutdown      func(context.Context) error
	executeSpan       *perf.Span
	executeEndOnce    sync.Once
	shutdownOnce      sync.Once
}

func newLifecycleState(
	rootCtx context.Context,
	rootSpan *perf.Span,
	perfCfg perfExportConfig,
	deps runDeps,
	perfShutdown func(context.Context) error,
) *lifecycleState {
	return &lifecycleState{
		rootCtx:           rootCtx,
		rootSpan:          rootSpan,
		perfCfg:           perfCfg,
		perfExport:        deps.perfExport,
		telemetryShutdown: deps.telemetryShutdown,
		perfShutdown:      perfShutdown,
	}
}

func (state *lifecycleState) setExecuteSpan(span *perf.Span) {
	state.executeSpan = span
}

func (state *lifecycleState) endExecute(success bool) {
	state.executeEndOnce.Do(func() {
		if state.executeSpan == nil {
			return
		}
		state.executeSpan.SetAttributes(attribute.Bool("success", success))
		state.executeSpan.End()
	})
}

func (state *lifecycleState) shutdown(trigger shutdownTrigger, sig os.Signal) {
	state.shutdownOnce.Do(func() {
		state.endExecute(false)

		attrs := []attribute.KeyValue{
			attribute.String("trigger", string(trigger)),
		}
		if sig != nil {
			attrs = append(attrs, attribute.String("signal", sig.String()))
		}

		_, shutdownSpan := perf.StartSpan(state.rootCtx, perfLifecycleShutdown, perf.WithAttributes(attrs...))
		shutdownSpan.End()
		state.rootSpan.End()

		state.telemetryShutdown(state.rootCtx)
		if state.perfCfg.enabled && state.perfExport != nil {
			exportErr := state.perfExport(state.perfCfg)
			if exportErr != nil && state.perfCfg.debug {
				log.Printf("perf export failed: %v", exportErr)
			}
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), perfShutdownTimeout)
		defer cancel()
		shutdownErr := state.perfShutdown(shutdownCtx)
		if shutdownErr != nil && state.perfCfg.debug {
			log.Printf("perf shutdown failed: %v", shutdownErr)
		}
	})
}

func resolveWorkingDir(getwd func() (string, error)) (string, error) {
	if getwd == nil {
		getwd = os.Getwd
	}
	cwd, err := getwd()
	if err != nil {
		return "", err
	}
	return cwd, nil
}

func configureTelemetry(perfCfg perfExportConfig, args []string) {
	telemetry.SetConfigPath(perfCfg.configPath)
	telemetry.SetPerfBaseDir(perfCfg.baseDir)
	telemetry.SetSessionNameHint(sessionNameHintFromArgs(args))
}

func resolvePerfHooks(deps runDeps) (func(perf.Config) error, func(context.Context) error) {
	perfInit := deps.perfInit
	if perfInit == nil {
		perfInit = perf.Init
	}
	perfShutdown := deps.perfShutdown
	if perfShutdown == nil {
		perfShutdown = perf.Shutdown
	}
	return perfInit, perfShutdown
}

func initPerf(perfInit func(perf.Config) error, perfCfg perfExportConfig) {
	initErr := perfInit(perf.Config{Enabled: true})
	if initErr != nil && perfCfg.debug {
		log.Printf("perf init failed: %v", initErr)
	}
}

type perfExportConfig struct {
	enabled bool
	debug   bool

	configPath string
	baseDir    string
	outDir     string
}

func perfExportConfigFromArgs(args []string, cwd string) perfExportConfig {
	parsedArgs := parsePerfExportArgs(args)
	return perfExportConfigFromParsedArgsWithAbs(parsedArgs, cwd, filepath.Abs)
}

func perfExportConfigFromParsedArgs(parsedArgs perfExportArgs, cwd string) perfExportConfig {
	return perfExportConfigFromParsedArgsWithAbs(parsedArgs, cwd, filepath.Abs)
}

func perfExportConfigFromParsedArgsWithAbs(parsedArgs perfExportArgs, cwd string, absPath func(string) (string, error)) perfExportConfig {
	resolvedConfig := resolvePerfConfigPath(parsedArgs, cwd, absPath)
	baseDir := filepath.Dir(resolvedConfig)
	outDir := resolvePerfOutDir(parsedArgs, baseDir)

	return perfExportConfig{
		enabled:    parsedArgs.perfEnabled,
		debug:      parsedArgs.debug,
		configPath: resolvedConfig,
		baseDir:    baseDir,
		outDir:     outDir,
	}
}

func resolvePerfConfigPath(parsedArgs perfExportArgs, cwd string, absPath func(string) (string, error)) string {
	resolvedConfig := parsedArgs.configPath
	if cwd != "" && !filepath.IsAbs(resolvedConfig) {
		resolvedConfig = filepath.Join(cwd, resolvedConfig)
	}
	resolvedConfig, err := absPath(resolvedConfig)
	if err != nil {
		return parsedArgs.configPath
	}
	return resolvedConfig
}

func resolvePerfOutDir(parsedArgs perfExportArgs, baseDir string) string {
	if strings.TrimSpace(parsedArgs.perfOutDir) == "" {
		return baseDir
	}
	if filepath.IsAbs(parsedArgs.perfOutDir) {
		return parsedArgs.perfOutDir
	}
	return filepath.Join(baseDir, parsedArgs.perfOutDir)
}

type perfExportArgs struct {
	configPath  string
	perfEnabled bool
	perfOutDir  string
	debug       bool
}

func parsePerfExportArgs(args []string) perfExportArgs {
	parsedArgs := perfExportArgs{
		configPath: "./modlist.json",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--perf":
			parsedArgs.perfEnabled = true
		case arg == "--debug" || arg == "-d":
			parsedArgs.debug = true
		case strings.HasPrefix(arg, "--config="):
			parsedArgs.configPath = strings.TrimPrefix(arg, "--config=")
		case strings.HasPrefix(arg, "--perf-out-dir="):
			parsedArgs.perfOutDir = strings.TrimPrefix(arg, "--perf-out-dir=")
		case arg == "--config" || arg == "-c":
			if i+1 < len(args) {
				i++
				parsedArgs.configPath = args[i]
			}
		case arg == "--perf-out-dir":
			if i+1 < len(args) {
				i++
				parsedArgs.perfOutDir = args[i]
			}
		}
	}

	return parsedArgs
}

func sessionNameHintFromArgs(args []string) string {
	command, ok := firstCommandFromArgs(args)
	if !ok {
		return "tui"
	}
	return command
}

func firstCommandFromArgs(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "--perf-out-dir=") {
			continue
		}
		if arg == "--config" || arg == "-c" || arg == "--perf-out-dir" {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg, true
	}
	return "", false
}

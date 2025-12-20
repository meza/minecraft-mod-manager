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
	deps := mainDepsValue.Load().(mainDeps)
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

// exitCodeError is a private error type that carries a specific exit code.
// Commands can return this error to signal non-standard exit codes
// (e.g., exit code 2 for the test command when version matches current).
type exitCodeError struct {
	code    int
	message string
}

func (e *exitCodeError) Error() string {
	if e.message != "" {
		return e.message
	}
	return fmt.Sprintf("exit code %d", e.code)
}

func (e *exitCodeError) ExitCode() int {
	return e.code
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
	getwd := deps.getwd
	if getwd == nil {
		getwd = os.Getwd
	}
	cwd, _ := getwd()
	perfCfg := perfExportConfigFromArgs(deps.args, cwd)
	telemetry.SetPerfBaseDir(perfCfg.baseDir)
	telemetry.SetSessionNameHint(sessionNameHintFromArgs(deps.args))

	perfInit := deps.perfInit
	if perfInit == nil {
		perfInit = perf.Init
	}
	perfShutdown := deps.perfShutdown
	if perfShutdown == nil {
		perfShutdown = perf.Shutdown
	}

	if err := perfInit(perf.Config{Enabled: true}); err != nil && perfCfg.debug {
		log.Printf("perf init failed: %v", err)
	}

	rootCtx, rootSpan := perf.StartSpan(context.Background(), "app.lifecycle")

	_, startupSpan := perf.StartSpan(rootCtx, perfLifecycleStartup)

	deps.telemetryInit()

	var shutdownOnce sync.Once
	var executeEndOnce sync.Once
	var executeSpan *perf.Span
	var executeCtx context.Context

	endExecute := func(success bool) {
		executeEndOnce.Do(func() {
			if executeSpan == nil {
				return
			}
			executeSpan.SetAttributes(attribute.Bool("success", success))
			executeSpan.End()
		})
	}

	shutdown := func(trigger shutdownTrigger, sig os.Signal) {
		shutdownOnce.Do(func() {
			endExecute(false)

			attrs := []attribute.KeyValue{
				attribute.String("trigger", string(trigger)),
			}
			if sig != nil {
				attrs = append(attrs, attribute.String("signal", sig.String()))
			}

			_, shutdownSpan := perf.StartSpan(rootCtx, perfLifecycleShutdown, perf.WithAttributes(attrs...))
			shutdownSpan.End()
			rootSpan.End()

			deps.telemetryShutdown(rootCtx)
			if perfCfg.enabled && deps.perfExport != nil {
				if err := deps.perfExport(perfCfg); err != nil && perfCfg.debug {
					log.Printf("perf export failed: %v", err)
				}
			}
			if err := perfShutdown(context.Background()); err != nil && perfCfg.debug {
				log.Printf("perf shutdown failed: %v", err)
			}
		})
	}

	handlerID := deps.register(func(sig os.Signal) {
		shutdown(shutdownTriggerSignal, sig)
	})
	defer deps.unregister(handlerID)
	defer shutdown(shutdownTriggerGraceful, nil)

	startupSpan.End()

	executeCtx, executeSpan = perf.StartSpan(rootCtx, perfLifecycleExecute)
	err := deps.execute(executeCtx)
	endExecute(err == nil)

	if err != nil {
		return getExitCode(err, 1)
	}

	return 0
}

type perfExportConfig struct {
	enabled bool
	debug   bool

	baseDir string
	outDir  string
}

func perfExportConfigFromArgs(args []string, cwd string) perfExportConfig {
	return perfExportConfigFromArgsWithAbs(args, cwd, filepath.Abs)
}

func perfExportConfigFromArgsWithAbs(args []string, cwd string, absPath func(string) (string, error)) perfExportConfig {
	configPath := "./modlist.json"
	perfEnabled := false
	perfOutDir := ""
	debug := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--perf":
			perfEnabled = true
		case arg == "--debug" || arg == "-d":
			debug = true
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
		case strings.HasPrefix(arg, "--perf-out-dir="):
			perfOutDir = strings.TrimPrefix(arg, "--perf-out-dir=")
		case arg == "--config" || arg == "-c":
			if i+1 < len(args) {
				i++
				configPath = args[i]
			}
		case arg == "--perf-out-dir":
			if i+1 < len(args) {
				i++
				perfOutDir = args[i]
			}
		}
	}

	resolvedConfig := configPath
	if cwd != "" && !filepath.IsAbs(resolvedConfig) {
		resolvedConfig = filepath.Join(cwd, resolvedConfig)
	}
	resolvedConfig, err := absPath(resolvedConfig)
	if err != nil {
		resolvedConfig = configPath
	}

	baseDir := filepath.Dir(resolvedConfig)
	outDir := baseDir
	if strings.TrimSpace(perfOutDir) != "" {
		if filepath.IsAbs(perfOutDir) {
			outDir = perfOutDir
		} else {
			outDir = filepath.Join(baseDir, perfOutDir)
		}
	}

	return perfExportConfig{
		enabled: perfEnabled,
		debug:   debug,
		baseDir: baseDir,
		outDir:  outDir,
	}
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

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/joho/godotenv/autoload"
	"github.com/meza/minecraft-mod-manager/cmd/mmm"
	"github.com/meza/minecraft-mod-manager/internal/lifecycle"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	exitCode := run()

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

const (
	perfLifecycleStartup  = "app.lifecycle.startup"
	perfLifecycleExecute  = "app.lifecycle.execute"
	perfLifecycleShutdown = "app.lifecycle.shutdown"
)

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
}

func runWithDeps(deps runDeps) int {
	getwd := deps.getwd
	if getwd == nil {
		getwd = os.Getwd
	}
	cwd, _ := getwd()
	perfCfg := perfExportConfigFromArgs(deps.args, cwd)

	if err := perf.Init(perf.Config{Enabled: true}); err != nil && perfCfg.debug {
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
			deps.telemetryShutdown(rootCtx)

			shutdownSpan.End()
			rootSpan.End()
			if perfCfg.enabled && deps.perfExport != nil {
				if err := deps.perfExport(perfCfg); err != nil && perfCfg.debug {
					log.Printf("perf export failed: %v", err)
				}
			}
			if err := perf.Shutdown(context.Background()); err != nil && perfCfg.debug {
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
		log.Printf("Error executing command: %v", err)
		return 1
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
	resolvedConfig, err := filepath.Abs(resolvedConfig)
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

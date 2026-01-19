package test

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

const defaultTestMaxConcurrency = 4

var runInteractiveInit = initCmd.RunInteractiveInit

type testOptions struct {
	ConfigPath  string
	GameVersion string
	Unattended  bool
	Quiet       bool
	Debug       bool
	LockSync    locksync.PolicyFlags
}

type initRequest struct {
	configPath string
}

type initRunner func(context.Context, *cobra.Command, initRequest) error

type testDeps struct {
	fs              afero.Fs
	logger          *logger.Logger
	output          *output.Output
	clients         platform.Clients
	minecraftClient httpclient.Doer
	fetchMod        fetcher
	latestVersion   latestVersionFetcher
	isValidVersion  versionValidator
	telemetry       func(telemetry.CommandTelemetry)
	runTea          func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error)
	runInit         initRunner
}

type fetcher func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error)

type latestVersionFetcher func(context.Context, httpclient.Doer) (string, error)

type versionValidator func(context.Context, string, httpclient.Doer) (bool, error)

var errInvalidVersion = errors.New("invalid minecraft version")
var errLatestVersionRequired = errors.New("could not determine latest version: please provide an explicit version")
var errVersionValidationUnavailable = errors.New("could not verify minecraft version")
var errUnsupportedMods = errors.New("unsupported mods")

type testRunner func(context.Context, *cobra.Command, testOptions, testDeps) (Result, error)

type Options = testOptions
type Deps = testDeps

type Result struct {
	TargetVersion   string
	ExitCode        int
	UnsupportedMods []models.Mod
	Interactive     bool
}

func Command() *cobra.Command {
	return commandWithRunner(runTest)
}

func commandWithRunner(runner testRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "test [game_version]",
		Aliases: []string{"t"},
		Short:   i18n.T("cmd.test.short", nil),
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestCommand(cmd, args, runner)
		},
	}

	return cmd
}

func runTestCommand(cmd *cobra.Command, args []string, runner testRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.test")

	opts, err := testOptionsFromFlags(cmd, args)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	deps := defaultTestDeps(cmd, opts)
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	result, err := runner(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	handleTestCommandError(cmd, err)
	recordTestTelemetry(deps.telemetry, result, mode, err)
	return err
}

func testOptionsFromFlags(cmd *cobra.Command, args []string) (testOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return testOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return testOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return testOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return testOptions{}, err
	}
	lockSync, err := locksync.PolicyFlagsFromFlags(cmd.Flags())
	if err != nil {
		return testOptions{}, err
	}

	return testOptions{
		ConfigPath:  configPath,
		GameVersion: resolveGameVersion(args),
		Unattended:  unattended,
		Quiet:       quiet,
		Debug:       debug,
		LockSync:    lockSync,
	}, nil
}

func resolveGameVersion(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "latest"
}

func newTestDeps(common cmddeps.CommonDeps) testDeps {
	return testDeps{
		fs:              common.FS,
		logger:          common.Logger,
		output:          common.Output,
		clients:         common.Clients,
		minecraftClient: common.MinecraftClient,
		fetchMod:        platform.FetchMod,
		latestVersion:   minecraft.GetLatestVersion,
		isValidVersion:  minecraft.IsValidVersion,
		telemetry:       telemetry.RecordCommand,
		runTea:          defaultRunTea,
	}
}

func defaultTestDeps(cmd *cobra.Command, opts testOptions) testDeps {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: opts.Quiet,
		Debug: opts.Debug,
	})

	deps := newTestDeps(common)
	deps.runInit = func(ctx context.Context, command *cobra.Command, request initRequest) error {
		if runInteractiveInit == nil {
			return errors.New("missing init runner")
		}
		return runInteractiveInit(ctx, command, initCmd.InteractiveInitDeps{
			FS:              common.FS,
			Output:          common.Output,
			Logger:          common.Logger,
			MinecraftClient: common.MinecraftClient,
			RunTea:          defaultRunTea,
		}, initCmd.InteractiveInitOptions{
			ConfigPath: request.configPath,
			Quiet:      opts.Quiet,
			Debug:      opts.Debug,
		})
	}

	return deps
}

func NewDeps(common cmddeps.CommonDeps) Deps {
	return newTestDeps(common)
}

func handleTestCommandError(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	cmd.SilenceUsage = true
	// Suppress cobra's error printing for errors that we have already logged.
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
}

func recordTestTelemetry(telemetryRecorder func(telemetry.CommandTelemetry), result Result, mode interaction.ExecutionMode, err error) {
	payload := telemetry.CommandTelemetry{
		Command:       "test",
		Success:       err == nil && result.ExitCode == 0,
		Error:         err,
		ExitCode:      result.ExitCode,
		Interactive:   mode.IsInteractive(),
		ExecutionMode: mode.String(),
		Extra: map[string]interface{}{
			"targetVersion": result.TargetVersion,
			"exitCode":      result.ExitCode,
		},
	}
	telemetryRecorder(payload)
}

package install

import (
	"context"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

func Command() *cobra.Command {
	return commandWithRunner(runInstall)
}

func commandWithRunner(runner installRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i"},
		Short:   i18n.T("cmd.install.short"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInstallCommand(cmd, runner)
		},
	}

	return cmd
}

// Run executes the install consistency check without emitting install telemetry.
// It is used by other commands (for example `update`) that need install semantics
// as a prerequisite.
func Run(ctx context.Context, cmd *cobra.Command, configPath string, quiet bool, debug bool) (Result, error) {
	quietForOutput := quiet && !debug
	out := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), quietForOutput)
	log := logger.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, debug)
	limiter := httpclient.DefaultLimiter()

	opts := installOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
	}

	return runInstall(ctx, cmd, opts, defaultInstallDeps(log, out, limiter, func(telemetry.CommandTelemetry) {}))
}

func runInstallCommand(cmd *cobra.Command, runner installRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.install")

	opts, err := installOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	quietMode := tui.QuietDisabled
	if opts.Quiet {
		quietMode = tui.QuietEnabled
	}
	useTUI := tui.ShouldUseTUI(quietMode, cmd.InOrStdin(), cmd.OutOrStdout())

	outWriter := cmd.OutOrStdout()
	errWriter := cmd.ErrOrStderr()
	var logProgram *tui.LogProgram
	if useTUI {
		logProgram = tui.StartLogProgram(cmd.InOrStdin(), outWriter)
		outWriter = logProgram.Writer()
	}

	quietForOutput := opts.Quiet && !opts.Debug
	out := output.New(outWriter, errWriter, quietForOutput)
	log := logger.New(outWriter, errWriter, false, opts.Debug)
	limiter := httpclient.DefaultLimiter()
	deps := defaultInstallDeps(log, out, limiter, telemetry.RecordCommand)

	result, err := runner(ctx, cmd, opts, deps)
	if useTUI {
		err = tui.MergeProgramError(err, logProgram.Stop())
	}
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	recordInstallTelemetry(deps.telemetry, result, err)
	return err
}

func installOptionsFromFlags(cmd *cobra.Command) (installOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return installOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return installOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return installOptions{}, err
	}

	return installOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
	}, nil
}

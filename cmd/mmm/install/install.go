package install

import (
	"context"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
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
		Short:   i18n.T("cmd.install.short", nil),
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
	opts := installOptions{
		ConfigPath: configPath,
		Quiet:      quiet,
		Debug:      debug,
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: quiet,
		Debug: debug,
	})
	return runInstall(ctx, cmd, opts, newInstallDeps(common, func(telemetry.CommandTelemetry) {}))
}

func runInstallCommand(cmd *cobra.Command, runner installRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.install")

	opts, err := installOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	useView := view.SupportsPrompting(cmd.InOrStdin(), cmd.OutOrStdout()) && !opts.Unattended

	outWriter := cmd.OutOrStdout()
	errWriter := cmd.ErrOrStderr()
	var logProgram *view.LogProgram
	if useView {
		logProgram = view.StartLogProgram(cmd.InOrStdin(), outWriter)
		outWriter = logProgram.Writer()
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		OutWriter: outWriter,
		ErrWriter: errWriter,
		Quiet:     opts.Quiet,
		Debug:     opts.Debug,
	})
	deps := newInstallDeps(common, telemetry.RecordCommand)

	result, err := runner(ctx, cmd, opts, deps)
	if useView {
		err = view.MergeProgramError(err, logProgram.Stop())
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
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
	unattended, err := cmd.Flags().GetBool("unattended")
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
		Unattended: unattended,
		Quiet:      quiet,
		Debug:      debug,
	}, nil
}

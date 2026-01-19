package install

import (
	"context"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/locksync"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
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

// RunOptions configures the install preflight used by other commands.
type RunOptions struct {
	ConfigPath   string
	Unattended   bool
	Quiet        bool
	Debug        bool
	LockSync     locksync.PolicyFlags
	SkipLockSync bool
}

// Run executes the install consistency check without emitting install telemetry.
// It is used by other commands (for example `update`) that need install semantics
// as a prerequisite.
func Run(ctx context.Context, cmd *cobra.Command, options RunOptions) (Result, error) {
	opts := installOptions(options)

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: options.Quiet,
		Debug: options.Debug,
	})
	return runInstall(ctx, cmd, opts, newInstallDeps(common, opts, func(telemetry.CommandTelemetry) {}))
}

func runInstallCommand(cmd *cobra.Command, runner installRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.install")

	opts, err := installOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: opts.Quiet,
		Debug: opts.Debug,
	})
	deps := newInstallDeps(common, opts, telemetry.RecordCommand)
	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	result, err := runner(ctx, cmd, opts, deps)
	applyInstallCommandErrorPolicy(cmd, err)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	recordInstallTelemetry(deps.telemetry, result, mode, err)
	return err
}

func applyInstallCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
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
	lockSync, err := locksync.PolicyFlagsFromFlags(cmd.Flags())
	if err != nil {
		return installOptions{}, err
	}

	return installOptions{
		ConfigPath: configPath,
		Unattended: unattended,
		Quiet:      quiet,
		Debug:      debug,
		LockSync:   lockSync,
	}, nil
}

package change

import (
	"errors"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/perf"
)

func Command() *cobra.Command {
	return commandWithRunner(runChange)
}

func commandWithRunner(runner changeRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change [game_version]",
		Short: i18n.T("cmd.change.short", nil),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChangeCommand(cmd, args, runner)
		},
	}

	cmd.Flags().BoolP("force", "f", false, i18n.T("cmd.change.flag.force", nil))
	cmd.Flags().Bool("keep-config", false, i18n.T("cmd.change.flag.keep_config", nil))
	cmd.Flags().Bool("prune-config", false, i18n.T("cmd.change.flag.prune_config", nil))
	cmd.Flags().Bool("disable-skipped", false, i18n.T("cmd.change.flag.disable_skipped", nil))

	return cmd
}

func runChangeCommand(cmd *cobra.Command, args []string, runner changeRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.change")

	opts, err := changeOptionsFromFlags(cmd, args)
	if err != nil {
		handledErr := handleChangeOptionsError(cmd, opts, err)
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return handledErr
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: opts.Quiet,
		Debug: opts.Debug,
	})
	deps := newChangeDeps(common)

	mode := interaction.ResolveExecutionMode(interaction.ExecutionModeInput{
		Unattended: opts.Unattended,
		In:         cmd.InOrStdin(),
		Out:        cmd.OutOrStdout(),
	})

	result, err := runner(ctx, cmd, opts, deps)
	applyChangeCommandErrorPolicy(cmd, err)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	recordChangeTelemetry(deps.telemetry, opts, result, mode, err)
	return err
}

func handleChangeOptionsError(cmd *cobra.Command, opts changeOptions, err error) error {
	var policyErr changePolicyFlagError
	if !errors.As(err, &policyErr) {
		return err
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: opts.Quiet,
		Debug: opts.Debug,
	})
	deps := newChangeDeps(common)
	if outputErr := writeChangePolicyFlagError(cmd, deps, policyErr); outputErr != nil {
		return outputErr
	}

	handledErr := clierrors.MarkHandled(policyErr)
	applyChangeCommandErrorPolicy(cmd, handledErr)
	return handledErr
}

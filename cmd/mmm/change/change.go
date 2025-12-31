package change

import (
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
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

	return cmd
}

func runChangeCommand(cmd *cobra.Command, args []string, runner changeRunner) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.change")

	opts, err := changeOptionsFromFlags(cmd, args)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: opts.Quiet,
		Debug: opts.Debug,
	})
	deps := newChangeDeps(common, cmd, cmd)

	result, err := runner(ctx, cmd, opts, deps)
	applyChangeCommandErrorPolicy(cmd, err)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	recordChangeTelemetry(deps.telemetry, opts, result, err)
	return err
}

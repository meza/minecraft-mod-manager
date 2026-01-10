package update

import (
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

const defaultUpdateMaxConcurrency = 4

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Aliases: []string{"u"},
		Short:   i18n.T("cmd.update.short", nil),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdateCommand(cmd)
		},
	}

	return cmd
}

func runUpdateCommand(cmd *cobra.Command) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.update")

	opts, err := updateOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: opts.Quiet,
		Debug: opts.Debug,
	})
	deps := newUpdateDeps(common, cmd, opts)
	counts, err := runUpdate(ctx, cmd, opts, deps)
	applyUpdateCommandErrorPolicy(cmd, err)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	recordUpdateTelemetry(deps.telemetry, counts.updated, counts.failed, err)
	return err
}

func applyUpdateCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func updateOptionsFromFlags(cmd *cobra.Command) (updateOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return updateOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return updateOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return updateOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return updateOptions{}, err
	}

	return updateOptions{
		ConfigPath: configPath,
		Unattended: unattended,
		Quiet:      quiet,
		Debug:      debug,
	}, nil
}

package update

import (
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
		Short:   i18n.T("cmd.update.short"),
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

	deps := defaultUpdateDeps(cmd, opts)
	counts, err := runUpdate(ctx, cmd, opts, deps)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	if err != nil {
		cmd.SilenceUsage = true
	}

	recordUpdateTelemetry(deps.telemetry, counts.updated, counts.failed, err)
	return err
}

func updateOptionsFromFlags(cmd *cobra.Command) (updateOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
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
		Quiet:      quiet,
		Debug:      debug,
	}, nil
}

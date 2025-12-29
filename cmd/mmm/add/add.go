// Package add implements the add command.
package add

import (
	"context"

	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
)

// Command builds the add command.
func Command() *cobra.Command {
	return commandWithRunner(runAdd)
}

func commandWithRunner(runner addRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <platform> <id>",
		Short: i18n.T("cmd.add.short", nil),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddCommand(cmd, args, runner)
		},
		Aliases:       []string{"a"},
		SilenceUsage:  false,
		SilenceErrors: false,
	}

	cmd.Flags().String("version", "", i18n.T("cmd.add.flag.version", nil))
	cmd.Flags().Bool("allow-version-fallback", false, i18n.T("cmd.add.flag.allow_version_fallback", nil))

	cmd.ValidArgsFunction = func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{string(models.CURSEFORGE), string(models.MODRINTH)}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func runAddCommand(cmd *cobra.Command, args []string, runner addRunner) error {
	ctx, span := startAddSpan(cmd.Context(), args)

	options, err := readAddOptions(cmd, args)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: options.Quiet,
		Debug: options.Debug,
	})
	deps := newAddDeps(common)

	telemetryPayload, err := runner(ctx, span, cmd, options, deps)
	errToReturn := normalizeAddError(err)
	endAddSpan(span, errToReturn)
	recordAddTelemetry(telemetryPayload, errToReturn)
	return errToReturn
}

func startAddSpan(ctx context.Context, args []string) (context.Context, *perf.Span) {
	return perf.StartSpan(ctx, "app.command.add",
		perf.WithAttributes(
			attribute.String("platform", args[0]),
			attribute.String("id", args[1]),
		),
	)
}

func endAddSpan(span *perf.Span, err error) {
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()
}

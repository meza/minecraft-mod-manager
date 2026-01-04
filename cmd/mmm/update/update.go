package update

import (
	"io"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/view"
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

	useView := view.SupportsPrompting(cmd.InOrStdin(), cmd.OutOrStdout()) && !opts.Unattended

	outWriter := cmd.OutOrStdout()
	errWriter := cmd.ErrOrStderr()
	var logProgram *view.LogProgram
	if useView {
		logProgram = view.StartLogProgram(cmd.InOrStdin(), outWriter)
		outWriter = logProgram.Writer()
	}

	installCmd := cmd
	if useView {
		installCmd = &cobra.Command{}
		installCmd.SetOut(wrapWriterWithFD(outWriter, cmd.OutOrStdout()))
		installCmd.SetErr(errWriter)
	}

	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		OutWriter: outWriter,
		ErrWriter: errWriter,
		Quiet:     opts.Quiet,
		Debug:     opts.Debug,
	})
	deps := newUpdateDeps(common, installCmd)
	counts, err := runUpdate(ctx, cmd, opts, deps)
	if useView {
		err = view.MergeProgramError(err, logProgram.Stop())
	}
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

type writerWithFD struct {
	io.Writer
	fd uintptr
}

func (writer writerWithFD) Fd() uintptr {
	return writer.fd
}

func wrapWriterWithFD(outputWriter io.Writer, terminalWriter io.Writer) io.Writer {
	fdWriter, ok := terminalWriter.(interface{ Fd() uintptr })
	if !ok {
		return outputWriter
	}
	return writerWithFD{
		Writer: outputWriter,
		fd:     fdWriter.Fd(),
	}
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

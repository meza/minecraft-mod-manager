package change

import (
	"io"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/tui"
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

	ioConfig := setupChangeIO(cmd, opts)
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		OutWriter: ioConfig.outWriter,
		ErrWriter: ioConfig.errWriter,
		Quiet:     opts.Quiet,
		Debug:     opts.Debug,
	})
	deps := newChangeDeps(common, ioConfig.testCmd, ioConfig.installCmd)

	result, err := runner(ctx, cmd, opts, deps)
	if ioConfig.logProgram != nil {
		err = tui.MergeProgramError(err, ioConfig.logProgram.Stop())
	}
	applyChangeCommandErrorPolicy(cmd, err)
	span.SetAttributes(attribute.Bool("success", err == nil))
	span.End()

	recordChangeTelemetry(deps.telemetry, opts, result, err)
	return err
}

type changeIO struct {
	outWriter  io.Writer
	errWriter  io.Writer
	logProgram *tui.LogProgram
	testCmd    *cobra.Command
	installCmd *cobra.Command
}

func setupChangeIO(cmd *cobra.Command, opts changeOptions) changeIO {
	promptMode := tui.PromptEnabled
	if opts.NonInteractive {
		promptMode = tui.PromptDisabled
	}
	useTUI := tui.ShouldUseTUI(promptMode, cmd.InOrStdin(), cmd.OutOrStdout())

	outWriter := cmd.OutOrStdout()
	errWriter := cmd.ErrOrStderr()
	var logProgram *tui.LogProgram

	testCmd := cmd
	installCmd := cmd
	if useTUI {
		logProgram = tui.StartLogProgram(cmd.InOrStdin(), outWriter)
		outWriter = logProgram.Writer()

		testCmd = &cobra.Command{}
		testCmd.SetOut(wrapWriterWithFD(outWriter, cmd.OutOrStdout()))
		testCmd.SetErr(errWriter)

		installCmd = &cobra.Command{}
		installCmd.SetOut(wrapWriterWithFD(outWriter, cmd.OutOrStdout()))
		installCmd.SetErr(errWriter)
	}

	return changeIO{
		outWriter:  outWriter,
		errWriter:  errWriter,
		logProgram: logProgram,
		testCmd:    testCmd,
		installCmd: installCmd,
	}
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

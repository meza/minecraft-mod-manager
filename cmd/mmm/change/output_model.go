package change

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

func runOutputLines(cmd *cobra.Command, deps changeDeps, writer io.Writer, lines []string) error {
	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}
	return view.RunOutputLines(runTea, view.OutputLinesModel{
		Lines:     lines,
		Output:    writer,
		Separator: view.SectionSeparatorParagraph,
	}, outputProgramOptions(cmd, writer)...)
}

func outputProgramOptions(cmd *cobra.Command, writer io.Writer) []tea.ProgramOption {
	outputWriter := writer
	if outputWriter == nil && cmd != nil {
		outputWriter = cmd.OutOrStdout()
	}
	return []tea.ProgramOption{
		tea.WithInput(nil),
		tea.WithOutput(outputWriter),
		tea.WithoutRenderer(),
	}
}

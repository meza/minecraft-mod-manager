package install

import (
	"io"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

type outputLinesModel = view.OutputLinesModel
type outputLineErrorMsg = view.OutputLineErrorMsg

func outputLineCmd(out io.Writer, line string) tea.Cmd {
	return view.OutputLineCmd(out, line)
}

type lockedWriter struct {
	writer io.Writer
	mutex  sync.Mutex
}

func newLockedWriter(writer io.Writer) io.Writer {
	return &lockedWriter{writer: writer}
}

func (writer *lockedWriter) Write(p []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return writer.writer.Write(p)
}

func outputLinesModelError(result tea.Model) error {
	return view.OutputLinesModelError(result)
}

func runOutputLines(cmd *cobra.Command, deps installDeps, writer io.Writer, lines []string) error {
	runTea := deps.runTea
	if runTea == nil {
		runTea = defaultRunTea
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

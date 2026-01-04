package list

import (
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/writeerrors"
	"github.com/spf13/cobra"
)

type outputLinesModel struct {
	lines  []string
	output io.Writer
	err    error
}

func (model outputLinesModel) Init() tea.Cmd {
	output := model.View()
	if strings.TrimSpace(output) == "" {
		return tea.Quit
	}
	return tea.Sequence(outputLineCmd(model.output, output), tea.Quit)
}

func (model outputLinesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case outputLineErrorMsg:
		model.err = typed.err
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model outputLinesModel) View() string {
	return renderViewSections(model.lines)
}

type outputLineErrorMsg struct {
	err error
}

func outputLineCmd(out io.Writer, line string) tea.Cmd {
	return func() tea.Msg {
		if out == nil {
			return outputLineErrorMsg{err: errors.New("output writer is nil")}
		}
		if _, err := fmt.Fprintln(out, line); err != nil && !writeerrors.IsBrokenPipe(err) {
			return outputLineErrorMsg{err: err}
		}
		return nil
	}
}

func outputLinesModelError(result tea.Model) error {
	switch typed := result.(type) {
	case *outputLinesModel:
		return typed.err
	case outputLinesModel:
		return typed.err
	default:
		return nil
	}
}

func runOutputLines(cmd *cobra.Command, deps listDeps, writer io.Writer, lines []string) error {
	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}
	result, err := runTea(outputLinesModel{
		lines:  lines,
		output: writer,
	}, outputProgramOptions(cmd, writer)...)
	if err != nil {
		return err
	}
	return outputLinesModelError(result)
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

func renderViewSections(sections []string) string {
	stringBuilder, ok := renderViewSectionsBuilder(sections)
	if !ok {
		return ""
	}
	return stringBuilder.String()
}

func renderViewSectionsBuilder(sections []string) (*strings.Builder, bool) {
	stringBuilder := &strings.Builder{}

	for _, section := range sections {
		if section == "" {
			continue
		}
		if stringBuilder.Len() > 0 {
			if err := listWriteString(stringBuilder, "\n\n"); err != nil {
				return stringBuilder, false
			}
		}
		if err := listWriteString(stringBuilder, section); err != nil {
			return stringBuilder, false
		}
	}

	return stringBuilder, true
}

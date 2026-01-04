package view

import (
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/writeerrors"
)

// OutputLinesModel renders a series of lines using a Bubble Tea program.
type OutputLinesModel struct {
	Lines     []string
	Output    io.Writer
	Separator string
	Err       error
}

// OutputLineErrorMsg reports an error while writing output.
type OutputLineErrorMsg struct {
	Err error
}

func (model OutputLinesModel) Init() tea.Cmd {
	output := model.View()
	if strings.TrimSpace(output) == "" {
		return tea.Quit
	}
	return tea.Sequence(OutputLineCmd(model.Output, output), tea.Quit)
}

func (model OutputLinesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case OutputLineErrorMsg:
		model.Err = typed.Err
		return model, tea.Quit
	default:
		return model, nil
	}
}

func (model OutputLinesModel) View() string {
	separator := model.Separator
	if separator == "" {
		separator = SectionSeparatorParagraph
	}
	return RenderViewSections(model.Lines, separator)
}

// OutputLineCmd writes a single output line to the writer.
func OutputLineCmd(out io.Writer, line string) tea.Cmd {
	return func() tea.Msg {
		if out == nil {
			return OutputLineErrorMsg{Err: errors.New("output writer is nil")}
		}
		if _, err := fmt.Fprintln(out, line); err != nil && !writeerrors.IsBrokenPipe(err) {
			return OutputLineErrorMsg{Err: err}
		}
		return nil
	}
}

// OutputLinesModelError extracts a write error from the model if present.
func OutputLinesModelError(result tea.Model) error {
	switch typed := result.(type) {
	case *OutputLinesModel:
		return typed.Err
	case OutputLinesModel:
		return typed.Err
	default:
		return nil
	}
}

// RunOutputLines runs the model using the provided runner and options.
func RunOutputLines(runTea func(tea.Model, ...tea.ProgramOption) (tea.Model, error), model OutputLinesModel, options ...tea.ProgramOption) error {
	result, err := runTea(model, options...)
	if err != nil {
		return err
	}
	return OutputLinesModelError(result)
}

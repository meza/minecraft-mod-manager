package update

import (
	"bytes"
	"errors"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestOutputLinesModelError(t *testing.T) {
	err := errors.New("write failed")
	model := outputLinesModel{Err: err}
	assert.ErrorIs(t, outputLinesModelError(model), err)
	assert.Nil(t, outputLinesModelError(stubTeaModel{}))
}

type stubTeaModel struct{}

func (stubTeaModel) Init() tea.Cmd                       { return nil }
func (stubTeaModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return stubTeaModel{}, nil }
func (stubTeaModel) View() string                        { return "" }

func TestOutputProgramOptionsBranches(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	options := outputProgramOptions(nil, nil)
	assert.Len(t, options, 3)

	options = outputProgramOptions(cmd, nil)
	assert.Len(t, options, 3)

	options = outputProgramOptions(cmd, io.Discard)
	assert.Len(t, options, 3)
}

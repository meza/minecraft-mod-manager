package prune

import (
	"bytes"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestOutputLineCmdWritesLine(t *testing.T) {
	buffer := &bytes.Buffer{}
	cmd := outputLineCmd(buffer, "hello")
	msg := cmd()
	assert.Nil(t, msg)
	assert.Contains(t, buffer.String(), "hello")
}

func TestOutputLineCmdReturnsErrorOnNilWriter(t *testing.T) {
	cmd := outputLineCmd(nil, "hello")
	msg := cmd()
	outputErr, ok := msg.(outputLineErrorMsg)
	if !assert.True(t, ok) {
		return
	}
	assert.Error(t, outputErr.Err)
}

func TestOutputLinesModelErrorHandlesTypes(t *testing.T) {
	primaryErr := errors.New("write failed")
	assert.ErrorIs(t, outputLinesModelError(outputLinesModel{Err: primaryErr}), primaryErr)
	assert.ErrorIs(t, outputLinesModelError(&outputLinesModel{Err: primaryErr}), primaryErr)
	assert.NoError(t, outputLinesModelError(fakeOutputModel{}))
}

func TestOutputProgramOptionsUsesCommandWriter(t *testing.T) {
	buffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buffer)
	options := outputProgramOptions(cmd, nil)
	assert.NotEmpty(t, options)
}

func TestOutputProgramOptionsHandlesNilCommand(t *testing.T) {
	options := outputProgramOptions(nil, &bytes.Buffer{})
	assert.NotEmpty(t, options)
}

type fakeOutputModel struct{}

func (model fakeOutputModel) Init() tea.Cmd {
	return nil
}

func (model fakeOutputModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return model, nil
}

func (model fakeOutputModel) View() string {
	return ""
}

func TestRunOutputLinesUsesDefaultRunner(t *testing.T) {
	originalRunner := runTeaProgram
	t.Cleanup(func() {
		runTeaProgram = originalRunner
	})
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return view.OutputLinesModel{}, nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	assert.NoError(t, runOutputLines(cmd, pruneDeps{}, cmd.OutOrStdout(), []string{"line"}))
}

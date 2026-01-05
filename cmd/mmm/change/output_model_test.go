package change

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunOutputLinesUsesProvidedRunner(t *testing.T) {
	called := false
	deps := changeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			called = true
			return model, nil
		},
	}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runOutputLines(cmd, deps, cmd.OutOrStdout(), []string{"line"})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestRunOutputLinesUsesFallbackRunner(t *testing.T) {
	original := runTeaProgram
	defer func() { runTeaProgram = original }()

	called := false
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		called = true
		return model, nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runOutputLines(cmd, changeDeps{}, cmd.OutOrStdout(), []string{"line"})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestOutputProgramOptionsUsesCmdOutputWhenWriterNil(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	options := outputProgramOptions(cmd, nil)
	assert.Len(t, options, 3)
}

func TestOutputProgramOptionsUsesWriterWhenProvided(t *testing.T) {
	options := outputProgramOptions(nil, &bytes.Buffer{})
	assert.Len(t, options, 3)
}

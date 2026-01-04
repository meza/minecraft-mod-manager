package list

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputLinesModelInitWithEmptyLinesQuits(t *testing.T) {
	model := outputLinesModel{lines: []string{}}
	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestOutputLineCmdReturnsErrorOnNilWriter(t *testing.T) {
	cmd := outputLineCmd(nil, "line")
	msg := cmd()
	typed, ok := msg.(outputLineErrorMsg)
	if !assert.True(t, ok) {
		return
	}
	assert.ErrorContains(t, typed.err, "output writer is nil")
}

func TestOutputLineCmdReturnsErrorOnWriteFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := outputLineCmd(errorWriter{err: writeErr}, "line")
	msg := cmd()
	typed, ok := msg.(outputLineErrorMsg)
	if !assert.True(t, ok) {
		return
	}
	assert.ErrorIs(t, typed.err, writeErr)
}

func TestOutputLinesModelErrorHandlesTypes(t *testing.T) {
	primaryErr := errors.New("primary")
	assert.ErrorIs(t, outputLinesModelError(outputLinesModel{err: primaryErr}), primaryErr)
	assert.ErrorIs(t, outputLinesModelError(&outputLinesModel{err: primaryErr}), primaryErr)
	assert.NoError(t, outputLinesModelError(fakeOutputModel{}))
}

func TestRenderViewSectionsAddsBlankLines(t *testing.T) {
	sections := []string{"one", "two"}
	assert.Equal(t, "one\n\ntwo", renderViewSections(sections))
}

func TestOutputLinesModelUpdateStoresError(t *testing.T) {
	writeErr := errors.New("write failed")
	model := outputLinesModel{}
	updated, cmd := model.Update(outputLineErrorMsg{err: writeErr})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.ErrorIs(t, updated.(outputLinesModel).err, writeErr)
}

func TestOutputLinesModelUpdateIgnoresOtherMessages(t *testing.T) {
	model := outputLinesModel{}
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.Empty(t, updated.(outputLinesModel).err)
}

func TestRunOutputLinesWritesToWriter(t *testing.T) {
	buffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(buffer)
	command.SetErr(io.Discard)

	deps := listDeps{runTea: defaultRunTea}
	err := runOutputLines(command, deps, buffer, []string{"hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buffer.String())
}

func TestRunOutputLinesUsesDefaultRunnerWhenNil(t *testing.T) {
	buffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(buffer)
	command.SetErr(io.Discard)

	err := runOutputLines(command, listDeps{}, buffer, []string{"hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buffer.String())
}

func TestRunOutputLinesReturnsRunTeaError(t *testing.T) {
	command := &cobra.Command{}
	command.SetOut(&bytes.Buffer{})
	command.SetErr(io.Discard)

	runErr := errors.New("run tea failed")
	deps := listDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}

	err := runOutputLines(command, deps, command.OutOrStdout(), []string{"hello"})
	assert.ErrorIs(t, err, runErr)
}

func TestOutputProgramOptionsUsesCommandOutputWhenWriterNil(t *testing.T) {
	buffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(buffer)
	options := outputProgramOptions(command, nil)
	assert.Len(t, options, 3)
}

func TestRenderViewSectionsReturnsEmptyOnWriteError(t *testing.T) {
	originalWriteString := listWriteString
	t.Cleanup(func() {
		listWriteString = originalWriteString
	})
	listWriteString = func(*strings.Builder, string) error {
		return errors.New("write failed")
	}

	assert.Equal(t, "", renderViewSections([]string{"one"}))
}

func TestRenderViewSectionsBuilderReturnsFalseOnWriteError(t *testing.T) {
	originalWriteString := listWriteString
	t.Cleanup(func() {
		listWriteString = originalWriteString
	})
	listWriteString = func(*strings.Builder, string) error {
		return errors.New("write failed")
	}

	builder, ok := renderViewSectionsBuilder([]string{"one"})
	assert.False(t, ok)
	assert.NotNil(t, builder)
}

func TestRenderViewSectionsBuilderReturnsFalseOnSeparatorWriteError(t *testing.T) {
	originalWriteString := listWriteString
	t.Cleanup(func() {
		listWriteString = originalWriteString
	})
	callCount := 0
	listWriteString = func(builder *strings.Builder, value string) error {
		callCount++
		if callCount == 2 {
			return errors.New("write failed")
		}
		_, err := builder.WriteString(value)
		return err
	}

	builder, ok := renderViewSectionsBuilder([]string{"one", "two"})
	assert.False(t, ok)
	assert.NotNil(t, builder)
}

func TestRenderViewSectionsBuilderSkipsEmptySections(t *testing.T) {
	builder, ok := renderViewSectionsBuilder([]string{"one", "", "two"})
	assert.True(t, ok)
	assert.Equal(t, "one\n\ntwo", builder.String())
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

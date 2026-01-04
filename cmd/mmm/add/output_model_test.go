package add

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

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type outputErrorWriter struct {
	err error
}

func (writer outputErrorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func TestOutputLinesModelInitWithEmptyLinesQuits(t *testing.T) {
	model := outputLinesModel{Lines: []string{}}
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
	assert.ErrorContains(t, typed.Err, "output writer is nil")
}

func TestOutputLineCmdReturnsErrorOnWriteFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := outputLineCmd(outputErrorWriter{err: writeErr}, "line")
	msg := cmd()
	typed, ok := msg.(outputLineErrorMsg)
	if !assert.True(t, ok) {
		return
	}
	assert.ErrorIs(t, typed.Err, writeErr)
}

func TestOutputLinesModelErrorHandlesTypes(t *testing.T) {
	primaryErr := errors.New("primary")
	assert.ErrorIs(t, outputLinesModelError(outputLinesModel{Err: primaryErr}), primaryErr)
	assert.ErrorIs(t, outputLinesModelError(&outputLinesModel{Err: primaryErr}), primaryErr)
	assert.NoError(t, outputLinesModelError(fakeOutputModel{}))
}

func TestOutputLinesModelUpdateStoresError(t *testing.T) {
	writeErr := errors.New("write failed")
	model := outputLinesModel{}
	updated, cmd := model.Update(outputLineErrorMsg{Err: writeErr})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.ErrorIs(t, updated.(outputLinesModel).Err, writeErr)
}

func TestOutputLinesModelUpdateIgnoresOtherMessages(t *testing.T) {
	model := outputLinesModel{}
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.Empty(t, updated.(outputLinesModel).Err)
}

func TestRunOutputLinesWritesToWriter(t *testing.T) {
	buffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(buffer)
	command.SetErr(io.Discard)

	deps := addDeps{runTea: defaultRunTea}
	err := runOutputLines(command, deps, buffer, []string{"hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buffer.String())
}

func TestRunOutputLinesUsesDefaultRunnerWhenNil(t *testing.T) {
	buffer := &bytes.Buffer{}
	command := &cobra.Command{}
	command.SetOut(buffer)
	command.SetErr(io.Discard)

	err := runOutputLines(command, addDeps{}, buffer, []string{"hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello\n", buffer.String())
}

func TestRunOutputLinesReturnsRunTeaError(t *testing.T) {
	command := &cobra.Command{}
	command.SetOut(&bytes.Buffer{})
	command.SetErr(io.Discard)

	runErr := errors.New("run tea failed")
	deps := addDeps{
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
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(io.Writer, string) error {
		return errors.New("write failed")
	}

	assert.Equal(t, "", view.RenderViewSections([]string{"one"}, view.SectionSeparatorParagraph))
}

func TestRenderViewSectionsBuilderReturnsFalseOnWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(io.Writer, string) error {
		return errors.New("write failed")
	}

	builder, ok := view.RenderViewSectionsBuilder([]string{"one"}, view.SectionSeparatorParagraph)
	assert.False(t, ok)
	assert.NotNil(t, builder)
}

func TestRenderViewSectionsBuilderReturnsFalseOnSeparatorWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	callCount := 0
	view.WriteString = func(writer io.Writer, value string) error {
		callCount++
		if callCount == 2 {
			return errors.New("write failed")
		}
		builder, ok := writer.(*strings.Builder)
		if !ok {
			return errors.New("unexpected writer")
		}
		_, err := builder.WriteString(value)
		return err
	}

	builder, ok := view.RenderViewSectionsBuilder([]string{"one", "two"}, view.SectionSeparatorParagraph)
	assert.False(t, ok)
	assert.NotNil(t, builder)
}

func TestRenderViewSectionsBuilderSkipsEmptySections(t *testing.T) {
	builder, ok := view.RenderViewSectionsBuilder([]string{"one", "", "two"}, view.SectionSeparatorParagraph)
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

package view

import (
	"bytes"
	"errors"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

type outputErrorWriter struct {
	err error
}

func (writer outputErrorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func TestOutputLinesModelInitWithEmptyLinesQuits(t *testing.T) {
	model := OutputLinesModel{Lines: []string{}}
	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestOutputLinesModelInitWithLinesRunsCommand(t *testing.T) {
	model := OutputLinesModel{Lines: []string{"hello"}, Output: io.Discard}
	cmd := model.Init()
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.False(t, isQuit)
}

func TestOutputLineCmdReturnsErrorOnNilWriter(t *testing.T) {
	cmd := OutputLineCmd(nil, "line")
	msg := cmd()
	typed, ok := msg.(OutputLineErrorMsg)
	if !assert.True(t, ok) {
		return
	}
	assert.ErrorContains(t, typed.Err, "output writer is nil")
}

func TestOutputLineCmdReturnsErrorOnWriteFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := OutputLineCmd(outputErrorWriter{err: writeErr}, "line")
	msg := cmd()
	typed, ok := msg.(OutputLineErrorMsg)
	if !assert.True(t, ok) {
		return
	}
	assert.ErrorIs(t, typed.Err, writeErr)
}

func TestOutputLinesModelErrorHandlesTypes(t *testing.T) {
	primaryErr := errors.New("primary")
	assert.ErrorIs(t, OutputLinesModelError(OutputLinesModel{Err: primaryErr}), primaryErr)
	assert.ErrorIs(t, OutputLinesModelError(&OutputLinesModel{Err: primaryErr}), primaryErr)
	assert.NoError(t, OutputLinesModelError(fakeOutputModel{}))
}

func TestOutputLinesModelUpdateStoresError(t *testing.T) {
	writeErr := errors.New("write failed")
	model := OutputLinesModel{}
	updated, cmd := model.Update(OutputLineErrorMsg{Err: writeErr})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.ErrorIs(t, updated.(OutputLinesModel).Err, writeErr)
}

func TestOutputLinesModelUpdateIgnoresOtherMessages(t *testing.T) {
	model := OutputLinesModel{}
	updated, cmd := model.Update(struct{}{})
	assert.Nil(t, cmd)
	assert.Empty(t, updated.(OutputLinesModel).Err)
}

func TestOutputLinesModelViewUsesDefaultSeparator(t *testing.T) {
	model := OutputLinesModel{Lines: []string{"one", "two"}}
	assert.Equal(t, "one\n\ntwo", model.View())
}

func TestRunOutputLinesReturnsModelError(t *testing.T) {
	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		if typed, ok := model.(OutputLinesModel); ok {
			typed.Err = errors.New("boom")
			return typed, nil
		}
		return model, nil
	}

	err := RunOutputLines(runTea, OutputLinesModel{Lines: []string{"hello"}, Output: io.Discard})
	assert.ErrorContains(t, err, "boom")
}

func TestRunOutputLinesReturnsRunnerError(t *testing.T) {
	runTea := func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("run failed")
	}

	err := RunOutputLines(runTea, OutputLinesModel{Lines: []string{"hello"}, Output: io.Discard})
	assert.ErrorContains(t, err, "run failed")
}

func TestRunOutputLinesWritesOutput(t *testing.T) {
	buffer := &bytes.Buffer{}
	runTea := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		typed := model.(OutputLinesModel)
		cmd := OutputLineCmd(buffer, typed.View())
		_ = cmd()
		return typed, nil
	}

	err := RunOutputLines(runTea, OutputLinesModel{Lines: []string{"hello"}, Output: buffer})
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", buffer.String())
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

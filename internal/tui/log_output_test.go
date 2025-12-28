package tui

import (
	"bytes"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

type fakeSender struct {
	messages []tea.Msg
}

func (sender *fakeSender) Send(msg tea.Msg) {
	sender.messages = append(sender.messages, msg)
}

func TestLogModelCollectsLines(t *testing.T) {
	model := NewLogModel()

	updated, cmd := model.Update(LogLineMsg{Line: "first"})
	model = updated.(LogModel)
	assert.Nil(t, cmd)
	assert.Equal(t, "first", model.View())

	updated, cmd = model.Update(LogLineMsg{Line: "second"})
	model = updated.(LogModel)
	assert.Nil(t, cmd)
	assert.Equal(t, "first\nsecond", model.View())
}

func TestLogModelDoneQuits(t *testing.T) {
	model := NewLogModel()

	updated, cmd := model.Update(LogDoneMsg{})
	assert.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.Equal(t, model, updated)
}

func TestLogModelIgnoresUnknownMessage(t *testing.T) {
	model := NewLogModel()

	updated, cmd := model.Update("noop")
	assert.Nil(t, cmd)
	assert.Equal(t, model, updated)
}

func TestLogLineWriterSplitsAndFlushes(t *testing.T) {
	sender := &fakeSender{}
	writer := NewLogLineWriter(sender)

	written, err := writer.Write([]byte("first\nsecond\nthird"))
	assert.NoError(t, err)
	assert.Equal(t, len("first\nsecond\nthird"), written)
	assert.Len(t, sender.messages, 2)

	assert.Equal(t, "first", sender.messages[0].(LogLineMsg).Line)
	assert.Equal(t, "second", sender.messages[1].(LogLineMsg).Line)

	writer.Flush()
	assert.Len(t, sender.messages, 3)
	assert.Equal(t, "third", sender.messages[2].(LogLineMsg).Line)
}

func TestLogLineWriterCloseStopsOutput(t *testing.T) {
	sender := &fakeSender{}
	writer := NewLogLineWriter(sender)
	writer.Close()

	written, err := writer.Write([]byte("ignored\n"))
	assert.NoError(t, err)
	assert.Equal(t, len("ignored\n"), written)
	assert.Empty(t, sender.messages)

	writer.Flush()
	assert.Empty(t, sender.messages)
}

func TestLogLineWriterNilSenderNoops(t *testing.T) {
	writer := NewLogLineWriter(nil)

	written, err := writer.Write([]byte("ignored\n"))
	assert.NoError(t, err)
	assert.Equal(t, len("ignored\n"), written)

	writer.Flush()
}

func TestLogProgramStopEndsProgram(t *testing.T) {
	restore := SetIsTerminalFuncForTesting(func(_ int) bool { return false })
	defer restore()

	program := StartLogProgram(bytes.NewBuffer(nil), &bytes.Buffer{})
	assert.NotNil(t, program.Writer())

	_, err := program.Writer().Write([]byte("hello\n"))
	assert.NoError(t, err)

	assert.NoError(t, program.Stop())
}

func TestLogProgramNilNoops(t *testing.T) {
	var program *LogProgram
	assert.Nil(t, program.Writer())
	assert.NoError(t, program.Stop())
}

func TestMergeProgramErrorPrefersProgramErrorWhenPrimaryNil(t *testing.T) {
	programErr := errors.New("program")
	assert.Equal(t, programErr, MergeProgramError(nil, programErr))
}

func TestMergeProgramErrorKeepsPrimary(t *testing.T) {
	primary := errors.New("primary")
	programErr := errors.New("program")
	assert.Equal(t, primary, MergeProgramError(primary, programErr))
}

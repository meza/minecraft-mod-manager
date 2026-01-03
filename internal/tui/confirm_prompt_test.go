package tui

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestRunConfirmPromptReturnsTrueForYes(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader("y\n"), &out, ConfirmPrompt{
		Question:    "Proceed?",
		DefaultHint: "y/N",
	})

	assert.NoError(t, err)
	assert.True(t, confirmed)
	assert.Equal(t, "Proceed? (y/N): ", out.String())
}

func TestRunConfirmPromptReturnsFalseForNoWithPrefix(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader("n\n"), &out, ConfirmPrompt{
		Prefix:   "?",
		Question: "Delete files?",
	})

	assert.NoError(t, err)
	assert.False(t, confirmed)
	assert.Equal(t, "? Delete files? ", out.String())
}

func TestRunConfirmPromptReturnsErrorOnWriteFailure(t *testing.T) {
	confirmed, err := RunConfirmPrompt(stringsReader("y\n"), errorWriter{}, ConfirmPrompt{
		Question:    "Proceed?",
		DefaultHint: "y/N",
	})

	assert.Error(t, err)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptReturnsErrorOnReadFailure(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(errorReader{}, &out, ConfirmPrompt{
		Question:    "Proceed?",
		DefaultHint: "y/N",
	})

	assert.Error(t, err)
	assert.False(t, confirmed)
}

func TestRunConfirmPromptReturnsErrorOnEOF(t *testing.T) {
	var out bytes.Buffer
	confirmed, err := RunConfirmPrompt(stringsReader(""), &out, ConfirmPrompt{
		Question:    "Proceed?",
		DefaultHint: "y/N",
	})

	assert.ErrorIs(t, err, io.EOF)
	assert.False(t, confirmed)
}

func stringsReader(value string) io.Reader {
	return bytes.NewBufferString(value)
}

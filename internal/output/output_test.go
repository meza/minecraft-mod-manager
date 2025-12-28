package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func TestOutputLogQuietSuppresses(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	out := New(&stdout, &stderr, true)
	err := out.Log("hello world", LogQuiet)

	assert.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestOutputLogForceBypassesQuiet(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	out := New(&stdout, &stderr, true)
	err := out.Log("hello world", LogForce)

	assert.NoError(t, err)
	assert.Equal(t, "hello world\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestOutputErrorAlwaysWrites(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	out := New(&stdout, &stderr, true)
	err := out.Error("bad thing")

	assert.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Equal(t, "bad thing\n", stderr.String())
}

func TestOutputErrorfAlwaysWrites(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	out := New(&stdout, &stderr, true)
	err := out.Errorf("bad %s", "thing")

	assert.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Equal(t, "bad thing", stderr.String())
}

func TestOutputReturnsWriterErrors(t *testing.T) {
	writeErr := errors.New("write failed")
	logWriter := errorWriter{err: writeErr}
	errWriter := errorWriter{err: writeErr}

	out := New(logWriter, errWriter, false)

	assert.Equal(t, writeErr, out.Log("hello world", LogForce))
	assert.Equal(t, writeErr, out.Error("bad thing"))
	assert.Equal(t, writeErr, out.Errorf("bad %s", "thing"))
}

func TestOutputIgnoresBrokenPipeErrors(t *testing.T) {
	logWriter := errorWriter{err: brokenPipeErr()}
	errWriter := errorWriter{err: brokenPipeErr()}

	out := New(logWriter, errWriter, false)

	assert.NoError(t, out.Log("hello world", LogForce))
	assert.NoError(t, out.Error("bad thing"))
	assert.NoError(t, out.Errorf("bad %s", "thing"))
}

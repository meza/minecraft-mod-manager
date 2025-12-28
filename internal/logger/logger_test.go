package logger

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

func TestLoggerLogQuietSuppresses(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	err := logger.Log("hello world", LogQuiet)

	assert.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerLogForceShowBypassesQuiet(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	err := logger.Log("hello world", LogForce)

	assert.NoError(t, err)
	assert.Equal(t, "hello world\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerLogBypassesQuietWhenDebugEnabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, true)
	err := logger.Log("hello world", LogQuiet)

	assert.NoError(t, err)
	assert.Equal(t, "hello world\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerDebugWritesToStdoutWhenEnabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, false, true)
	err := logger.Debug("hello debug")

	assert.NoError(t, err)
	assert.Equal(t, "hello debug\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerDebugDoesNotWriteWhenDisabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, false, false)
	err := logger.Debug("hello debug")

	assert.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerErrorAlwaysWritesToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	err := logger.Error("bad thing")

	assert.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Equal(t, "bad thing\n", stderr.String())
}

func TestLoggerErrorfAlwaysWritesToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	err := logger.Errorf("bad %s", "thing")

	assert.NoError(t, err)
	assert.Empty(t, stdout.String())
	assert.Equal(t, "bad thing", stderr.String())
}

func TestLoggerHandlesWriterErrors(t *testing.T) {
	writeErr := errors.New("write failed")
	logWriter := errorWriter{err: writeErr}
	errWriter := errorWriter{err: writeErr}

	logger := New(logWriter, errWriter, false, true)

	assert.Equal(t, writeErr, logger.Log("hello world", LogForce))
	assert.Equal(t, writeErr, logger.Debug("hello debug"))
	assert.Equal(t, writeErr, logger.Error("bad thing"))
	assert.Equal(t, writeErr, logger.Errorf("bad %s", "thing"))
}

func TestLoggerIgnoresBrokenPipeErrors(t *testing.T) {
	logWriter := errorWriter{err: brokenPipeErr()}
	errWriter := errorWriter{err: brokenPipeErr()}

	logger := New(logWriter, errWriter, false, true)

	assert.NoError(t, logger.Log("hello world", LogForce))
	assert.NoError(t, logger.Debug("hello debug"))
	assert.NoError(t, logger.Error("bad thing"))
	assert.NoError(t, logger.Errorf("bad %s", "thing"))
}

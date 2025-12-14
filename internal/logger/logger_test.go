package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerLogQuietSuppresses(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	logger.Log("hello world", false)

	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerLogForceShowBypassesQuiet(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	logger.Log("hello world", true)

	assert.Equal(t, "hello world\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerLogBypassesQuietWhenDebugEnabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, true)
	logger.Log("hello world", false)

	assert.Equal(t, "hello world\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerDebugWritesToStdoutWhenEnabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, false, true)
	logger.Debug("hello debug")

	assert.Equal(t, "hello debug\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerDebugDoesNotWriteWhenDisabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, false, false)
	logger.Debug("hello debug")

	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestLoggerErrorAlwaysWritesToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	logger.Error("bad thing")

	assert.Empty(t, stdout.String())
	assert.Equal(t, "bad thing\n", stderr.String())
}

func TestLoggerErrorfAlwaysWritesToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	logger := New(&stdout, &stderr, true, false)
	logger.Errorf("bad %s", "thing")

	assert.Empty(t, stdout.String())
	assert.Equal(t, "bad thing", stderr.String())
}

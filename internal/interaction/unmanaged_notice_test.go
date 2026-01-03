package interaction

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/output"
)

func TestLogUnmanagedNoticeReturnsErrorWhenOutputMissing(t *testing.T) {
	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Message: "missing output",
	})

	assert.Error(t, err)
}

func TestLogUnmanagedNoticeUsesErrorOutput(t *testing.T) {
	var out bytes.Buffer
	var errBuffer bytes.Buffer
	outWriter := output.New(&out, &errBuffer, false)

	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Output:   outWriter,
		Message:  "unmanaged error",
		UseError: true,
	})

	assert.NoError(t, err)
	assert.Empty(t, out.String())
	assert.Equal(t, "unmanaged error\n", errBuffer.String())
}

func TestLogUnmanagedNoticeUsesLogOutput(t *testing.T) {
	var out bytes.Buffer
	var errBuffer bytes.Buffer
	outWriter := output.New(&out, &errBuffer, false)

	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Output:     outWriter,
		Message:    "unmanaged log",
		UseError:   false,
		Visibility: output.LogForce,
	})

	assert.NoError(t, err)
	assert.Equal(t, "unmanaged log\n", out.String())
	assert.Empty(t, errBuffer.String())
}

func TestLogUnmanagedNoticeDefaultsToLogForce(t *testing.T) {
	var out bytes.Buffer
	var errBuffer bytes.Buffer
	outWriter := output.New(&out, &errBuffer, true)

	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Output:  outWriter,
		Message: "forced output",
	})

	assert.NoError(t, err)
	assert.Equal(t, "forced output\n", out.String())
	assert.Empty(t, errBuffer.String())
}

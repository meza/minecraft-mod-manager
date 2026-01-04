package interaction

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

type fakeFDReader struct{}

func (fakeFDReader) Fd() uintptr              { return 1 }
func (fakeFDReader) Read([]byte) (int, error) { return 0, nil }

type fakeFDWriter struct{}

func (fakeFDWriter) Fd() uintptr               { return 1 }
func (fakeFDWriter) Write([]byte) (int, error) { return 0, nil }

func TestResolveExecutionModeUnattended(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	mode := ResolveExecutionMode(ExecutionModeInput{
		Unattended: true,
		In:         fakeFDReader{},
		Out:        fakeFDWriter{},
	})
	assert.Equal(t, ExecutionModeUnattended, mode)
}

func TestResolveExecutionModeInteractive(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restore)

	mode := ResolveExecutionMode(ExecutionModeInput{
		Unattended: false,
		In:         fakeFDReader{},
		Out:        fakeFDWriter{},
	})
	assert.Equal(t, ExecutionModeInteractive, mode)
}

func TestResolveExecutionModeNonTTY(t *testing.T) {
	restore := view.SetIsTerminalFuncForTesting(func(int) bool { return false })
	t.Cleanup(restore)

	mode := ResolveExecutionMode(ExecutionModeInput{
		Unattended: false,
		In:         bytes.NewBuffer(nil),
		Out:        bytes.NewBuffer(nil),
	})
	assert.Equal(t, ExecutionModeNonTTY, mode)
}

func TestExecutionModeIsInteractive(t *testing.T) {
	assert.True(t, ExecutionModeInteractive.IsInteractive())
	assert.False(t, ExecutionModeUnattended.IsInteractive())
	assert.False(t, ExecutionModeNonTTY.IsInteractive())
}

func TestExecutionModeString(t *testing.T) {
	assert.Equal(t, "interactive", ExecutionModeInteractive.String())
	assert.Equal(t, "unattended", ExecutionModeUnattended.String())
	assert.Equal(t, "non_tty", ExecutionModeNonTTY.String())
	assert.Equal(t, "unknown", ExecutionMode(99).String())
}

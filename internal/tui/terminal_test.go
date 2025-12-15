package tui

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeFD struct{}

func (fakeFD) Fd() uintptr { return 1 }

type fakeReader struct{ io.Reader }
type fakeWriter struct{ io.Writer }

func (f fakeReader) Fd() uintptr { return 1 }
func (f fakeWriter) Fd() uintptr { return 1 }

func TestShouldUseTUIHonorsQuiet(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.False(t, ShouldUseTUI(true, fakeReader{}, fakeWriter{}))
}

func TestShouldUseTUIRequiresTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, false)
	defer restore()

	assert.False(t, ShouldUseTUI(false, fakeReader{}, fakeWriter{}))
}

func TestShouldUseTUIWhenTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.True(t, ShouldUseTUI(false, fakeReader{}, fakeWriter{}))
}

func TestProgramOptionsDisablesRendererWithoutTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, false)
	defer restore()

	opts := ProgramOptions(fakeReader{}, fakeWriter{})
	assert.Len(t, opts, 3)
}

func TestProgramOptionsKeepsRendererWithTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	opts := ProgramOptions(fakeReader{}, fakeWriter{})
	assert.Len(t, opts, 2)
}

func TestIsTerminalReaderWithoutFD(t *testing.T) {
	assert.False(t, IsTerminalReader(strings.NewReader("data")))
}

func TestIsTerminalWriterWithoutFD(t *testing.T) {
	assert.False(t, IsTerminalWriter(&strings.Builder{}))
}

func mockTerminalDetection(t *testing.T, result bool) func() {
	t.Helper()
	original := isTerminalFunc
	isTerminalFunc = func(_ int) bool { return result }
	return func() { isTerminalFunc = original }
}

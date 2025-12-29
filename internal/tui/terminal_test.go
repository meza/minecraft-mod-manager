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

func (reader fakeReader) Fd() uintptr { return 1 }
func (writer fakeWriter) Fd() uintptr { return 1 }

func TestShouldUseTUIDisablesWhenPromptDisabled(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.False(t, ShouldUseTUI(PromptDisabled, fakeReader{}, fakeWriter{}))
}

func TestShouldUseTUIRequiresTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, false)
	defer restore()

	assert.False(t, ShouldUseTUI(PromptEnabled, fakeReader{}, fakeWriter{}))
}

func TestShouldUseTUIWhenTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.True(t, ShouldUseTUI(PromptEnabled, fakeReader{}, fakeWriter{}))
}

func TestShouldPromptDisablesWhenPromptDisabled(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.False(t, ShouldPrompt(PromptDisabled, fakeReader{}, fakeWriter{}))
}

func TestShouldPromptRequiresTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, false)
	defer restore()

	assert.False(t, ShouldPrompt(PromptEnabled, fakeReader{}, fakeWriter{}))
}

func TestShouldPromptWhenTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.True(t, ShouldPrompt(PromptEnabled, fakeReader{}, fakeWriter{}))
}

func TestQuietModeEnabled(t *testing.T) {
	assert.False(t, QuietDisabled.Enabled())
	assert.True(t, QuietEnabled.Enabled())
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

func TestSetIsTerminalFuncForTestingRestores(t *testing.T) {
	previous := isTerminalFunc
	defer func() { isTerminalFunc = previous }()

	isTerminalFunc = func(_ int) bool { return false }

	restore := SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	assert.True(t, isTerminalFunc(0))

	restore()
	assert.False(t, isTerminalFunc(0))
}

func mockTerminalDetection(t *testing.T, result bool) func() {
	t.Helper()
	original := isTerminalFunc
	isTerminalFunc = func(_ int) bool { return result }
	return func() { isTerminalFunc = original }
}

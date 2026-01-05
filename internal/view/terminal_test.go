package view

import (
	"io"
	"strings"
	"testing"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

type fakeFD struct{}

func (fakeFD) Fd() uintptr { return 1 }

type fakeReader struct{ io.Reader }
type fakeWriter struct{ io.Writer }

func (reader fakeReader) Fd() uintptr { return 1 }
func (writer fakeWriter) Fd() uintptr { return 1 }

func TestSupportsPromptingRequiresTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, false)
	defer restore()

	assert.False(t, SupportsPrompting(fakeReader{}, fakeWriter{}))
}

func TestSupportsPromptingWhenTerminal(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.True(t, SupportsPrompting(fakeReader{}, fakeWriter{}))
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

func TestSupportsUnicodeUsesDefaultDetector(t *testing.T) {
	assert.IsType(t, true, SupportsUnicode())
}

func TestSupportsUnicodeWhenOverrideReturnsTrue(t *testing.T) {
	restore := SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	defer restore()

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeWhenOverrideReturnsFalse(t *testing.T) {
	restore := SetIsTerminalFuncForTesting(func(_ int) bool { return false })
	defer restore()

	assert.False(t, SupportsUnicode())
}

func TestSupportsUnicodeWhenNotTerminal(t *testing.T) {
	previousFunc := isTerminalFunc
	previousOverride := isTerminalFuncOverridden
	defer func() {
		isTerminalFunc = previousFunc
		isTerminalFuncOverridden = previousOverride
	}()

	isTerminalFunc = func(_ int) bool { return false }
	isTerminalFuncOverridden = false
	assert.False(t, SupportsUnicode())
}

func TestSupportsUnicodeWhenTerminalUsesProfile(t *testing.T) {
	previousFunc := isTerminalFunc
	previousOverride := isTerminalFuncOverridden
	defer func() {
		isTerminalFunc = previousFunc
		isTerminalFuncOverridden = previousOverride
	}()

	isTerminalFunc = func(_ int) bool { return true }
	isTerminalFuncOverridden = false
	assert.IsType(t, true, SupportsUnicode())
}

func TestSupportsColorRequiresTerminalWriter(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	restoreProfile := SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	defer restoreProfile()

	assert.False(t, SupportsColor(&strings.Builder{}))
}

func TestSupportsColorWhenProfileIsAscii(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	restoreProfile := SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	defer restoreProfile()

	assert.False(t, SupportsColor(fakeWriter{}))
}

func TestSupportsColorWhenTerminalAndProfileSupportsColor(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	restoreProfile := SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	defer restoreProfile()

	assert.True(t, SupportsColor(fakeWriter{}))
}

func TestSupportsControlSequencesRequiresTerminalWriter(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.False(t, SupportsControlSequences(&strings.Builder{}))
}

func TestSupportsControlSequencesWhenTerminalWriter(t *testing.T) {
	restore := mockTerminalDetection(t, true)
	defer restore()

	assert.True(t, SupportsControlSequences(fakeWriter{}))
}

func mockTerminalDetection(t *testing.T, result bool) func() {
	t.Helper()
	original := isTerminalFunc
	isTerminalFunc = func(_ int) bool { return result }
	return func() { isTerminalFunc = original }
}

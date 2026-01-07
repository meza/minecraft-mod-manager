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

func TestSupportsUnicodeUsesLocaleUTF8(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.UTF-8")

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeReturnsFalseForCLocale(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "C")

	assert.False(t, SupportsUnicode())
}

func TestSupportsUnicodeDefaultsToTrueWithoutLocale(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LANG", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeUsesLCAll(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("LANG", "C")

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeUsesLCCType(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "en_US.UTF-8")

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeReturnsFalseForCPrefixedLocale(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LANG", "C.ISO8859-1")

	assert.False(t, SupportsUnicode())
}

func TestSupportsUnicodeReturnsFalseForPosixLocale(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "POSIX")

	assert.False(t, SupportsUnicode())
}

func TestSupportsUnicodeReturnsTrueForUTF8NoDash(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.UTF8")

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeReturnsTrueForNonUTF8Locale(t *testing.T) {
	previous := unicodeSupportFunc
	t.Cleanup(func() { unicodeSupportFunc = previous })
	unicodeSupportFunc = defaultUnicodeSupport
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.ISO8859-1")

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeWhenOverrideReturnsTrue(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return true })
	defer restore()

	assert.True(t, SupportsUnicode())
}

func TestSupportsUnicodeWhenOverrideReturnsFalse(t *testing.T) {
	restore := SetUnicodeSupportFuncForTesting(func() bool { return false })
	defer restore()

	assert.False(t, SupportsUnicode())
}

func TestSupportsColorWhenProfileIsAscii(t *testing.T) {
	restoreProfile := SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	defer restoreProfile()

	assert.False(t, SupportsColor(&strings.Builder{}))
}

func TestSupportsColorWhenProfileSupportsColor(t *testing.T) {
	restoreProfile := SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	defer restoreProfile()

	assert.True(t, SupportsColor(&strings.Builder{}))
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

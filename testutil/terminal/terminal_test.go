package terminal

import (
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestBufferOperations(t *testing.T) {
	buffer := NewBuffer()
	_, writeErr := buffer.Write([]byte("hello"))
	require.NoError(t, writeErr)
	require.Equal(t, "hello", buffer.String())
	require.Equal(t, []byte("hello"), buffer.Bytes())

	chunk := make([]byte, 2)
	count, readErr := buffer.Read(chunk)
	require.NoError(t, readErr)
	require.Equal(t, 2, count)
	require.Equal(t, "he", string(chunk))
	require.Equal(t, "llo", buffer.String())

	buffer.Reset()
	require.Empty(t, buffer.String())
}

func TestSnapshotReaderUpdatesBetweenReads(t *testing.T) {
	buffer := NewBuffer()
	_, writeErr := buffer.Write([]byte("abc"))
	require.NoError(t, writeErr)

	reader := NewSnapshotReader(buffer.Bytes)
	data, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Equal(t, "abc", string(data))

	buffer.Reset()
	_, writeErr = buffer.Write([]byte("def"))
	require.NoError(t, writeErr)

	data, readErr = io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Equal(t, "def", string(data))
}

func TestSnapshotReaderNilSource(t *testing.T) {
	reader := NewSnapshotReader(nil)
	data, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Empty(t, data)
}

func TestSnapshotReaderEmptySource(t *testing.T) {
	reader := NewSnapshotReader(func() []byte { return []byte{} })
	data, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)
	require.Empty(t, data)
}

func TestSnapshotReaderPartialReads(t *testing.T) {
	reader := NewSnapshotReader(func() []byte { return []byte("abcd") })
	chunk := make([]byte, 2)

	count, readErr := reader.Read(chunk)
	require.NoError(t, readErr)
	require.Equal(t, 2, count)
	require.Equal(t, "ab", string(chunk))

	count, readErr = reader.Read(chunk)
	require.ErrorIs(t, readErr, io.EOF)
	require.Equal(t, 2, count)
	require.Equal(t, "cd", string(chunk))
}

func TestDeviceAndTerminalDetection(t *testing.T) {
	t.Run("tty", func(t *testing.T) {
		input := NewDeviceWithFD(10)
		output := NewDeviceWithFD(20)
		ApplyTerminalDetection(t, input, output, TTYCapabilities())
		require.True(t, view.SupportsPrompting(input, output))
	})

	t.Run("non-tty", func(t *testing.T) {
		input := NewDeviceWithFD(30)
		output := NewDeviceWithFD(40)
		ApplyTerminalDetection(t, input, output, NonTTYCapabilities())
		require.False(t, view.SupportsPrompting(input, output))
	})
}

func TestApplyTerminalDetectionSingleDevice(t *testing.T) {
	output := NewDeviceWithFD(12)
	ApplyTerminalDetection(t, nil, output, TTYCapabilities())
	require.False(t, view.SupportsPrompting(nil, output))

	ApplyTerminalDetectionWithFDs(t, 1, 2, TTYCapabilities())
	other := NewDeviceWithFD(3)
	require.False(t, view.IsTerminalWriter(other))
}

func TestApplyTerminalDetectionNoDevices(t *testing.T) {
	ApplyTerminalDetection(t, nil, nil, TTYCapabilities())

	restore := view.SetIsTerminalFuncForTesting(func(fd int) bool { return fd == 42 })
	t.Cleanup(restore)

	ApplyTerminalDetectionWithFDs(nil, 1, 2, TTYCapabilities())
	other := NewDeviceWithFD(42)
	require.True(t, view.IsTerminalWriter(other))
}

func TestDeviceBasics(t *testing.T) {
	device := NewDevice()
	require.NotZero(t, device.Fd())
	require.NotNil(t, device.Buffer())

	_, writeErr := device.Write([]byte("ok"))
	require.NoError(t, writeErr)
	require.Equal(t, "ok", device.String())
	require.Equal(t, []byte("ok"), device.Bytes())

	read := make([]byte, 2)
	count, readErr := device.Read(read)
	require.NoError(t, readErr)
	require.Equal(t, 2, count)
	require.Equal(t, "ok", string(read))

	device.Reset()
	require.Empty(t, device.String())
}

func TestCapabilitiesHelpers(t *testing.T) {
	tty := TTYCapabilities()
	require.True(t, tty.InputIsTerminal)
	require.True(t, tty.OutputIsTerminal)

	nonTTY := NonTTYCapabilities()
	require.False(t, nonTTY.InputIsTerminal)
	require.False(t, nonTTY.OutputIsTerminal)
}

func TestNormalizeOutput(t *testing.T) {
	input := "line1\r\n\x1b[31mline2\x1b[0m\r\n"
	normalized := NormalizeOutput(input, NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	require.Equal(t, "line1\nline2", normalized)

	limited := NormalizeOutput("a\nb", NormalizeOptions{RowLimit: 3, PadRows: true})
	require.Equal(t, "a\nb\n", limited)

	cropped := NormalizeOutput("a\nb\nc", NormalizeOptions{RowLimit: 2})
	require.Equal(t, "b\nc", cropped)

	trimmed := NormalizeOutput("\n\n", NormalizeOptions{TrimTrailingEmptyLines: true})
	require.Empty(t, trimmed)
}

func TestEncodeSGRMouseSequence(t *testing.T) {
	sequence := EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress, X: 0, Y: 0})
	require.Equal(t, "\x1b[<0;1;1M", sequence)

	release := EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 0, Y: 0})
	require.Equal(t, "\x1b[<0;1;1m", release)

	wheel := EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress, X: 2, Y: 3, Shift: true})
	require.True(t, strings.HasPrefix(wheel, "\x1b[<"))
	require.Contains(t, wheel, ";3;4")

	motion := EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion, X: 2, Y: 2, Alt: true, Ctrl: true})
	require.Contains(t, motion, ";3;3")

	wheelRelease := EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionRelease, X: 1, Y: 1})
	require.Contains(t, wheelRelease, "M")

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonRight, Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonMiddle, Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonWheelLeft, Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonWheelRight, Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonBackward, Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonForward, Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButtonNone, Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)

	sequence = EncodeSGRMouseSequence(MouseEvent{Button: tea.MouseButton(255), Action: tea.MouseActionPress, X: 1, Y: 1})
	require.NotEmpty(t, sequence)
}

func TestSizeString(t *testing.T) {
	size := Size{Columns: 80, Rows: 25}
	require.Equal(t, "80x25", size.String())
}

func TestApplyFixturesReturnsDeterministicRand(t *testing.T) {
	randValue := ApplyFixtures(t, WithRandSeed(1337), WithColorProfile(termenv.TrueColor), WithUnicodeEnabled(false), WithoutMMMTestEnv())
	require.NotNil(t, randValue)
	//nolint:gosec // Deterministic math/rand for test expectation.
	expected := rand.New(rand.NewSource(1337)).Int63()
	require.Equal(t, expected, randValue.Int63())
}

func TestApplyFixturesOverridesViewSupport(t *testing.T) {
	ApplyFixtures(t, WithColorProfile(termenv.TrueColor), WithUnicodeEnabled(false))
	require.True(t, view.SupportsColor(io.Discard))
	require.False(t, view.SupportsUnicode())
}

func TestApplyFixturesNilTest(t *testing.T) {
	require.Nil(t, ApplyFixtures(nil))
}

func TestApplyFixturesDefault(t *testing.T) {
	randValue := ApplyFixtures(t)
	require.Nil(t, randValue)

	value, present := os.LookupEnv("MMM_TEST")
	require.True(t, present)
	require.Equal(t, "true", value)
}

func TestApplyFixturesBlocksLiveHTTP(t *testing.T) {
	ApplyFixtures(t)

	request, err := http.NewRequest(http.MethodGet, "https://example.invalid", nil)
	require.NoError(t, err)

	response, err := http.DefaultTransport.RoundTrip(request)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "vcr: live HTTP disabled in tests")
}

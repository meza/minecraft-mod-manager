package pty

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type closerFunc func() error

func (closer closerFunc) Close() error {
	return closer()
}

func TestValidateSizeValid(t *testing.T) {
	require.NoError(t, validateSize(terminal.Size{Columns: 80, Rows: 25}))
}

func TestValidateSizeInvalid(t *testing.T) {
	require.Error(t, validateSize(terminal.Size{Columns: 0, Rows: 10}))

	maxUint16 := int(^uint16(0))
	require.Error(t, validateSize(terminal.Size{Columns: maxUint16 + 1, Rows: 1}))
}

func TestApplyPTYSizeNil(t *testing.T) {
	require.NoError(t, applyPTYSize(ptySetup{}, nil))
}

func TestApplyPTYSizeInvalid(t *testing.T) {
	err := applyPTYSize(ptySetup{}, &terminal.Size{Columns: 0, Rows: 1})
	require.Error(t, err)
	require.Contains(t, err.Error(), "pty size invalid")
}

func TestApplyPTYSizeSetError(t *testing.T) {
	file, err := os.CreateTemp("", "pty-size")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	original := setPTYSize
	setPTYSize = func(_ terminalFile, _ terminal.Size) error {
		return errors.New("boom")
	}
	t.Cleanup(func() { setPTYSize = original })

	setup := ptySetup{master: file}
	err = applyPTYSize(setup, &terminal.Size{Columns: 80, Rows: 25})
	require.Error(t, err)
	require.Contains(t, err.Error(), "pty set size failed")
}

func TestNewSessionNil(t *testing.T) {
	require.Nil(t, NewSession(nil))
}

func TestNewSessionOpenError(t *testing.T) {
	originalOpen := openPTY
	originalFatalf := sessionFatalf
	t.Cleanup(func() {
		openPTY = originalOpen
		sessionFatalf = originalFatalf
	})

	openPTY = func() (ptySetup, error) {
		return ptySetup{}, errors.New("boom")
	}

	called := false
	sessionFatalf = func(test testing.TB, format string, args ...any) {
		called = true
	}

	require.Nil(t, NewSession(t))
	require.True(t, called)
}

func TestNewSessionSizeError(t *testing.T) {
	originalSet := setPTYSize
	originalFatalf := sessionFatalf
	t.Cleanup(func() {
		setPTYSize = originalSet
		sessionFatalf = originalFatalf
	})

	setPTYSize = func(_ terminalFile, _ terminal.Size) error {
		return errors.New("boom")
	}

	called := false
	sessionFatalf = func(test testing.TB, format string, args ...any) {
		called = true
	}

	require.Nil(t, NewSession(t, WithSize(terminal.Size{Columns: 80, Rows: 25})))
	require.True(t, called)
}

func TestStartCaptureNil(t *testing.T) {
	startCapture(nil)
	registerCleanup(nil, nil)
}

func TestCloseFileIgnoresClosedFile(t *testing.T) {
	file, err := os.CreateTemp("", "pty-close")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	require.NoError(t, closeFile(file))
	require.NoError(t, closeFile(file))
}

func TestCloseFileReturnsOtherErrors(t *testing.T) {
	file, err := os.CreateTemp("", "pty-close-error")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	originalCloser := fileCloser
	fileCloser = func(_ terminalFile) error {
		return errors.New("boom")
	}
	t.Cleanup(func() {
		fileCloser = originalCloser
		require.NoError(t, originalCloser(file))
	})

	require.Error(t, closeFile(file))
}

func TestCloseFileTimesOutOnBlockedClose(t *testing.T) {
	file, err := os.CreateTemp("", "pty-close-timeout")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	block := make(chan struct{})
	originalCloser := fileCloser
	originalTimeout := closeTimeout
	fileCloser = func(_ terminalFile) error {
		<-block
		return nil
	}
	closeTimeout = 10 * time.Millisecond
	t.Cleanup(func() {
		closeTimeout = originalTimeout
		fileCloser = originalCloser
		close(block)
		require.NoError(t, originalCloser(file))
	})

	err = closeFile(file)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out closing PTY handle")
}

func TestCloseWithErrorIncludesCloseErrors(t *testing.T) {
	badFile := os.NewFile(^uintptr(0), "bad")
	err := closeWithError(ptySetup{master: badFile}, errors.New("boom"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}

func TestRegisterCleanupReportsCloseError(t *testing.T) {
	session := &Session{
		readDone:  make(chan struct{}),
		readError: make(chan error, 1),
	}
	close(session.readDone)

	originalHandler := cleanupErrorHandler
	called := false
	cleanupErrorHandler = func(test testing.TB, err error) {
		called = true
	}
	t.Cleanup(func() {
		cleanupErrorHandler = originalHandler
		require.True(t, called)
	})

	registerCleanup(t, session)
}

func TestSessionNilGuards(t *testing.T) {
	var session *Session

	_, err := session.SendInput([]byte("x"))
	require.Error(t, err)

	require.Error(t, session.Resize(terminal.Size{Columns: 1, Rows: 1}))
	require.Nil(t, session.OutputBytes())
	require.Empty(t, session.OutputString())
	require.Nil(t, session.OutputReader())
	require.Nil(t, session.Master())
	require.Nil(t, session.Input())
	require.Nil(t, session.Output())
	require.Nil(t, session.Slave())

	session.WaitForOutput(t, func(_ []byte) bool { return true })
	session.WaitForOutputAndClose(t, func(_ []byte) bool { return true })
	require.NoError(t, session.Close())
}

func TestSessionSlaveUsesOutputWhenSplit(t *testing.T) {
	input, err := os.CreateTemp("", "pty-input")
	require.NoError(t, err)
	defer os.Remove(input.Name())
	defer input.Close()

	output, err := os.CreateTemp("", "pty-output")
	require.NoError(t, err)
	defer os.Remove(output.Name())
	defer output.Close()

	session := &Session{input: input, output: output}

	require.Equal(t, output, session.Slave())
}

func TestSessionSlaveUsesInputWhenShared(t *testing.T) {
	shared, err := os.CreateTemp("", "pty-shared")
	require.NoError(t, err)
	defer os.Remove(shared.Name())
	defer shared.Close()

	session := &Session{input: shared, output: shared}

	require.Equal(t, shared, session.Slave())
}

func TestSessionCloseIsIdempotent(t *testing.T) {
	session := NewSession(t, WithSize(terminal.Size{Columns: 80, Rows: 25}))
	require.NotNil(t, session)
	require.NotNil(t, session.Master())
	require.NotNil(t, session.Input())
	require.NotNil(t, session.Output())

	require.NoError(t, session.Resize(terminal.Size{Columns: 100, Rows: 30}))
	require.Empty(t, session.OutputBytes())
	require.Equal(t, string(session.OutputBytes()), session.OutputString())
	require.NotNil(t, session.OutputReader())

	require.NoError(t, session.Close())
	require.NoError(t, session.Close())
}

func TestSessionCloseWithoutReader(t *testing.T) {
	master, err := os.CreateTemp("", "pty-master")
	require.NoError(t, err)
	defer os.Remove(master.Name())

	input, err := os.CreateTemp("", "pty-input")
	require.NoError(t, err)
	defer os.Remove(input.Name())

	output, err := os.CreateTemp("", "pty-output")
	require.NoError(t, err)
	defer os.Remove(output.Name())

	session := &Session{master: master, input: input, output: output}
	require.NoError(t, session.Close())
}

func TestSessionCloseReportsExtraCloserError(t *testing.T) {
	session := &Session{
		extraClosers: []io.Closer{
			closerFunc(func() error { return errors.New("boom") }),
			nil,
		},
	}

	err := session.Close()
	require.Error(t, err)
}

func TestCloseSetupReportsExtraCloserError(t *testing.T) {
	setup := ptySetup{
		extraClosers: []io.Closer{
			closerFunc(func() error { return errors.New("boom") }),
			nil,
		},
	}

	err := closeSetup(setup)
	require.Error(t, err)
}

func TestSessionCloseMissingReadError(t *testing.T) {
	session := &Session{
		readDone:  make(chan struct{}),
		readError: make(chan error, 1),
	}
	close(session.readDone)

	err := session.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing PTY read error")
}

func TestSessionCloseWithoutReadErrorChannel(t *testing.T) {
	session := &Session{
		readDone: make(chan struct{}),
	}
	close(session.readDone)

	require.NoError(t, session.Close())
}

func TestSessionCloseTimeout(t *testing.T) {
	session := &Session{
		readDone:  make(chan struct{}),
		readError: make(chan error, 1),
	}

	err := session.Close()
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out waiting for PTY output")
}

func TestSessionResizeInvalidSize(t *testing.T) {
	session := NewSession(t, WithSize(terminal.Size{Columns: 80, Rows: 25}))
	require.NotNil(t, session)

	require.Error(t, session.Resize(terminal.Size{Columns: 0, Rows: 25}))
	require.NoError(t, session.Close())
}

func TestSessionNonTTYCapabilities(t *testing.T) {
	session := NewSession(t, WithCapabilities(terminal.NonTTYCapabilities()))
	require.NotNil(t, session)

	require.NoError(t, session.Close())
}

func TestNormalizeReadError(t *testing.T) {
	require.NoError(t, normalizeReadError(nil))
	require.NoError(t, normalizeReadError(os.ErrClosed))
	require.Error(t, normalizeReadError(errors.New("boom")))
}

func TestWaitForIntervalOption(t *testing.T) {
	session := NewSession(t, WithSize(terminal.Size{Columns: 80, Rows: 25}))
	require.NotNil(t, session)

	session.WaitForOutput(t, func(_ []byte) bool { return true }, WithWaitInterval(time.Millisecond))
	require.NoError(t, session.Close())
}

func TestWaitForOutputAndClose(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := NewSession(t, WithSize(terminal.Size{Columns: 80, Rows: 25}))
	require.NotNil(t, session)

	session.WaitForOutputAndClose(t, func(_ []byte) bool { return true })
	require.NoError(t, session.Close())
}

func TestWaitForOutputAndCloseReportsCloseError(t *testing.T) {
	session := &Session{
		outputReader: terminal.NewSnapshotReader(func() []byte { return []byte("ready") }),
		readDone:     make(chan struct{}),
		readError:    make(chan error, 1),
	}
	close(session.readDone)
	session.readError <- errors.New("boom")

	originalHandler := cleanupErrorHandler
	called := false
	cleanupErrorHandler = func(test testing.TB, err error) {
		called = true
	}
	t.Cleanup(func() {
		cleanupErrorHandler = originalHandler
		require.True(t, called)
	})

	session.WaitForOutputAndClose(t, func(data []byte) bool { return len(data) > 0 })
	require.Error(t, session.Close())
}

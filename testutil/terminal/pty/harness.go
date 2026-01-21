package pty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	charmteatest "github.com/charmbracelet/x/exp/teatest"

	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type terminalFile interface {
	io.Reader
	io.Writer
	Fd() uintptr
	Close() error
}

type ptySetup struct {
	master       terminalFile
	input        terminalFile
	output       terminalFile
	extraClosers []io.Closer
}

// Session manages a PTY for interactive terminal snapshot testing.
type Session struct {
	master       terminalFile
	input        terminalFile
	output       terminalFile
	extraClosers []io.Closer
	outputBuffer *terminal.Buffer
	outputReader *terminal.SnapshotReader
	readDone     chan struct{}
	readError    chan error
	closeOnce    sync.Once
	closeErr     error
}

var (
	openPTY             = openPTYDefault
	setPTYSize          = setPTYSizeDefault
	fileCloser          = func(file terminalFile) error { return file.Close() }
	closeTimeout        = 5 * time.Second
	sessionFatalf       = func(test testing.TB, format string, args ...any) { test.Fatalf(format, args...) }
	cleanupErrorHandler = func(test testing.TB, err error) { test.Errorf("pty session close failed: %v", err) }
)

// Option configures a PTY session.
type Option func(options *Options)

// Options describe session configuration.
type Options struct {
	Size         *terminal.Size
	Capabilities terminal.Capabilities
}

// WithSize sets the initial PTY size.
func WithSize(size terminal.Size) Option {
	return func(options *Options) {
		options.Size = &size
	}
}

// WithCapabilities overrides terminal detection for the slave.
func WithCapabilities(capabilities terminal.Capabilities) Option {
	return func(options *Options) {
		options.Capabilities = capabilities
	}
}

// NewSession opens a PTY and starts capturing output.
func NewSession(test testing.TB, options ...Option) *Session {
	if test == nil {
		return nil
	}

	configured := Options{Capabilities: terminal.TTYCapabilities()}
	for _, option := range options {
		option(&configured)
	}

	setup, err := openPTY()
	if err != nil {
		sessionFatalf(test, "pty open failed: %v", err)
		return nil
	}

	if sizeErr := applyPTYSize(setup, configured.Size); sizeErr != nil {
		sessionFatalf(test, "%v", sizeErr)
		return nil
	}

	terminal.ApplyTerminalDetectionWithFDs(test, int(setup.input.Fd()), int(setup.output.Fd()), configured.Capabilities)

	session := createSession(setup)
	startCapture(session)
	registerCleanup(test, session)

	return session
}

func applyPTYSize(setup ptySetup, size *terminal.Size) error {
	if size == nil {
		return nil
	}
	sizeErr := validateSize(*size)
	if sizeErr != nil {
		return closeWithError(setup, fmt.Errorf("pty size invalid: %w", sizeErr))
	}
	if setErr := setPTYSize(setup.master, *size); setErr != nil {
		return closeWithError(setup, fmt.Errorf("pty set size failed: %w", setErr))
	}
	return nil
}

func closeWithError(setup ptySetup, err error) error {
	return errors.Join(err, closeSetup(setup))
}

func createSession(setup ptySetup) *Session {
	outputBuffer := terminal.NewBuffer()
	outputReader := terminal.NewSnapshotReader(outputBuffer.Bytes)

	return &Session{
		master:       setup.master,
		input:        setup.input,
		output:       setup.output,
		extraClosers: setup.extraClosers,
		outputBuffer: outputBuffer,
		outputReader: outputReader,
		readDone:     make(chan struct{}),
		readError:    make(chan error, 1),
	}
}

func startCapture(session *Session) {
	if session == nil {
		return
	}
	go func() {
		_, copyErr := io.Copy(session.outputBuffer, session.master)
		session.readError <- copyErr
		close(session.readDone)
	}()
}

func registerCleanup(test testing.TB, session *Session) {
	if test == nil || session == nil {
		return
	}
	test.Cleanup(func() {
		if closeErr := session.Close(); closeErr != nil {
			cleanupErrorHandler(test, closeErr)
		}
	})
}

// Master returns the PTY master.
func (session *Session) Master() terminalFile {
	if session == nil {
		return nil
	}
	return session.master
}

// Input returns the PTY input handle.
func (session *Session) Input() terminalFile {
	if session == nil {
		return nil
	}
	return session.input
}

// Output returns the PTY output handle.
func (session *Session) Output() terminalFile {
	if session == nil {
		return nil
	}
	return session.output
}

// Slave returns the PTY slave handle for legacy usage.
func (session *Session) Slave() terminalFile {
	if session == nil {
		return nil
	}
	if session.input == session.output {
		return session.input
	}
	return session.output
}

// Resize updates the PTY size.
func (session *Session) Resize(size terminal.Size) error {
	if session == nil || session.master == nil {
		return errors.New("pty session closed")
	}
	sizeErr := validateSize(size)
	if sizeErr != nil {
		return sizeErr
	}
	return setPTYSize(session.master, size)
}

// SendInput writes raw input to the PTY master.
func (session *Session) SendInput(value []byte) (int, error) {
	if session == nil || session.master == nil {
		return 0, errors.New("pty session closed")
	}
	return session.master.Write(value)
}

// OutputBytes returns a snapshot of captured output.
func (session *Session) OutputBytes() []byte {
	if session == nil || session.outputBuffer == nil {
		return nil
	}
	return session.outputBuffer.Bytes()
}

// OutputString returns a snapshot of captured output.
func (session *Session) OutputString() string {
	return string(session.OutputBytes())
}

// OutputReader returns a reader that snapshots output per read cycle.
func (session *Session) OutputReader() *terminal.SnapshotReader {
	if session == nil {
		return nil
	}
	return session.outputReader
}

// WaitForOutput waits for output to satisfy the predicate.
func (session *Session) WaitForOutput(test testing.TB, predicate func([]byte) bool, options ...WaitOption) {
	if session == nil || test == nil {
		return
	}

	waitOptions := waitConfig{duration: time.Second, checkInterval: 50 * time.Millisecond}
	for _, option := range options {
		option(&waitOptions)
	}

	var waitForOptions []charmteatest.WaitForOption
	waitForOptions = append(waitForOptions, charmteatest.WithDuration(waitOptions.duration))
	waitForOptions = append(waitForOptions, charmteatest.WithCheckInterval(waitOptions.checkInterval))

	charmteatest.WaitFor(test, session.OutputReader(), predicate, waitForOptions...)
}

// WaitForOutputAndClose waits until the output satisfies the predicate, then closes the session.
func (session *Session) WaitForOutputAndClose(test testing.TB, predicate func([]byte) bool, options ...WaitOption) {
	if session == nil || test == nil {
		return
	}
	session.WaitForOutput(test, predicate, options...)
	if closeErr := session.Close(); closeErr != nil {
		cleanupErrorHandler(test, closeErr)
	}
}

// Close shuts down the PTY and waits for output capture.
func (session *Session) Close() error {
	if session == nil {
		return nil
	}
	session.closeOnce.Do(func() {
		session.closeHandles()
		session.closeExtras()
		session.waitForReadCompletion()
	})
	return session.closeErr
}

func (session *Session) closeHandles() {
	session.joinCloseErr(closeFile(session.input))
	session.joinCloseErr(closeFile(session.output))
	session.joinCloseErr(closeFile(session.master))
}

func (session *Session) closeExtras() {
	for _, extraCloser := range session.extraClosers {
		if extraCloser == nil {
			continue
		}
		session.joinCloseErr(extraCloser.Close())
	}
}

func (session *Session) waitForReadCompletion() {
	if session.readDone == nil {
		return
	}
	select {
	case <-session.readDone:
		session.captureReadError()
	case <-time.After(2 * time.Second):
		session.joinCloseErr(errors.New("timed out waiting for PTY output"))
	}
}

func (session *Session) captureReadError() {
	if session.readError == nil {
		return
	}
	select {
	case readErr := <-session.readError:
		session.joinCloseErr(normalizeReadError(readErr))
	default:
		session.joinCloseErr(errors.New("missing PTY read error"))
	}
}

func (session *Session) joinCloseErr(err error) {
	if err == nil {
		return
	}
	session.closeErr = errors.Join(session.closeErr, err)
}

func validateSize(size terminal.Size) error {
	if size.Columns <= 0 || size.Rows <= 0 {
		return fmt.Errorf("terminal size must be positive: %s", size)
	}
	maxUint16 := int(^uint16(0))
	if size.Columns > maxUint16 || size.Rows > maxUint16 {
		return fmt.Errorf("terminal size out of range: %s", size)
	}
	return nil
}

func closeSetup(setup ptySetup) error {
	closerErr := closeFile(setup.input)
	closerErr = errors.Join(closerErr, closeFile(setup.output))
	closerErr = errors.Join(closerErr, closeFile(setup.master))
	for _, extraCloser := range setup.extraClosers {
		if extraCloser == nil {
			continue
		}
		closerErr = errors.Join(closerErr, extraCloser.Close())
	}
	return closerErr
}

func closeFile(file terminalFile) error {
	if file == nil {
		return nil
	}
	closer := fileCloser
	timeout := closeTimeout
	closeErr := make(chan error, 1)
	go func() {
		closeErr <- closer(file)
	}()
	select {
	case err := <-closeErr:
		if err != nil && !errors.Is(err, os.ErrClosed) {
			return err
		}
	case <-time.After(timeout):
		return errors.New("timed out closing PTY handle")
	}
	return nil
}

// WaitOption configures WaitForOutput.
type WaitOption func(config *waitConfig)

type waitConfig struct {
	duration      time.Duration
	checkInterval time.Duration
}

// WithWaitDuration sets the maximum wait duration.
func WithWaitDuration(duration time.Duration) WaitOption {
	return func(config *waitConfig) {
		config.duration = duration
	}
}

// WithWaitInterval sets the polling interval.
func WithWaitInterval(interval time.Duration) WaitOption {
	return func(config *waitConfig) {
		config.checkInterval = interval
	}
}

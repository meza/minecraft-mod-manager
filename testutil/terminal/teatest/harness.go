package teatest

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	charmteatest "github.com/charmbracelet/x/exp/teatest"

	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

var sessionFatalf = func(test testing.TB, format string, args ...any) { test.Fatalf(format, args...) }

// Session manages an in-process Bubble Tea program with deterministic I/O capture.
type Session struct {
	program      *tea.Program
	inputDevice  *terminal.Device
	outputDevice *terminal.Device
	outputReader *terminal.SnapshotReader

	finalModel tea.Model
	runError   error
	doneCh     chan struct{}
}

// Option configures a Session.
type Option func(options *Options)

// Options describe session configuration.
type Options struct {
	Capabilities   terminal.Capabilities
	InitialSize    *terminal.Size
	ProgramOptions []tea.ProgramOption
}

// WithCapabilities sets terminal capabilities for the session.
func WithCapabilities(capabilities terminal.Capabilities) Option {
	return func(options *Options) {
		options.Capabilities = capabilities
	}
}

// WithInitialSize sets the initial window size for the session.
func WithInitialSize(size terminal.Size) Option {
	return func(options *Options) {
		options.InitialSize = &size
	}
}

// WithProgramOptions appends Bubble Tea program options for the session.
func WithProgramOptions(programOptions ...tea.ProgramOption) Option {
	return func(options *Options) {
		options.ProgramOptions = append(options.ProgramOptions, programOptions...)
	}
}

// NewSession starts a Bubble Tea program wired with MMM terminal defaults.
func NewSession(test testing.TB, model tea.Model, options ...Option) *Session {
	if test == nil {
		return nil
	}

	configured := Options{
		Capabilities: terminal.TTYCapabilities(),
	}
	for _, option := range options {
		option(&configured)
	}

	inputDevice := terminal.NewDevice()
	outputDevice := terminal.NewDevice()
	terminal.ApplyTerminalDetection(test, inputDevice, outputDevice, configured.Capabilities)

	programOptions := view.ProgramOptions(inputDevice, outputDevice)
	programOptions = append(programOptions, tea.WithoutSignals())
	programOptions = append(programOptions, configured.ProgramOptions...)

	program := tea.NewProgram(model, programOptions...)

	session := &Session{
		program:      program,
		inputDevice:  inputDevice,
		outputDevice: outputDevice,
		outputReader: terminal.NewSnapshotReader(outputDevice.Bytes),
		doneCh:       make(chan struct{}),
	}

	go func() {
		finalModel, runError := program.Run()
		session.finalModel = finalModel
		session.runError = runError
		close(session.doneCh)
	}()

	if configured.InitialSize != nil {
		program.Send(tea.WindowSizeMsg{Width: configured.InitialSize.Columns, Height: configured.InitialSize.Rows})
	}

	return session
}

// Send sends a Bubble Tea message to the program.
func (session *Session) Send(message tea.Msg) {
	if session == nil || session.program == nil {
		return
	}
	session.program.Send(message)
}

// SendKey sends a keyboard message to the program.
func (session *Session) SendKey(message tea.KeyMsg) {
	session.Send(message)
}

// Type sends a series of rune key messages to the program.
func (session *Session) Type(text string) {
	for _, value := range text {
		session.Send(tea.KeyMsg{Runes: []rune{value}, Type: tea.KeyRunes})
	}
}

// SendMouse sends a mouse message to the program.
func (session *Session) SendMouse(message tea.MouseMsg) {
	session.Send(message)
}

// Resize sends a WindowSizeMsg to the program.
func (session *Session) Resize(size terminal.Size) {
	session.Send(tea.WindowSizeMsg{Width: size.Columns, Height: size.Rows})
}

// OutputBytes returns a snapshot of the current output buffer.
func (session *Session) OutputBytes() []byte {
	if session == nil || session.outputDevice == nil {
		return nil
	}
	return session.outputDevice.Bytes()
}

// OutputString returns a snapshot of the current output buffer as a string.
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

// WaitForOutput waits until the output satisfies the predicate.
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

// WaitFinished waits for the program to finish or time out.
func (session *Session) WaitFinished(test testing.TB, options ...FinalOption) {
	if session == nil || test == nil {
		return
	}
	finalOptions := buildFinalOptions(options)

	if finalOptions.timeout > 0 {
		select {
		case <-session.doneCh:
		case <-time.After(finalOptions.timeout):
			if finalOptions.onTimeout != nil {
				finalOptions.onTimeout(test)
				return
			}
			sessionFatalf(test, "timeout after %s", finalOptions.timeout)
		}
		return
	}

	<-session.doneCh
}

// FinalModel waits for completion and returns the final model.
func (session *Session) FinalModel(test testing.TB, options ...FinalOption) tea.Model {
	session.WaitFinished(test, options...)
	return session.finalModel
}

// FinalOutput waits for completion and returns the final output.
func (session *Session) FinalOutput(test testing.TB, options ...FinalOption) []byte {
	session.WaitFinished(test, options...)
	return session.OutputBytes()
}

// RunError returns the program run error after completion.
func (session *Session) RunError() error {
	if session == nil {
		return nil
	}
	return session.runError
}

// Quit requests program shutdown.
func (session *Session) Quit() {
	if session == nil || session.program == nil {
		return
	}
	session.program.Quit()
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

// FinalOption configures wait behavior for completion.
type FinalOption func(config *finalConfig)

type finalConfig struct {
	timeout   time.Duration
	onTimeout func(test testing.TB)
}

func buildFinalOptions(options []FinalOption) finalConfig {
	configured := finalConfig{}
	for _, option := range options {
		option(&configured)
	}
	return configured
}

// WithFinalTimeout sets a timeout for waiting on completion.
func WithFinalTimeout(timeout time.Duration) FinalOption {
	return func(config *finalConfig) {
		config.timeout = timeout
	}
}

// WithTimeoutHandler sets the handler invoked on timeout.
func WithTimeoutHandler(handler func(test testing.TB)) FinalOption {
	return func(config *finalConfig) {
		config.onTimeout = handler
	}
}

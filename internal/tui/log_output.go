package tui

import (
	"bytes"
	"io"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type logSender interface {
	Send(tea.Msg)
}

// LogLineMsg appends a line to the log output model.
type LogLineMsg struct {
	Line string
}

// LogDoneMsg signals that the log program should exit.
type LogDoneMsg struct{}

// LogModel is a minimal Bubble Tea model for rendering log lines.
type LogModel struct {
	lines []string
}

// NewLogModel returns an empty LogModel.
func NewLogModel() LogModel {
	return LogModel{}
}

// Init returns nil because log output does not require a startup command.
func (model LogModel) Init() tea.Cmd {
	return nil
}

// Update handles log line and done messages for the log output model.
func (model LogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case LogLineMsg:
		model.lines = append(model.lines, typed.Line)
		return model, nil
	case LogDoneMsg:
		return model, tea.Quit
	default:
		return model, nil
	}
}

// View renders the collected log lines.
func (model LogModel) View() string {
	return strings.Join(model.lines, "\n")
}

// LogLineWriter buffers output and sends LogLineMsg events to a Bubble Tea program.
type LogLineWriter struct {
	sender logSender
	buffer bytes.Buffer
	mutex  sync.Mutex
	closed bool
}

// NewLogLineWriter returns a LogLineWriter that emits log lines to sender.
func NewLogLineWriter(sender logSender) *LogLineWriter {
	return &LogLineWriter{sender: sender}
}

// LogProgram wraps a Bubble Tea program that renders log output.
type LogProgram struct {
	program *tea.Program
	writer  *LogLineWriter
	errors  chan error
}

func logProgramFilter(_ tea.Model, msg tea.Msg) tea.Msg {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		return nil
	default:
		return msg
	}
}

// StartLogProgram starts a Bubble Tea log program with the provided input/output.
func StartLogProgram(input io.Reader, output io.Writer) *LogProgram {
	options := []tea.ProgramOption{
		tea.WithInput(input),
		tea.WithOutput(output),
		tea.WithFilter(logProgramFilter),
	}
	program := tea.NewProgram(NewLogModel(), options...)
	writer := NewLogLineWriter(program)
	errors := make(chan error, 1)
	go func() {
		_, programErr := program.Run()
		errors <- programErr
	}()
	return &LogProgram{
		program: program,
		writer:  writer,
		errors:  errors,
	}
}

// Writer returns the log line writer for this program.
func (program *LogProgram) Writer() *LogLineWriter {
	if program == nil {
		return nil
	}
	return program.writer
}

// Stop flushes buffered output, signals program shutdown, and waits for completion.
func (program *LogProgram) Stop() error {
	if program == nil {
		return nil
	}
	program.writer.Flush()
	program.program.Send(LogDoneMsg{})
	program.writer.Close()
	return <-program.errors
}

// MergeProgramError returns programErr when primary is nil, otherwise returns primary.
func MergeProgramError(primary error, programErr error) error {
	if primary == nil && programErr != nil {
		return programErr
	}
	return primary
}

// Write buffers output and emits log lines when newline delimiters are seen.
func (writer *LogLineWriter) Write(p []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	if writer.closed || writer.sender == nil {
		return len(p), nil
	}

	_, _ = writer.buffer.Write(p)

	for {
		data := writer.buffer.Bytes()
		newlineIndex := bytes.IndexByte(data, '\n')
		if newlineIndex < 0 {
			break
		}

		line := string(data[:newlineIndex])
		writer.buffer.Next(newlineIndex + 1)
		writer.sender.Send(LogLineMsg{Line: line})
	}

	return len(p), nil
}

// Flush sends any remaining buffered output as a log line.
func (writer *LogLineWriter) Flush() {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	if writer.closed || writer.sender == nil {
		return
	}

	if writer.buffer.Len() == 0 {
		return
	}

	writer.sender.Send(LogLineMsg{Line: writer.buffer.String()})
	writer.buffer.Reset()
}

// Close stops accepting output and clears any buffered data.
func (writer *LogLineWriter) Close() {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	writer.closed = true
	writer.buffer.Reset()
}

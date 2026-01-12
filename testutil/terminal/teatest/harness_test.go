package teatest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type demoModel struct {
	lastEvent string
	columns   int
	rows      int
	mouseHits int
}

func (model demoModel) Init() tea.Cmd {
	return nil
}

func (model demoModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := message.(type) {
	case tea.KeyMsg:
		if typed.Type == tea.KeyRunes && len(typed.Runes) > 0 {
			model.lastEvent = fmt.Sprintf("key:%s", string(typed.Runes))
		}
	case tea.MouseMsg:
		model.mouseHits++
		model.lastEvent = "mouse"
	case tea.WindowSizeMsg:
		model.columns = typed.Width
		model.rows = typed.Height
		model.lastEvent = "resize"
	}
	return model, nil
}

func (model demoModel) View() string {
	return fmt.Sprintf("%s size=%dx%d mouse=%d", model.lastEvent, model.columns, model.rows, model.mouseHits)
}

type quitModel struct{}

func (quitModel) Init() tea.Cmd {
	return tea.Quit
}

func (quitModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return quitModel{}, nil
}

func (quitModel) View() string {
	return "done"
}

type stallModel struct{}

func (stallModel) Init() tea.Cmd {
	return nil
}

func (stallModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return stallModel{}, nil
}

func (stallModel) View() string {
	return "waiting"
}

func TestSessionCapturesOutputAndInputs(t *testing.T) {
	terminal.ApplyFixtures(t)

	initialSize := terminal.Size{Columns: 80, Rows: 25}
	session := NewSession(t, demoModel{}, WithInitialSize(initialSize))

	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "size=80x25")
	}, WithWaitDuration(2*time.Second))

	session.Type("a")
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "key:a")
	}, WithWaitDuration(2*time.Second))

	session.SendMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 1, Y: 1})
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "mouse=1")
	}, WithWaitDuration(2*time.Second))

	session.Resize(terminal.Size{Columns: 100, Rows: 40})
	session.WaitForOutput(t, func(output []byte) bool {
		return strings.Contains(string(output), "size=100x40")
	}, WithWaitDuration(2*time.Second), WithWaitInterval(10*time.Millisecond))

	session.Quit()
	session.WaitFinished(t, WithFinalTimeout(2*time.Second))
	require.NoError(t, session.RunError())
}

func TestNewSessionNil(t *testing.T) {
	require.Nil(t, NewSession(nil, demoModel{}))
}

func TestSessionFinalOutputAndModel(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := NewSession(t, quitModel{}, WithProgramOptions(tea.WithAltScreen()))
	finalModel := session.FinalModel(t, WithFinalTimeout(2*time.Second))
	require.IsType(t, quitModel{}, finalModel)

	finalOutput := session.FinalOutput(t, WithFinalTimeout(2*time.Second))
	require.NotNil(t, finalOutput)
	require.NoError(t, session.RunError())
}

func TestSessionNonTTYCapabilities(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := NewSession(t, demoModel{}, WithCapabilities(terminal.NonTTYCapabilities()))
	require.False(t, view.SupportsPrompting(session.inputDevice, session.outputDevice))

	session.Quit()
	session.WaitFinished(t, WithFinalTimeout(2*time.Second))
}

func TestSessionWaitFinishedNoTimeout(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := NewSession(t, quitModel{})
	session.WaitFinished(t)
	require.NoError(t, session.RunError())
}

func TestSessionTimeoutWithoutHandler(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := NewSession(t, stallModel{})
	originalFatalf := sessionFatalf
	called := false
	sessionFatalf = func(test testing.TB, format string, args ...any) {
		called = true
	}
	t.Cleanup(func() {
		sessionFatalf = originalFatalf
	})

	session.WaitFinished(t, WithFinalTimeout(10*time.Millisecond))
	require.True(t, called)

	session.Quit()
	session.WaitFinished(t, WithFinalTimeout(2*time.Second))
}

func TestSessionTimeoutHandler(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := NewSession(t, stallModel{})
	timeoutCalled := false
	session.WaitFinished(t, WithFinalTimeout(10*time.Millisecond), WithTimeoutHandler(func(test testing.TB) {
		timeoutCalled = true
	}))
	require.True(t, timeoutCalled)

	session.Quit()
	session.WaitFinished(t, WithFinalTimeout(2*time.Second))
}

func TestSessionNilGuards(t *testing.T) {
	var session *Session

	session.Send(nil)
	session.SendKey(tea.KeyMsg{})
	session.SendMouse(tea.MouseMsg{})
	session.Type("hi")
	session.Resize(terminal.Size{Columns: 1, Rows: 1})
	session.Quit()
	session.WaitForOutput(t, func(_ []byte) bool { return true })
	session.WaitFinished(t, WithFinalTimeout(time.Millisecond))
	require.Nil(t, session.OutputBytes())
	require.Empty(t, session.OutputString())
	require.Nil(t, session.OutputReader())
	require.Nil(t, session.RunError())
}

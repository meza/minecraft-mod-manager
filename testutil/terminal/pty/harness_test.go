package pty

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
)

type ptyModel struct {
	lastKey string
}

func (model ptyModel) Init() tea.Cmd {
	return nil
}

func (model ptyModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if typed, ok := message.(tea.KeyMsg); ok {
		model.lastKey = typed.String()
		if typed.String() == "q" {
			return model, tea.Quit
		}
	}
	return model, nil
}

func (model ptyModel) View() string {
	return "last=" + model.lastKey
}

func TestSessionCapturesOutput(t *testing.T) {
	terminal.ApplyFixtures(t)

	session := NewSession(t, WithSize(terminal.Size{Columns: 80, Rows: 25}))
	require.NotNil(t, session)

	program := tea.NewProgram(ptyModel{}, view.ProgramOptions(session.Input(), session.Output())...)
	programDone := make(chan error, 1)
	go func() {
		_, runError := program.Run()
		programDone <- runError
	}()

	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{StripControlSequences: true})
		return strings.Contains(normalized, "last=")
	}, WithWaitDuration(2*time.Second))

	_, writeError := session.SendInput([]byte("q"))
	require.NoError(t, writeError)

	select {
	case runError := <-programDone:
		require.NoError(t, runError)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for program completion")
	}
}

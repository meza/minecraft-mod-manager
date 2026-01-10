package install

import (
	"bytes"
	"context"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type unexpectedModel struct{}

func (unexpectedModel) Init() tea.Cmd { return nil }

func (unexpectedModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return unexpectedModel{}, nil
}

func (unexpectedModel) View() string { return "" }

func TestDefaultRunInstallProgram(t *testing.T) {
	model := newInstallModel(context.Background(), view.ColorDisabled, []installItem{}, map[string]int{}, nil, func(context.Context, httpclient.Sender) installExecutionOutcome {
		return installExecutionOutcome{errType: installExecutionErrorNone}
	}, nil)

	result, err := defaultRunInstallProgram(model,
		tea.WithInput(bytes.NewBuffer(nil)),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
	)
	assert.NoError(t, err)

	outcome, err := installOutcomeFromModel(result)
	assert.NoError(t, err)
	assert.Equal(t, installExecutionErrorNone, outcome.errType)
}

func TestDefaultRunInstallTranscriptProgram(t *testing.T) {
	model := newInstallTranscriptModel(context.Background(), view.ColorDisabled, []installItem{}, map[string]int{}, io.Discard, func(context.Context, httpclient.Sender) installExecutionOutcome {
		return installExecutionOutcome{errType: installExecutionErrorNone}
	})

	result, err := defaultRunInstallTranscriptProgram(model,
		tea.WithInput(bytes.NewBuffer(nil)),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
	)
	assert.NoError(t, err)

	outcome, err := installOutcomeFromModel(result)
	assert.NoError(t, err)
	assert.Equal(t, installExecutionErrorNone, outcome.errType)
}

func TestInstallOutcomeFromModelUnexpected(t *testing.T) {
	_, err := installOutcomeFromModel(unexpectedModel{})
	assert.Error(t, err)
}

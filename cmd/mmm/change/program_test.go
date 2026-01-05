package change

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

func (unexpectedModel) Init() tea.Cmd {
	return nil
}

func (unexpectedModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return unexpectedModel{}, nil
}

func (unexpectedModel) View() string {
	return ""
}

func TestDefaultRunChangeProgram(t *testing.T) {
	model := newChangeModel(changeModelInput{
		ctx:        context.Background(),
		target:     "1.19.4",
		colorMode:  view.ColorDisabled,
		items:      []changeItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, httpclient.Sender) changeOutcome {
			return changeOutcome{Stage: changeStageSuccess}
		},
	})

	result, err := defaultRunChangeProgram(model,
		tea.WithInput(bytes.NewBuffer(nil)),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
	)
	assert.NoError(t, err)

	outcome, err := changeOutcomeFromModel(result)
	assert.NoError(t, err)
	assert.Equal(t, changeStageSuccess, outcome.Stage)
}

func TestChangeOutcomeFromModelUnexpected(t *testing.T) {
	_, err := changeOutcomeFromModel(unexpectedModel{})
	assert.Error(t, err)
}

package scan

import (
	"context"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestScanOutcomeFromModelScanModel(t *testing.T) {
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      []scanItem{},
		indexByKey: map[string]int{},
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	})
	model.outcome = scanExecutionOutcome{unknown: []string{"alpha.jar"}}

	outcome, err := scanOutcomeFromModel(model)
	assert.NoError(t, err)
	assert.Equal(t, []string{"alpha.jar"}, outcome.unknown)
}

func TestScanOutcomeFromModelTranscriptModel(t *testing.T) {
	model := newScanTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		[]scanItem{},
		map[string]int{},
		nil,
		false,
		false,
		func(context.Context, scanExecSender) scanExecutionOutcome { return scanExecutionOutcome{} },
	)
	model.outcome = scanExecutionOutcome{unknown: []string{"beta.jar"}}

	outcome, err := scanOutcomeFromModel(model)
	assert.NoError(t, err)
	assert.Equal(t, []string{"beta.jar"}, outcome.unknown)
}

func TestScanOutcomeFromModelUnexpectedType(t *testing.T) {
	_, err := scanOutcomeFromModel(scanFakePromptModel{})
	assert.Error(t, err)
}

func TestDefaultRunScanProgramReturnsModel(t *testing.T) {
	items := []scanItem{{FileName: "alpha.jar", Status: scanItemStatusPending}}
	model := newScanModel(scanModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: scanIndexByFile(items),
		execRunner: func(context.Context, scanExecSender) scanExecutionOutcome {
			return scanExecutionOutcome{items: items}
		},
	})

	result, err := defaultRunScanProgram(model, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	assert.NoError(t, err)
	_, ok := result.(*scanModel)
	assert.True(t, ok)
}

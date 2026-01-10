package update

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestDefaultRunUpdateProgramAndTranscript(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	items := []updateItem{
		{ConfigIndex: 0, DisplayName: "Alpha", Status: updateItemStatusUpToDate},
	}
	indexByKey := map[int]int{0: 0}

	model := newUpdateModel(updateModelInput{
		ctx:        context.Background(),
		colorMode:  view.ColorDisabled,
		items:      items,
		indexByKey: indexByKey,
		execRunner: func(context.Context, updateExecSender) updateExecutionOutcome {
			return updateExecutionOutcome{items: items, errType: updateExecutionErrorNone}
		},
	})

	result, err := defaultRunUpdateProgram(model, outputProgramOptions(nil, io.Discard)...)
	assert.NoError(t, err)
	typed, ok := result.(*updateModel)
	assert.True(t, ok)
	assert.Equal(t, updateExecutionErrorNone, typed.outcome.errType)

	buffer := &bytes.Buffer{}
	transcriptModel := newUpdateTranscriptModel(
		context.Background(),
		view.ColorDisabled,
		items,
		indexByKey,
		buffer,
		func(context.Context, updateExecSender) updateExecutionOutcome {
			return updateExecutionOutcome{items: items, errType: updateExecutionErrorNone}
		},
	)
	result, err = defaultRunUpdateTranscriptProgram(transcriptModel, outputProgramOptions(nil, buffer)...)
	assert.NoError(t, err)
	typedTranscript, ok := result.(*updateTranscriptModel)
	assert.True(t, ok)
	assert.Equal(t, updateExecutionErrorNone, typedTranscript.outcome.errType)
	assert.Contains(t, buffer.String(), "cmd.update.summary.success")
}

func TestUpdateOutcomeFromModel(t *testing.T) {
	outcome := updateExecutionOutcome{errType: updateExecutionErrorNone}
	model := &updateModel{outcome: outcome}
	result, err := updateOutcomeFromModel(model)
	assert.NoError(t, err)
	assert.Equal(t, updateExecutionErrorNone, result.errType)

	transcript := &updateTranscriptModel{outcome: outcome}
	result, err = updateOutcomeFromModel(transcript)
	assert.NoError(t, err)
	assert.Equal(t, updateExecutionErrorNone, result.errType)

	_, err = updateOutcomeFromModel(outputLinesModel{})
	assert.Error(t, err)
}

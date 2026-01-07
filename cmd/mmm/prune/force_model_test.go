package prune

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestPruneForceModelViewShowsDeleting(t *testing.T) {
	viewOutput := newPruneForceModel(view.ColorDisabled, []string{"/mods/file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	}).View()

	assert.Contains(t, viewOutput, i18n.T("cmd.prune.header.deleting", nil))
}

func TestPruneForceModelViewShowsSuccess(t *testing.T) {
	model := newPruneForceModel(view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	model.results = []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}}
	model.done = true

	viewOutput := model.View()
	assert.Contains(t, viewOutput, i18n.T("cmd.prune.header.deleted", nil))
	assert.Contains(t, viewOutput, "file.jar")
	assert.Contains(t, viewOutput, i18n.T("cmd.prune.summary.success", nil))
}

func TestPruneForceModelViewShowsFailure(t *testing.T) {
	model := newPruneForceModel(view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	model.results = []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusFailed, Err: errors.New("boom")}}
	model.deleteErr = errors.New("boom")
	model.done = true

	viewOutput := model.View()
	assert.Contains(t, viewOutput, i18n.T("cmd.prune.header.deleting", nil))
	assert.Contains(t, viewOutput, "delete failed")
	assert.Contains(t, viewOutput, i18n.T("cmd.prune.summary.incomplete", nil))
}

func TestPruneForceResultFromModelHandlesPointer(t *testing.T) {
	model := &pruneForceModel{
		results: []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}},
	}
	results, deleteErr, err := pruneForceResultFromModel(model)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NoError(t, deleteErr)
}

func TestPruneForceResultFromModelReturnsErrorForUnexpectedModel(t *testing.T) {
	results, deleteErr, err := pruneForceResultFromModel(unexpectedModel{})
	assert.Error(t, err)
	assert.Empty(t, results)
	assert.NoError(t, deleteErr)
}

func TestPruneForceModelInitUsesDeleteFunc(t *testing.T) {
	model := newPruneForceModel(view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}}, nil
	})
	msg := model.Init()()
	result, ok := msg.(pruneForceResult)
	require.True(t, ok)
	assert.Len(t, result.results, 1)
}

func TestPruneForceModelUpdateAppliesResult(t *testing.T) {
	model := newPruneForceModel(view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	updated, cmd := model.Update(pruneForceResult{
		results: []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}},
		err:     errors.New("boom"),
	})
	assert.NotNil(t, cmd)

	updatedModel, ok := updated.(pruneForceModel)
	require.True(t, ok)
	assert.True(t, updatedModel.done)
	assert.Len(t, updatedModel.results, 1)
	assert.Error(t, updatedModel.deleteErr)
}

func TestPruneForceModelUpdateIgnoresOtherMessages(t *testing.T) {
	model := newPruneForceModel(view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})

	updated, cmd := model.Update("ignored")
	assert.Nil(t, cmd)

	updatedModel, ok := updated.(pruneForceModel)
	require.True(t, ok)
	assert.False(t, updatedModel.done)
}

func TestRenderDeleteFailedViewIncludesSummary(t *testing.T) {
	output := renderDeleteFailedView(view.ColorDisabled, []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusFailed, Err: errors.New("boom")}})
	assert.True(t, strings.Contains(output, i18n.T("cmd.prune.summary.incomplete", nil)))
}

func TestRenderDeleteSuccessViewIncludesSummary(t *testing.T) {
	output := renderDeleteSuccessView(view.ColorDisabled, []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}})
	assert.True(t, strings.Contains(output, i18n.T("cmd.prune.summary.success", nil)))
}

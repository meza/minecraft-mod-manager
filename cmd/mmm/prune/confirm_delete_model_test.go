package prune

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

func TestPruneConfirmDeleteModelViewShowsPrompt(t *testing.T) {
	model := newPruneConfirmDeleteModel("list", "question", view.ColorDisabled, []string{"file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	})

	viewOutput := model.View()
	assert.True(t, strings.Contains(viewOutput, "list"))
	assert.True(t, strings.Contains(viewOutput, "question"))
}

func TestPruneConfirmDeleteModelViewShowsPromptWithoutList(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, []string{"file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	})

	viewOutput := model.View()
	assert.False(t, strings.Contains(viewOutput, "list"))
	assert.True(t, strings.Contains(viewOutput, "question"))
}

func TestPruneConfirmDeleteModelInitReturnsNil(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	assert.Nil(t, model.Init())
}

func TestPruneConfirmDeleteModelViewShowsDeleting(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, []string{"file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	model.confirmed = true

	viewOutput := model.View()
	assert.True(t, strings.Contains(viewOutput, i18n.T("cmd.prune.header.deleting", nil)))
}

func TestPruneConfirmDeleteModelViewShowsSuccess(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	model.results = []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}}
	model.done = true

	viewOutput := model.View()
	assert.True(t, strings.Contains(viewOutput, i18n.T("cmd.prune.header.deleted", nil)))
	assert.True(t, strings.Contains(viewOutput, i18n.T("cmd.prune.summary.success", nil)))
}

func TestPruneConfirmDeleteModelViewShowsFailure(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	model.results = []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusFailed, Err: errors.New("boom")}}
	model.deleteErr = errors.New("boom")
	model.done = true

	viewOutput := model.View()
	assert.True(t, strings.Contains(viewOutput, i18n.T("cmd.prune.header.deleting", nil)))
	assert.True(t, strings.Contains(viewOutput, i18n.T("cmd.prune.summary.incomplete", nil)))
}

func TestPruneConfirmDeleteModelUpdateConfirmNoQuits(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	updated, cmd := model.Update(confirmSelectedMessage{confirmed: false})
	assert.NotNil(t, cmd)

	updatedModel, ok := updated.(pruneConfirmDeleteModel)
	require.True(t, ok)
	assert.False(t, updatedModel.confirmed)
}

func TestPruneConfirmDeleteModelUpdatePassesThroughOtherMessages(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	assert.NotNil(t, cmd)

	_, ok := updated.(pruneConfirmDeleteModel)
	require.True(t, ok)
}

func TestPruneConfirmDeleteModelUpdateCancelQuits(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd)

	updatedModel, ok := updated.(pruneConfirmDeleteModel)
	require.True(t, ok)
	assert.True(t, updatedModel.canceled)
}

func TestPruneConfirmDeleteModelUpdateConfirmYesRunsDelete(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}}, nil
	})

	updated, cmd := model.Update(confirmSelectedMessage{confirmed: true})
	assert.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(pruneDeleteResult)
	require.True(t, ok)
	assert.Len(t, result.results, 1)

	updatedModel, ok := updated.(pruneConfirmDeleteModel)
	require.True(t, ok)
	assert.True(t, updatedModel.confirmed)
}

func TestPruneConfirmDeleteModelUpdateDeleteResultQuits(t *testing.T) {
	model := newPruneConfirmDeleteModel("", "question", view.ColorDisabled, nil, func() ([]pruneFileResult, error) {
		return nil, nil
	})
	updated, cmd := model.Update(pruneDeleteResult{
		results: []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}},
		err:     errors.New("boom"),
	})
	assert.NotNil(t, cmd)

	updatedModel, ok := updated.(pruneConfirmDeleteModel)
	require.True(t, ok)
	assert.True(t, updatedModel.done)
	assert.Error(t, updatedModel.deleteErr)
}

func TestPruneConfirmDeleteResultReturnsErrorForUnexpectedModel(t *testing.T) {
	outcome, err := pruneConfirmDeleteResult(unexpectedModel{})
	assert.Error(t, err)
	assert.False(t, outcome.confirmed)
	assert.False(t, outcome.canceled)
	assert.Empty(t, outcome.results)
	assert.NoError(t, outcome.deleteErr)
}

func TestPruneConfirmDeleteResultHandlesPointer(t *testing.T) {
	model := &pruneConfirmDeleteModel{
		confirmed: true,
		canceled:  false,
		results:   []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}},
	}

	outcome, err := pruneConfirmDeleteResult(model)
	require.NoError(t, err)
	assert.True(t, outcome.confirmed)
	assert.False(t, outcome.canceled)
	assert.Len(t, outcome.results, 1)
	assert.NoError(t, outcome.deleteErr)
}

func TestRunConfirmDeleteFlowReturnsErrorWhenRunTeaFails(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&fakeTerminalWriter{})

	runErr := errors.New("run failed")
	_, err := runConfirmDeleteFlow(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
	}, view.ColorDisabled, []string{"file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	})

	assert.ErrorIs(t, err, runErr)
}

func TestRunConfirmDeleteFlowReturnsErrorForUnexpectedModel(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&fakeTerminalWriter{})

	_, err := runConfirmDeleteFlow(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return unexpectedModel{}, nil
		},
	}, view.ColorDisabled, []string{"file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	})

	assert.Error(t, err)
}

func TestRunConfirmDeleteFlowReturnsResults(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&fakeTerminalWriter{})

	results := []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}}
	outcome, err := runConfirmDeleteFlow(cmd, pruneDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return pruneConfirmDeleteModel{
				confirmed: true,
				results:   results,
				done:      true,
			}, nil
		},
	}, view.ColorDisabled, []string{"file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	})

	require.NoError(t, err)
	assert.True(t, outcome.confirmed)
	assert.False(t, outcome.canceled)
	assert.Equal(t, results, outcome.results)
	assert.NoError(t, outcome.deleteErr)
}

func TestRunConfirmDeleteFlowUsesFallbackRunner(t *testing.T) {
	originalRunner := runTeaProgram
	t.Cleanup(func() {
		runTeaProgram = originalRunner
	})

	ran := false
	runTeaProgram = func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
		ran = true
		return pruneConfirmDeleteModel{
			confirmed: true,
			results:   []pruneFileResult{{Path: "/mods/file.jar", Status: pruneFileStatusDeleted}},
			done:      true,
		}, nil
	}

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{})
	cmd.SetOut(&fakeTerminalWriter{})
	cmd.SetErr(&fakeTerminalWriter{})

	outcome, err := runConfirmDeleteFlow(cmd, pruneDeps{}, view.ColorDisabled, []string{"file.jar"}, func() ([]pruneFileResult, error) {
		return nil, nil
	})

	require.NoError(t, err)
	assert.True(t, ran)
	assert.True(t, outcome.confirmed)
	assert.False(t, outcome.canceled)
	assert.Len(t, outcome.results, 1)
	assert.NoError(t, outcome.deleteErr)
}

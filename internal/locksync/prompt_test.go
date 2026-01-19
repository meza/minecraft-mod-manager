package locksync

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLockSyncPromptReturnsErrorWhenRunnerMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	_, err := runLockSyncPrompt(lockSyncPromptInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing lock sync prompt runner")
}

func TestRunLockSyncPromptReturnsErrorOnRunTeaFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	expectedErr := errors.New("run failed")
	_, err := runLockSyncPrompt(lockSyncPromptInput{
		listView: "list",
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, expectedErr
		},
	})
	assert.ErrorIs(t, err, expectedErr)
}

func TestLockSyncPromptModelSelectsPolicy(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newLockSyncPromptModel("list")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)

	typed, ok := updated.(lockSyncPromptModel)
	require.True(t, ok)
	assert.True(t, typed.answered)
	assert.Equal(t, PolicyAdd, typed.policy)
	assert.Contains(t, typed.View(), "cmd.lock_sync.answer.add")
}

func TestLockSyncPromptModelCancelSetsCanceled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newLockSyncPromptModel("list")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})

	typed, ok := updated.(lockSyncPromptModel)
	require.True(t, ok)
	assert.True(t, typed.canceled)
}

func TestLockSyncPromptModelQuitSetsCanceled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newLockSyncPromptModel("list")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	typed, ok := updated.(lockSyncPromptModel)
	require.True(t, ok)
	assert.True(t, typed.canceled)
}

func TestLockSyncPromptModelUpdatesWindowSize(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newLockSyncPromptModel("list")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120})

	typed, ok := updated.(lockSyncPromptModel)
	require.True(t, ok)
	assert.Equal(t, 120, typed.list.Width())
}

func TestLockSyncPromptModelEnterWithUnknownItemDoesNotAnswer(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newLockSyncPromptModel("list")
	model.list.SetItems([]list.Item{stubListItem{}})
	model.list.Select(0)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	typed, ok := updated.(lockSyncPromptModel)
	require.True(t, ok)
	assert.False(t, typed.answered)
}

func TestLockSyncPromptModelDelegatesUnhandledKeyToList(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newLockSyncPromptModel("list")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	_, ok := updated.(lockSyncPromptModel)
	assert.True(t, ok)
}

func TestLockSyncPromptViewUsesListWhenUnanswered(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newLockSyncPromptModel("list")
	view := model.View()
	assert.Contains(t, view, "list")
}

func TestJoinPromptLinesHandlesEmptyInputs(t *testing.T) {
	assert.Equal(t, "prompt", joinPromptLines("", "prompt"))
	assert.Equal(t, "list", joinPromptLines("list", ""))
	assert.Equal(t, "list\n\nprompt", joinPromptLines("list", "prompt"))
}

func TestLockSyncActionLineFallbackUsesSkipLabel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	line := lockSyncActionLine(PolicyUnknown)
	assert.Contains(t, line, "cmd.lock_sync.action.applied")
	assert.Contains(t, line, "cmd.lock_sync.answer.skip")
}

func TestLockSyncActionLineUsesPolicyLabel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	line := lockSyncActionLine(PolicyAdd)
	assert.Contains(t, line, "cmd.lock_sync.action.applied")
	assert.Contains(t, line, "cmd.lock_sync.answer.add")
}

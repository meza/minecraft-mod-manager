package test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
)

func TestTestVersionPromptModelCancelsOnEsc(t *testing.T) {
	model := newTestVersionPromptModel(context.Background(), noopDoer{}, testVersionPromptInput{
		headline: "headline",
		question: "Question?",
	})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.True(t, updated.(testVersionPromptModel).canceled)
}

func TestTestVersionPromptModelAcceptsSelectedMessage(t *testing.T) {
	model := newTestVersionPromptModel(context.Background(), noopDoer{}, testVersionPromptInput{
		headline: "headline",
		question: "Question?",
	})

	updated, cmd := model.Update(initCmd.GameVersionSelectedMessage{GameVersion: "1.20.1"})
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
	assert.Equal(t, "1.20.1", updated.(testVersionPromptModel).selected)
}

func TestTestVersionPromptModelViewIncludesHeadline(t *testing.T) {
	model := newTestVersionPromptModel(context.Background(), noopDoer{}, testVersionPromptInput{
		headline: "headline",
		question: "Question?",
	})

	assert.Contains(t, model.View(), "headline")
}

func TestTestVersionPromptModelViewWithoutHeadline(t *testing.T) {
	model := newTestVersionPromptModel(context.Background(), noopDoer{}, testVersionPromptInput{
		question: "Question?",
	})

	assert.NotContains(t, model.View(), "headline")
}

func TestTestVersionPromptModelInitReturnsNil(t *testing.T) {
	model := newTestVersionPromptModel(context.Background(), noopDoer{}, testVersionPromptInput{
		headline: "headline",
		question: "Question?",
	})

	assert.Nil(t, model.Init())
}

func TestTestVersionPromptModelUpdatePassesThroughPrompt(t *testing.T) {
	model := newTestVersionPromptModel(context.Background(), noopDoer{}, testVersionPromptInput{
		question: "Question?",
	})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	assert.False(t, updated.(testVersionPromptModel).canceled)
}

func TestRunTestVersionPromptUsesRunner(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	deps := testDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return testVersionPromptModel{selected: "1.20.1"}, nil
		},
		minecraftClient: noopDoer{},
	}

	version, canceled, err := runTestVersionPrompt(context.Background(), command, deps, testVersionPromptInput{
		question: "Question?",
	})
	require.NoError(t, err)
	assert.False(t, canceled)
	assert.Equal(t, "1.20.1", version)
}

func TestRunTestVersionPromptUsesDefaultRunner(t *testing.T) {
	originalRunner := runTeaProgram
	t.Cleanup(func() { runTeaProgram = originalRunner })
	runTeaProgram = func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
		return testVersionPromptModel{selected: "1.20.1"}, nil
	}

	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	version, canceled, err := runTestVersionPrompt(context.Background(), command, testDeps{
		minecraftClient: noopDoer{},
	}, testVersionPromptInput{
		question: "Question?",
	})
	require.NoError(t, err)
	assert.False(t, canceled)
	assert.Equal(t, "1.20.1", version)
}

func TestTestVersionPromptResultUnexpectedModel(t *testing.T) {
	_, _, err := testVersionPromptResult(fakePromptModel{})
	assert.Error(t, err)
}

func TestTestVersionPromptResultHandlesPointerModel(t *testing.T) {
	model := &testVersionPromptModel{selected: "1.20.1", canceled: true}
	version, canceled, err := testVersionPromptResult(model)
	require.NoError(t, err)
	assert.Equal(t, "1.20.1", version)
	assert.True(t, canceled)
}

func TestRunTestVersionPromptReturnsRunnerError(t *testing.T) {
	command := &cobra.Command{}
	command.SetIn(&bytes.Buffer{})
	command.SetOut(&bytes.Buffer{})

	runErr := errors.New("run tea failed")
	deps := testDeps{
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return nil, runErr
		},
		minecraftClient: noopDoer{},
	}

	_, _, err := runTestVersionPrompt(context.Background(), command, deps, testVersionPromptInput{
		question: "Question?",
	})
	assert.ErrorIs(t, err, runErr)
}

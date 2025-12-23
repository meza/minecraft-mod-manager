package add

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestAddTUIListItemFilterValueEmpty(t *testing.T) {
	item := addTUIListItem{value: "abc"}
	assert.Equal(t, "", item.FilterValue())
}

func TestAddTUIListDelegateUpdateNoop(t *testing.T) {
	var delegate addTUIListDelegate
	assert.Nil(t, delegate.Update(nil, nil))
}

func TestAddTUIListDelegateRenderSelectedAndUnselected(t *testing.T) {
	var delegate addTUIListDelegate
	model := newPlatformListModel("question", "", false, 20)
	items := model.Items()
	if !assert.Len(t, items, 2) {
		return
	}

	selected := &bytes.Buffer{}
	model.Select(0)
	delegate.Render(selected, model, 0, items[0])
	assert.NotEmpty(t, selected.String())

	unselected := &bytes.Buffer{}
	delegate.Render(unselected, model, 1, items[1])
	assert.NotEmpty(t, unselected.String())
}

func TestAddTUIListDelegateRenderHandlesWriteError(t *testing.T) {
	var delegate addTUIListDelegate
	model := newPlatformListModel("question", "", false, 20)
	items := model.Items()
	if !assert.Len(t, items, 2) {
		return
	}

	model.Select(0)
	writerErr := errors.New("write failed")
	assert.NotPanics(t, func() {
		delegate.Render(errorWriter{err: writerErr}, model, 0, items[0])
	})

	assert.NotPanics(t, func() {
		delegate.Render(errorWriter{err: writerErr}, model, 1, items[1])
	})
}

func TestAddTUIListDelegateRenderIgnoresUnknownItem(t *testing.T) {
	var delegate addTUIListDelegate
	model := list.New([]list.Item{fakeListItem{}}, addTUIListDelegate{}, 10, 5)
	buffer := &bytes.Buffer{}
	delegate.Render(buffer, model, 0, fakeListItem{})
	assert.Equal(t, "", buffer.String())
}

func TestAddTUIModelInitReturnsQuitForDone(t *testing.T) {
	model := addTUIModel{state: addTUIStateDone}
	cmd := model.Init()
	assert.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
}

func TestAddTUIModelInitReturnsNilForActiveState(t *testing.T) {
	model := addTUIModel{state: addTUIStateUnknownPlatformSelect}
	assert.Nil(t, model.Init())
}

func TestAddTUIModelUpdateHandlesWindowSize(t *testing.T) {
	model := addTUIModel{
		state: addTUIStateUnknownPlatformSelect,
		list:  newPlatformListModel("question", "", false, 10),
	}
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120})
	typed := updated.(addTUIModel)
	assert.Equal(t, 120, typed.width)
}

func TestAddTUIModelUpdateCtrlCAborts(t *testing.T) {
	model := addTUIModel{state: addTUIStateUnknownPlatformSelect}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateCtrlCAddsSpanEvent(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	model := addTUIModel{
		ctx:         ctx,
		sessionSpan: span,
		state:       addTUIStateUnknownPlatformSelect,
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateEscAddsSpanEvent(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	model := addTUIModel{
		ctx:         ctx,
		sessionSpan: span,
		state:       addTUIStateUnknownPlatformSelect,
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateEscAbortsWhenNoHistory(t *testing.T) {
	model := addTUIModel{state: addTUIStateUnknownPlatformSelect}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateEscGoesBackWithHistory(t *testing.T) {
	initPerf(t)
	model := addTUIModel{
		state:             addTUIStateModNotFoundConfirm,
		candidatePlatform: models.CURSEFORGE,
		candidateProject:  "abc",
		history: []addTUIHistory{
			{state: addTUIStateUnknownPlatformSelect, candidatePlatform: models.MODRINTH, candidateProject: "def"},
		},
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateUnknownPlatformSelect, typed.state)
	assert.Equal(t, models.MODRINTH, typed.candidatePlatform)
	assert.Equal(t, "def", typed.candidateProject)
	assert.Nil(t, cmd)
}

func TestAddTUIModelUpdateDefaultStateNoop(t *testing.T) {
	model := addTUIModel{state: addTUIStateFatalError}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateFatalError, typed.state)
	assert.Nil(t, cmd)
}

func TestAddTUIModelUpdateListCancelAborts(t *testing.T) {
	initPerf(t)
	model := addTUIModel{
		state: addTUIStateUnknownPlatformSelect,
		list:  newPlatformListModel("question", "", true, 20),
		fetchCmd: func(models.Platform, string) tea.Cmd {
			return nil
		},
	}
	model.list.Select(2)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateListSelectsPlatform(t *testing.T) {
	initPerf(t)
	var called bool
	model := addTUIModel{
		state: addTUIStateUnknownPlatformSelect,
		list:  newPlatformListModel("question", "", false, 20),
		fetchCmd: func(models.Platform, string) tea.Cmd {
			called = true
			return nil
		},
	}
	model.list.Select(0)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.True(t, called)
	assert.Equal(t, models.CURSEFORGE, typed.candidatePlatform)
}

func TestAddTUIModelUpdateListSelectsPlatformWithSpan(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	var called bool
	model := addTUIModel{
		ctx:            ctx,
		sessionSpan:    span,
		state:          addTUIStateUnknownPlatformSelect,
		list:           newPlatformListModel("question", "", false, 20),
		failureProject: "abc",
		fetchCmd: func(models.Platform, string) tea.Cmd {
			called = true
			return nil
		},
	}
	model.list.Select(0)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.True(t, called)
	assert.Equal(t, models.CURSEFORGE, typed.candidatePlatform)
}

func TestAddTUIModelUpdateListCancelAddsSpanEvent(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	model := addTUIModel{
		ctx:         ctx,
		sessionSpan: span,
		state:       addTUIStateUnknownPlatformSelect,
		list:        newPlatformListModel("question", "", true, 20),
		fetchCmd: func(models.Platform, string) tea.Cmd {
			return nil
		},
	}
	model.list.Select(2)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateListIgnoresUnknownItem(t *testing.T) {
	model := addTUIModel{
		state: addTUIStateUnknownPlatformSelect,
		list:  list.New([]list.Item{fakeListItem{}}, addTUIListDelegate{}, 10, 5),
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateUnknownPlatformSelect, typed.state)
	assert.Nil(t, cmd)
}

func TestAddTUIModelUpdateListSelectMovesToProjectEntry(t *testing.T) {
	initPerf(t)
	model := addTUIModel{
		state: addTUIStateModNotFoundSelectPlatform,
		list:  newPlatformListModel("question", "", false, 20),
	}
	model.list.Select(1)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateModNotFoundEnterProjectID, typed.state)
	assert.Equal(t, models.MODRINTH, typed.candidatePlatform)
}

func TestAddTUIModelUpdateListPassesThroughToList(t *testing.T) {
	model := addTUIModel{
		state: addTUIStateUnknownPlatformSelect,
		list:  newPlatformListModel("question", "", false, 20),
	}
	model.list.Select(0)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateUnknownPlatformSelect, typed.state)
	assert.Equal(t, 1, typed.list.Index())
}

func TestAddTUIModelUpdateInputUsesPlaceholder(t *testing.T) {
	initPerf(t)
	model := addTUIModel{
		state:            addTUIStateModNotFoundEnterProjectID,
		candidateProject: "abc",
		fetchCmd: func(models.Platform, string) tea.Cmd {
			return nil
		},
	}
	model.input = newProjectIDInputModel("prompt", "fallback")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, "fallback", typed.candidateProject)
}

func TestAddTUIModelUpdateInputEmptyNoop(t *testing.T) {
	model := addTUIModel{
		state: addTUIStateModNotFoundEnterProjectID,
		input: textinput.Model{},
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateModNotFoundEnterProjectID, typed.state)
	assert.Nil(t, cmd)
}

func TestAddTUIModelUpdateInputPassesThroughToInput(t *testing.T) {
	input := textinput.New()
	input.Focus()
	model := addTUIModel{
		state: addTUIStateModNotFoundEnterProjectID,
		input: input,
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	typed := updated.(addTUIModel)
	assert.Equal(t, "a", typed.input.Value())
}

func TestAddTUIModelUpdateInputAddsSpanEvent(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	var called bool
	input := textinput.New()
	input.SetValue("proj")
	model := addTUIModel{
		ctx:         ctx,
		sessionSpan: span,
		state:       addTUIStateModNotFoundEnterProjectID,
		input:       input,
		fetchCmd: func(models.Platform, string) tea.Cmd {
			called = true
			return nil
		},
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.True(t, called)
	assert.Equal(t, "proj", typed.candidateProject)
}

func TestAddTUIModelUpdateInputNoFileSetsAlternatePlatform(t *testing.T) {
	initPerf(t)
	model := addTUIModel{
		state:            addTUIStateNoFileEnterProjectID,
		failurePlatform:  models.CURSEFORGE,
		candidateProject: "abc",
		fetchCmd: func(models.Platform, string) tea.Cmd {
			return nil
		},
	}
	model.input = newProjectIDInputModel("prompt", "mod-id")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, models.MODRINTH, typed.candidatePlatform)
	assert.Equal(t, "mod-id", typed.candidateProject)
}

func TestAddTUIModelUpdateConfirmYesMovesState(t *testing.T) {
	initPerf(t)
	model := addTUIModel{state: addTUIStateModNotFoundConfirm}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateModNotFoundSelectPlatform, typed.state)
}

func TestAddTUIModelUpdateConfirmYesAddsSpanEvent(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	model := addTUIModel{
		ctx:         ctx,
		sessionSpan: span,
		state:       addTUIStateModNotFoundConfirm,
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateModNotFoundSelectPlatform, typed.state)
}

func TestAddTUIModelUpdateConfirmYesMovesStateForNoFile(t *testing.T) {
	initPerf(t)
	model := addTUIModel{state: addTUIStateNoFileConfirm}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateNoFileEnterProjectID, typed.state)
}

func TestAddTUIModelUpdateConfirmNoAborts(t *testing.T) {
	model := addTUIModel{state: addTUIStateModNotFoundConfirm}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateConfirmNoAddsSpanEvent(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	model := addTUIModel{
		ctx:         ctx,
		sessionSpan: span,
		state:       addTUIStateModNotFoundConfirm,
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateConfirmNoDirectAddsSpanEvent(t *testing.T) {
	initPerf(t)

	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.add.session")
	t.Cleanup(span.End)

	model := addTUIModel{
		ctx:         ctx,
		sessionSpan: span,
		state:       addTUIStateModNotFoundConfirm,
	}
	updated, cmd := model.updateConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateAborted, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelUpdateConfirmNonKeyNoop(t *testing.T) {
	model := addTUIModel{state: addTUIStateModNotFoundConfirm}
	updated, cmd := model.updateConfirm(struct{}{})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateModNotFoundConfirm, typed.state)
	assert.Nil(t, cmd)
}

func TestAddTUIModelHandleFetchResultSuccess(t *testing.T) {
	model := addTUIModel{state: addTUIStateUnknownPlatformSelect}
	updated, cmd := model.handleFetchResult(addTUIFetchResultMsg{
		platform:  models.MODRINTH,
		projectID: "abc",
		remote:    platform.RemoteMod{Name: "Example", FileName: "example.jar"},
	})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateDone, typed.state)
	assert.Equal(t, models.MODRINTH, typed.resolvedPlatform)
	assert.Equal(t, "abc", typed.resolvedProject)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelHandleFetchResultUnknownPlatform(t *testing.T) {
	initPerf(t)
	model := addTUIModel{}
	updated, _ := model.handleFetchResult(addTUIFetchResultMsg{
		platform:  models.MODRINTH,
		projectID: "abc",
		err:       &platform.UnknownPlatformError{Platform: "invalid"},
	})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateUnknownPlatformSelect, typed.state)
}

func TestAddTUIModelHandleFetchResultModNotFound(t *testing.T) {
	initPerf(t)
	model := addTUIModel{}
	updated, _ := model.handleFetchResult(addTUIFetchResultMsg{
		platform:  models.MODRINTH,
		projectID: "abc",
		err:       &platform.ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"},
	})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateModNotFoundConfirm, typed.state)
}

func TestAddTUIModelHandleFetchResultNoFile(t *testing.T) {
	initPerf(t)
	model := addTUIModel{}
	updated, _ := model.handleFetchResult(addTUIFetchResultMsg{
		platform:  models.MODRINTH,
		projectID: "abc",
		err:       &platform.NoCompatibleFileError{Platform: models.MODRINTH, ProjectID: "abc"},
	})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateNoFileConfirm, typed.state)
}

func TestAddTUIModelHandleFetchResultDefaultError(t *testing.T) {
	model := addTUIModel{}
	updated, cmd := model.handleFetchResult(addTUIFetchResultMsg{
		platform:  models.MODRINTH,
		projectID: "abc",
		err:       errors.New("boom"),
	})
	typed := updated.(addTUIModel)
	assert.Equal(t, addTUIStateFatalError, typed.state)
	assert.NotNil(t, cmd)
}

func TestAddTUIModelViewStates(t *testing.T) {
	model := addTUIModel{state: addTUIStateDone}
	assert.Equal(t, "", model.View())

	model = addTUIModel{
		state: addTUIStateUnknownPlatformSelect,
		list:  newPlatformListModel("question", "", false, 20),
	}
	assert.NotEmpty(t, model.View())

	model = addTUIModel{state: addTUIStateModNotFoundConfirm, confirmMessage: "confirm", confirmDefault: true}
	assert.NotEmpty(t, model.View())

	model = addTUIModel{
		state: addTUIStateModNotFoundSelectPlatform,
		list:  newPlatformListModel("question", "", false, 20),
	}
	assert.NotEmpty(t, model.View())

	model = addTUIModel{state: addTUIStateModNotFoundEnterProjectID, input: textinput.New()}
	assert.NotEmpty(t, model.View())

	model = addTUIModel{state: addTUIStateNoFileConfirm, confirmMessage: "confirm", confirmDefault: false}
	assert.NotEmpty(t, model.View())

	model = addTUIModel{state: addTUIStateNoFileEnterProjectID, input: textinput.New()}
	assert.NotEmpty(t, model.View())

	model = addTUIModel{state: addTUIStateFatalError}
	assert.Equal(t, "", model.View())

	model = addTUIModel{state: addTUIStateFatalError, err: errors.New("boom")}
	assert.Contains(t, model.View(), "boom")

	model = addTUIModel{state: addTUIState(99)}
	assert.Equal(t, "", model.View())
}

func TestAddTUIModelEnterStateFatalErrorNoop(t *testing.T) {
	initPerf(t)
	model := addTUIModel{ctx: context.Background()}
	model.enterState(addTUIStateFatalError)
}

func TestAddTUIStartWaitSkipsDone(t *testing.T) {
	initPerf(t)
	model := addTUIModel{ctx: context.Background()}
	model.startWait(addTUIStateDone)
	assert.Nil(t, model.waitSpan)
}

func TestAddTUIBeginFetchEndsOverlappingSpan(t *testing.T) {
	initPerf(t)
	model := addTUIModel{ctx: context.Background()}
	_, span := perf.StartSpan(context.Background(), "tui.add.fetch")
	model.fetchSpan = span
	model.beginFetch("action", models.MODRINTH, "abc")
	assert.NotNil(t, model.fetchSpan)
}

func TestAddTUIEndFetchNoSpanNoop(t *testing.T) {
	model := addTUIModel{}
	model.endFetch(addTUIFetchResultMsg{err: errors.New("boom")})
}

func TestNewPlatformListModelSelectsDefaultAndIncludesCancel(t *testing.T) {
	model := newPlatformListModel("question", string(models.MODRINTH), true, 20)
	items := model.Items()
	values := make([]string, 0, len(items))
	for _, item := range items {
		if typed, ok := item.(addTUIListItem); ok {
			values = append(values, typed.value)
		}
	}
	assert.Contains(t, values, "cancel")
	assert.Equal(t, 1, model.Index())
}

func TestNewProjectIDInputModelMinWidth(t *testing.T) {
	model := newProjectIDInputModel("prompt", "x")
	assert.Equal(t, 10, model.Width)
}

func TestNewProjectIDInputModelUsesPlaceholderWidth(t *testing.T) {
	placeholder := "long-placeholder"
	model := newProjectIDInputModel("prompt", placeholder)
	assert.Equal(t, len(placeholder), model.Width)
}

func TestRenderConfirmSuffixes(t *testing.T) {
	assert.True(t, strings.HasSuffix(renderConfirm("message", false), "(y/N)"))
	assert.True(t, strings.HasSuffix(renderConfirm("message", true), "(Y/n)"))
}

func TestRenderInputReturnsView(t *testing.T) {
	input := textinput.New()
	input.Prompt = "?"
	assert.Equal(t, input.View(), renderInput(input))
}

func TestAddTUIResultStates(t *testing.T) {
	_, _, _, err := addTUIModel{state: addTUIStateDone}.result()
	assert.Error(t, err)

	_, _, _, err = addTUIModel{state: addTUIStateAborted}.result()
	assert.True(t, errors.Is(err, errAborted))

	_, _, _, err = addTUIModel{state: addTUIStateFatalError, err: errors.New("boom")}.result()
	assert.EqualError(t, err, "boom")

	model := addTUIModel{
		state:            addTUIStateDone,
		remoteMod:        platform.RemoteMod{FileName: "mod.jar"},
		resolvedPlatform: models.MODRINTH,
		resolvedProject:  "abc",
	}
	remote, platformValue, projectID, err := model.result()
	assert.NoError(t, err)
	assert.Equal(t, "mod.jar", remote.FileName)
	assert.Equal(t, models.MODRINTH, platformValue)
	assert.Equal(t, "abc", projectID)
}

func TestAddTUIResultErrorsWhenNotFinished(t *testing.T) {
	_, _, _, err := addTUIModel{state: addTUIStateUnknownPlatformSelect}.result()
	assert.EqualError(t, err, "add TUI did not finish")
}

func TestResolveRemoteModWithTUIMissingRunTea(t *testing.T) {
	ctx := context.Background()
	remote, platformValue, projectID, err := resolveRemoteModWithTUI(ctx, nil, addTUIStateUnknownPlatformSelect, models.ModsJSON{}, addOptions{}, models.MODRINTH, "abc", addDeps{}, strings.NewReader(""), io.Discard)
	assert.Error(t, err)
	assert.Empty(t, remote.FileName)
	assert.Equal(t, models.MODRINTH, platformValue)
	assert.Equal(t, "abc", projectID)
}

func TestResolveRemoteModWithTUIRunTeaError(t *testing.T) {
	ctx := context.Background()
	deps := addDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("boom")
		},
	}
	_, _, _, err := resolveRemoteModWithTUI(ctx, nil, addTUIStateUnknownPlatformSelect, models.ModsJSON{}, addOptions{}, models.MODRINTH, "abc", deps, strings.NewReader(""), io.Discard)
	assert.Error(t, err)
}

func TestResolveRemoteModWithTUIUnexpectedModel(t *testing.T) {
	ctx := context.Background()
	deps := addDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return fakeTeaModel{}, nil
		},
	}
	_, _, _, err := resolveRemoteModWithTUI(ctx, nil, addTUIStateUnknownPlatformSelect, models.ModsJSON{}, addOptions{}, models.MODRINTH, "abc", deps, strings.NewReader(""), io.Discard)
	assert.Error(t, err)
}

func TestResolveRemoteModWithTUIResultError(t *testing.T) {
	ctx := context.Background()
	deps := addDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{state: addTUIStateDone}, nil
		},
	}
	_, _, _, err := resolveRemoteModWithTUI(ctx, nil, addTUIStateUnknownPlatformSelect, models.ModsJSON{}, addOptions{}, models.MODRINTH, "abc", deps, strings.NewReader(""), io.Discard)
	assert.Error(t, err)
}

func TestResolveRemoteModWithTUIResultAborted(t *testing.T) {
	ctx := context.Background()
	deps := addDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return addTUIModel{state: addTUIStateAborted}, nil
		},
	}
	_, _, _, err := resolveRemoteModWithTUI(ctx, nil, addTUIStateUnknownPlatformSelect, models.ModsJSON{}, addOptions{}, models.MODRINTH, "abc", deps, strings.NewReader(""), io.Discard)
	assert.True(t, errors.Is(err, errAborted))
}

func TestAddTUIStateNameForUnknown(t *testing.T) {
	model := addTUIModel{}
	assert.Equal(t, "unknown", model.stateNameFor(addTUIState(99)))
}

func TestAddTUIStateNameForAllStates(t *testing.T) {
	model := addTUIModel{}
	assert.Equal(t, "unknown_platform_select", model.stateNameFor(addTUIStateUnknownPlatformSelect))
	assert.Equal(t, "mod_not_found_confirm", model.stateNameFor(addTUIStateModNotFoundConfirm))
	assert.Equal(t, "mod_not_found_select_platform", model.stateNameFor(addTUIStateModNotFoundSelectPlatform))
	assert.Equal(t, "mod_not_found_enter_project_id", model.stateNameFor(addTUIStateModNotFoundEnterProjectID))
	assert.Equal(t, "no_file_confirm", model.stateNameFor(addTUIStateNoFileConfirm))
	assert.Equal(t, "no_file_enter_project_id", model.stateNameFor(addTUIStateNoFileEnterProjectID))
	assert.Equal(t, "fatal_error", model.stateNameFor(addTUIStateFatalError))
	assert.Equal(t, "done", model.stateNameFor(addTUIStateDone))
	assert.Equal(t, "aborted", model.stateNameFor(addTUIStateAborted))
}

type fakeListItem struct{}

func (fakeListItem) FilterValue() string { return "" }

type fakeTeaModel struct{}

func (fakeTeaModel) Init() tea.Cmd { return nil }

func (fakeTeaModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return fakeTeaModel{}, nil }

func (fakeTeaModel) View() string { return "fake" }

func initPerf(t *testing.T) {
	t.Helper()
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))
}

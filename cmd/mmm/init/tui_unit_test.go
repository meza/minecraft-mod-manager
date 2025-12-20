package init

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
)

func TestGameVersionModelInitReturnsNil(t *testing.T) {
	model := GameVersionModel{}
	assert.Nil(t, model.Init())
}

func TestGameVersionModelUpdateHandlesQuitKeys(t *testing.T) {
	input := textinput.New()
	input.Blur()
	model := GameVersionModel{input: input}
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd)

	input = textinput.New()
	model = GameVersionModel{input: input}
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
}

func TestGameVersionModelUpdateEnterEmptySetsError(t *testing.T) {
	model := GameVersionModel{
		input: textinput.New(),
		validate: func(string) error {
			return nil
		},
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
}

func TestGameVersionModelUpdateEnterInvalidSetsError(t *testing.T) {
	model := GameVersionModel{
		input: textinput.New(),
		validate: func(string) error {
			return errors.New("invalid")
		},
	}
	model.input.SetValue("nope")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
}

func TestGameVersionModelUpdateEnterValidSetsValue(t *testing.T) {
	model := GameVersionModel{
		input: textinput.New(),
		validate: func(string) error {
			return nil
		},
	}
	model.input.SetValue("1.21.1")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "1.21.1", updated.Value)
	msg := cmd()
	_, ok := msg.(GameVersionSelectedMessage)
	assert.True(t, ok)
}

func TestGameVersionModelUpdateTabFillsPlaceholder(t *testing.T) {
	model := GameVersionModel{input: textinput.New()}
	model.input.Focus()
	model.input.Placeholder = "1.20.4"
	model.input.SetValue("")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, "1.20.4", updated.input.Value())
}

func TestGameVersionModelUpdateDefaultClearsError(t *testing.T) {
	model := GameVersionModel{input: textinput.New(), error: errors.New("boom")}
	model.input.Focus()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, updated.error)
}

func TestGameVersionModelViewForValueAndError(t *testing.T) {
	model := GameVersionModel{input: textinput.New(), Value: "1.21.1"}
	assert.Contains(t, model.View(), "1.21.1")

	model = GameVersionModel{input: textinput.New(), error: errors.New("boom")}
	assert.Contains(t, model.View(), "boom")
}

func TestGameVersionSelectedMessage(t *testing.T) {
	model := GameVersionModel{input: textinput.New(), Value: "1.21.1"}
	msg := model.gameVersionSelected()()
	typed := msg.(GameVersionSelectedMessage)
	assert.Equal(t, "1.21.1", typed.GameVersion)
}

func TestNewGameVersionModelWidthsAndSuggestions(t *testing.T) {
	minecraft.ClearManifestCache()
	model := NewGameVersionModel(context.Background(), manifestDoer([]string{"1.0.0", "1.1.0"}), "long-version")
	assert.GreaterOrEqual(t, model.input.Width, len("long-version"))
	assert.True(t, model.input.ShowSuggestions)

	minecraft.ClearManifestCache()
	model = NewGameVersionModel(context.Background(), doerFunc(func(*http.Request) (*http.Response, error) {
		body := `{"latest":{"release":"1.0.0"},"versions":[]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	}), "")
	assert.False(t, model.input.ShowSuggestions)
}

func TestValidateMinecraftVersion(t *testing.T) {
	minecraft.ClearManifestCache()
	err := validateMinecraftVersion(context.Background(), "", manifestDoer([]string{"1.21.1"}))
	assert.Error(t, err)

	minecraft.ClearManifestCache()
	err = validateMinecraftVersion(context.Background(), "nope", manifestDoer([]string{"1.21.1"}))
	assert.Error(t, err)

	minecraft.ClearManifestCache()
	err = validateMinecraftVersion(context.Background(), "1.21.1", manifestDoer([]string{"1.21.1"}))
	assert.NoError(t, err)
}

func TestLoaderModelInitAndUpdate(t *testing.T) {
	model := LoaderModel{}
	assert.Nil(t, model.Init())

	model = NewLoaderModel("")
	model.list.Select(0)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, updated.Value)

	model = LoaderModel{list: list.New([]list.Item{fakeListItem{}}, itemDelegate{}, 10, 5)}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Empty(t, updated.Value)
	assert.Nil(t, cmd)
}

func TestLoaderModelViewSelected(t *testing.T) {
	model := NewLoaderModel("fabric")
	assert.Contains(t, model.View(), "fabric")
}

func TestLoaderModelTitle(t *testing.T) {
	model := NewLoaderModel("")
	assert.Equal(t, model.list.Title, model.Title())
}

func TestLoaderModelSelectedMessage(t *testing.T) {
	model := LoaderModel{Value: models.FABRIC}
	msg := model.loaderSelected()()
	typed := msg.(LoaderSelectedMessage)
	assert.Equal(t, models.FABRIC, typed.Loader)
}

func TestLoaderDelegateRender(t *testing.T) {
	model := NewLoaderModel("")
	items := model.list.Items()
	delegate := itemDelegate{}

	buffer := &bytes.Buffer{}
	delegate.Render(buffer, model.list, 0, items[0])
	assert.NotEmpty(t, buffer.String())

	buffer.Reset()
	delegate.Render(buffer, model.list, 0, fakeListItem{})
	assert.Equal(t, "", buffer.String())
}

func TestLoaderTypeFilterValue(t *testing.T) {
	item := loaderType("fabric")
	assert.Equal(t, "", item.FilterValue())
}

func TestReleaseTypesModelInitAndUpdate(t *testing.T) {
	model := NewReleaseTypesModel([]models.ReleaseType{models.Release})
	assert.Nil(t, model.Init())

	model.list.Select(2)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assert.False(t, updated.selected[models.Release])

	model = ReleaseTypesModel{
		list:     model.list,
		selected: map[models.ReleaseType]bool{},
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)

	model = NewReleaseTypesModel([]models.ReleaseType{models.Release})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, updated.Value)
}

func TestReleaseTypesModelHelpKeys(t *testing.T) {
	model := NewReleaseTypesModel([]models.ReleaseType{models.Release})
	assert.NotEmpty(t, model.list.AdditionalFullHelpKeys())
}

func TestReleaseTypesModelViewWithError(t *testing.T) {
	model := NewReleaseTypesModel([]models.ReleaseType{models.Release})
	model.error = errors.New("boom")
	assert.Contains(t, model.View(), "boom")
}

func TestReleaseTypesSelectedMessage(t *testing.T) {
	model := ReleaseTypesModel{Value: []models.ReleaseType{models.Release}}
	msg := model.releaseTypesSelected()()
	typed := msg.(ReleaseTypesSelectedMessage)
	assert.Equal(t, []models.ReleaseType{models.Release}, typed.ReleaseTypes)
}

func TestReleaseTypesToggleSelectedIgnoresUnknownItem(t *testing.T) {
	model := ReleaseTypesModel{list: list.New([]list.Item{fakeListItem{}}, releaseTypeDelegate{}, 10, 5)}
	model.toggleSelected()
}

func TestReleaseTypesFormat(t *testing.T) {
	assert.Equal(t, "release, beta", formatReleaseTypes([]models.ReleaseType{models.Release, models.Beta}))
}

func TestReleaseTypeDelegateRender(t *testing.T) {
	model := NewReleaseTypesModel([]models.ReleaseType{models.Release})
	items := model.list.Items()
	delegate := releaseTypeDelegate{}

	buffer := &bytes.Buffer{}
	delegate.Render(buffer, model.list, 0, items[0])
	assert.NotEmpty(t, buffer.String())

	buffer.Reset()
	delegate.Render(buffer, model.list, 0, fakeListItem{})
	assert.Equal(t, "", buffer.String())
}

func TestReleaseTypeItemFilterValue(t *testing.T) {
	item := releaseTypeItem{value: models.Release}
	assert.Equal(t, "release", item.FilterValue())
}

func TestModsFolderModelInitAndUpdate(t *testing.T) {
	model := ModsFolderModel{}
	assert.Nil(t, model.Init())

	input := textinput.New()
	input.Blur()
	model = ModsFolderModel{input: input}
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd)

	model = ModsFolderModel{input: textinput.New()}
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)

	input = textinput.New()
	input.Focus()
	model = ModsFolderModel{input: input}
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
}

func TestModsFolderModelUpdateTabAndEnter(t *testing.T) {
	model := ModsFolderModel{
		input: textinput.New(),
		validate: func(string) error {
			return nil
		},
	}
	model.input.Focus()
	model.input.Placeholder = "mods"
	model.input.SetValue("")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, "mods", updated.input.Value())

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "mods", updated.Value)
	assert.NotNil(t, cmd)
}

func TestModsFolderModelUpdateEnterEmptyAndInvalid(t *testing.T) {
	model := ModsFolderModel{
		input: textinput.New(),
		validate: func(string) error {
			return errors.New("invalid")
		},
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)

	model.input.SetValue("mods")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
}

func TestModsFolderModelUpdateClearsErrorOnInput(t *testing.T) {
	input := textinput.New()
	input.Focus()
	model := ModsFolderModel{
		input: input,
		error: errors.New("boom"),
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, updated.error)
}

func TestModsFolderModelViewForValueAndError(t *testing.T) {
	model := ModsFolderModel{input: textinput.New(), Value: "mods"}
	assert.Contains(t, model.View(), "mods")

	model = ModsFolderModel{input: textinput.New(), error: errors.New("boom")}
	assert.Contains(t, model.View(), "boom")
}

func TestNewModsFolderModelPrefill(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")
	assert.NoError(t, fs.MkdirAll("/cfg/mods", 0755))

	model := NewModsFolderModel("mods", meta, fs, true)
	assert.Equal(t, "mods", model.Value)
	assert.GreaterOrEqual(t, model.input.Width, 10)
}

func TestNewModsFolderModelPrefillInvalidDoesNotSetValue(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	model := NewModsFolderModel("mods", meta, fs, true)
	assert.Equal(t, "", model.Value)
}

func TestNewModsFolderModelMinWidth(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	model := NewModsFolderModel("m", meta, fs, false)
	assert.Equal(t, 10, model.input.Width)
}

func TestCommandModelInitAndView(t *testing.T) {
	model := CommandModel{state: done}
	cmd := model.Init()
	assert.NotNil(t, cmd)

	model = CommandModel{state: stateLoader}
	assert.Nil(t, model.Init())

	model = CommandModel{
		state:           stateLoader,
		initialProvided: providedFlags{},
		loaderQuestion:  NewLoaderModel(""),
	}
	assert.NotEmpty(t, model.View())
}

func TestCommandModelUpdateSelectMessages(t *testing.T) {
	initPerf(t)
	model := CommandModel{
		state:  stateLoader,
		result: initOptions{Provided: providedFlags{}},
	}

	updated, _ := model.Update(LoaderSelectedMessage{Loader: models.FABRIC})
	typed := updated.(CommandModel)
	assert.Equal(t, models.FABRIC, typed.result.Loader)
	assert.True(t, typed.result.Provided.Loader)
}

func TestCommandModelUpdateSelectAddsSpanEvent(t *testing.T) {
	initPerf(t)
	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.init.session")
	t.Cleanup(span.End)

	model := CommandModel{
		state:       stateLoader,
		ctx:         ctx,
		sessionSpan: span,
		result:      initOptions{Provided: providedFlags{}},
	}

	updated, _ := model.Update(LoaderSelectedMessage{Loader: models.FABRIC})
	typed := updated.(CommandModel)
	assert.Equal(t, models.FABRIC, typed.result.Loader)
}

func TestCommandModelUpdateAbort(t *testing.T) {
	model := CommandModel{state: stateLoader, loaderQuestion: NewLoaderModel("")}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(CommandModel)
	assert.Error(t, typed.err)
	assert.NotNil(t, cmd)
}

func TestCommandModelUpdateAbortAddsSpanEvent(t *testing.T) {
	initPerf(t)
	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "tui.init.session")
	t.Cleanup(span.End)

	model := CommandModel{
		state:          stateLoader,
		ctx:            ctx,
		sessionSpan:    span,
		loaderQuestion: NewLoaderModel(""),
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(CommandModel)
	assert.Error(t, typed.err)
	assert.NotNil(t, cmd)
}

func TestCommandModelUpdateModsFolderCompletes(t *testing.T) {
	initPerf(t)
	model := CommandModel{
		state: stateModsFolder,
		result: initOptions{
			Provided: providedFlags{
				Loader:       true,
				GameVersion:  true,
				ReleaseTypes: true,
			},
		},
	}
	updated, cmd := model.Update(ModsFolderSelectedMessage{ModsFolder: "mods"})
	typed := updated.(CommandModel)
	assert.Equal(t, done, typed.state)
	assert.NotNil(t, cmd)
}

func TestCommandModelUpdateDefaultStateQuits(t *testing.T) {
	model := CommandModel{state: state(99)}
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.NotNil(t, cmd)
}

func TestCommandModelSetStateNoopWhenEntered(t *testing.T) {
	model := CommandModel{state: stateLoader, entered: true}
	model.setState(stateLoader)
	assert.Equal(t, stateLoader, model.state)
}

func TestCommandModelStateNameUnknown(t *testing.T) {
	model := CommandModel{state: state(99)}
	assert.Equal(t, "unknown", model.stateName())
}

func TestCommandModelStartWaitDoneNoSpan(t *testing.T) {
	initPerf(t)
	model := CommandModel{state: done}
	model.startWait()
	assert.Nil(t, model.waitSpan)
}

func TestCommandModelEndWaitNoSpanNoop(t *testing.T) {
	model := CommandModel{}
	model.endWait("action")
}

func TestNextMissingState(t *testing.T) {
	result := initOptions{Provided: providedFlags{}}
	assert.Equal(t, stateLoader, nextMissingState(result))

	result.Provided.Loader = true
	assert.Equal(t, stateGameVersion, nextMissingState(result))

	result.Provided.GameVersion = true
	assert.Equal(t, stateReleaseTypes, nextMissingState(result))

	result.Provided.ReleaseTypes = true
	assert.Equal(t, stateModsFolder, nextMissingState(result))

	result.Provided.ModsFolder = true
	assert.Equal(t, done, nextMissingState(result))
}

type fakeListItem struct{}

func (fakeListItem) FilterValue() string { return "" }

package init

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
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
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestGameVersionModelInitReturnsNil(t *testing.T) {
	model := GameVersionModel{}
	assert.Nil(t, model.Init())
}

func TestGameVersionModelUpdateHandlesEscQuit(t *testing.T) {
	input := textinput.New()
	input.Blur()
	model := GameVersionModel{input: input}
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}

	input = textinput.New()
	model = GameVersionModel{input: input}
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
}

func TestGameVersionModelUpdateQWhenFocusedUpdatesInput(t *testing.T) {
	input := textinput.New()
	input.Focus()
	model := GameVersionModel{input: input}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
	assert.Equal(t, "q", updated.input.Value())
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

func TestConfigPathModelUpdateEnterEmptySetsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := NewConfigPathModel(afero.NewMemMapFs())
	assert.Nil(t, model.Init())
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
}

func TestConfigPathModelUpdateEnterValidReturnsMessage(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := NewConfigPathModel(afero.NewMemMapFs())
	model.input.SetValue("modlist2.json")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "modlist2.json", updated.Value)
	msg := cmd()
	typed := msg.(ConfigPathSelectedMessage)
	assert.Equal(t, "modlist2.json", typed.ConfigPath)
}

func TestConfigPathModelUpdateEnterExistingPathSetsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	configPath := filepath.FromSlash("/cfg/modlist.json")
	assert.NoError(t, fs.MkdirAll(filepath.Dir(configPath), 0755))
	assert.NoError(t, afero.WriteFile(fs, configPath, []byte(`{"existing":true}`), 0644))

	model := NewConfigPathModel(fs)
	model.input.SetValue(configPath)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
	assert.Contains(t, updated.error.Error(), "cmd.init.error.config-path.exists")
}

func TestConfigPathModelUpdateEnterStatErrorSetsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	statErrFs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}
	model := NewConfigPathModel(statErrFs)
	model.input.SetValue(filepath.FromSlash("/cfg/modlist.json"))
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
	assert.Contains(t, updated.error.Error(), "stat failed")
}

func TestConfigPathModelUpdateClearsErrorOnInput(t *testing.T) {
	model := ConfigPathModel{input: textinput.New(), error: errors.New("boom")}
	model.input.Focus()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, updated.error)
}

func TestConfigPathModelViewForValueAndError(t *testing.T) {
	model := ConfigPathModel{input: textinput.New(), Value: "modlist2.json"}
	assert.Contains(t, model.View(), "modlist2.json")

	model = ConfigPathModel{input: textinput.New(), error: errors.New("boom")}
	assert.Contains(t, model.View(), "boom")
}

func TestConfigPathModelUpdateEscQuits(t *testing.T) {
	model := NewConfigPathModel(afero.NewMemMapFs())
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
}

func TestConfigPathModelUpdateQWhenBlurredDoesNotQuit(t *testing.T) {
	model := NewConfigPathModel(afero.NewMemMapFs())
	model.input.Blur()
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
}

func TestConfigPathModelUpdateQWhenFocusedUpdatesInput(t *testing.T) {
	model := NewConfigPathModel(afero.NewMemMapFs())
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
	assert.Equal(t, "q", updated.input.Value())
}

func TestConfirmPromptModelUpdateQWhenFocusedUpdatesInput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
	assert.Equal(t, "q", updated.input.Value())
}

func TestConfirmPromptModelUpdateQWhenBlurredDoesNotQuit(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.input.Blur()
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
}

func TestModsFolderModelUpdateQWhenFocusedUpdatesInput(t *testing.T) {
	model := NewModsFolderModel(modsFolderModelInput{
		modsFolder: "mods",
		meta:       config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		fs:         afero.NewMemMapFs(),
	})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
	assert.Equal(t, "q", updated.input.Value())
}

func TestConfirmPromptModelDefaultsToNo(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	assert.Nil(t, model.Init())
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.noOption.short, updated.Value)
	msg := cmd()
	typed := msg.(ConfirmWriteSelectedMessage)
	assert.False(t, typed.Confirmed)
}

func TestConfirmPromptModelUpdateClearsErrorOnInput(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.error = errors.New("boom")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Nil(t, updated.error)
}

func TestConfirmPromptModelAcceptsYesShort(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.input.SetValue(model.yesOption.short)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.yesOption.short, updated.Value)
	msg := cmd()
	typed := msg.(ConfirmWriteSelectedMessage)
	assert.True(t, typed.Confirmed)
}

func TestConfirmPromptModelAcceptsYesLabel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.input.SetValue(model.yesOption.label)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.yesOption.short, updated.Value)
	msg := cmd()
	typed := msg.(ConfirmWriteSelectedMessage)
	assert.True(t, typed.Confirmed)
}

func TestConfirmPromptModelAcceptsNoLabel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.input.SetValue(model.noOption.label)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.noOption.short, updated.Value)
	msg := cmd()
	typed := msg.(ConfirmWriteSelectedMessage)
	assert.False(t, typed.Confirmed)
}

func TestConfirmPromptModelConfigOverwriteMessage(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfigOverwritePromptModel("question")
	model.input.SetValue(model.yesOption.short)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, model.yesOption.short, updated.Value)
	msg := cmd()
	typed := msg.(ConfigOverwriteSelectedMessage)
	assert.True(t, typed.Overwrite)
}

func TestConfirmPromptModelInvalidChoiceSetsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.input.SetValue("maybe")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
	assert.Nil(t, cmd)
}

func TestConfirmPromptModelConfirmSelectedFallback(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.messageBuilder = nil
	msg := model.confirmSelected(true)()
	typed := msg.(ConfirmWriteSelectedMessage)
	assert.True(t, typed.Confirmed)
}

func TestConfirmPromptModelViewForValueAndError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	model.Value = model.noOption.short
	assert.Contains(t, model.View(), model.noOption.short)

	model = newConfirmWritePromptModel("question")
	model.error = errors.New("boom")
	assert.Contains(t, model.View(), "boom")
}

func TestConfirmPromptModelUpdateEscQuits(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := newConfirmWritePromptModel("question")
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
}

func TestCommandModelHandleModsFolderSelectedDoesNotQuitImmediatelyWhenDone(t *testing.T) {
	model := CommandModel{
		ctx:   context.Background(),
		state: stateModsFolder,
		result: initOptions{
			Loader:       models.FABRIC,
			GameVersion:  "1.20.1",
			ReleaseTypes: []models.ReleaseType{models.Release},
			Provided: providedFlags{
				Loader:       true,
				GameVersion:  true,
				ReleaseTypes: true,
			},
		},
	}

	updated, cmd := model.handleModsFolderSelected(ModsFolderSelectedMessage{ModsFolder: "mods"})

	assert.Equal(t, done, updated.state)
	assert.Nil(t, cmd)
}

func TestCommandModelHandleModsFolderSelectedContinuesWhenNotDone(t *testing.T) {
	model := CommandModel{
		ctx:   context.Background(),
		state: stateModsFolder,
		result: initOptions{
			Loader:       models.FABRIC,
			GameVersion:  "1.20.1",
			ReleaseTypes: []models.ReleaseType{models.Release},
			Provided: providedFlags{
				Loader:      true,
				GameVersion: true,
			},
		},
	}

	updated, cmd := model.handleModsFolderSelected(ModsFolderSelectedMessage{ModsFolder: "mods"})

	assert.Equal(t, stateReleaseTypes, updated.state)
	assert.Nil(t, cmd)
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

func TestGameVersionModelUpdateLatestResolveError(t *testing.T) {
	model := GameVersionModel{
		input: textinput.New(),
		validate: func(string) error {
			return nil
		},
		resolveLatest: func() (string, error) {
			return "", errors.New("offline")
		},
	}
	model.input.SetValue("latest")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, updated.error)
	assert.Empty(t, updated.Value)
}

func TestGameVersionModelUpdateLatestResolves(t *testing.T) {
	model := GameVersionModel{
		input: textinput.New(),
		validate: func(string) error {
			return nil
		},
		resolveLatest: func() (string, error) {
			return "1.21.1", nil
		},
	}
	model.input.SetValue("latest")
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "1.21.1", updated.Value)
	msg := cmd()
	typed := msg.(GameVersionSelectedMessage)
	assert.Equal(t, "1.21.1", typed.GameVersion)
}

func TestCommandModelHandleConfigOverwriteSelectedOverwriteTrue(t *testing.T) {
	model := CommandModel{
		ctx:                  context.Background(),
		state:                stateConfigExists,
		showConfigExists:     true,
		configExistsQuestion: newConfigOverwritePromptModel("question"),
		configPathQuestion:   NewConfigPathModel(afero.NewMemMapFs()),
		result:               initOptions{Provided: providedFlags{}},
	}
	model.configExistsQuestion.Value = "answered"

	updated := model.handleConfigOverwriteSelected(ConfigOverwriteSelectedMessage{Overwrite: true})
	assert.False(t, updated.showConfigPath)
	assert.Equal(t, stateLoader, updated.state)
}

func TestCommandModelHandleConfigOverwriteSelectedAddsSpanEvent(t *testing.T) {
	initPerf(t)
	ctx, span := perf.StartSpan(context.Background(), "interactive.init.session")
	defer span.End()

	model := CommandModel{
		ctx:                  ctx,
		sessionSpan:          span,
		state:                stateConfigExists,
		showConfigExists:     true,
		configExistsQuestion: newConfigOverwritePromptModel("question"),
		configPathQuestion:   NewConfigPathModel(afero.NewMemMapFs()),
		result:               initOptions{Provided: providedFlags{}},
	}
	model.configExistsQuestion.Value = "answered"

	updated := model.handleConfigOverwriteSelected(ConfigOverwriteSelectedMessage{Overwrite: false})
	assert.True(t, updated.showConfigPath)
}

func TestCommandModelHandleConfigOverwriteSelectedOverwriteFalse(t *testing.T) {
	model := CommandModel{
		ctx:                  context.Background(),
		state:                stateConfigExists,
		showConfigExists:     true,
		configExistsQuestion: newConfigOverwritePromptModel("question"),
		configPathQuestion:   NewConfigPathModel(afero.NewMemMapFs()),
		result:               initOptions{Provided: providedFlags{}},
	}
	model.configExistsQuestion.Value = "answered"

	updated := model.handleConfigOverwriteSelected(ConfigOverwriteSelectedMessage{Overwrite: false})
	assert.True(t, updated.showConfigPath)
	assert.Equal(t, stateConfigPath, updated.state)
}

func TestCommandModelHandleConfigPathSelectedUpdatesMeta(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	model := CommandModel{
		ctx:                context.Background(),
		meta:               meta,
		fs:                 fs,
		configPathQuestion: NewConfigPathModel(afero.NewMemMapFs()),
		result: initOptions{
			ModsFolder: "mods",
			Provided:   providedFlags{},
		},
	}

	updated := model.handleConfigPathSelected(ConfigPathSelectedMessage{ConfigPath: filepath.FromSlash("/new/modlist.json")})
	assert.Equal(t, filepath.FromSlash("/new/modlist.json"), updated.result.ConfigPath)
	assert.Equal(t, filepath.FromSlash("/new/modlist.json"), updated.meta.ConfigPath)
	assert.Equal(t, stateLoader, updated.state)
}

func TestCommandModelHandleConfigPathSelectedInvalidatesModsFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)

	updated := model.handleConfigPathSelected(ConfigPathSelectedMessage{ConfigPath: filepath.FromSlash("/other/modlist.json")})
	assert.Equal(t, stateModsFolder, updated.state)
	assert.False(t, updated.result.Provided.ModsFolder)
	assert.False(t, updated.initialProvided.ModsFolder)
}

func TestCommandModelRevalidateModsFolderSkipsWhenNotProvided(t *testing.T) {
	model := CommandModel{
		result: initOptions{
			ModsFolder: "",
			Provided: providedFlags{
				ModsFolder: false,
			},
		},
		initialProvided: providedFlags{
			ModsFolder: true,
		},
	}

	model.revalidateModsFolder()
	assert.False(t, model.result.Provided.ModsFolder)
	assert.True(t, model.initialProvided.ModsFolder)
}

func TestCommandModelRevalidateModsFolderClearsEmptyValue(t *testing.T) {
	model := CommandModel{
		result: initOptions{
			ModsFolder: "",
			Provided: providedFlags{
				ModsFolder: true,
			},
		},
		initialProvided: providedFlags{
			ModsFolder: true,
		},
	}

	model.revalidateModsFolder()
	assert.False(t, model.result.Provided.ModsFolder)
	assert.False(t, model.initialProvided.ModsFolder)
}

func TestCommandModelRevalidateModsFolderKeepsValidValue(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	model := CommandModel{
		fs:   fs,
		meta: meta,
		result: initOptions{
			ModsFolder: "mods",
			Provided: providedFlags{
				ModsFolder: true,
			},
		},
		initialProvided: providedFlags{
			ModsFolder: true,
		},
	}

	model.revalidateModsFolder()
	assert.True(t, model.result.Provided.ModsFolder)
	assert.True(t, model.initialProvided.ModsFolder)
	assert.Equal(t, "mods", model.result.ModsFolder)
}

func TestCommandModelHandleConfigPathSelectedAddsSpanEvent(t *testing.T) {
	initPerf(t)
	ctx, span := perf.StartSpan(context.Background(), "interactive.init.session")
	defer span.End()

	model := CommandModel{
		ctx:                ctx,
		sessionSpan:        span,
		meta:               config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
		configPathQuestion: NewConfigPathModel(afero.NewMemMapFs()),
		fs:                 afero.NewMemMapFs(),
		result:             initOptions{ModsFolder: "mods"},
	}

	updated := model.handleConfigPathSelected(ConfigPathSelectedMessage{ConfigPath: filepath.FromSlash("/new/modlist.json")})
	assert.Equal(t, stateLoader, updated.state)
}

func TestCommandModelRefreshModsFolderModelUsesUpdatedMeta(t *testing.T) {
	fs := afero.NewMemMapFs()
	originalMeta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	updatedMeta := config.NewMetadata(filepath.FromSlash("/longer/path/modlist.json"))

	model := CommandModel{
		fs:   fs,
		meta: originalMeta,
		result: initOptions{
			ModsFolder: "mods",
		},
	}
	model.refreshModsFolderModel()
	originalWidth := model.modsFolderQuestion.input.Width

	model.meta = updatedMeta
	model.refreshModsFolderModel()
	assert.Greater(t, model.modsFolderQuestion.input.Width, originalWidth)
}

func TestCommandModelHandleConfirmWriteSelectedCancel(t *testing.T) {
	model := CommandModel{
		ctx:   context.Background(),
		state: stateConfirmWrite,
	}

	updated, cmd := model.handleConfirmWriteSelected(ConfirmWriteSelectedMessage{Confirmed: false})
	assert.Equal(t, done, updated.state)
	assert.ErrorIs(t, updated.err, ErrInitCanceled)
	assert.NotNil(t, cmd)
}

func TestCommandModelHandleModsFolderSelectedAddsSpanEvent(t *testing.T) {
	initPerf(t)
	ctx, span := perf.StartSpan(context.Background(), "interactive.init.session")
	defer span.End()

	model := CommandModel{
		ctx:         ctx,
		state:       stateModsFolder,
		sessionSpan: span,
		result: initOptions{
			Loader:       models.FABRIC,
			GameVersion:  "1.21.1",
			ReleaseTypes: []models.ReleaseType{models.Release},
			Provided: providedFlags{
				Loader:       true,
				GameVersion:  true,
				ReleaseTypes: true,
			},
		},
	}

	updated, _ := model.handleModsFolderSelected(ModsFolderSelectedMessage{ModsFolder: "mods"})
	assert.Equal(t, done, updated.state)
}

func TestCommandModelUpdateCtrlCAborts(t *testing.T) {
	model := CommandModel{
		ctx:                  context.Background(),
		state:                stateConfigExists,
		configExistsQuestion: newConfigOverwritePromptModel("question"),
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(CommandModel)
	assert.ErrorIs(t, typed.err, ErrInitCanceled)
	assert.NotNil(t, cmd)
}

func TestCommandModelUpdateConfigOverwriteMessage(t *testing.T) {
	model := CommandModel{
		ctx:                  context.Background(),
		state:                stateConfigExists,
		showConfigExists:     true,
		configExistsQuestion: newConfigOverwritePromptModel("question"),
		configPathQuestion:   NewConfigPathModel(afero.NewMemMapFs()),
		loaderQuestion:       NewLoaderModel(""),
	}
	model.configExistsQuestion.Value = "answered"

	updated, _ := model.Update(ConfigOverwriteSelectedMessage{Overwrite: true})
	typed := updated.(CommandModel)
	assert.Equal(t, stateLoader, typed.state)
}

func TestCommandModelUpdateConfigPathMessage(t *testing.T) {
	fs := afero.NewMemMapFs()
	model := CommandModel{
		ctx:                context.Background(),
		state:              stateConfigPath,
		showConfigPath:     true,
		configPathQuestion: NewConfigPathModel(afero.NewMemMapFs()),
		loaderQuestion:     NewLoaderModel(""),
		fs:                 fs,
		result:             initOptions{ModsFolder: "mods"},
		meta:               config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")),
	}
	model.configPathQuestion.Value = "answered"

	updated, _ := model.Update(ConfigPathSelectedMessage{ConfigPath: filepath.FromSlash("/new/modlist.json")})
	typed := updated.(CommandModel)
	assert.Equal(t, stateLoader, typed.state)
}

func TestCommandModelViewReturnsEmptyOnWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		return errors.New("write failed")
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		ModsFolder:   "mods",
		ReleaseTypes: []models.ReleaseType{models.Release},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)
	model.state = stateGameVersion

	assert.Equal(t, "", model.View())
}

func TestCommandModelViewReturnsEmptyOnConfigExistsWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		return errors.New("write failed")
	}

	model := CommandModel{
		state:                stateConfigExists,
		showConfigExists:     true,
		configExistsQuestion: newConfigOverwritePromptModel("cmd.init.prompt.config-overwrite.question"),
	}

	assert.Equal(t, "", model.View())
}

func TestCommandModelViewReturnsEmptyOnConfigPathWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		return errors.New("write failed")
	}

	model := CommandModel{
		state:              stateConfigPath,
		showConfigPath:     true,
		configPathQuestion: NewConfigPathModel(afero.NewMemMapFs()),
	}

	assert.Equal(t, "", model.View())
}

func TestRenderViewSectionsWithTrailingNewlineError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		if value == "\n" {
			return errors.New("write failed")
		}
		return writeStringToBuilder(writer, value)
	}

	result := view.RenderViewSectionsWithTrailingNewline([]string{"section"}, view.SectionSeparatorLine)
	assert.Equal(t, "", result)
}

func TestRenderViewSectionsBuilderError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		if value == "boom" {
			return errors.New("write failed")
		}
		return writeStringToBuilder(writer, value)
	}

	builder, ok := view.RenderViewSectionsBuilder([]string{"boom"}, view.SectionSeparatorLine)
	assert.False(t, ok)
	assert.Equal(t, "", builder.String())
}

func TestRenderViewSectionsBuilderSuccess(t *testing.T) {
	builder, ok := view.RenderViewSectionsBuilder([]string{"one", "two"}, view.SectionSeparatorLine)
	assert.True(t, ok)
	assert.Contains(t, builder.String(), "one")
	assert.Contains(t, builder.String(), "two")
}

func TestRenderViewSectionsBuilderSkipsEmptySection(t *testing.T) {
	builder, ok := view.RenderViewSectionsBuilder([]string{"", "one"}, view.SectionSeparatorLine)
	assert.True(t, ok)
	assert.Equal(t, "one", builder.String())
}

func TestRenderViewSectionsWithTrailingNewlineSuccess(t *testing.T) {
	result := view.RenderViewSectionsWithTrailingNewline([]string{"one"}, view.SectionSeparatorLine)
	assert.Contains(t, result, "one")
	assert.Equal(t, "\n", result[len(result)-1:])
}

func TestRenderViewSectionsWithTrailingNewlineBuilderError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		if value == "boom" {
			return errors.New("write failed")
		}
		return writeStringToBuilder(writer, value)
	}

	result := view.RenderViewSectionsWithTrailingNewline([]string{"boom"}, view.SectionSeparatorLine)
	assert.Equal(t, "", result)
}

func TestCommandModelViewReturnsEmptyOnSecondaryWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		if value == "\n" {
			return errors.New("write failed")
		}
		return writeStringToBuilder(writer, value)
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		ModsFolder:   "mods",
		ReleaseTypes: []models.ReleaseType{models.Release},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)
	model.initialProvided.Loader = true
	model.gameVersionQuestion.Value = "1.21.1"
	model.modsFolderQuestion.Value = "mods"
	model.state = stateModsFolder

	assert.Equal(t, "", model.View())
}

func TestCommandModelViewReturnsEmptyOnTrailingNewlineError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	view.WriteString = func(writer io.Writer, value string) error {
		if value == "\n" {
			return errors.New("write failed")
		}
		return writeStringToBuilder(writer, value)
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := CommandModel{
		state: done,
		modsFolderQuestion: NewModsFolderModel(modsFolderModelInput{
			modsFolder: "mods",
			meta:       meta,
			fs:         fs,
			prefill:    true,
		}),
		result: initOptions{
			ModsFolder: "mods",
			Provided: providedFlags{
				ModsFolder: true,
			},
		},
		initialProvided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
		},
	}

	assert.Equal(t, "", model.View())
}

func TestCommandModelViewReturnsEmptyOnReleaseTypesWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	callCount := 0
	view.WriteString = func(writer io.Writer, value string) error {
		callCount++
		if callCount == 3 {
			return errors.New("write failed")
		}
		return nil
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		ModsFolder:   "mods",
		ReleaseTypes: []models.ReleaseType{models.Release},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)
	model.state = stateReleaseTypes

	assert.Equal(t, "", model.View())
}

func TestCommandModelViewReturnsEmptyOnModsFolderWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})
	callCount := 0
	view.WriteString = func(writer io.Writer, value string) error {
		callCount++
		if callCount == 4 {
			return errors.New("write failed")
		}
		return nil
	}

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		ModsFolder:   "mods",
		ReleaseTypes: []models.ReleaseType{models.Release},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)
	model.state = stateModsFolder

	assert.Equal(t, "", model.View())
}

func TestViewSectionsForState(t *testing.T) {
	sections := viewSections{
		loader:       "loader",
		gameVersion:  "gameVersion",
		releaseTypes: "releaseTypes",
		modsFolder:   "modsFolder",
		confirmWrite: "confirmWrite",
	}

	assert.Equal(t, []string{"gameVersion"}, sections.forState(stateGameVersion))
	assert.Equal(t, []string{"gameVersion", "releaseTypes"}, sections.forState(stateReleaseTypes))
	assert.Equal(t, []string{"gameVersion", "releaseTypes", "modsFolder"}, sections.forState(stateModsFolder))
	assert.Equal(t, []string{"gameVersion", "releaseTypes", "modsFolder", "confirmWrite"}, sections.forState(done))
	assert.Nil(t, sections.forState(stateLoader))
}

func TestCommandModelViewSkipsProvidedLoader(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath: meta.ConfigPath,
		Loader:     models.FABRIC,
		Provided:   providedFlags{Loader: true},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)
	model.state = stateLoader

	assert.Equal(t, "", model.View())
}

func TestCommandModelViewConfigExists(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := CommandModel{
		state:                stateConfigExists,
		showConfigExists:     true,
		configExistsQuestion: newConfigOverwritePromptModel("cmd.init.prompt.config-overwrite.question"),
	}

	view := model.View()
	assert.Contains(t, view, "cmd.init.prompt.config-overwrite.question")
}

func TestCommandModelViewConfigPath(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := CommandModel{
		state:              stateConfigPath,
		showConfigPath:     true,
		configPathQuestion: NewConfigPathModel(afero.NewMemMapFs()),
	}

	view := model.View()
	assert.Contains(t, view, "cmd.init.prompt.config-path.question")
}

func TestCommandModelViewConfigPathWithConfigExistsAnswer(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := CommandModel{
		state:                stateConfigPath,
		showConfigExists:     true,
		showConfigPath:       true,
		configExistsQuestion: newConfigOverwritePromptModel("cmd.init.prompt.config-overwrite.question"),
		configPathQuestion:   NewConfigPathModel(afero.NewMemMapFs()),
	}
	model.configExistsQuestion.Value = "y"

	view := model.View()
	assert.Contains(t, view, "cmd.init.prompt.config-overwrite.question")
	assert.Contains(t, view, "cmd.init.prompt.config-path.question")
}

func TestCommandModelViewLoaderState(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := CommandModel{
		state:          stateLoader,
		loaderQuestion: NewLoaderModel(""),
	}

	view := model.View()
	assert.Contains(t, view, "bukkit")
}

func TestCommandModelViewConfirmWriteState(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	model := CommandModel{
		state:                stateConfirmWrite,
		showConfirmWrite:     true,
		confirmWriteQuestion: newConfirmWritePromptModel("cmd.init.prompt.confirm-write.question"),
	}

	view := model.View()
	assert.Contains(t, view, "cmd.init.prompt.confirm-write.question")
}

func TestCommandModelViewGameVersionState(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	minecraft.ClearManifestCache()

	model := CommandModel{
		state:               stateGameVersion,
		loaderQuestion:      NewLoaderModel(""),
		gameVersionQuestion: NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), ""),
	}

	view := model.View()
	assert.Contains(t, view, "cmd.init.prompt.game-version.question")
}

func TestCommandModelViewReleaseTypesState(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	minecraft.ClearManifestCache()

	model := CommandModel{
		state:                stateReleaseTypes,
		loaderQuestion:       NewLoaderModel(""),
		gameVersionQuestion:  NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), ""),
		releaseTypesQuestion: NewReleaseTypesModel([]models.ReleaseType{models.Release}),
	}

	view := model.View()
	assert.Contains(t, view, "alpha")
}

func TestCommandModelViewModsFolderState(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	minecraft.ClearManifestCache()

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	model := CommandModel{
		state:                stateModsFolder,
		loaderQuestion:       NewLoaderModel(""),
		gameVersionQuestion:  NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), ""),
		releaseTypesQuestion: NewReleaseTypesModel([]models.ReleaseType{models.Release}),
		modsFolderQuestion: NewModsFolderModel(modsFolderModelInput{
			modsFolder: "mods",
			meta:       meta,
			fs:         afero.NewMemMapFs(),
			prefill:    false,
		}),
	}

	view := model.View()
	assert.Contains(t, view, "cmd.init.prompt.mods-folder.question")
}

func TestCommandModelViewDoneWithoutSectionsReturnsEmpty(t *testing.T) {
	model := CommandModel{
		state: done,
		initialProvided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}
	assert.Equal(t, "", model.View())
}

func TestCommandModelViewDoneStateRendersSummary(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		ModsFolder:   "mods",
		ReleaseTypes: []models.ReleaseType{models.Release},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)
	model.state = done

	assert.NotEmpty(t, model.View())
}

func TestCommandModelViewSkipsProvidedGameVersion(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:  meta.ConfigPath,
		GameVersion: "1.21.1",
		Provided:    providedFlags{GameVersion: true},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)
	model.state = stateGameVersion

	assert.NotEmpty(t, model.View())
}

func TestNewModelTreatsInvalidProvidedGameVersionAsMissing(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "nope",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)

	assert.False(t, model.initialProvided.GameVersion)
	assert.Equal(t, stateGameVersion, model.state)
}

func TestNewModelTreatsInvalidProvidedModsFolderAsMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "missing",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)

	assert.False(t, model.initialProvided.ModsFolder)
	assert.Equal(t, stateModsFolder, model.state)
}

func TestNewModelUsesDefaultModsFolderWhenOtherValuesProvided(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   false,
		},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)

	assert.True(t, model.initialProvided.ModsFolder)
	assert.Equal(t, stateConfirmWrite, model.state)
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
	model := GameVersionModel{
		input: textinput.New(),
		error: errors.New("boom"),
	}
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

func TestNewGameVersionModelUsesProvidedVersionWhenValid(t *testing.T) {
	minecraft.ClearManifestCache()
	model := NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), "1.21.1")
	assert.Equal(t, "1.21.1", model.Value)
}

func TestNewGameVersionModelSkipsLatestProvidedValue(t *testing.T) {
	minecraft.ClearManifestCache()
	model := NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), "latest")
	assert.Empty(t, model.Value)
	assert.Equal(t, "", model.input.Value())
}

func TestNewGameVersionModelHandlesLatestLookupError(t *testing.T) {
	minecraft.ClearManifestCache()
	model := NewGameVersionModel(context.Background(), doerFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("offline")
	}), "")
	assert.Equal(t, "", model.input.Placeholder)
}

func TestNewGameVersionModelInvalidProvidedVersionDoesNotSetValue(t *testing.T) {
	minecraft.ClearManifestCache()
	model := NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), "nope")
	assert.Equal(t, "nope", model.input.Value())
	assert.Empty(t, model.Value)
}

func TestNewGameVersionModelValidateAndResolveLatest(t *testing.T) {
	minecraft.ClearManifestCache()
	model := NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), "")
	assert.NoError(t, model.validate("1.21.1"))

	latest, err := model.resolveLatest()
	assert.NoError(t, err)
	assert.Equal(t, "1.21.1", latest)
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

	minecraft.ClearManifestCache()
	err = validateMinecraftVersion(context.Background(), "1.21.1", doerFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("offline")
	}))
	assert.Error(t, err)
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

func TestLoaderDelegateRenderHandlesWriteError(t *testing.T) {
	model := NewLoaderModel("")
	items := model.list.Items()
	delegate := itemDelegate{}
	if len(items) < 2 {
		t.Fatalf("expected at least two items, got %d", len(items))
	}

	model.list.Select(0)
	writerErr := errors.New("write failed")
	assert.NotPanics(t, func() {
		delegate.Render(errorWriter{err: writerErr}, model.list, 0, items[0])
	})
	assert.NotPanics(t, func() {
		delegate.Render(errorWriter{err: writerErr}, model.list, 1, items[1])
	})
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

func TestReleaseTypeDelegateRenderHandlesWriteError(t *testing.T) {
	model := NewReleaseTypesModel([]models.ReleaseType{models.Release})
	items := model.list.Items()
	delegate := releaseTypeDelegate{}
	if len(items) < 2 {
		t.Fatalf("expected at least two items, got %d", len(items))
	}

	model.list.Select(0)
	writerErr := errors.New("write failed")
	assert.NotPanics(t, func() {
		delegate.Render(errorWriter{err: writerErr}, model.list, 0, items[0])
	})
	assert.NotPanics(t, func() {
		delegate.Render(errorWriter{err: writerErr}, model.list, 1, items[1])
	})
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
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}

	model = ModsFolderModel{input: textinput.New()}
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)

	input = textinput.New()
	input.Focus()
	model = ModsFolderModel{input: input}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		assert.False(t, isQuit)
	}
	assert.Equal(t, "q", updated.input.Value())
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

	model := NewModsFolderModel(modsFolderModelInput{modsFolder: "mods", meta: meta, fs: fs, prefill: true})
	assert.Equal(t, "mods", model.Value)
	assert.GreaterOrEqual(t, model.input.Width, 10)
}

func TestNewModsFolderModelPrefillInvalidDoesNotSetValue(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("/cfg/modlist.json")

	model := NewModsFolderModel(modsFolderModelInput{modsFolder: "mods", meta: meta, fs: fs, prefill: true})
	assert.Equal(t, "", model.Value)
}

func TestNewModsFolderModelMinWidth(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	model := NewModsFolderModel(modsFolderModelInput{modsFolder: "m", meta: meta, fs: fs, prefill: false})
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
		ctx:    context.Background(),
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
	_, span := perf.StartSpan(ctx, "interactive.init.session")
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
	model := CommandModel{ctx: context.Background(), state: stateLoader, loaderQuestion: NewLoaderModel("")}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(CommandModel)
	assert.Error(t, typed.err)
	assert.NotNil(t, cmd)
}

func TestCommandModelUpdateAbortAddsSpanEvent(t *testing.T) {
	initPerf(t)
	ctx := context.Background()
	_, span := perf.StartSpan(ctx, "interactive.init.session")
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
		ctx:   context.Background(),
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
	model := CommandModel{ctx: context.Background(), state: state(99)}
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.NotNil(t, cmd)
}

func TestCommandModelUpdateCurrentStateConfigExists(t *testing.T) {
	model := CommandModel{
		state:                stateConfigExists,
		showConfigExists:     true,
		configExistsQuestion: newConfigOverwritePromptModel("question"),
	}

	updated, _ := model.updateCurrentState(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	assert.Equal(t, "y", updated.configExistsQuestion.input.Value())
}

func TestCommandModelUpdateCurrentStateConfigPath(t *testing.T) {
	model := CommandModel{
		state:              stateConfigPath,
		showConfigPath:     true,
		configPathQuestion: NewConfigPathModel(afero.NewMemMapFs()),
	}

	updated, _ := model.updateCurrentState(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, "a", updated.configPathQuestion.input.Value())
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

func TestCommandModelStateNameConfigStates(t *testing.T) {
	model := CommandModel{state: stateConfigExists}
	assert.Equal(t, "config_exists", model.stateName())
	model.state = stateConfigPath
	assert.Equal(t, "config_path", model.stateName())
	model.state = stateConfirmWrite
	assert.Equal(t, "confirm_write", model.stateName())
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
	model := &CommandModel{
		result:           initOptions{Provided: providedFlags{}},
		showConfirmWrite: false,
	}
	assert.Equal(t, stateLoader, nextMissingState(model))

	model.showConfigExists = true
	assert.Equal(t, stateConfigExists, nextMissingState(model))
	model.configExistsQuestion.Value = "answered"

	model.showConfigPath = true
	assert.Equal(t, stateConfigPath, nextMissingState(model))
	model.configPathQuestion.Value = "config.json"

	model.result.Provided.Loader = true
	assert.Equal(t, stateGameVersion, nextMissingState(model))

	model.result.Provided.GameVersion = true
	assert.Equal(t, stateReleaseTypes, nextMissingState(model))

	model.result.Provided.ReleaseTypes = true
	assert.Equal(t, stateModsFolder, nextMissingState(model))

	model.result.Provided.ModsFolder = true
	assert.Equal(t, done, nextMissingState(model))
}

func writeStringToBuilder(writer io.Writer, value string) error {
	builder, ok := writer.(*strings.Builder)
	if !ok {
		return errors.New("unexpected writer")
	}
	_, err := builder.WriteString(value)
	return err
}

type fakeListItem struct{}

func (fakeListItem) FilterValue() string { return "" }

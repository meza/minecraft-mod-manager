package init

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

// GameVersionSelectedMessage signals a selected game version.
type GameVersionSelectedMessage struct {
	GameVersion string
}

// GameVersionModel drives the game version prompt UI.
type GameVersionModel struct {
	tea.Model
	input  textinput.Model
	help   help.Model
	keymap tui.TranslatedInputKeyMap
	error  error
	Value  string

	validate func(string) error
}

// Init implements tea.Model.
func (model GameVersionModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (model GameVersionModel) Update(msg tea.Msg) (GameVersionModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		updated, cmd, handled := model.handleKeyMsg(keyMsg)
		model = updated
		if handled {
			return model, cmd
		}
	}

	updatedInput, cmd := model.input.Update(msg)
	model.input = updatedInput
	return model, cmd
}

// View renders the game version prompt.
func (model GameVersionModel) View() string {
	if model.Value != "" {
		return fmt.Sprintf("%s%s", model.input.Prompt, tui.SelectedItemStyle.Render(model.Value))
	}

	errorString := ""

	if model.error != nil {
		errorString = tui.ErrorStyle.Render(" <- " + model.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", model.input.View(), errorString, model.help.View(model.keymap))
}

func (model GameVersionModel) handleKeyMsg(msg tea.KeyMsg) (GameVersionModel, tea.Cmd, bool) {
	switch msg.String() {
	case "q":
		if !model.input.Focused() {
			return model, tea.Quit, true
		}
		return model, nil, false
	case "esc":
		return model, tea.Quit, true
	case "enter":
		return model.handleEnterKey()
	case "tab":
		if model.input.Focused() && model.input.Value() == "" {
			model.input.SetValue(model.input.Placeholder)
		}
		return model, nil, false
	default:
		if model.input.Focused() {
			model.error = nil
		}
		return model, nil, false
	}
}

func (model GameVersionModel) handleEnterKey() (GameVersionModel, tea.Cmd, bool) {
	value := strings.TrimSpace(model.input.Value())
	if value == "" {
		value = strings.TrimSpace(model.input.Placeholder)
	}

	if value == "" {
		model.error = fmt.Errorf("%s", i18n.T("cmd.init.tui.game-version.error"))
		return model, nil, true
	}

	err := model.validate(value)
	if err != nil {
		model.error = err
		return model, nil, true
	}

	model.Value = value
	model.input.SetValue(value)
	return model, model.gameVersionSelected(), true
}

func (model GameVersionModel) gameVersionSelected() tea.Cmd {
	model.input.Blur()
	return func() tea.Msg {
		return GameVersionSelectedMessage{GameVersion: model.Value}
	}
}

// NewGameVersionModel builds a game version prompt model.
func NewGameVersionModel(ctx context.Context, minecraftClient httpclient.Doer, gameVersion string) GameVersionModel {
	latestVersion, err := minecraft.GetLatestVersion(ctx, minecraftClient)
	if err != nil {
		latestVersion = ""
	}
	allVersions := minecraft.GetAllMineCraftVersions(ctx, minecraftClient)

	inputModel := textinput.New()
	inputModel.Prompt = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(i18n.T("cmd.init.tui.game-version.question")) + " "
	inputModel.Placeholder = latestVersion
	inputModel.PlaceholderStyle = tui.PlaceholderStyle
	width := len(inputModel.Placeholder)
	if len(gameVersion) > width {
		width = len(gameVersion)
	}
	const minWidth = 10
	if width < minWidth {
		width = minWidth
	}
	inputModel.Width = width
	if len(allVersions) > 0 {
		inputModel.ShowSuggestions = true
	}
	inputModel.SetSuggestions(allVersions)
	inputModel.Focus()

	model := GameVersionModel{
		input:  inputModel,
		help:   help.New(),
		keymap: tui.TranslatedInputKeyMap{},
		validate: func(value string) error {
			return validateMinecraftVersion(ctx, value, minecraftClient)
		},
	}

	if gameVersion != "" && !strings.EqualFold(gameVersion, "latest") && model.validate(gameVersion) == nil {
		model.Value = gameVersion
		model.input.SetValue(gameVersion)
	}

	return model
}

func validateMinecraftVersion(ctx context.Context, value string, client httpclient.Doer) error {
	if value == "" {
		return fmt.Errorf("%s", i18n.T("cmd.init.tui.game-version.error"))
	}

	if !minecraft.IsValidVersion(ctx, value, client) {
		return fmt.Errorf("%s", i18n.T("cmd.init.tui.game-version.invalid"))
	}
	return nil
}

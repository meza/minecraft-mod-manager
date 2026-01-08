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
	"github.com/meza/minecraft-mod-manager/internal/view"
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
	keymap view.TranslatedInputKeyMap
	error  error
	Value  string

	validate      func(string) error
	resolveLatest func() (string, error)
	messages      GameVersionPromptMessages
}

// GameVersionPromptMessages holds prompt copy for the game version selector.
type GameVersionPromptMessages struct {
	Empty             string
	Invalid           string
	Unavailable       string
	LatestUnavailable string
}

// GameVersionPromptOptions configures the prompt question and validation copy.
type GameVersionPromptOptions struct {
	Question     string
	Messages     GameVersionPromptMessages
	InitialError string
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
		return fmt.Sprintf("%s%s", model.input.Prompt, view.SelectedItemStyle.Render(model.Value))
	}

	errorString := ""
	if model.error != nil {
		errorString = view.ErrorStyle.Render(" <- " + model.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", model.input.View(), errorString, model.help.View(model.keymap))
}

func (model GameVersionModel) handleKeyMsg(msg tea.KeyMsg) (GameVersionModel, tea.Cmd, bool) {
	switch msg.String() {
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
		model.error = fmt.Errorf("%s", model.messages.Empty)
		return model, nil, true
	}

	if strings.EqualFold(value, "latest") {
		resolved, err := model.resolveLatest()
		if err != nil {
			model.error = fmt.Errorf("%s", model.messages.LatestUnavailable)
			return model, nil, true
		}
		value = resolved
		model.input.SetValue(value)
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
	return NewGameVersionPromptModel(ctx, minecraftClient, gameVersion, GameVersionPromptOptions{
		Question: i18n.T("cmd.init.prompt.game-version.question", nil),
		Messages: defaultGameVersionPromptMessages(),
	})
}

// NewGameVersionPromptModel builds a game version prompt model with custom messaging.
func NewGameVersionPromptModel(
	ctx context.Context,
	minecraftClient httpclient.Doer,
	gameVersion string,
	options GameVersionPromptOptions,
) GameVersionModel {
	options = normalizeGameVersionPromptOptions(options)
	latestVersion := fetchLatestVersion(ctx, minecraftClient)
	allVersions := minecraft.GetAllMinecraftVersions(ctx, minecraftClient)

	inputModel := buildGameVersionInputModel(latestVersion, allVersions, gameVersion, options.Question)

	model := GameVersionModel{
		input:  inputModel,
		help:   help.New(),
		keymap: view.TranslatedInputKeyMap{},
		validate: func(value string) error {
			return validateMinecraftVersionWithMessages(ctx, value, minecraftClient, options.Messages)
		},
		resolveLatest: func() (string, error) {
			return minecraft.GetLatestVersion(ctx, minecraftClient)
		},
		messages: options.Messages,
	}

	applyInitialGameVersion(&model, gameVersion)
	if strings.TrimSpace(options.InitialError) != "" {
		model.error = fmt.Errorf("%s", options.InitialError)
	}

	return model
}

func normalizeGameVersionPromptOptions(options GameVersionPromptOptions) GameVersionPromptOptions {
	if strings.TrimSpace(options.Question) == "" {
		options.Question = i18n.T("cmd.init.prompt.game-version.question", nil)
	}

	defaultMessages := defaultGameVersionPromptMessages()
	if strings.TrimSpace(options.Messages.Empty) == "" {
		options.Messages.Empty = defaultMessages.Empty
	}
	if strings.TrimSpace(options.Messages.Invalid) == "" {
		options.Messages.Invalid = defaultMessages.Invalid
	}
	if strings.TrimSpace(options.Messages.Unavailable) == "" {
		options.Messages.Unavailable = defaultMessages.Unavailable
	}
	if strings.TrimSpace(options.Messages.LatestUnavailable) == "" {
		options.Messages.LatestUnavailable = defaultMessages.LatestUnavailable
	}

	return options
}

func fetchLatestVersion(ctx context.Context, minecraftClient httpclient.Doer) string {
	latestVersion, err := minecraft.GetLatestVersion(ctx, minecraftClient)
	if err != nil {
		return ""
	}
	return latestVersion
}

func buildGameVersionInputModel(
	latestVersion string,
	allVersions []string,
	gameVersion string,
	question string,
) textinput.Model {
	inputModel := textinput.New()
	inputModel.Prompt = view.QuestionStyle.Render("? ") + view.TitleStyle.Render(question) + " "
	inputModel.Placeholder = latestVersion
	inputModel.PlaceholderStyle = view.PlaceholderStyle
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
	if gameVersion != "" && !strings.EqualFold(gameVersion, "latest") {
		inputModel.SetValue(gameVersion)
	}
	return inputModel
}

func applyInitialGameVersion(model *GameVersionModel, gameVersion string) {
	if gameVersion == "" || strings.EqualFold(gameVersion, "latest") {
		return
	}

	if err := model.validate(gameVersion); err == nil {
		model.Value = gameVersion
	}
}

func defaultGameVersionPromptMessages() GameVersionPromptMessages {
	return GameVersionPromptMessages{
		Empty:             i18n.T("cmd.init.prompt.game-version.error", nil),
		Invalid:           i18n.T("cmd.init.prompt.game-version.invalid", nil),
		Unavailable:       i18n.T("cmd.init.prompt.game-version.unavailable", nil),
		LatestUnavailable: i18n.T("cmd.init.prompt.game-version.latest-unavailable", nil),
	}
}

func validateMinecraftVersion(ctx context.Context, value string, client httpclient.Doer) error {
	return validateMinecraftVersionWithMessages(ctx, value, client, defaultGameVersionPromptMessages())
}

func validateMinecraftVersionWithMessages(
	ctx context.Context,
	value string,
	client httpclient.Doer,
	messages GameVersionPromptMessages,
) error {
	if value == "" {
		return fmt.Errorf("%s", messages.Empty)
	}

	valid, validationErr := minecraft.IsValidVersion(ctx, value, client)
	if !valid && validationErr == nil {
		return fmt.Errorf("%s", messages.Invalid)
	}
	if validationErr != nil {
		return fmt.Errorf("%s", messages.Unavailable)
	}
	return nil
}

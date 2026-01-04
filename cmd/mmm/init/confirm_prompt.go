package init

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type confirmOption struct {
	label string
	short string
}

type confirmPromptModel struct {
	input  textinput.Model
	help   help.Model
	keymap view.TranslatedInputKeyMap
	error  error
	Value  string

	yesOption      confirmOption
	noOption       confirmOption
	messageBuilder func(bool) tea.Msg
}

type ConfigOverwriteSelectedMessage struct {
	Overwrite bool
}

type ConfirmWriteSelectedMessage struct {
	Confirmed bool
}

func newConfigOverwritePromptModel(question string) confirmPromptModel {
	return newConfirmPromptModel(question, func(confirmed bool) tea.Msg {
		return ConfigOverwriteSelectedMessage{Overwrite: confirmed}
	})
}

func newConfirmWritePromptModel(question string) confirmPromptModel {
	return newConfirmPromptModel(question, func(confirmed bool) tea.Msg {
		return ConfirmWriteSelectedMessage{Confirmed: confirmed}
	})
}

func newConfirmPromptModel(question string, messageBuilder func(bool) tea.Msg) confirmPromptModel {
	yesOption := confirmOption{
		label: i18n.T("cmd.init.prompt.option.yes.label", nil),
		short: i18n.T("cmd.init.prompt.option.yes.short", nil),
	}
	noOption := confirmOption{
		label: i18n.T("cmd.init.prompt.option.no.label", nil),
		short: i18n.T("cmd.init.prompt.option.no.short", nil),
	}

	promptText := buildConfirmPrompt(question, yesOption.short, noOption.short)

	inputModel := textinput.New()
	inputModel.Prompt = promptText
	inputModel.Width = 10
	inputModel.Focus()

	return confirmPromptModel{
		input:          inputModel,
		help:           help.New(),
		keymap:         view.TranslatedInputKeyMap{},
		yesOption:      yesOption,
		noOption:       noOption,
		messageBuilder: messageBuilder,
	}
}

func buildConfirmPrompt(question string, yesShort string, noShort string) string {
	questionPrefix := view.QuestionStyle.Render("? ")
	questionText := view.TitleStyle.Render(question)
	suffixTemplate := i18n.T("cmd.init.prompt.confirm.suffix", nil)
	suffix := suffixTemplate
	if strings.Contains(suffixTemplate, "%") {
		suffix = fmt.Sprintf(suffixTemplate, yesShort, noShort, noShort)
	} else if !strings.HasPrefix(suffixTemplate, " ") {
		suffix = " " + suffixTemplate
	}
	if !strings.HasSuffix(suffix, " ") {
		suffix += " "
	}
	return questionPrefix + questionText + suffix
}

func (model confirmPromptModel) Init() tea.Cmd {
	return nil
}

func (model confirmPromptModel) Update(msg tea.Msg) (confirmPromptModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		updated, cmd, handled := model.handleKeyMsg(keyMsg)
		model = updated
		if handled {
			return model, cmd
		}
	}

	var cmd tea.Cmd
	model.input, cmd = model.input.Update(msg)
	return model, cmd
}

func (model confirmPromptModel) View() string {
	if model.Value != "" {
		return fmt.Sprintf("%s%s", model.input.Prompt, view.SelectedItemStyle.Render(model.Value))
	}

	errorString := ""
	if model.error != nil {
		errorString = view.ErrorStyle.Render(" <- " + model.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", model.input.View(), errorString, model.help.View(model.keymap))
}

func (model confirmPromptModel) handleKeyMsg(msg tea.KeyMsg) (confirmPromptModel, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		return model, tea.Quit, true
	case "enter":
		return model.handleEnterKey()
	default:
		if model.input.Focused() {
			model.error = nil
		}
		return model, nil, false
	}
}

func (model confirmPromptModel) handleEnterKey() (confirmPromptModel, tea.Cmd, bool) {
	rawValue := strings.TrimSpace(model.input.Value())
	if rawValue == "" {
		model.Value = model.noOption.short
		model.input.SetValue(model.Value)
		return model, model.confirmSelected(false), true
	}

	if model.matchesOption(rawValue, model.yesOption) {
		model.Value = model.yesOption.short
		model.input.SetValue(model.Value)
		return model, model.confirmSelected(true), true
	}

	if model.matchesOption(rawValue, model.noOption) {
		model.Value = model.noOption.short
		model.input.SetValue(model.Value)
		return model, model.confirmSelected(false), true
	}

	model.error = fmt.Errorf("%s", i18n.T("cmd.init.prompt.error.invalid_choice", &i18n.Tvars{
		Data: &i18n.TData{
			"yesShort": model.yesOption.short,
			"noShort":  model.noOption.short,
		},
	}))
	return model, nil, true
}

func (model confirmPromptModel) confirmSelected(confirmed bool) tea.Cmd {
	model.input.Blur()
	return func() tea.Msg {
		if model.messageBuilder == nil {
			return ConfirmWriteSelectedMessage{Confirmed: confirmed}
		}
		return model.messageBuilder(confirmed)
	}
}

func (model confirmPromptModel) matchesOption(value string, option confirmOption) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.EqualFold(value, strings.ToLower(option.short)) ||
		strings.EqualFold(value, strings.ToLower(option.label))
}

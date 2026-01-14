package prune

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/cobra"
)

type confirmOption struct {
	label string
	short string
}

type confirmSelectedMessage struct {
	confirmed bool
}

type confirmPromptModel struct {
	input  textinput.Model
	help   help.Model
	keymap view.TranslatedInputKeyMap
	error  error
	value  string

	yesOption confirmOption
	noOption  confirmOption
}

func newConfirmPromptModel(question string) confirmPromptModel {
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
		input:     inputModel,
		help:      help.New(),
		keymap:    view.TranslatedInputKeyMap{},
		yesOption: yesOption,
		noOption:  noOption,
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
	if model.value != "" {
		return fmt.Sprintf("%s%s", model.input.Prompt, view.SelectedItemStyle.Render(model.value))
	}

	errorString := ""
	if model.error != nil {
		errorString = view.ErrorStyle.Render(" <- " + model.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", model.input.View(), errorString, model.help.View(model.keymap))
}

func (model confirmPromptModel) handleKeyMsg(msg tea.KeyMsg) (confirmPromptModel, tea.Cmd, bool) {
	switch msg.String() {
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
		model.value = model.noOption.short
		model.input.SetValue(model.value)
		return model, model.confirmSelected(false), true
	}

	if model.matchesOption(rawValue, model.yesOption) {
		model.value = model.yesOption.short
		model.input.SetValue(model.value)
		return model, model.confirmSelected(true), true
	}

	if model.matchesOption(rawValue, model.noOption) {
		model.value = model.noOption.short
		model.input.SetValue(model.value)
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
		return confirmSelectedMessage{confirmed: confirmed}
	}
}

func (model confirmPromptModel) matchesOption(value string, option confirmOption) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.EqualFold(value, strings.ToLower(option.short)) ||
		strings.EqualFold(value, strings.ToLower(option.label))
}

type configInitModel struct {
	headline  string
	prompt    confirmPromptModel
	canceled  bool
	confirmed bool
}

func newConfigInitModel(headline string, question string) configInitModel {
	return configInitModel{
		headline: headline,
		prompt:   newConfirmPromptModel(question),
	}
}

func (model configInitModel) Init() tea.Cmd {
	return nil
}

func (model configInitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" {
			model.canceled = true
			return model, tea.Quit
		}
	case confirmSelectedMessage:
		model.confirmed = typed.confirmed
		return model, tea.Quit
	}

	updatedPrompt, cmd := model.prompt.Update(msg)
	model.prompt = updatedPrompt
	return model, cmd
}

func (model configInitModel) View() string {
	if model.headline == "" {
		return model.prompt.View()
	}
	return model.headline + "\n\n" + model.prompt.View()
}

func runConfigInitPrompt(cmd *cobra.Command, deps pruneDeps, meta config.Metadata) (confirmed bool, canceled bool, err error) {
	colorMode := colorModeForWriter(cmd)
	headline := messageWithIcon(view.FinalErrorIcon(colorMode), i18n.T("cmd.config.error.missing", &i18n.Tvars{
		Data: &i18n.TData{"configPath": meta.ConfigPath},
	}))
	if colorMode.Enabled() {
		headline = view.ErrorStyle.Render(headline)
	}

	question := i18n.T("cmd.config.confirm_init", nil)
	model := newConfigInitModel(headline, question)

	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}

	result, err := runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return false, false, err
	}
	return configInitResult(result)
}

func configInitResult(result tea.Model) (confirmed bool, canceled bool, err error) {
	switch typed := result.(type) {
	case configInitModel:
		return typed.confirmed, typed.canceled, nil
	case *configInitModel:
		return typed.confirmed, typed.canceled, nil
	default:
		return false, false, errors.New("unexpected prompt model")
	}
}

type pruneConfirmModel struct {
	listView  string
	prompt    confirmPromptModel
	canceled  bool
	confirmed bool
}

func newPruneConfirmModel(listView string, question string) pruneConfirmModel {
	return pruneConfirmModel{
		listView: listView,
		prompt:   newConfirmPromptModel(question),
	}
}

func (model pruneConfirmModel) Init() tea.Cmd {
	return nil
}

func (model pruneConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" {
			model.canceled = true
			return model, tea.Quit
		}
	case confirmSelectedMessage:
		model.confirmed = typed.confirmed
		return model, tea.Quit
	}

	updatedPrompt, cmd := model.prompt.Update(msg)
	model.prompt = updatedPrompt
	return model, cmd
}

func (model pruneConfirmModel) View() string {
	if model.listView == "" {
		return model.prompt.View()
	}
	sections := []string{model.listView, model.prompt.View()}
	return view.RenderViewSections(sections, view.SectionSeparatorParagraph)
}

func runDeletePrompt(cmd *cobra.Command, deps pruneDeps, colorMode view.ColorMode, unmanaged []string) (confirmed bool, canceled bool, err error) {
	listView := renderUnmanagedList(colorMode, unmanaged)
	question := i18n.T("cmd.prune.confirm", nil)
	model := newPruneConfirmModel(listView, question)

	runTea := deps.runTea
	if runTea == nil {
		runTea = runTeaProgram
	}

	result, err := runTea(model, view.ProgramOptions(cmd.InOrStdin(), cmd.OutOrStdout())...)
	if err != nil {
		return false, false, err
	}
	return pruneConfirmResult(result)
}

func pruneConfirmResult(result tea.Model) (confirmed bool, canceled bool, err error) {
	switch typed := result.(type) {
	case pruneConfirmModel:
		return typed.confirmed, typed.canceled, nil
	case *pruneConfirmModel:
		return typed.confirmed, typed.canceled, nil
	default:
		return false, false, errors.New("unexpected prompt model")
	}
}

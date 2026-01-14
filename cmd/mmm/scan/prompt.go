package scan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type scanConfirmOption struct {
	label string
	short string
}

type scanConfirmKeyMap struct{}

func (scanConfirmKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{view.Accept(), view.QuitWithEsc()}
}

func (scanConfirmKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{view.Accept(), view.QuitWithEsc()}}
}

type scanConfirmPromptModel struct {
	input  textinput.Model
	help   help.Model
	keymap scanConfirmKeyMap
	error  error
	Value  string

	yesOption scanConfirmOption
	noOption  scanConfirmOption
}

type scanConfirmSelectedMessage struct {
	confirmed bool
}

type scanAdoptionPromptModel struct {
	sections    []string
	prompt      scanConfirmPromptModel
	canceled    bool
	confirmed   bool
	viewport    viewport.Model
	windowW     int
	windowH     int
	focusBottom bool
}

func newScanConfirmPromptModel(question string) scanConfirmPromptModel {
	yesOption := scanConfirmOption{
		label: i18n.T("cmd.init.prompt.option.yes.label", nil),
		short: i18n.T("cmd.init.prompt.option.yes.short", nil),
	}
	noOption := scanConfirmOption{
		label: i18n.T("cmd.init.prompt.option.no.label", nil),
		short: i18n.T("cmd.init.prompt.option.no.short", nil),
	}

	promptText := buildScanConfirmPrompt(question, yesOption.short, noOption.short)

	inputModel := textinput.New()
	inputModel.Prompt = promptText
	inputModel.Width = 10
	inputModel.Focus()

	return scanConfirmPromptModel{
		input:     inputModel,
		help:      help.New(),
		keymap:    scanConfirmKeyMap{},
		yesOption: yesOption,
		noOption:  noOption,
	}
}

func buildScanConfirmPrompt(question string, yesShort string, noShort string) string {
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

func (model scanConfirmPromptModel) Init() tea.Cmd {
	return nil
}

func (model scanConfirmPromptModel) Update(msg tea.Msg) (scanConfirmPromptModel, tea.Cmd) {
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

func (model scanConfirmPromptModel) View() string {
	if model.Value != "" {
		return fmt.Sprintf("%s%s", model.input.Prompt, view.SelectedItemStyle.Render(model.Value))
	}

	errorString := ""
	if model.error != nil {
		errorString = view.ErrorStyle.Render(" <- " + model.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", model.input.View(), errorString, model.help.View(model.keymap))
}

func (model scanConfirmPromptModel) handleKeyMsg(msg tea.KeyMsg) (scanConfirmPromptModel, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		return model, tea.Quit, true
	case "ctrl+c":
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

func (model scanConfirmPromptModel) handleEnterKey() (scanConfirmPromptModel, tea.Cmd, bool) {
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

func (model scanConfirmPromptModel) confirmSelected(confirmed bool) tea.Cmd {
	model.input.Blur()
	return func() tea.Msg {
		return scanConfirmSelectedMessage{confirmed: confirmed}
	}
}

func (model scanConfirmPromptModel) matchesOption(value string, option scanConfirmOption) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.EqualFold(value, strings.ToLower(option.short)) ||
		strings.EqualFold(value, strings.ToLower(option.label))
}

func newScanAdoptionPromptModel(sections []string, question string) *scanAdoptionPromptModel {
	viewportModel := viewport.New(0, 0)
	viewportModel.MouseWheelEnabled = true
	return &scanAdoptionPromptModel{
		sections:    sections,
		prompt:      newScanConfirmPromptModel(question),
		viewport:    viewportModel,
		focusBottom: true,
	}
}

func (model *scanAdoptionPromptModel) Init() tea.Cmd {
	return nil
}

func (model *scanAdoptionPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		model.windowW = typed.Width
		model.windowH = typed.Height
		model.focusBottom = true
		return model, nil
	case tea.KeyMsg:
		if typed.String() == "ctrl+c" || typed.String() == "esc" {
			model.canceled = true
			return model, tea.Quit
		}
	case scanConfirmSelectedMessage:
		model.confirmed = typed.confirmed
		return model, tea.Quit
	}

	updatedPrompt, cmd := model.prompt.Update(msg)
	model.prompt = updatedPrompt

	viewportCmd := model.updateViewportForMessage(msg)
	return model, tea.Batch(cmd, viewportCmd)
}

func (model *scanAdoptionPromptModel) View() string {
	sections := append(cloneStrings(model.sections), model.prompt.View())
	content := view.RenderViewSections(sections, view.SectionSeparatorParagraph)
	if model.windowH <= 0 || content == "" {
		return content
	}
	model.updateViewport(content, model.windowH)
	return model.viewport.View()
}

func (model *scanAdoptionPromptModel) updateViewport(content string, height int) {
	model.viewport.SetContent(content)

	contentHeight := lipgloss.Height(content)
	viewportHeight := view.ClampViewportHeight(height)

	model.viewport.Height = viewportHeight
	if model.windowW > 0 {
		model.viewport.Width = model.windowW
	}
	maxOffset := view.MaxViewportOffset(contentHeight, viewportHeight)
	if model.focusBottom {
		model.viewport.SetYOffset(maxOffset)
		model.focusBottom = false
		return
	}
	model.viewport.SetYOffset(model.viewport.YOffset)
}

func (model *scanAdoptionPromptModel) updateViewportForMessage(msg tea.Msg) tea.Cmd {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		updated, cmd := model.viewport.Update(typed)
		model.viewport = updated
		return cmd
	case tea.MouseMsg:
		updated, cmd := model.viewport.Update(typed)
		model.viewport = updated
		return cmd
	default:
		return nil
	}
}

func scanAdoptionPromptResult(result tea.Model) (scanAdoptionPromptModel, error) {
	switch typed := result.(type) {
	case *scanAdoptionPromptModel:
		return *typed, nil
	default:
		return scanAdoptionPromptModel{}, errors.New("unexpected prompt model")
	}
}

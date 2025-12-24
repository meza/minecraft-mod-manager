package init

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/tui"
	"github.com/spf13/afero"
)

type ModsFolderSelectedMessage struct {
	ModsFolder string
}

type ModsFolderModel struct {
	input  textinput.Model
	help   help.Model
	keymap tui.TranslatedInputKeyMap
	error  error
	Value  string

	validate func(string) error
}

func NewModsFolderModel(modsFolder string, meta config.Metadata, fs afero.Fs, prefill bool) ModsFolderModel {
	inputModel := textinput.New()
	inputModel.Prompt = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(i18n.T("cmd.init.tui.mods-folder.question")) + " "
	resolvedModsFolder := meta.ModsFolderPath(models.ModsJSON{ModsFolder: modsFolder})
	inputModel.Placeholder = modsFolder
	// Ensure the placeholder fits so the full path is visible to the user.
	minWidth := len(resolvedModsFolder) + 2
	if minWidth < 10 {
		minWidth = 10
	}
	inputModel.Width = minWidth
	inputModel.PlaceholderStyle = tui.PlaceholderStyle
	inputModel.Focus()
	if prefill {
		inputModel.SetValue(modsFolder)
	}

	model := ModsFolderModel{
		input:  inputModel,
		help:   help.New(),
		keymap: tui.TranslatedInputKeyMap{},
		validate: func(value string) error {
			return validateModsFolder(fs, meta, value)
		},
	}

	if prefill && model.validate(modsFolder) == nil {
		model.Value = modsFolder
	}

	return model
}

func (model ModsFolderModel) Init() tea.Cmd {
	return nil
}

func (model ModsFolderModel) Update(msg tea.Msg) (ModsFolderModel, tea.Cmd) {
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

func (model ModsFolderModel) View() string {
	if model.Value != "" {
		return fmt.Sprintf("%s%s", model.input.Prompt, tui.SelectedItemStyle.Render(model.Value))
	}

	errorString := ""
	if model.error != nil {
		errorString = tui.ErrorStyle.Render(" <- " + model.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", model.input.View(), errorString, model.help.View(model.keymap))
}

func (model ModsFolderModel) modsFolderSelected() tea.Cmd {
	model.input.Blur()
	return func() tea.Msg {
		return ModsFolderSelectedMessage{ModsFolder: filepath.Clean(model.Value)}
	}
}

func (model ModsFolderModel) handleKeyMsg(msg tea.KeyMsg) (ModsFolderModel, tea.Cmd, bool) {
	switch msg.String() {
	case "q":
		if !model.input.Focused() {
			return model, tea.Quit, true
		}
		return model, nil, false
	case "esc":
		return model, tea.Quit, true
	case "tab":
		if model.input.Focused() && model.input.Value() == "" {
			model.input.SetValue(model.input.Placeholder)
		}
		return model, nil, false
	case "enter":
		return model.handleEnterKey()
	default:
		if model.input.Focused() {
			model.error = nil
		}
		return model, nil, false
	}
}

func (model ModsFolderModel) handleEnterKey() (ModsFolderModel, tea.Cmd, bool) {
	value := strings.TrimSpace(model.input.Value())
	if value == "" {
		value = strings.TrimSpace(model.input.Placeholder)
	}
	if value == "" {
		model.error = fmt.Errorf("mods folder cannot be empty")
		return model, nil, true
	}

	if err := model.validate(value); err != nil {
		model.error = err
		return model, nil, true
	}

	model.Value = value
	return model, model.modsFolderSelected(), true
}

package init

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
	"github.com/spf13/afero"
)

type ConfigPathSelectedMessage struct {
	ConfigPath string
}

type ConfigPathModel struct {
	input  textinput.Model
	help   help.Model
	keymap view.TranslatedInputKeyMap
	error  error
	Value  string

	validate func(string) error
}

func NewConfigPathModel(fs afero.Fs) ConfigPathModel {
	inputModel := textinput.New()
	inputModel.Prompt = view.QuestionStyle.Render("? ") + view.TitleStyle.Render(i18n.T("cmd.init.prompt.config-path.question", nil)) + " "
	inputModel.Width = 10
	inputModel.Focus()

	return ConfigPathModel{
		input:  inputModel,
		help:   help.New(),
		keymap: view.TranslatedInputKeyMap{},
		validate: func(value string) error {
			configPath := config.NewMetadata(value).ConfigPath
			exists, err := afero.Exists(fs, configPath)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("%s", i18n.T("cmd.init.error.config-path.exists", &i18n.Tvars{
					Data: &i18n.TData{"path": configPath},
				}))
			}
			return nil
		},
	}
}

func (model ConfigPathModel) Init() tea.Cmd {
	return nil
}

func (model ConfigPathModel) Update(msg tea.Msg) (ConfigPathModel, tea.Cmd) {
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

func (model ConfigPathModel) View() string {
	if model.Value != "" {
		return fmt.Sprintf("%s%s", model.input.Prompt, view.SelectedItemStyle.Render(model.Value))
	}

	errorString := ""
	if model.error != nil {
		errorString = view.ErrorStyle.Render(" <- " + model.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", model.input.View(), errorString, model.help.View(model.keymap))
}

func (model ConfigPathModel) handleKeyMsg(msg tea.KeyMsg) (ConfigPathModel, tea.Cmd, bool) {
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

func (model ConfigPathModel) handleEnterKey() (ConfigPathModel, tea.Cmd, bool) {
	value := strings.TrimSpace(model.input.Value())
	if value == "" {
		model.error = errors.New(i18n.T("cmd.init.error.config-path.empty", nil))
		return model, nil, true
	}

	if model.validate != nil {
		if err := model.validate(value); err != nil {
			model.error = err
			return model, nil, true
		}
	}

	model.Value = value
	model.input.SetValue(value)
	return model, model.configPathSelected(), true
}

func (model ConfigPathModel) configPathSelected() tea.Cmd {
	model.input.Blur()
	return func() tea.Msg {
		return ConfigPathSelectedMessage{ConfigPath: model.Value}
	}
}

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
	m := textinput.New()
	m.Prompt = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(i18n.T("cmd.init.tui.mods-folder.question")) + " "
	resolvedModsFolder := meta.ModsFolderPath(models.ModsJSON{ModsFolder: modsFolder})
	m.Placeholder = modsFolder
	// Ensure the placeholder fits so the full path is visible to the user.
	minWidth := len(resolvedModsFolder) + 2
	if minWidth < 10 {
		minWidth = 10
	}
	m.Width = minWidth
	m.PlaceholderStyle = tui.PlaceholderStyle
	m.Focus()
	if prefill {
		m.SetValue(modsFolder)
	}

	model := ModsFolderModel{
		input:  m,
		help:   help.New(),
		keymap: tui.TranslatedInputKeyMap{},
		validate: func(value string) error {
			_, err := validateModsFolder(fs, meta, value)
			return err
		},
	}

	if prefill && model.validate(modsFolder) == nil {
		model.Value = modsFolder
	}

	return model
}

func (m ModsFolderModel) Init() tea.Cmd {
	return nil
}

func (m ModsFolderModel) Update(msg tea.Msg) (ModsFolderModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if !m.input.Focused() {
				return m, tea.Quit
			}
		case "esc":
			return m, tea.Quit
		case "tab":
			if m.input.Focused() && m.input.Value() == "" {
				m.input.SetValue(m.input.Placeholder)
			}
		case "enter":
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				value = strings.TrimSpace(m.input.Placeholder)
			}
			if value == "" {
				m.error = fmt.Errorf("mods folder cannot be empty")
				return m, nil
			}

			if err := m.validate(value); err != nil {
				m.error = err
				return m, nil
			}

			m.Value = value
			return m, m.modsFolderSelected()
		default:
			if m.input.Focused() {
				m.error = nil
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ModsFolderModel) View() string {
	if m.Value != "" {
		return fmt.Sprintf("%s%s", m.input.Prompt, tui.SelectedItemStyle.Render(m.Value))
	}

	errorString := ""
	if m.error != nil {
		errorString = tui.ErrorStyle.Render(" <- " + m.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", m.input.View(), errorString, m.help.View(m.keymap))
}

func (m ModsFolderModel) modsFolderSelected() tea.Cmd {
	m.input.Blur()
	return func() tea.Msg {
		return ModsFolderSelectedMessage{ModsFolder: filepath.Clean(m.Value)}
	}
}

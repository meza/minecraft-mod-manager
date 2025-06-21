package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
)

type ModsFolderSelectedMessage struct{ ModsFolder string }

type ModsFolderModel struct {
	input  textinput.Model
	help   help.Model
	keymap TranslatedInputKeyMap
	Value  string
}

func NewModsFolderModel(folder string) ModsFolderModel {
	m := textinput.New()
	m.Prompt = QuestionStyle.Render("? ") + TitleStyle.Render(i18n.T("cmd.init.tui.mods-folder.question")) + " "
	m.Placeholder = "mods"
	m.PlaceholderStyle = PlaceholderStyle
	m.Focus()
	model := ModsFolderModel{input: m, help: help.New(), keymap: TranslatedInputKeyMap{}}
	if folder != "" {
		model.Value = folder
	}
	return model
}

func (m ModsFolderModel) Init() tea.Cmd { return nil }

func (m ModsFolderModel) Update(msg tea.Msg) (ModsFolderModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.input.Value() != "" {
				m.Value = m.input.Value()
				return m, m.modsFolderSelected()
			}
		case "tab":
			if m.input.Value() == "" {
				m.input.SetValue(m.input.Placeholder)
			}
		}
	}
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ModsFolderModel) View() string {
	if m.Value != "" {
		return fmt.Sprintf("%s%s", m.input.Prompt, SelectedItemStyle.Render(m.Value))
	}
	return fmt.Sprintf("%s\n\n%s", m.input.View(), m.help.View(m.keymap))
}

func (m ModsFolderModel) modsFolderSelected() tea.Cmd {
	m.input.Blur()
	return func() tea.Msg { return ModsFolderSelectedMessage{ModsFolder: m.Value} }
}

func (m ModsFolderModel) Help() string { return m.help.View(m.keymap) }

func (m ModsFolderModel) SetSize(width, _ int) ModsFolderModel {
	m.input.Width = width
	return m
}

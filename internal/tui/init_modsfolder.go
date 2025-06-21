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
	input   textinput.Model
	help    help.Model
	keymap  TranslatedInputKeyMap
	focused bool
	Value   string
	err     error
}

func (m ModsFolderModel) Init() tea.Cmd { return nil }

func (m ModsFolderModel) Focus() { m.focused = true }
func (m ModsFolderModel) Blur()  { m.focused = false }

func (m ModsFolderModel) Update(msg tea.Msg) (ModsFolderModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.input.Value() != "" {
				m.Value = m.input.Value()
				m.input.Blur()
				return m, m.modsFolderSelected()
			}
		}
	}
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m ModsFolderModel) modsFolderSelected() tea.Cmd {
	return func() tea.Msg { return ModsFolderSelectedMessage{ModsFolder: m.Value} }
}

func (m ModsFolderModel) View() string {
	if m.Value != "" {
		return fmt.Sprintf("%s%s", m.input.Prompt, SelectedItemStyle.Render(m.Value))
	}
	if m.err != nil {
		return fmt.Sprintf("%s\n%s", m.input.View(), ErrorStyle.Render(m.err.Error()))
	}
	if m.focused {
		return fmt.Sprintf("%s\n\n%s", m.input.View(), m.help.View(m.keymap))
	}
	return m.input.View()
}

func (m ModsFolderModel) HelpView() string {
	if m.focused {
		return m.help.View(m.keymap)
	}
	return ""
}

func NewModsFolderModel(modsFolder string) ModsFolderModel {
	ti := textinput.New()
	ti.Prompt = QuestionStyle.Render("? ") + TitleStyle.Render(i18n.T("cmd.init.tui.mods-folder.question")) + " "
	ti.SetValue(modsFolder)
	ti.Focus()
	return ModsFolderModel{input: ti, help: help.New(), keymap: TranslatedInputKeyMap{}, focused: true}
}

package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"net/http"
)

type GameVersionSelectedMessage struct {
	GameVersion string
}

type GameVersionModel struct {
	tea.Model
	input  textinput.Model
	help   help.Model
	keymap TranslatedInputKeyMap
	error  error
	Value  string
}

var validVersionFn = minecraft.IsValidVersion

func (m GameVersionModel) Init() tea.Cmd {
	return nil
}

func (m GameVersionModel) Update(msg tea.Msg) (GameVersionModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if !m.input.Focused() {
				return m, tea.Quit
			}

		case "esc":
			return m, tea.Quit
		case "enter":
			if m.input.Value() != "" {
				err := isValidMinecraftVersion(m.input.Value())
				if err != nil {
					m.error = err
				} else {
					m.Value = m.input.Value()
					return m, m.gameVersionSelected()
				}
			}
		case "tab":
			if m.input.Focused() {
				if m.input.Value() == "" {
					m.input.SetValue(m.input.Placeholder)
				}
			}
		default:
			if m.input.Focused() {
				m.error = nil
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m GameVersionModel) View() string {
	if m.Value != "" {
		return fmt.Sprintf("%s%s", m.input.Prompt, SelectedItemStyle.Render(m.Value))
	}

	errorString := ""

	if m.error != nil {
		errorString = ErrorStyle.Render(" <- " + m.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", m.input.View(), errorString, m.help.View(m.keymap))
}

func (m GameVersionModel) gameVersionSelected() tea.Cmd {
	m.input.Blur()
	return func() tea.Msg {
		return GameVersionSelectedMessage{GameVersion: m.Value}
	}
}

func NewGameVersionModel(gameVersion string) GameVersionModel {

	latestVersion, _ := minecraft.GetLatestVersion(http.DefaultClient)
	allVersions := minecraft.GetAllMineCraftVersions(http.DefaultClient)

	m := textinput.New()
	m.Prompt = QuestionStyle.Render("? ") + TitleStyle.Render(i18n.T("cmd.init.tui.game-version.question")) + " "
	m.Placeholder = latestVersion
	m.PlaceholderStyle = PlaceholderStyle
	m.ShowSuggestions = true
	m.SetSuggestions(allVersions)
	m.Focus()

	model := GameVersionModel{
		input:  m,
		help:   help.New(),
		keymap: TranslatedInputKeyMap{},
	}

	if minecraft.IsValidVersion(gameVersion, http.DefaultClient) {
		model.Value = gameVersion
	}

	return model
}

func isValidMinecraftVersion(value string) error {
	if value == "" {
		return fmt.Errorf(i18n.T("cmd.init.tui.game-version.error"))
	}

	if !validVersionFn(value, http.DefaultClient) {
		return fmt.Errorf(i18n.T("cmd.init.tui.game-version.invalid"))
	}
	return nil
}

func (m GameVersionModel) Help() string { return m.help.View(m.keymap) }

func (m GameVersionModel) SetSize(width, _ int) GameVersionModel {
	m.input.Width = width
	return m
}

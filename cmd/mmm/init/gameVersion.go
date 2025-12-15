package init

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

type GameVersionSelectedMessage struct {
	GameVersion string
}

type GameVersionModel struct {
	tea.Model
	input  textinput.Model
	help   help.Model
	keymap tui.TranslatedInputKeyMap
	error  error
	Value  string

	validate func(string) error
}

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
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				value = strings.TrimSpace(m.input.Placeholder)
			}

			if value == "" {
				m.error = fmt.Errorf("%s", i18n.T("cmd.init.tui.game-version.error"))
				return m, nil
			}

			err := m.validate(value)
			if err != nil {
				m.error = err
			} else {
				m.Value = value
				m.input.SetValue(value)
				return m, m.gameVersionSelected()
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
		return fmt.Sprintf("%s%s", m.input.Prompt, tui.SelectedItemStyle.Render(m.Value))
	}

	errorString := ""

	if m.error != nil {
		errorString = tui.ErrorStyle.Render(" <- " + m.error.Error())
	}

	return fmt.Sprintf("%s%s\n\n%s", m.input.View(), errorString, m.help.View(m.keymap))
}

func (m GameVersionModel) gameVersionSelected() tea.Cmd {
	m.input.Blur()
	return func() tea.Msg {
		return GameVersionSelectedMessage{GameVersion: m.Value}
	}
}

func NewGameVersionModel(minecraftClient httpClient.Doer, gameVersion string) GameVersionModel {
	latestVersion, _ := minecraft.GetLatestVersion(minecraftClient)
	allVersions := minecraft.GetAllMineCraftVersions(minecraftClient)

	m := textinput.New()
	m.Prompt = tui.QuestionStyle.Render("? ") + tui.TitleStyle.Render(i18n.T("cmd.init.tui.game-version.question")) + " "
	m.Placeholder = latestVersion
	m.PlaceholderStyle = tui.PlaceholderStyle
	width := len(m.Placeholder)
	if len(gameVersion) > width {
		width = len(gameVersion)
	}
	const minWidth = 10
	if width < minWidth {
		width = minWidth
	}
	m.Width = width
	if len(allVersions) > 0 {
		m.ShowSuggestions = true
	}
	m.SetSuggestions(allVersions)
	m.Focus()

	model := GameVersionModel{
		input:  m,
		help:   help.New(),
		keymap: tui.TranslatedInputKeyMap{},
		validate: func(value string) error {
			return validateMinecraftVersion(value, minecraftClient)
		},
	}

	if gameVersion != "" && !strings.EqualFold(gameVersion, "latest") && model.validate(gameVersion) == nil {
		model.Value = gameVersion
		model.input.SetValue(gameVersion)
	}

	return model
}
func validateMinecraftVersion(value string, client httpClient.Doer) error {
	if value == "" {
		return fmt.Errorf("%s", i18n.T("cmd.init.tui.game-version.error"))
	}

	if !minecraft.IsValidVersion(value, client) {
		return fmt.Errorf("%s", i18n.T("cmd.init.tui.game-version.invalid"))
	}
	return nil
}

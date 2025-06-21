package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type state int

const (
	stateModsFolder state = iota
	stateLoader
	stateGameVersion
	done
)

type InitModel struct {
	state               state
	width, height       int
	modsFolderQuestion  ModsFolderModel
	loaderQuestion      LoaderModel
	gameVersionQuestion GameVersionModel
}

func (m InitModel) Init() tea.Cmd {
	return nil
}
func (m InitModel) View() string {
	var sb strings.Builder
	sb.WriteString(m.modsFolderQuestion.View())
	if m.state >= stateLoader {
		sb.WriteString("\n")
		sb.WriteString(m.loaderQuestion.View())
	}
	if m.state >= stateGameVersion {
		sb.WriteString("\n")
		sb.WriteString(m.gameVersionQuestion.View())
	}
	return sb.String()
}

func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ModsFolderSelectedMessage:
		m.state = stateLoader
	case LoaderSelectedMessage:
		m.state = stateGameVersion
	case GameVersionSelectedMessage:
		m.state = done
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			cmds = append(cmds, tea.Quit)
		}
	}

	switch m.state {
	case stateModsFolder:
		m.modsFolderQuestion, cmd = m.modsFolderQuestion.Update(msg)
	case stateLoader:
		m.loaderQuestion, cmd = m.loaderQuestion.Update(msg)
	case stateGameVersion:
		m.gameVersionQuestion, cmd = m.gameVersionQuestion.Update(msg)
	default:
		return m, tea.Quit
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func NewInitModel(loader string, gameVersion string, releaseTypes string, modsFolder string) *InitModel {
	model := &InitModel{
		modsFolderQuestion:  NewModsFolderModel(modsFolder),
		loaderQuestion:      NewLoaderModel(loader),
		gameVersionQuestion: NewGameVersionModel(gameVersion),
		//selectedReleaseTypes: parseReleaseTypes(releaseTypes),

	}

	return model

}

func (m InitModel) Help() string {
	switch m.state {
	case stateModsFolder:
		return m.modsFolderQuestion.Help()
	case stateLoader:
		return m.loaderQuestion.Help()
	case stateGameVersion:
		return m.gameVersionQuestion.Help()
	default:
		return ""
	}
}

func (m *InitModel) SetSize(width, height int) {
	m.width, m.height = width, height
	ws := tea.WindowSizeMsg{Width: width, Height: height}
	m.modsFolderQuestion, _ = m.modsFolderQuestion.Update(ws)
	m.loaderQuestion, _ = m.loaderQuestion.Update(ws)
	m.gameVersionQuestion, _ = m.gameVersionQuestion.Update(ws)
}

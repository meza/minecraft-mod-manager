package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type state int

const (
	stateLoader state = iota
	stateGameVersion
	done
)

type InitModel struct {
	state               state
	loaderQuestion      LoaderModel
	gameVersionQuestion GameVersionModel
}

func (m InitModel) Init() tea.Cmd {
	return nil
}

func (m InitModel) View() string {
	stringBuilder := strings.Builder{}
	stringBuilder.WriteString(m.loaderQuestion.View())

	switch m.state {
	case stateLoader:
		return stringBuilder.String()
	case stateGameVersion:
		stringBuilder.WriteString("\n")
		stringBuilder.WriteString(m.gameVersionQuestion.View())
	case done:
		stringBuilder.WriteString("\n")
		stringBuilder.WriteString(m.gameVersionQuestion.View())
	}

	stringBuilder.WriteString("\n")
	stringBuilder.WriteString("\n")
	return stringBuilder.String()

}

func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	var cmd tea.Cmd

	switch msg := msg.(type) {
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
		loaderQuestion:      NewLoaderModel(loader),
		gameVersionQuestion: NewGameVersionModel(gameVersion),
		//selectedReleaseTypes: parseReleaseTypes(releaseTypes),

	}

	return model

}

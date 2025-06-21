package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

type initStep int

const (
	stepFields initStep = iota
	stepDone
)

type InitModel struct {
	step                initStep
	focus               int
	modsFolder          ModsFolderModel
	loaderQuestion      LoaderModel
	gameVersionQuestion GameVersionModel
	width               int
}

func NewInitModel(loader string, gameVersion string, releaseTypes string, modsFolder string) *InitModel {
	m := &InitModel{
		modsFolder:          NewModsFolderModel(modsFolder),
		loaderQuestion:      NewLoaderModel(loader),
		gameVersionQuestion: NewGameVersionModel(gameVersion),
	}
	m.focus = 0
	m.focusCurrent()
	return m
}

func (m *InitModel) focusCurrent() {
	switch m.focus {
	case 0:
		m.modsFolder.Focus()
	case 1:
		m.loaderQuestion.Focus()
	case 2:
		m.gameVersionQuestion.Focus()
	}
}

func (m *InitModel) blurCurrent() {
	switch m.focus {
	case 0:
		m.modsFolder.Blur()
	case 1:
		m.loaderQuestion.Blur()
	case 2:
		m.gameVersionQuestion.Blur()
	}
}

func (m InitModel) Init() tea.Cmd { return nil }

func (m *InitModel) nextField() {
	m.blurCurrent()
	m.focus = (m.focus + 1) % 3
	m.focusCurrent()
}

func (m *InitModel) prevField() {
	m.blurCurrent()
	if m.focus == 0 {
		m.focus = 2
	} else {
		m.focus--
	}
	m.focusCurrent()
}

func (m *InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			m.nextField()
			return m, nil
		case tea.KeyShiftTab:
			m.prevField()
			return m, nil
		case tea.KeyRight:
			if m.step == stepFields {
				m.step = stepDone
			}
		case tea.KeyLeft:
			if m.step == stepDone {
				m.step = stepFields
			}
		case tea.KeyEnter:
			if m.step == stepDone {
				return m, tea.Quit
			}
		}
	}
	if m.step == stepFields {
		switch m.focus {
		case 0:
			m.modsFolder, cmd = m.modsFolder.Update(msg)
		case 1:
			m.loaderQuestion, cmd = m.loaderQuestion.Update(msg)
		case 2:
			m.gameVersionQuestion, cmd = m.gameVersionQuestion.Update(msg)
		}
	}
	return m, cmd
}

func (m InitModel) currentHelp() string {
	switch m.focus {
	case 0:
		return m.modsFolder.HelpView()
	case 1:
		return m.loaderQuestion.HelpView()
	case 2:
		return m.gameVersionQuestion.HelpView()
	}
	return ""
}

func (m InitModel) View() string {
	var b strings.Builder
	b.WriteString(m.modsFolder.View())
	b.WriteString("\n")
	b.WriteString(m.loaderQuestion.View())
	b.WriteString("\n")
	b.WriteString(m.gameVersionQuestion.View())
	b.WriteString("\n\n")
	b.WriteString(m.currentHelp())
	if m.step == stepFields {
		b.WriteString("\n\n[Next →]")
	} else {
		b.WriteString("\n\n[← Back] [Finish]")
	}
	return b.String()
}

func (m *InitModel) SetSize(width int) {
	m.width = width
	m.loaderQuestion.list.SetWidth(width)
}

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
)

type dummyModel struct{}

func (d dummyModel) Init() tea.Cmd                           { return nil }
func (d dummyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return d, nil }
func (d dummyModel) View() string                            { return "content" }

func TestNewAppModel(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	m := NewAppModel([]Component{{Name: "Init", Model: dummyModel{}}})
	snaps.MatchSnapshot(t, m.View())
}

package test

import (
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

type logCollector struct {
	model  tui.LogModel
	writer *tui.LogLineWriter
}

func newLogCollector() *logCollector {
	collector := &logCollector{model: tui.NewLogModel()}
	collector.writer = tui.NewLogLineWriter(collector)
	return collector
}

func (collector *logCollector) Send(msg tea.Msg) {
	updated, _ := collector.model.Update(msg)
	collector.model = updated.(tui.LogModel)
}

func (collector *logCollector) View() string {
	collector.writer.Flush()
	return collector.model.View()
}

func TestTestTUILogSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := tui.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	collector := newLogCollector()
	deps := testDeps{
		output: output.New(collector.writer, io.Discard, false),
	}

	unsupported := []modCheckOutcome{
		{Mod: models.Mod{ID: "mod-a", Name: "Alpha Mod", Type: models.MODRINTH}},
		{Mod: models.Mod{ID: "mod-b", Name: "Beta Mod", Type: models.CURSEFORGE}},
	}

	exitCode, err := reportUnsupportedMods("1.20.1", unsupported, deps, tui.ColorEnabled)
	assert.ErrorIs(t, err, errUnsupportedMods)
	assert.Equal(t, 1, exitCode)

	snaps.MatchSnapshot(t, collector.View())
}

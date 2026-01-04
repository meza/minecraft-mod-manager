package install

import (
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
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

func TestInstallTUILogSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreUnicode := tui.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	collector := newLogCollector()
	out := output.New(collector.writer, io.Discard, false)

	err := out.Log(i18n.T("cmd.install.download.missing", &i18n.Tvars{
		Data: &i18n.TData{
			"name":     "Sodium",
			"platform": string(models.MODRINTH),
		},
	}), output.LogForce)
	assert.NoError(t, err)

	err = out.Log(messageWithIcon(tui.SuccessIcon(tui.ColorEnabled), i18n.T("cmd.install.success", nil)), output.LogForce)
	assert.NoError(t, err)

	snaps.MatchSnapshot(t, collector.View())
}

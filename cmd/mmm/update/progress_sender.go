package update

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
)

type updateProgressSender struct {
	index  int
	sender updateExecSender
}

func (sender updateProgressSender) Send(msg tea.Msg) {
	switch typed := msg.(type) {
	case httpclient.DownloadProgressMsg:
		sender.sender.Send(updateItemProgressMsg{
			index: sender.index,
			progress: updateProgress{
				ratio:      typed.Ratio,
				downloaded: typed.Downloaded,
				total:      typed.Total,
			},
		})
	default:
		return
	}
}

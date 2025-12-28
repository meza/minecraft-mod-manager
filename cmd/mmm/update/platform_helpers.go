package update

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type noopSender struct{}

func (sender *noopSender) Send(msg tea.Msg) { _ = msg }

func downloadClient(clients platform.Clients) httpclient.Doer {
	if clients.Curseforge != nil {
		return clients.Curseforge
	}
	return clients.Modrinth
}

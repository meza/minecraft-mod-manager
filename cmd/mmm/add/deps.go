package add

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func newAddDeps(common cmddeps.CommonDeps) addDeps {
	return addDeps{
		fs:              common.FS,
		clients:         common.Clients,
		minecraftClient: common.MinecraftClient,
		logger:          common.Logger,
		output:          common.Output,
		fetchMod:        platform.FetchMod,
		downloader:      httpclient.DownloadFile,
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return tea.NewProgram(model, options...).Run()
		},
	}
}

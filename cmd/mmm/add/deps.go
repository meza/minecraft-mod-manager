package add

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/afero"
	"golang.org/x/time/rate"
)

func defaultAddDeps(log *logger.Logger, limiter *rate.Limiter) addDeps {
	return addDeps{
		fs:              afero.NewOsFs(),
		clients:         platform.DefaultClients(limiter),
		minecraftClient: httpclient.NewRLClient(limiter),
		logger:          log,
		fetchMod:        platform.FetchMod,
		downloader:      httpclient.DownloadFile,
		runTea: func(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
			return tea.NewProgram(model, options...).Run()
		},
	}
}

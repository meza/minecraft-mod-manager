package change

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func newChangeDeps(common cmddeps.CommonDeps) changeDeps {
	return changeDeps{
		fs:              common.FS,
		output:          common.Output,
		logger:          common.Logger,
		clients:         common.Clients,
		downloadClient:  httpclient.NewRLClient(common.Limiter),
		minecraftClient: common.MinecraftClient,
		limiter:         common.Limiter,
		downloader:      httpclient.DownloadFile,
		fetchMod:        platform.FetchMod,
		latestVersion:   minecraft.GetLatestVersion,
		isValidVersion:  minecraft.IsValidVersion,
		runTea:          defaultRunTea,

		readConfig:  config.ReadConfig,
		ensureLock:  config.EnsureLock,
		writeConfig: config.WriteConfig,
		writeLock:   config.WriteLock,
		removeFile: func(fs afero.Fs, path string) error {
			return fs.Remove(path)
		},
		renameFile: func(fs afero.Fs, source string, destination string) error {
			return fs.Rename(source, destination)
		},
		removeAll: func(fs afero.Fs, path string) error {
			return fs.RemoveAll(path)
		},
		mkdirAll: func(fs afero.Fs, path string, perm os.FileMode) error {
			return fs.MkdirAll(path, perm)
		},

		telemetry: telemetry.RecordCommand,
	}
}

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	return tea.NewProgram(model, options...).Run()
}

var runTeaProgram = defaultRunTea

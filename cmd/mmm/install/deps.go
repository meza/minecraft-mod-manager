package install

import (
	curseforgeFingerprint "github.com/meza/curseforge-fingerprint-go"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/spf13/afero"
	"golang.org/x/time/rate"
)

func defaultInstallDeps(log *logger.Logger, out *output.Output, limiter *rate.Limiter, telemetryRecorder func(telemetry.CommandTelemetry)) installDeps {
	return installDeps{
		fs:         afero.NewOsFs(),
		logger:     log,
		output:     out,
		clients:    platform.DefaultClients(limiter),
		downloader: httpclient.DownloadFile,
		fetchMod:   platform.FetchMod,
		telemetry:  telemetryRecorder,

		curseforgeFingerprint:      curseforgeFingerprint.GetFingerprintFor,
		modrinthVersionForSha:      defaultModrinthVersionForSha,
		modrinthProjectTitle:       defaultModrinthProjectTitle,
		curseforgeFingerprintMatch: defaultCurseforgeFingerprintMatch,
		curseforgeProjectName:      defaultCurseforgeProjectName,
	}
}

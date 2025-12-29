package install

import (
	curseforgeFingerprint "github.com/meza/curseforge-fingerprint-go"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
)

func newInstallDeps(common cmddeps.CommonDeps, telemetryRecorder func(telemetry.CommandTelemetry)) installDeps {
	return installDeps{
		fs:         common.FS,
		logger:     common.Logger,
		output:     common.Output,
		clients:    common.Clients,
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

package platform

import (
	"errors"
	"fmt"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"golang.org/x/time/rate"
)

type RemoteMod struct {
	Name        string
	FileName    string
	ReleaseDate string
	Hash        string
	DownloadURL string
}

type FetchOptions struct {
	AllowedReleaseTypes []models.ReleaseType
	GameVersion         string
	Loader              models.Loader
	AllowFallback       bool
	FixedVersion        string
}

type Clients struct {
	Modrinth   httpClient.Doer
	Curseforge httpClient.Doer
}

func DefaultClients(limiter *rate.Limiter) Clients {
	if limiter == nil {
		limiter = rate.NewLimiter(rate.Inf, 0)
	}
	client := httpClient.NewRLClient(limiter)
	return Clients{
		Modrinth:   client,
		Curseforge: client,
	}
}

func FetchMod(platform models.Platform, projectID string, opts FetchOptions, clients Clients) (RemoteMod, error) {
	details := perf.PerformanceDetails{
		"platform":       string(platform),
		"project_id":     projectID,
		"loader":         string(opts.Loader),
		"game_version":   opts.GameVersion,
		"allow_fallback": opts.AllowFallback,
		"fixed_version":  opts.FixedVersion,
	}
	region := perf.StartRegionWithDetails("platform.fetch_mod", &details)

	var remote RemoteMod
	var err error
	switch platform {
	case models.MODRINTH:
		remote, err = fetchModrinth(projectID, opts, clients.Modrinth)
	case models.CURSEFORGE:
		remote, err = fetchCurseforge(projectID, opts, clients.Curseforge)
	default:
		err = &UnknownPlatformError{Platform: string(platform)}
	}

	details["success"] = err == nil
	if err != nil {
		details["error_type"] = fmt.Sprintf("%T", err)
	}
	region.End()

	if err != nil {
		return RemoteMod{}, err
	}
	return remote, nil
}

func mapProjectNotFound(platform models.Platform, projectID string, err error) error {
	var notFound *globalErrors.ProjectNotFoundError
	if errors.As(err, &notFound) {
		return &ModNotFoundError{
			Platform:  platform,
			ProjectID: projectID,
		}
	}
	return err
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

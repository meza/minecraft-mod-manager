package platform

import (
	"errors"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
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
	switch platform {
	case models.MODRINTH:
		return fetchModrinth(projectID, opts, clients.Modrinth)
	case models.CURSEFORGE:
		return fetchCurseforge(projectID, opts, clients.Curseforge)
	default:
		return RemoteMod{}, &UnknownPlatformError{Platform: string(platform)}
	}
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

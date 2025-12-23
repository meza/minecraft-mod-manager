package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"go.opentelemetry.io/otel/attribute"
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
	Modrinth   httpclient.Doer
	Curseforge httpclient.Doer
}

func DefaultClients(limiter *rate.Limiter) Clients {
	if limiter == nil {
		limiter = rate.NewLimiter(rate.Inf, 0)
	}
	client := httpclient.NewRLClient(limiter)
	return Clients{
		Modrinth:   client,
		Curseforge: client,
	}
}

func FetchMod(ctx context.Context, platform models.Platform, projectID string, opts FetchOptions, clients Clients) (RemoteMod, error) {
	ctx, span := perf.StartSpan(ctx, "platform.fetch_mod",
		perf.WithAttributes(
			attribute.String("platform", string(platform)),
			attribute.String("project_id", projectID),
			attribute.String("loader", string(opts.Loader)),
			attribute.String("game_version", opts.GameVersion),
			attribute.Bool("allow_fallback", opts.AllowFallback),
			attribute.String("fixed_version", opts.FixedVersion),
		),
	)

	var remote RemoteMod
	var err error
	switch platform {
	case models.MODRINTH:
		remote, err = fetchModrinth(ctx, projectID, opts, clients.Modrinth)
	case models.CURSEFORGE:
		remote, err = fetchCurseforge(ctx, projectID, opts, clients.Curseforge)
	default:
		err = &UnknownPlatformError{Platform: string(platform)}
	}

	span.SetAttributes(attribute.Bool("success", err == nil))
	if err != nil {
		span.SetAttributes(attribute.String("error_type", fmt.Sprintf("%T", err)))
	}
	span.End()

	if err != nil {
		return RemoteMod{}, err
	}
	return remote, nil
}

func mapProjectNotFound(platform models.Platform, projectID string, err error) error {
	var notFound *globalerrors.ProjectNotFoundError
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

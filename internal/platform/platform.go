package platform

import (
	"context"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"
)

type Clients struct {
	Modrinth   httpclient.Doer
	Curseforge httpclient.Doer
}

// PreferredDownloadClient selects the best available download client.
// It favors the CurseForge client when available, otherwise it returns Modrinth.
func PreferredDownloadClient(clients Clients) httpclient.Doer {
	if clients.Curseforge != nil {
		return clients.Curseforge
	}
	return clients.Modrinth
}

func DefaultClients(limiter *rate.Limiter) Clients {
	if limiter == nil {
		limiter = httpclient.DefaultLimiter()
	}
	client := httpclient.NewRLClient(limiter)
	return Clients{
		Modrinth:   client,
		Curseforge: client,
	}
}

// FetchMod is the cross-platform entrypoint for resolving a project to RemoteMod.
// It dispatches to the provider-specific FetchRemoteMod implementation and returns
// the normalized RemoteMod output.
//
// It returns UnknownPlatformError for unsupported platforms, ModNotFoundError when
// the project is missing, NoCompatibleFileError when no eligible file exists, or
// underlying provider API/validation errors.
//
// Example:
//
//	remote, err := platform.FetchMod(ctx, models.MODRINTH, "AANobbMI", opts, clients)
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
		remote, err = modrinth.FetchRemoteMod(ctx, projectID, opts, clients.Modrinth)
	case models.CURSEFORGE:
		remote, err = curseforge.FetchRemoteMod(ctx, projectID, opts, clients.Curseforge)
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

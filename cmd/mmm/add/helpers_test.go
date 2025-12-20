package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestModNameForConfig_ReturnsProjectIDWhenMissing(t *testing.T) {
	cfg := models.ModsJson{
		Mods: []models.Mod{
			{Type: models.MODRINTH, ID: "abc", Name: "Example"},
		},
	}

	assert.Equal(t, "xyz", modNameForConfig(cfg, models.MODRINTH, "xyz"))
	assert.Equal(t, "Example", modNameForConfig(cfg, models.MODRINTH, "abc"))
}

func TestDownloadClientPrefersCurseforge(t *testing.T) {
	modrinthClient := platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)).Modrinth
	curseforgeClient := platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)).Curseforge

	clients := platform.Clients{Modrinth: modrinthClient}
	assert.Equal(t, modrinthClient, downloadClient(clients))

	clients.Curseforge = curseforgeClient
	assert.Equal(t, curseforgeClient, downloadClient(clients))
}

func TestResolveRemoteMod_WrappedErrorReturnsError(t *testing.T) {
	ctx := context.Background()
	cfg := models.ModsJson{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
	}
	deps := addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		logger:  logger.New(io.Discard, io.Discard, true, false),
		fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
			return platform.RemoteMod{}, fmt.Errorf("outer: %w", errors.New("inner"))
		},
	}

	_, _, _, err := resolveRemoteMod(ctx, nil, cfg, addOptions{Quiet: true}, models.MODRINTH, "abc", deps, false, strings.NewReader(""), io.Discard)
	assert.Error(t, err)
}

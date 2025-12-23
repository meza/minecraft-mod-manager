// Package minecraft provides Minecraft version lookups.
package minecraft

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/perf"
)

type latest struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

type version struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
}

type versionManifest struct {
	Latest   latest    `json:"latest"`
	Versions []version `json:"versions"`
}

var versionManifestURL = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
var latestManifest *versionManifest
var newRequestWithContext = http.NewRequestWithContext

func ClearManifestCache() {
	latestManifest = nil
}

func getMinecraftVersionManifest(ctx context.Context, client httpclient.Doer) (*versionManifest, error) {
	_, span := perf.StartSpan(ctx, "api.minecraft.version_manifest.get")
	defer span.End()
	if latestManifest != nil {
		return latestManifest, nil
	}

	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, "GET", versionManifestURL, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, ErrManifestNotFound
	}

	var decodedManifest versionManifest
	decodeErr := json.NewDecoder(response.Body).Decode(&decodedManifest)
	closeErr := response.Body.Close()
	if decodeErr != nil {
		return nil, ErrManifestNotFound
	}
	if closeErr != nil {
		return nil, closeErr
	}
	latestManifest = &decodedManifest
	return latestManifest, nil
}

func GetLatestVersion(ctx context.Context, client httpclient.Doer) (string, error) {
	manifest, err := getMinecraftVersionManifest(ctx, client)

	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return "", httpclient.WrapTimeoutError(err)
		}
		return "", ErrCouldNotDetermineLatestVersion
	}

	return manifest.Latest.Release, nil
}

func IsValidVersion(ctx context.Context, version string, client httpclient.Doer) bool {
	if version == "" {
		return false
	}

	manifest, err := getMinecraftVersionManifest(ctx, client)

	if err != nil {
		// If we couldn't get the manifest, we can't determine if the version is valid
		// so we return true to allow the user to try to download the version anyway
		return true
	}

	for _, v := range manifest.Versions {
		if v.ID == version {
			return true
		}
	}

	return false
}

func GetAllMineCraftVersions(ctx context.Context, client httpclient.Doer) []string {
	manifest, err := getMinecraftVersionManifest(ctx, client)

	if err != nil {
		return []string{}
	}

	versions := make([]string, 0)
	for _, v := range manifest.Versions {
		versions = append(versions, v.ID)
	}

	return versions
}

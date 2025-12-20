package minecraft

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/perf"
)

type latest struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

type version struct {
	Id          string    `json:"id"`
	Type        string    `json:"type"`
	Url         string    `json:"url"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
}

type versionManifest struct {
	Latest   latest    `json:"latest"`
	Versions []version `json:"versions"`
}

var versionManifestUrl = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
var latestManifest *versionManifest

func ClearManifestCache() {
	latestManifest = nil
}

func getMinecraftVersionManifest(ctx context.Context, client httpClient.Doer) (*versionManifest, error) {
	_, span := perf.StartSpan(ctx, "api.minecraft.version_manifest.get")
	defer span.End()
	if latestManifest != nil {
		return latestManifest, nil
	}

	timeoutCtx, cancel := httpClient.WithMetadataTimeout(ctx)
	defer cancel()
	request, _ := http.NewRequestWithContext(timeoutCtx, "GET", versionManifestUrl, nil)

	response, err := client.Do(request)
	if err != nil {
		if httpClient.IsTimeoutError(err) {
			return nil, httpClient.WrapTimeoutError(err)
		}
		return nil, ManifestNotFound
	}

	defer response.Body.Close()

	var manifest versionManifest
	err = json.NewDecoder(response.Body).Decode(&manifest)
	if err != nil {
		return nil, ManifestNotFound
	}
	latestManifest = &manifest
	return latestManifest, nil
}

func GetLatestVersion(ctx context.Context, client httpClient.Doer) (string, error) {
	manifest, err := getMinecraftVersionManifest(ctx, client)

	if err != nil {
		if httpClient.IsTimeoutError(err) {
			return "", httpClient.WrapTimeoutError(err)
		}
		return "", CouldNotDetermineLatestVersion
	}

	return manifest.Latest.Release, nil
}

func IsValidVersion(ctx context.Context, version string, client httpClient.Doer) bool {
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
		if v.Id == version {
			return true
		}
	}

	return false
}

func GetAllMineCraftVersions(ctx context.Context, client httpClient.Doer) []string {
	manifest, err := getMinecraftVersionManifest(ctx, client)

	if err != nil {
		return []string{}
	}

	versions := make([]string, 0)
	for _, v := range manifest.Versions {
		versions = append(versions, v.Id)
	}

	return versions
}

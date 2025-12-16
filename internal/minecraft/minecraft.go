package minecraft

import (
	"encoding/json"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"net/http"
	"time"
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

func getMinecraftVersionManifest(client httpClient.Doer) (*versionManifest, error) {
	defer perf.StartRegion("api.minecraft.version_manifest.get").End()
	if latestManifest != nil {
		return latestManifest, nil
	}

	request, _ := http.NewRequest("GET", versionManifestUrl, nil)

	response, err := client.Do(request)
	if err != nil {
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

func GetLatestVersion(client httpClient.Doer) (string, error) {
	manifest, err := getMinecraftVersionManifest(client)

	if err != nil {
		return "", CouldNotDetermineLatestVersion
	}

	return manifest.Latest.Release, nil
}

func IsValidVersion(version string, client httpClient.Doer) bool {
	if version == "" {
		return false
	}

	manifest, err := getMinecraftVersionManifest(client)

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

func GetAllMineCraftVersions(client httpClient.Doer) []string {
	manifest, err := getMinecraftVersionManifest(client)

	if err != nil {
		return []string{}
	}

	versions := make([]string, 0)
	for _, v := range manifest.Versions {
		versions = append(versions, v.Id)
	}

	return versions
}

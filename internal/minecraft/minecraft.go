package minecraft

import (
	"encoding/json"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
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

func getMinecraftVersionManifest(client httpClient.Doer) (*versionManifest, error) {
	const versionManifestUrl = "https://launchermeta.mojang.com/mc/game/version_manifest.json"

	request, _ := http.NewRequest("GET", versionManifestUrl, nil)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	var manifest versionManifest
	err = json.NewDecoder(response.Body).Decode(&manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

func GetLatestVersion(client httpClient.Doer) string {
	manifest, err := getMinecraftVersionManifest(client)

	if err != nil {
		return ""
	}

	return manifest.Latest.Release
}

func IsValidVersion(version string, client httpClient.Doer) bool {
	manifest, err := getMinecraftVersionManifest(client)

	if err != nil {
		panic(err)
	}

	for _, v := range manifest.Versions {
		if v.Id == version {
			return true
		}
	}

	return false
}

func GetAllMineCraftVersions(client httpClient.Doer) []string {
	manifest, _ := getMinecraftVersionManifest(client)

	versions := make([]string, 0)
	for _, v := range manifest.Versions {
		versions = append(versions, v.Id)
	}

	return versions
}

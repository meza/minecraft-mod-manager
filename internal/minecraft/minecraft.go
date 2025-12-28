// Package minecraft provides Minecraft version lookups.
package minecraft

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
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
var newRequestWithContext = http.NewRequestWithContext

type manifestCache struct {
	mutex    sync.RWMutex
	manifest *versionManifest
}

func (cache *manifestCache) clear() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.manifest = nil
}

func (cache *manifestCache) get() (*versionManifest, bool) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	if cache.manifest == nil {
		return nil, false
	}
	return cache.manifest, true
}

func (cache *manifestCache) set(manifest *versionManifest) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.manifest = manifest
}

var manifestCacheState = &manifestCache{}

func ClearManifestCache() {
	manifestCacheState.clear()
}

func getMinecraftVersionManifest(ctx context.Context, client httpclient.Doer) (*versionManifest, error) {
	_, span := perf.StartSpan(ctx, "api.minecraft.version_manifest.get")
	defer span.End()
	if manifest, ok := manifestCacheState.get(); ok {
		return manifest, nil
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
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		responseErr := httpclient.NewResponseError(response)
		closeErr := response.Body.Close()
		if closeErr != nil {
			return nil, errors.Join(responseErr, closeErr)
		}
		return nil, responseErr
	}

	var decodedManifest versionManifest
	decodeErr := json.NewDecoder(response.Body).Decode(&decodedManifest)
	closeErr := response.Body.Close()
	if decodeErr != nil {
		if closeErr != nil {
			return nil, errors.Join(decodeErr, closeErr)
		}
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	manifestCacheState.set(&decodedManifest)
	return &decodedManifest, nil
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

func IsValidVersion(ctx context.Context, version string, client httpclient.Doer) (bool, error) {
	if version == "" {
		return false, nil
	}

	manifest, err := getMinecraftVersionManifest(ctx, client)

	if err != nil {
		return false, err
	}

	for _, manifestVersion := range manifest.Versions {
		if manifestVersion.ID == version {
			return true, nil
		}
	}

	return false, nil
}

func GetAllMinecraftVersions(ctx context.Context, client httpclient.Doer) []string {
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

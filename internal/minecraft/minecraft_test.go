package minecraft

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"shanhu.io/g/https/httpstest"
	"testing"
)

func TestMinecraft(t *testing.T) {
	t.Run("GetLatestVersion_1", func(t *testing.T) {
		ClearManifestCache()
		mockServer, _ := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Write([]byte(`{"latest":{"release":"1.21.2"}}`))
		}))
		defer mockServer.Close()

		ver, _ := GetLatestVersion(context.Background(), mockServer.Client())

		assert.Equal(t, "1.21.2", ver)
	})

	t.Run("IsValidVersion", func(t *testing.T) {
		ClearManifestCache()
		mockServer, _ := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Write([]byte(`{"versions": [{
      "id": "24w34a",
      "type": "snapshot",
      "url": "https://piston-meta.mojang.com/v1/packages/17e3b903641353554e4b1728df2b62b97562d0ab/24w34a.json",
      "time": "2024-08-21T14:24:24+00:00",
      "releaseTime": "2024-08-21T14:14:13+00:00"
    },
    {
      "id": "24w33a",
      "type": "snapshot",
      "url": "https://piston-meta.mojang.com/v1/packages/3c8612a383ea5e0e86d8d0a4c84b3c56c90e7095/24w33a.json",
      "time": "2024-08-21T13:00:55+00:00",
      "releaseTime": "2024-08-15T12:39:34+00:00"
    },
    {
      "id": "1.21.1",
      "type": "release",
      "url": "https://piston-meta.mojang.com/v1/packages/d1937ef3108629ae7b60e468b3846e6e02ddeebb/1.21.1.json",
      "time": "2024-08-21T13:00:55+00:00",
      "releaseTime": "2024-08-08T12:24:45+00:00"
    }]}`))
		}))
		defer mockServer.Close()

		assert.True(t, IsValidVersion(context.Background(), "1.21.1", mockServer.Client()))
		assert.False(t, IsValidVersion(context.Background(), "1.21.2", mockServer.Client()))
		assert.False(t, IsValidVersion(context.Background(), "", mockServer.Client()))
		assert.False(t, IsValidVersion(context.Background(), "1.21.3", mockServer.Client()))
		assert.True(t, IsValidVersion(context.Background(), "24w33a", mockServer.Client()))

	})

	t.Run("GetAllMineCraftVersions", func(t *testing.T) {
		ClearManifestCache()
		mockServer, _ := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Write([]byte(`{"versions": [{
      "id": "24w34a",
      "type": "snapshot",
      "url": "https://piston-meta.mojang.com/v1/packages/17e3b903641353554e4b1728df2b62b97562d0ab/24w34a.json",
      "time": "2024-08-21T14:24:24+00:00",
      "releaseTime": "2024-08-21T14:14:13+00:00"
    },
    {
      "id": "24w33a",
      "type": "snapshot",
      "url": "https://piston-meta.mojang.com/v1/packages/3c8612a383ea5e0e86d8d0a4c84b3c56c90e7095/24w33a.json",
      "time": "2024-08-21T13:00:55+00:00",
      "releaseTime": "2024-08-15T12:39:34+00:00"
    },
    {
      "id": "1.21.1",
      "type": "release",
      "url": "https://piston-meta.mojang.com/v1/packages/d1937ef3108629ae7b60e468b3846e6e02ddeebb/1.21.1.json",
      "time": "2024-08-21T13:00:55+00:00",
      "releaseTime": "2024-08-08T12:24:45+00:00"
    }]}`))
		}))
		defer mockServer.Close()

		assert.Equal(t, []string{"24w34a", "24w33a", "1.21.1"}, GetAllMineCraftVersions(context.Background(), mockServer.Client()))
	})

	t.Run("GetLatestVersion_Error", func(t *testing.T) {
		ClearManifestCache()
		mockServer, _ := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer mockServer.Close()

		ver, err := GetLatestVersion(context.Background(), mockServer.Client())

		assert.Empty(t, ver)
		assert.ErrorIs(t, err, CouldNotDetermineLatestVersion)
	})

	t.Run("IsValidVersion_Error", func(t *testing.T) {
		ClearManifestCache()
		mockServer, _ := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer mockServer.Close()

		assert.True(t, IsValidVersion(context.Background(), "1.21.1", mockServer.Client()))
	})

	t.Run("GetAllMineCraftVersions_Error", func(t *testing.T) {
		ClearManifestCache()
		oldUrl := versionManifestUrl
		versionManifestUrl = "xxx"
		mockServer, _ := httpstest.NewServer([]string{}, nil)
		defer mockServer.Close()
		defer func() { versionManifestUrl = oldUrl }()

		assert.Empty(t, GetAllMineCraftVersions(context.Background(), mockServer.Client()))
	})

	t.Run("Caching", func(t *testing.T) {
		ClearManifestCache()
		callCount := 0
		mockServer, _ := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Write([]byte(`{"latest":{"release":"1.21.2"}}`))
		}))
		defer mockServer.Close()

		client := mockServer.Client()

		// First call to populate the cache
		_, err := getMinecraftVersionManifest(context.Background(), client)
		assert.NoError(t, err)

		// Second call should use the cached manifest
		_, err = getMinecraftVersionManifest(context.Background(), client)
		assert.NoError(t, err)

		ClearManifestCache()

		// Third call should refetch after clearing cache
		_, err = getMinecraftVersionManifest(context.Background(), client)
		assert.NoError(t, err)

		assert.Equal(t, 2, callCount, "server should be called twice (cache cleared once)")
	})
}

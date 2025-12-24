package minecraft

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/stretchr/testify/assert"
	"shanhu.io/g/https/httpstest"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (doer doerFunc) Do(req *http.Request) (*http.Response, error) {
	return doer(req)
}

type closeErrorBody struct {
	reader   *strings.Reader
	closeErr error
}

func newCloseErrorBody(payload string, closeErr error) *closeErrorBody {
	return &closeErrorBody{
		reader:   strings.NewReader(payload),
		closeErr: closeErr,
	}
}

func (body *closeErrorBody) Read(p []byte) (int, error) {
	return body.reader.Read(p)
}

func (body *closeErrorBody) Close() error {
	if body.closeErr != nil {
		return body.closeErr
	}
	return nil
}

func writeResponse(t *testing.T, writer http.ResponseWriter, payload string) {
	t.Helper()
	if _, err := writer.Write([]byte(payload)); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func TestMinecraft(t *testing.T) {
	t.Run("GetLatestVersion_1", func(t *testing.T) {
		ClearManifestCache()
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			writeResponse(t, w, `{"latest":{"release":"1.21.2"}}`)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		ver, err := GetLatestVersion(context.Background(), mockServer.Client())
		assert.NoError(t, err)

		assert.Equal(t, "1.21.2", ver)
	})

	t.Run("IsValidVersion", func(t *testing.T) {
		ClearManifestCache()
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			writeResponse(t, w, `{"versions": [{
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
    }]}`)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		assert.True(t, IsValidVersion(context.Background(), "1.21.1", mockServer.Client()))
		assert.False(t, IsValidVersion(context.Background(), "1.21.2", mockServer.Client()))
		assert.False(t, IsValidVersion(context.Background(), "", mockServer.Client()))
		assert.False(t, IsValidVersion(context.Background(), "1.21.3", mockServer.Client()))
		assert.True(t, IsValidVersion(context.Background(), "24w33a", mockServer.Client()))

	})

	t.Run("GetAllMineCraftVersions", func(t *testing.T) {
		ClearManifestCache()
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			writeResponse(t, w, `{"versions": [{
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
    }]}`)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		assert.Equal(t, []string{"24w34a", "24w33a", "1.21.1"}, GetAllMineCraftVersions(context.Background(), mockServer.Client()))
	})

	t.Run("GetLatestVersion_Error", func(t *testing.T) {
		ClearManifestCache()
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			http.Error(w, "not found", http.StatusNotFound)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		ver, err := GetLatestVersion(context.Background(), mockServer.Client())

		assert.Empty(t, ver)
		assert.ErrorIs(t, err, ErrCouldNotDetermineLatestVersion)
	})

	t.Run("GetLatestVersion_Timeout", func(t *testing.T) {
		ClearManifestCache()
		ver, err := GetLatestVersion(context.Background(), doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}))

		assert.Empty(t, ver)
		var timeoutErr *httpclient.TimeoutError
		assert.ErrorAs(t, err, &timeoutErr)
	})

	t.Run("IsValidVersion_Error", func(t *testing.T) {
		ClearManifestCache()
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			http.Error(w, "not found", http.StatusNotFound)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		assert.True(t, IsValidVersion(context.Background(), "1.21.1", mockServer.Client()))
	})

	t.Run("GetAllMineCraftVersions_Error", func(t *testing.T) {
		ClearManifestCache()
		oldURL := versionManifestURL
		versionManifestURL = "xxx"
		mockServer, err := httpstest.NewServer([]string{}, nil)
		assert.NoError(t, err)
		defer mockServer.Close()
		defer func() { versionManifestURL = oldURL }()

		assert.Empty(t, GetAllMineCraftVersions(context.Background(), mockServer.Client()))
	})

	t.Run("Caching", func(t *testing.T) {
		ClearManifestCache()
		callCount := 0
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			writeResponse(t, w, `{"latest":{"release":"1.21.2"}}`)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		client := mockServer.Client()

		// First call to populate the cache
		_, err = getMinecraftVersionManifest(context.Background(), client)
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

	t.Run("GetManifestReturnsErrorOnRequestBuildFailure", func(t *testing.T) {
		ClearManifestCache()
		originalRequest := newRequestWithContext
		newRequestWithContext = func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("request failed")
		}
		defer func() {
			newRequestWithContext = originalRequest
		}()

		manifest, err := getMinecraftVersionManifest(context.Background(), doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected doer call")
		}))
		assert.Error(t, err)
		assert.Nil(t, manifest)
	})

	t.Run("GetManifestReturnsErrorOnCloseFailure", func(t *testing.T) {
		ClearManifestCache()
		closeErr := errors.New("close failed")
		body := newCloseErrorBody(`{"latest":{"release":"1.21.2"}}`, closeErr)
		manifest, err := getMinecraftVersionManifest(context.Background(), doerFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       body,
			}, nil
		}))
		assert.ErrorIs(t, err, closeErr)
		assert.Nil(t, manifest)
		assert.Nil(t, latestManifest)
	})
}

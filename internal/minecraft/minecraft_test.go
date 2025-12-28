package minecraft

import (
	"context"
	"encoding/json"
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

		valid, err := IsValidVersion(context.Background(), "1.21.1", mockServer.Client())
		assert.NoError(t, err)
		assert.True(t, valid)

		valid, err = IsValidVersion(context.Background(), "1.21.2", mockServer.Client())
		assert.NoError(t, err)
		assert.False(t, valid)

		valid, err = IsValidVersion(context.Background(), "", mockServer.Client())
		assert.NoError(t, err)
		assert.False(t, valid)

		valid, err = IsValidVersion(context.Background(), "1.21.3", mockServer.Client())
		assert.NoError(t, err)
		assert.False(t, valid)

		valid, err = IsValidVersion(context.Background(), "24w33a", mockServer.Client())
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("GetAllMinecraftVersions", func(t *testing.T) {
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

		assert.Equal(t, []string{"24w34a", "24w33a", "1.21.1"}, GetAllMinecraftVersions(context.Background(), mockServer.Client()))
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

		valid, err := IsValidVersion(context.Background(), "1.21.1", mockServer.Client())
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("GetAllMinecraftVersions_Error", func(t *testing.T) {
		ClearManifestCache()
		oldURL := versionManifestURL
		versionManifestURL = "xxx"
		mockServer, err := httpstest.NewServer([]string{}, nil)
		assert.NoError(t, err)
		defer mockServer.Close()
		defer func() { versionManifestURL = oldURL }()

		assert.Empty(t, GetAllMinecraftVersions(context.Background(), mockServer.Client()))
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

		// Third call should still use the cached manifest
		_, err = getMinecraftVersionManifest(context.Background(), client)
		assert.NoError(t, err)

		assert.Equal(t, 1, callCount, "server should be called once (cached for lifecycle)")
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
		assert.Nil(t, manifestCacheState.manifest)
	})

	t.Run("GetManifestReturnsErrorOnStatusFailure", func(t *testing.T) {
		ClearManifestCache()
		manifest, err := getMinecraftVersionManifest(context.Background(), doerFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("server error")),
				Request:    req,
			}, nil
		}))
		assert.Nil(t, manifest)
		var responseErr *httpclient.ResponseError
		assert.ErrorAs(t, err, &responseErr)
		assert.Equal(t, http.StatusInternalServerError, responseErr.StatusCode)
		assert.Nil(t, manifestCacheState.manifest)
	})

	t.Run("GetManifestReturnsJoinedErrorOnStatusCloseFailure", func(t *testing.T) {
		ClearManifestCache()
		closeErr := errors.New("close failed")
		manifest, err := getMinecraftVersionManifest(context.Background(), doerFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       newCloseErrorBody("server error", closeErr),
				Request:    req,
			}, nil
		}))
		assert.Nil(t, manifest)
		var responseErr *httpclient.ResponseError
		assert.ErrorAs(t, err, &responseErr)
		assert.ErrorIs(t, err, closeErr)
		assert.Nil(t, manifestCacheState.manifest)
	})

	t.Run("GetManifestReturnsErrorOnDecodeFailure", func(t *testing.T) {
		ClearManifestCache()
		manifest, err := getMinecraftVersionManifest(context.Background(), doerFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{invalid")),
			}, nil
		}))
		assert.Nil(t, manifest)
		assert.Error(t, err)
		assert.Nil(t, manifestCacheState.manifest)
	})

	t.Run("GetManifestReturnsJoinedErrorOnDecodeCloseFailure", func(t *testing.T) {
		ClearManifestCache()
		closeErr := errors.New("close failed")
		manifest, err := getMinecraftVersionManifest(context.Background(), doerFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       newCloseErrorBody("{invalid", closeErr),
			}, nil
		}))
		assert.Nil(t, manifest)
		var syntaxErr *json.SyntaxError
		assert.ErrorAs(t, err, &syntaxErr)
		assert.ErrorIs(t, err, closeErr)
		assert.Nil(t, manifestCacheState.manifest)
	})

	t.Run("NextPatchDownFallsBackWithinSeries", func(t *testing.T) {
		ClearManifestCache()
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			writeResponse(t, w, `{"versions": [
      {"id": "1.21.3", "type": "release"},
      {"id": "1.21.2", "type": "release"},
      {"id": "1.21.1", "type": "release"},
      {"id": "1.20.9", "type": "release"},
      {"id": "26.1-snapshot-1", "type": "snapshot"}
    ]}`)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		next, ok, err := NextPatchDown(context.Background(), "1.21.3", mockServer.Client())
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "1.21.2", next)
	})

	t.Run("NextPatchDownRejectsUnknownRelease", func(t *testing.T) {
		ClearManifestCache()
		mockServer, err := httpstest.NewServer([]string{
			"launchermeta.mojang.com",
		}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/mc/game/version_manifest.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			writeResponse(t, w, `{"versions": [
      {"id": "26.1.3", "type": "release"},
      {"id": "26.1.2", "type": "release"}
    ]}`)
		}))
		assert.NoError(t, err)
		defer mockServer.Close()

		_, _, err = NextPatchDown(context.Background(), "26.1.4", mockServer.Client())
		var invalidVersion *InvalidReleaseVersionError
		assert.ErrorAs(t, err, &invalidVersion)
	})
}

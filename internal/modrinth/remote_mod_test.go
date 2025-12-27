package modrinth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/testutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestFetchRemoteModPrefersPrimaryFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/project/test-mod":
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
		case "/v2/project/test-mod/version":
			response := []map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-01T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes":   map[string]string{"sha1": "abc"},
							"url":      "https://example.com/secondary.jar",
							"filename": "secondary.jar",
							"primary":  false,
						},
						{
							"hashes":   map[string]string{"sha1": "def"},
							"url":      "https://example.com/primary.jar",
							"filename": "primary.jar",
							"primary":  true,
						},
					},
				},
			}
			writeJSONResponse(t, writer, response)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	assert.NoError(t, err)
	assert.Equal(t, "primary.jar", mod.FileName)
	assert.Equal(t, "def", mod.Hash)
	assert.Equal(t, "https://example.com/primary.jar", mod.DownloadURL)
}

func TestFetchRemoteModFallsBackToFirstFileWhenNoPrimary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/project/test-mod":
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
		case "/v2/project/test-mod/version":
			response := []map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-01T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes":   map[string]string{"sha1": "abc"},
							"url":      "https://example.com/first.jar",
							"filename": "first.jar",
						},
						{
							"hashes":   map[string]string{"sha1": "def"},
							"url":      "https://example.com/second.jar",
							"filename": "second.jar",
						},
					},
				},
			}
			writeJSONResponse(t, writer, response)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	assert.NoError(t, err)
	assert.Equal(t, "first.jar", mod.FileName)
	assert.Equal(t, "abc", mod.Hash)
}

func TestFetchRemoteModSelectsNewestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/project/test-mod":
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
		case "/v2/project/test-mod/version":
			response := []map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-01T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes":   map[string]string{"sha1": "old"},
							"url":      "https://example.com/older.jar",
							"filename": "older.jar",
						},
					},
				},
				{
					"project_id":     "test-mod",
					"version_number": "1.1.0",
					"version_type":   "release",
					"date_published": "2024-09-01T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes":   map[string]string{"sha1": "new"},
							"url":      "https://example.com/newer.jar",
							"filename": "newer.jar",
						},
					},
				},
			}
			writeJSONResponse(t, writer, response)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	assert.NoError(t, err)
	assert.Equal(t, "newer.jar", mod.FileName)
	assert.Equal(t, "new", mod.Hash)
}

func TestFetchRemoteModUsesFixedVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/project/test-mod":
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
		case "/v2/project/test-mod/version":
			response := []map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "fixed-version",
					"version_type":   "beta",
					"date_published": "2024-08-01T12:00:00Z",
					"game_versions":  []string{"1.19.4"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes":   map[string]string{"sha1": "abc"},
							"url":      "https://example.com/fixed.jar",
							"filename": "fixed.jar",
						},
					},
				},
			}
			writeJSONResponse(t, writer, response)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		FixedVersion:        "fixed-version",
	}, client)

	assert.NoError(t, err)
	assert.Equal(t, "fixed.jar", mod.FileName)
}

func TestFetchRemoteModFallsBackWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		if request.URL.Path == "/v2/project/test-mod/version" {
			gameVersions := request.URL.Query().Get("game_versions")
			if gameVersions == `["1.20.2"]` {
				writeJSONResponse(t, writer, []map[string]interface{}{})
				return
			}
			writeJSONResponse(t, writer, []map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-02T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes":   map[string]string{"sha1": "abc"},
							"url":      "https://example.com/file.jar",
							"filename": "file.jar",
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.2",
		Loader:              models.FABRIC,
		AllowFallback:       true,
	}, client)

	assert.NoError(t, err)
	assert.Equal(t, "file.jar", mod.FileName)
}

func TestFetchRemoteModReturnsNoCompatibleFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		if request.URL.Path == "/v2/project/test-mod/version" {
			writeJSONResponse(t, writer, []map[string]interface{}{})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFetchRemoteModReturnsNoCompatibleWithoutFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		if request.URL.Path == "/v2/project/test-mod/version" {
			writeJSONResponse(t, writer, []map[string]interface{}{})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.2",
		Loader:              models.FABRIC,
	}, client)

	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFetchRemoteModRejectsEmptyFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		if request.URL.Path == "/v2/project/test-mod/version" {
			writeJSONResponse(t, writer, []map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-01T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files":          []map[string]interface{}{},
				},
			})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFetchRemoteModReturnsNoCompatibleWhenFallbackStops(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		if request.URL.Path == "/v2/project/test-mod/version" {
			writeJSONResponse(t, writer, []map[string]interface{}{})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       true,
	}, client)

	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFetchRemoteModFailsOnMissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		if request.URL.Path == "/v2/project/test-mod/version" {
			writeJSONResponse(t, writer, []map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-01T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes":   map[string]string{"sha1": ""},
							"url":      "https://example.com/file.jar",
							"filename": "file.jar",
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFetchRemoteModMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "missing", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var notFound *models.ModNotFoundError
	assert.ErrorAs(t, err, &notFound)
}

func TestFetchRemoteModMapsNotFoundFromVersionLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/project/test-mod":
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
		case "/v2/project/test-mod/version":
			writer.WriteHeader(http.StatusNotFound)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var notFound *models.ModNotFoundError
	assert.ErrorAs(t, err, &notFound)
}

func TestSelectPrimaryFileHandlesEmptyFiles(t *testing.T) {
	_, ok := selectPrimaryFile(nil)
	assert.False(t, ok)
}

func TestFilterVersionsSkipsUnmatchedReleaseAndVersion(t *testing.T) {
	versions := Versions{
		{
			VersionNumber: "1.0.0",
			Type:          models.Beta,
			GameVersions:  []string{"1.19.4"},
		},
	}

	result := filterVersions(versions, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
	}, "1.20.1")

	assert.Empty(t, result)
}

func TestFilterVersionsSkipsMismatchedGameVersion(t *testing.T) {
	versions := Versions{
		{
			VersionNumber: "1.0.0",
			Type:          models.Release,
			GameVersions:  []string{"1.19.4"},
		},
	}

	result := filterVersions(versions, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
	}, "1.20.1")

	assert.Empty(t, result)
}

func TestFilterVersionsFixedVersionMismatch(t *testing.T) {
	versions := Versions{
		{VersionNumber: "1.0.0"},
	}

	result := filterVersions(versions, models.FetchOptions{
		FixedVersion: "2.0.0",
	}, "1.20.1")

	assert.Empty(t, result)
}

func TestFetchRemoteModReturnsApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var apiError *globalerrors.ProjectAPIError
	assert.ErrorAs(t, err, &apiError)
}

func TestFetchRemoteModReturnsApiErrorWhenVersionLookupFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/project/test-mod":
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
		case "/v2/project/test-mod/version":
			writer.WriteHeader(http.StatusInternalServerError)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "test-mod", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var apiError *globalerrors.ProjectAPIError
	assert.ErrorAs(t, err, &apiError)
}

func TestContainsHelpersMatch(t *testing.T) {
	assert.True(t, containsReleaseType([]models.ReleaseType{models.Release}, models.Release))
	assert.True(t, containsGameVersion([]string{"1.20.1"}, "1.20.1"))
}

func TestContainsGameVersionMismatch(t *testing.T) {
	assert.False(t, containsGameVersion([]string{"1.19.4"}, "1.20.1"))
}

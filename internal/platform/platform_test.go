package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestFetchModrinth_Succeeds(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/project/test-mod":
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
		case r.URL.Path == "/v2/project/test-mod/version":
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
							"hashes": map[string]string{
								"sha1": "abc",
							},
							"url":      "https://example.com/file.jar",
							"filename": "file.jar",
						},
					},
				},
				{
					"project_id":     "test-mod",
					"version_number": "0.9.0",
					"version_type":   "release",
					"date_published": "2024-07-01T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes": map[string]string{
								"sha1": "zzz",
							},
							"url":      "https://example.com/older.jar",
							"filename": "older.jar",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	mod, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       false,
	}, Clients{Modrinth: client})

	assert.NoError(t, err)
	assert.Equal(t, "Test Mod", mod.Name)
	assert.Equal(t, "file.jar", mod.FileName)
	assert.Equal(t, "abc", mod.Hash)
	assert.Equal(t, "https://example.com/file.jar", mod.DownloadURL)
	assert.Equal(t, "2024-08-01T12:00:00Z", mod.ReleaseDate)

	assertPerfSpanExists(t, "platform.fetch_mod")
	assertPerfAttrEquals(t, "platform.fetch_mod", "platform", string(models.MODRINTH))
	assertPerfAttrEquals(t, "platform.fetch_mod", "project_id", "test-mod")
	assertPerfAttrEquals(t, "platform.fetch_mod", "success", true)
}

func TestFetchMod_UnknownPlatformRecordsPerf(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	_, err := FetchMod(context.Background(), "unknown", "abc", FetchOptions{}, Clients{})
	assert.Error(t, err)

	assertPerfSpanExists(t, "platform.fetch_mod")
	assertPerfAttrEquals(t, "platform.fetch_mod", "platform", "unknown")
	assertPerfAttrEquals(t, "platform.fetch_mod", "success", false)
	assertPerfAttrContains(t, "platform.fetch_mod", "error_type", "UnknownPlatformError")
}

func assertPerfSpanExists(t *testing.T, name string) {
	t.Helper()
	spans, err := perf.GetSpans()
	assert.NoError(t, err)
	_, ok := perf.FindSpanByName(spans, name)
	assert.True(t, ok, "expected span %q", name)
}

func assertPerfAttrEquals(t *testing.T, spanName string, key string, expected interface{}) {
	t.Helper()
	spans, err := perf.GetSpans()
	assert.NoError(t, err)
	span, ok := perf.FindSpanByName(spans, spanName)
	assert.True(t, ok, "expected span %q", spanName)
	actual := span.Attributes[key]
	assert.Equal(t, expected, actual)
}

func assertPerfAttrContains(t *testing.T, spanName string, key string, needle string) {
	t.Helper()
	spans, err := perf.GetSpans()
	assert.NoError(t, err)
	span, ok := perf.FindSpanByName(spans, spanName)
	assert.True(t, ok, "expected span %q", spanName)
	value, ok := span.Attributes[key].(string)
	if !ok {
		t.Fatalf("expected span %q attribute %q to be string", spanName, key)
	}
	assert.Contains(t, value, needle)
}

func TestFetchModrinth_Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}

		if r.URL.Path == "/v2/project/test-mod/version" {
			gameVersions := r.URL.Query().Get("game_versions")
			if gameVersions == `["1.20.2"]` {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-02T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes": map[string]string{
								"sha1": "abc",
							},
							"url":      "https://example.com/file.jar",
							"filename": "file.jar",
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	mod, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.2",
		Loader:              models.FABRIC,
		AllowFallback:       true,
	}, Clients{Modrinth: client})

	assert.NoError(t, err)
	assert.Equal(t, "file.jar", mod.FileName)
}

func TestFetchModrinth_FixedVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}

		if r.URL.Path == "/v2/project/test-mod/version" {
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"project_id":     "test-mod",
					"version_number": "1.0.0",
					"version_type":   "release",
					"date_published": "2024-08-02T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes": map[string]string{
								"sha1": "abc",
							},
							"url":      "https://example.com/file.jar",
							"filename": "file.jar",
						},
					},
				},
				{
					"project_id":     "test-mod",
					"version_number": "2.0.0",
					"version_type":   "release",
					"date_published": "2024-08-03T12:00:00Z",
					"game_versions":  []string{"1.20.1"},
					"loaders":        []string{"fabric"},
					"files": []map[string]interface{}{
						{
							"hashes": map[string]string{
								"sha1": "def",
							},
							"url":      "https://example.com/other.jar",
							"filename": "other.jar",
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	mod, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       false,
		FixedVersion:        "2.0.0",
	}, Clients{Modrinth: client})

	assert.NoError(t, err)
	assert.Equal(t, "other.jar", mod.FileName)
	assert.Equal(t, "def", mod.Hash)
}

func TestFetchModrinth_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "missing-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
	_, ok := err.(*ModNotFoundError)
	assert.True(t, ok)
}

func TestFetchModrinth_NoFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}

		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       false,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
	_, ok := err.(*NoCompatibleFileError)
	assert.True(t, ok)
}

func TestFetchCurseforge_Succeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/mods/1234":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/mods/1234/files"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"fileName":    "cf.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"hashes": []map[string]interface{}{
							{"algo": 1, "value": "cfhash"},
						},
						"downloadUrl": "https://example.com/cf.jar",
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
						"isAvailable": true,
					},
					{
						"fileName":    "old.jar",
						"fileDate":    "2024-07-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"hashes": []map[string]interface{}{
							{"algo": 1, "value": "oldhash"},
						},
						"downloadUrl": "https://example.com/old.jar",
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
						"isAvailable": true,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	mod, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       false,
	}, Clients{Curseforge: client})

	assert.NoError(t, err)
	assert.Equal(t, "cf.jar", mod.FileName)
	assert.Equal(t, "cfhash", mod.Hash)
	assert.Equal(t, "CF Mod", mod.Name)
}

func TestFetchCurseforge_Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/mods/1234":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/mods/1234/files"):
			gameVersion := r.URL.Query().Get("gameVersion")
			if gameVersion == "1.20.2" {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"fileName":    "cf.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  10,
						"hashes": []map[string]interface{}{
							{"algo": 1, "value": "cfhash"},
						},
						"downloadUrl": "https://example.com/cf.jar",
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
						"isAvailable": true,
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	mod, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.2",
		Loader:              models.FABRIC,
		AllowFallback:       true,
	}, Clients{Curseforge: client})

	assert.NoError(t, err)
	assert.Equal(t, "cf.jar", mod.FileName)
}

func TestFetchCurseforge_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.CURSEFORGE, "missing", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Curseforge: client})

	assert.Error(t, err)
	_, ok := err.(*ModNotFoundError)
	assert.True(t, ok)
}

func TestFetchCurseforge_NoFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mods/1234" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
			return
		}
		if r.URL.Path == "/mods/1234/files" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		}
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       false,
	}, Clients{Curseforge: client})

	assert.Error(t, err)
	_, ok := err.(*NoCompatibleFileError)
	assert.True(t, ok)
}

func TestFetchCurseforge_MissingDownloadURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods/1234":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case "/mods/1234/files":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"fileName":    "cf.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"hashes": []map[string]interface{}{
							{"algo": 1, "value": "cfhash"},
						},
						"downloadUrl": "",
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
						"isAvailable": true,
					},
				},
			})
		}
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Curseforge: client})

	assert.Error(t, err)
}

func TestFetchCurseforge_MissingHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mods/1234":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case "/mods/1234/files":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"fileName":    "cf.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"hashes": []map[string]interface{}{
							{"algo": 2, "value": "md5"},
						},
						"downloadUrl": "https://example.com/cf.jar",
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
						"isAvailable": true,
					},
				},
			})
		}
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)
	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Curseforge: client})

	assert.Error(t, err)
}

func TestFetchCurseforge_UnsupportedLoader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"id":   1234,
				"name": "CF Mod",
			},
		})
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)
	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.BUKKIT,
	}, Clients{Curseforge: client})

	assert.Error(t, err)
}

func TestFetchCurseforgeFilesErrors(t *testing.T) {
	notFoundServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundServer.Close()

	t.Setenv("CURSEFORGE_API_URL", notFoundServer.URL)
	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))
	_, err := fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, client)
	assert.Error(t, err)

	badJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer badJSONServer.Close()

	t.Setenv("CURSEFORGE_API_URL", badJSONServer.URL)
	_, err = fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, client)
	assert.Error(t, err)

	errorDoer := errorDoer{}
	_, err = fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, errorDoer)
	assert.Error(t, err)
}

func TestFetchMod_UnknownPlatform(t *testing.T) {
	_, err := FetchMod(context.Background(), "unknown", "abc", FetchOptions{}, Clients{})
	assert.Error(t, err)
}

func TestDefaultClients(t *testing.T) {
	clients := DefaultClients(nil)
	assert.NotNil(t, clients.Modrinth)
	assert.NotNil(t, clients.Curseforge)
}

func TestCurseforgeHelpers(t *testing.T) {
	_, ok := curseforgeReleaseType(curseforge.FileReleaseType(99))
	assert.False(t, ok)

	rt, ok := curseforgeReleaseType(curseforge.Beta)
	assert.True(t, ok)
	assert.Equal(t, models.Beta, rt)

	rt, ok = curseforgeReleaseType(curseforge.Alpha)
	assert.True(t, ok)
	assert.Equal(t, models.Alpha, rt)

	hash, err := getCurseforgeHash([]curseforge.FileHash{{Algorithm: curseforge.MD5, Hash: ""}}, curseforge.SHA1)
	assert.Error(t, err)
	assert.Empty(t, hash)

	_, err = getCurseforgeHash([]curseforge.FileHash{{Algorithm: curseforge.SHA1, Hash: ""}}, curseforge.SHA1)
	assert.Error(t, err)

	files := filterCurseforgeFiles([]curseforge.File{
		{
			FileName:    "file.jar",
			FileStatus:  curseforge.Rejected,
			ReleaseType: curseforge.Release,
			IsAvailable: false,
		},
	}, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.20.1")
	assert.Len(t, files, 0)
}

func TestMapProjectNotFoundPassthrough(t *testing.T) {
	err := errors.New("boom")
	assert.Equal(t, err, mapProjectNotFound(models.MODRINTH, "abc", err))
}

func TestErrorMessages(t *testing.T) {
	assert.Contains(t, (&UnknownPlatformError{Platform: "x"}).Error(), "unknown platform")
	assert.Contains(t, (&ModNotFoundError{Platform: models.MODRINTH, ProjectID: "abc"}).Error(), "mod not found")
	assert.Contains(t, (&NoCompatibleFileError{Platform: models.CURSEFORGE, ProjectID: "abc"}).Error(), "no compatible file")
}

func TestVersionFormatZeroPatch(t *testing.T) {
	parts := versionParts{major: 1, minor: 20, patch: 0}
	assert.Equal(t, "1.20", parts.format(0))
}

func TestCurseforgeLoaderFromLoaderUnsupported(t *testing.T) {
	_, err := curseforgeLoaderFromLoader(models.BUKKIT)
	assert.Error(t, err)

	_, err = curseforgeLoaderFromLoader(models.FORGE)
	assert.NoError(t, err)
	_, err = curseforgeLoaderFromLoader(models.QUILT)
	assert.NoError(t, err)
	_, err = curseforgeLoaderFromLoader(models.CAULDRON)
	assert.NoError(t, err)
	_, err = curseforgeLoaderFromLoader(models.LITELOADER)
	assert.NoError(t, err)
	_, err = curseforgeLoaderFromLoader(models.NEOFORGE)
	assert.NoError(t, err)
}

func TestContainsHelpers(t *testing.T) {
	assert.False(t, containsReleaseType([]models.ReleaseType{models.Release}, models.Alpha))
	assert.False(t, containsGameVersion([]string{"1.20.1"}, "1.19.4"))
}

func TestFetchModrinth_MissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"project_id":     "test-mod",
				"version_number": "1.0.0",
				"version_type":   "release",
				"date_published": "2024-08-02T12:00:00Z",
				"game_versions":  []string{"1.20.1"},
				"loaders":        []string{"fabric"},
				"files": []map[string]interface{}{
					{
						"hashes": map[string]string{
							"sha1": "",
						},
						"url":      "",
						"filename": "file.jar",
					},
				},
			},
		})
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
}

func TestFetchModrinth_FixedVersionNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"project_id":     "test-mod",
				"version_number": "1.0.0",
				"version_type":   "release",
				"date_published": "2024-08-02T12:00:00Z",
				"game_versions":  []string{"1.20.1"},
				"loaders":        []string{"fabric"},
				"files": []map[string]interface{}{
					{
						"hashes": map[string]string{
							"sha1": "abc",
						},
						"url":      "https://example.com",
						"filename": "file.jar",
					},
				},
			},
		})
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		FixedVersion:        "2.0.0",
	}, Clients{Modrinth: client})

	assert.Error(t, err)
}

func TestFetchModrinth_NoFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"project_id":     "test-mod",
				"version_number": "1.0.0",
				"version_type":   "release",
				"date_published": "2024-08-02T12:00:00Z",
				"game_versions":  []string{"1.20.1"},
				"loaders":        []string{"fabric"},
				"files":          []map[string]interface{}{},
			},
		})
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
}

func TestFetchModrinth_VersionApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
}

func TestFetchCurseforge_FixedVersionFilter(t *testing.T) {
	files := filterCurseforgeFiles([]curseforge.File{
		{
			FileName:    "match.jar",
			ReleaseType: curseforge.Release,
			FileStatus:  curseforge.Approved,
			Hashes: []curseforge.FileHash{
				{Algorithm: curseforge.SHA1, Hash: "abc"},
			},
			DownloadUrl: "https://example.com/match.jar",
			SortableGameVersions: []curseforge.SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
			IsAvailable: true,
		},
		{
			FileName:    "other.jar",
			ReleaseType: curseforge.Release,
			FileStatus:  curseforge.Approved,
			Hashes: []curseforge.FileHash{
				{Algorithm: curseforge.SHA1, Hash: "def"},
			},
			DownloadUrl: "https://example.com/other.jar",
			SortableGameVersions: []curseforge.SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
			IsAvailable: true,
		},
	}, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		FixedVersion:        "match.jar",
	}, "1.20.1")

	assert.Len(t, files, 1)
	assert.Equal(t, "match.jar", files[0].FileName)
}

func TestFetchCurseforge_FixedVersionMismatch(t *testing.T) {
	files := filterCurseforgeFiles([]curseforge.File{
		{
			FileName:    "other.jar",
			ReleaseType: curseforge.Release,
			FileStatus:  curseforge.Approved,
			Hashes: []curseforge.FileHash{
				{Algorithm: curseforge.SHA1, Hash: "def"},
			},
			DownloadUrl: "https://example.com/other.jar",
			SortableGameVersions: []curseforge.SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
			IsAvailable: true,
		},
	}, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		FixedVersion:        "missing.jar",
	}, "1.20.1")

	assert.Len(t, files, 0)
}

func TestFilterCurseforgeFilesInvalidReleaseType(t *testing.T) {
	files := filterCurseforgeFiles([]curseforge.File{
		{
			FileName:    "file.jar",
			ReleaseType: curseforge.FileReleaseType(99),
			FileStatus:  curseforge.Approved,
			DownloadUrl: "https://example.com/file.jar",
			Hashes: []curseforge.FileHash{
				{Algorithm: curseforge.SHA1, Hash: "abc"},
			},
			SortableGameVersions: []curseforge.SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
			IsAvailable: true,
		},
	}, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.20.1")

	assert.Len(t, files, 0)
}

func TestFilterCurseforgeFilesVersionMismatch(t *testing.T) {
	files := filterCurseforgeFiles([]curseforge.File{
		{
			FileName:    "file.jar",
			ReleaseType: curseforge.Release,
			FileStatus:  curseforge.Approved,
			Hashes: []curseforge.FileHash{
				{Algorithm: curseforge.SHA1, Hash: "abc"},
			},
			DownloadUrl: "https://example.com/file.jar",
			SortableGameVersions: []curseforge.SortableGameVersion{
				{GameVersionName: "1.19.4"},
			},
			IsAvailable: true,
		},
	}, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.20.1")

	assert.Len(t, files, 0)
}

func TestFilterCurseforgeFilesUnavailable(t *testing.T) {
	files := filterCurseforgeFiles([]curseforge.File{
		{
			FileName:    "file.jar",
			ReleaseType: curseforge.Release,
			FileStatus:  curseforge.Approved,
			Hashes: []curseforge.FileHash{
				{Algorithm: curseforge.SHA1, Hash: "abc"},
			},
			DownloadUrl: "https://example.com/file.jar",
			SortableGameVersions: []curseforge.SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
			IsAvailable: false,
		},
	}, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.20.1")

	assert.Len(t, files, 0)
}

func TestFetchCurseforgeFilesUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)
	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, client)
	assert.Error(t, err)
}

func TestFetchModrinth_FallbackCannotGoDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20", // no patch part so fallback stops
		Loader:              models.FABRIC,
		AllowFallback:       true,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
}

func TestFetchCurseforge_ProjectNotFoundFromFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mods/1234" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	t.Setenv("CURSEFORGE_API_URL", server.URL)
	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Curseforge: client})

	assert.Error(t, err)
}

func TestFetchModrinth_VersionsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			_, _ = w.Write([]byte(`{"title":"Test Mod","id":"test-mod"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	t.Setenv("MODRINTH_API_URL", server.URL)

	client := httpClient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
}

func TestFilterModrinthVersionsBranches(t *testing.T) {
	versions := modrinth.Versions{
		{
			VersionNumber: "1.0.0",
			Type:          models.Release,
			GameVersions:  []string{"1.20.1"},
		},
	}

	result := filterModrinthVersions(versions, FetchOptions{
		FixedVersion: "1.0.0",
	}, "1.20.1")
	assert.Len(t, result, 1)

	result = filterModrinthVersions(versions, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Alpha},
	}, "1.20.1")
	assert.Len(t, result, 0)

	result = filterModrinthVersions(versions, FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.19.4")
	assert.Len(t, result, 0)
}

func TestFetchCurseforge_DoError(t *testing.T) {
	_, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Curseforge: errorDoer{}})

	assert.Error(t, err)
}

type errorDoer struct{}

func (errorDoer) Do(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("doer error")
}

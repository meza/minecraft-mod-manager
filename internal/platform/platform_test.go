package platform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/testutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func writeJSONResponse(t *testing.T, writer http.ResponseWriter, payload any) {
	t.Helper()
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		t.Fatalf("write json response: %v", err)
	}
}

func writeStringResponse(t *testing.T, writer http.ResponseWriter, payload string) {
	t.Helper()
	if _, err := writer.Write([]byte(payload)); err != nil {
		t.Fatalf("write string response: %v", err)
	}
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

func (c *closeErrorBody) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *closeErrorBody) Close() error {
	if c.closeErr != nil {
		return c.closeErr
	}
	return nil
}

func TestFetchModrinth_Succeeds(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/project/test-mod":
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
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
			writeJSONResponse(t, w, response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}

		if r.URL.Path == "/v2/project/test-mod/version" {
			gameVersions := r.URL.Query().Get("game_versions")
			if gameVersions == `["1.20.2"]` {
				writeJSONResponse(t, w, []map[string]interface{}{})
				return
			}
			writeJSONResponse(t, w, []map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.2",
		Loader:              models.FABRIC,
		AllowFallback:       true,
	}, Clients{Modrinth: client})

	assert.NoError(t, err)
	assert.Equal(t, "file.jar", mod.FileName)
}

func TestParseIntReturnsZeroOnInvalidValue(t *testing.T) {
	assert.Equal(t, 0, parseInt("not-a-number"))
}

func TestFetchCurseforgeFilesReturnsErrorOnRequestBuildFailure(t *testing.T) {
	originalRequest := newRequestWithContext
	newRequestWithContext = func(context.Context, string, string, io.Reader) (*http.Request, error) {
		return nil, errors.New("request failed")
	}
	t.Cleanup(func() {
		newRequestWithContext = originalRequest
	})

	files, err := fetchCurseforgeFiles(context.Background(), "12345", "1.20.1", curseforge.Fabric, errorDoer{})
	assert.Error(t, err)
	assert.Nil(t, files)
}

func TestFetchCurseforgeFilesReturnsErrorOnResponseCloseFailure(t *testing.T) {
	closeErr := errors.New("close failed")
	body := newCloseErrorBody(`{"data":[]}`, closeErr)
	files, err := fetchCurseforgeFiles(context.Background(), "12345", "1.20.1", curseforge.Fabric, responseDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
			Header:     make(http.Header),
		},
	})
	assert.ErrorIs(t, err, closeErr)
	assert.NotNil(t, files)
}

func TestFetchModrinth_FixedVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}

		if r.URL.Path == "/v2/project/test-mod/version" {
			writeJSONResponse(t, w, []map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}

		writeJSONResponse(t, w, []map[string]interface{}{})
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
		case r.URL.Path == "/v1/mods/1234":
			writeJSONResponse(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/mods/1234/files"):
			writeJSONResponse(t, w, map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
		case r.URL.Path == "/v1/mods/1234":
			writeJSONResponse(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/mods/1234/files"):
			gameVersion := r.URL.Query().Get("gameVersion")
			if gameVersion == "1.20.2" {
				writeJSONResponse(t, w, map[string]interface{}{"data": []interface{}{}})
				return
			}
			writeJSONResponse(t, w, map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
		if r.URL.Path == "/v1/mods/1234" {
			writeJSONResponse(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
			return
		}
		if r.URL.Path == "/v1/mods/1234/files" {
			writeJSONResponse(t, w, map[string]interface{}{"data": []interface{}{}})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
		case "/v1/mods/1234":
			writeJSONResponse(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case "/v1/mods/1234/files":
			writeJSONResponse(t, w, map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
		case "/v1/mods/1234":
			writeJSONResponse(t, w, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case "/v1/mods/1234/files":
			writeJSONResponse(t, w, map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Curseforge: client})

	assert.Error(t, err)
}

func TestFetchCurseforge_UnsupportedLoader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(t, w, map[string]interface{}{
			"data": map[string]interface{}{
				"id":   1234,
				"name": "CF Mod",
			},
		})
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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

	baseClient := httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0))
	_, err := fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, testutil.MustNewHostRewriteDoer(notFoundServer.URL, baseClient))
	assert.Error(t, err)

	badJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeStringResponse(t, w, "{")
	}))
	defer badJSONServer.Close()

	_, err = fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, testutil.MustNewHostRewriteDoer(badJSONServer.URL, baseClient))
	assert.Error(t, err)

	errorDoer := errorDoer{}
	_, err = fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, errorDoer)
	assert.Error(t, err)

	timeoutDoer := timeoutDoer{}
	_, err = fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, timeoutDoer)
	assert.Error(t, err)
	var timeoutErr *httpclient.TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
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
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		writeJSONResponse(t, w, []map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		writeJSONResponse(t, w, []map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		writeJSONResponse(t, w, []map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Modrinth: client})

	assert.Error(t, err)
}

func TestFetchModrinth_VersionAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
			DownloadURL: "https://example.com/match.jar",
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
			DownloadURL: "https://example.com/other.jar",
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
			DownloadURL: "https://example.com/other.jar",
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
			DownloadURL: "https://example.com/file.jar",
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
			DownloadURL: "https://example.com/file.jar",
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
			DownloadURL: "https://example.com/file.jar",
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

	client := httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := fetchCurseforgeFiles(context.Background(), "1234", "1.20.1", curseforge.Forge, testutil.MustNewHostRewriteDoer(server.URL, client))
	assert.Error(t, err)
}

func TestFetchModrinth_FallbackCannotGoDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/project/test-mod" {
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		writeJSONResponse(t, w, []map[string]interface{}{})
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
		if r.URL.Path == "/v1/mods/1234" {
			writeJSONResponse(t, w, map[string]interface{}{
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

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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
			writeStringResponse(t, w, `{"title":"Test Mod","id":"test-mod"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

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

type responseDoer struct {
	response *http.Response
	err      error
}

func (r responseDoer) Do(_ *http.Request) (*http.Response, error) {
	return r.response, r.err
}

type timeoutDoer struct{}

func (timeoutDoer) Do(_ *http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
}

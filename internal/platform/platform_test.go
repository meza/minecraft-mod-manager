package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/testutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestFetchModRecordsPerfOnUnknownPlatform(t *testing.T) {
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

func TestFetchModUsesModrinthProvider(t *testing.T) {
	perf.Reset()
	t.Cleanup(perf.Reset)
	assert.NoError(t, perf.Init(perf.Config{Enabled: true}))

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/project/test-mod":
			writeStringResponse(t, writer, `{"title":"Test Mod","id":"test-mod"}`)
		case "/v2/project/test-mod/version":
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
							"hashes":   map[string]string{"sha1": "abc"},
							"url":      "https://example.com/file.jar",
							"filename": "file.jar",
						},
					},
				},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))
	mod, err := FetchMod(context.Background(), models.MODRINTH, "test-mod", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Modrinth: client})

	assert.NoError(t, err)
	assert.Equal(t, "file.jar", mod.FileName)
	assertPerfAttrEquals(t, "platform.fetch_mod", "success", true)
}

func TestFetchModUsesCurseforgeProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/mods/1234":
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
		case "/v1/mods/1234/files":
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":          2,
						"fileName":    "file.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"isAvailable": true,
						"downloadUrl": "https://example.com/file.jar",
						"hashes":      []map[string]interface{}{{"algo": 1, "value": "bbb"}},
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
					},
				},
				"pagination": map[string]interface{}{
					"index":       0,
					"pageSize":    50,
					"resultCount": 1,
					"totalCount":  1,
				},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))
	mod, err := FetchMod(context.Background(), models.CURSEFORGE, "1234", FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, Clients{Curseforge: client})

	assert.NoError(t, err)
	assert.Equal(t, "file.jar", mod.FileName)
}

func TestDefaultClients(t *testing.T) {
	clients := DefaultClients(nil)
	assert.NotNil(t, clients.Modrinth)
	assert.NotNil(t, clients.Curseforge)
}

func TestUnknownPlatformErrorMessage(t *testing.T) {
	assert.Contains(t, (&UnknownPlatformError{Platform: "x"}).Error(), "unknown platform")
}

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

package curseforge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/testutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestFetchRemoteModSucceeds(t *testing.T) {
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
						"id":          1,
						"fileName":    "older.jar",
						"fileDate":    "2024-07-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"isAvailable": true,
						"downloadUrl": "https://example.com/older.jar",
						"hashes":      []map[string]interface{}{{"algo": 1, "value": "aaa"}},
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
					},
					{
						"id":          2,
						"fileName":    "newer.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"isAvailable": true,
						"downloadUrl": "https://example.com/newer.jar",
						"hashes":      []map[string]interface{}{{"algo": 1, "value": "bbb"}},
						"sortableGameVersions": []map[string]interface{}{
							{"gameVersionName": "1.20.1"},
						},
					},
				},
				"pagination": map[string]interface{}{
					"index":       0,
					"pageSize":    50,
					"resultCount": 2,
					"totalCount":  2,
				},
			})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	assert.NoError(t, err)
	assert.Equal(t, "CF Mod", mod.Name)
	assert.Equal(t, "newer.jar", mod.FileName)
	assert.Equal(t, "bbb", mod.Hash)
	assert.Equal(t, "https://example.com/newer.jar", mod.DownloadURL)
}

func TestFetchRemoteModFallsBackWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/mods/1234" {
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
			return
		}
		if request.URL.Path == "/v1/mods/1234/files" {
			gameVersion := request.URL.Query().Get("gameVersion")
			if gameVersion == "1.20.2" {
				writeJSONResponse(t, writer, map[string]interface{}{
					"data": []interface{}{},
					"pagination": map[string]interface{}{
						"index":       0,
						"pageSize":    50,
						"resultCount": 0,
						"totalCount":  0,
					},
				})
				return
			}
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":          2,
						"fileName":    "fallback.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"isAvailable": true,
						"downloadUrl": "https://example.com/fallback.jar",
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
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	mod, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.2",
		Loader:              models.FABRIC,
		AllowFallback:       true,
	}, client)

	assert.NoError(t, err)
	assert.Equal(t, "fallback.jar", mod.FileName)
}

func TestFetchRemoteModReturnsNoCompatibleFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/mods/1234" {
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
			return
		}
		if request.URL.Path == "/v1/mods/1234/files" {
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": []interface{}{},
				"pagination": map[string]interface{}{
					"index":       0,
					"pageSize":    50,
					"resultCount": 0,
					"totalCount":  0,
				},
			})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFetchRemoteModReturnsNoCompatibleWhenFallbackStops(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/mods/1234" {
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1234,
					"name": "CF Mod",
				},
			})
			return
		}
		if request.URL.Path == "/v1/mods/1234/files" {
			writeJSONResponse(t, writer, map[string]interface{}{
				"data": []interface{}{},
				"pagination": map[string]interface{}{
					"index":       0,
					"pageSize":    50,
					"resultCount": 0,
					"totalCount":  0,
				},
			})
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
		AllowFallback:       true,
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

func TestFetchRemoteModRejectsMissingHash(t *testing.T) {
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
						"fileName":    "missing.jar",
						"fileDate":    "2024-08-01T12:00:00Z",
						"releaseType": 1,
						"fileStatus":  4,
						"isAvailable": true,
						"downloadUrl": "https://example.com/missing.jar",
						"hashes":      []map[string]interface{}{{"algo": 1, "value": ""}},
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

	_, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFetchRemoteModReturnsNotFoundForInvalidProjectID(t *testing.T) {
	client := httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchRemoteMod(context.Background(), "not-a-number", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var notFound *models.ModNotFoundError
	assert.ErrorAs(t, err, &notFound)
}

func TestFetchRemoteModRejectsUnsupportedLoader(t *testing.T) {
	client := httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0))

	_, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.BUKKIT,
	}, client)

	assert.Error(t, err)
}

func TestModLoaderTypeFromLoader(t *testing.T) {
	tests := []struct {
		loader   models.Loader
		expected ModLoaderType
		hasError bool
	}{
		{loader: models.FABRIC, expected: Fabric},
		{loader: models.QUILT, expected: Quilt},
		{loader: models.FORGE, expected: Forge},
		{loader: models.CAULDRON, expected: Cauldron},
		{loader: models.LITELOADER, expected: LiteLoader},
		{loader: models.NEOFORGE, expected: NeoForge},
		{loader: models.BUKKIT, hasError: true},
	}

	for _, test := range tests {
		modLoaderType, err := modLoaderTypeFromLoader(test.loader)
		if test.hasError {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, test.expected, modLoaderType)
	}
}

func TestGetCurseforgeHash(t *testing.T) {
	hash, err := getCurseforgeHash([]FileHash{{Algorithm: SHA1, Hash: "abc"}})
	assert.NoError(t, err)
	assert.Equal(t, "abc", hash)

	_, err = getCurseforgeHash([]FileHash{{Algorithm: SHA1, Hash: ""}})
	assert.Error(t, err)

	_, err = getCurseforgeHash([]FileHash{{Algorithm: MD5, Hash: "abc"}})
	assert.Error(t, err)
}

func TestFilterCurseforgeFilesFiltersByReleaseTypeAndVersion(t *testing.T) {
	date, err := time.Parse(time.RFC3339, "2024-08-01T12:00:00Z")
	assert.NoError(t, err)
	files := []File{
		{
			FileName:    "ok.jar",
			FileDate:    date,
			ReleaseType: Release,
			FileStatus:  Approved,
			IsAvailable: true,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "abc"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
		},
		{
			FileName:    "skip.jar",
			FileDate:    date,
			ReleaseType: Beta,
			FileStatus:  Approved,
			IsAvailable: true,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "def"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
		},
	}

	filtered := filterCurseforgeFiles(files, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
	}, "1.20.1")

	assert.Len(t, filtered, 1)
	assert.Equal(t, "ok.jar", filtered[0].FileName)
}

func TestFilterCurseforgeFilesSkipsMismatchedGameVersion(t *testing.T) {
	files := []File{
		{
			FileName:    "skip.jar",
			ReleaseType: Release,
			FileStatus:  Approved,
			IsAvailable: true,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "abc"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.19.4"},
			},
		},
	}

	filtered := filterCurseforgeFiles(files, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.20.1")

	assert.Empty(t, filtered)
}

func TestFilterCurseforgeFilesSkipsUnknownReleaseType(t *testing.T) {
	files := []File{
		{
			FileName:    "skip.jar",
			ReleaseType: FileReleaseType(99),
			FileStatus:  Approved,
			IsAvailable: true,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "abc"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
		},
	}

	filtered := filterCurseforgeFiles(files, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.20.1")

	assert.Empty(t, filtered)
}

func TestCurseforgeReleaseTypeUnknown(t *testing.T) {
	_, ok := curseforgeReleaseType(FileReleaseType(99))
	assert.False(t, ok)
}

func TestFetchRemoteModMapsProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var notFound *models.ModNotFoundError
	assert.ErrorAs(t, err, &notFound)
}

func TestFetchRemoteModMapsNotFoundFromFiles(t *testing.T) {
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
			writer.WriteHeader(http.StatusNotFound)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var notFound *models.ModNotFoundError
	assert.ErrorAs(t, err, &notFound)
}

func TestFetchRemoteModReturnsApiError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := testutil.MustNewHostRewriteDoer(server.URL, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))

	_, err := FetchRemoteMod(context.Background(), "1234", models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		GameVersion:         "1.20.1",
		Loader:              models.FABRIC,
	}, client)

	var apiError *globalerrors.ProjectAPIError
	assert.ErrorAs(t, err, &apiError)
}

func TestBuildRemoteModRejectsMissingURL(t *testing.T) {
	project := &Project{Name: "CF Mod"}
	_, err := buildRemoteMod(project, File{
		FileName:    "missing.jar",
		FileDate:    time.Now(),
		ReleaseType: Release,
		FileStatus:  Approved,
		IsAvailable: true,
		Hashes:      []FileHash{{Algorithm: SHA1, Hash: "abc"}},
	}, "1234")
	var noCompatible *models.NoCompatibleFileError
	assert.ErrorAs(t, err, &noCompatible)
}

func TestFilterCurseforgeFilesHonorsFixedVersion(t *testing.T) {
	date := time.Now()
	files := []File{
		{
			FileName:    "match.jar",
			FileDate:    date,
			ReleaseType: Release,
			FileStatus:  Approved,
			IsAvailable: true,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "abc"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
		},
		{
			FileName:    "other.jar",
			FileDate:    date,
			ReleaseType: Release,
			FileStatus:  Approved,
			IsAvailable: true,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "def"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
		},
	}

	filtered := filterCurseforgeFiles(files, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		FixedVersion:        "match.jar",
	}, "1.20.1")

	assert.Len(t, filtered, 1)
	assert.Equal(t, "match.jar", filtered[0].FileName)
}

func TestFilterCurseforgeFilesFixedVersionMismatch(t *testing.T) {
	files := []File{
		{
			FileName:    "other.jar",
			ReleaseType: Release,
			FileStatus:  Approved,
			IsAvailable: true,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "def"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
		},
	}

	filtered := filterCurseforgeFiles(files, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
		FixedVersion:        "missing.jar",
	}, "1.20.1")

	assert.Empty(t, filtered)
}

func TestFilterCurseforgeFilesSkipsUnavailable(t *testing.T) {
	files := []File{
		{
			FileName:    "skip.jar",
			ReleaseType: Release,
			FileStatus:  Rejected,
			IsAvailable: false,
			Hashes:      []FileHash{{Algorithm: SHA1, Hash: "abc"}},
			SortableGameVersions: []SortableGameVersion{
				{GameVersionName: "1.20.1"},
			},
		},
	}

	filtered := filterCurseforgeFiles(files, models.FetchOptions{
		AllowedReleaseTypes: []models.ReleaseType{models.Release},
	}, "1.20.1")

	assert.Empty(t, filtered)
}

func TestFileHasVersionMismatch(t *testing.T) {
	file := File{
		SortableGameVersions: []SortableGameVersion{
			{GameVersionName: "1.19.4"},
		},
	}

	assert.False(t, fileHasVersion(file, "1.20.1"))
}

func TestCurseforgeReleaseTypeMappings(t *testing.T) {
	releaseType, ok := curseforgeReleaseType(Release)
	assert.True(t, ok)
	assert.Equal(t, models.Release, releaseType)

	releaseType, ok = curseforgeReleaseType(Beta)
	assert.True(t, ok)
	assert.Equal(t, models.Beta, releaseType)

	releaseType, ok = curseforgeReleaseType(Alpha)
	assert.True(t, ok)
	assert.Equal(t, models.Alpha, releaseType)
}

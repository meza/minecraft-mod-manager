package curseforge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	stdErrors "errors"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/testutil"
	pkgErrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetFilesForProject(t *testing.T) {
	// Define the mock response JSON
	mockResponse := `{
  "data": [
    {
      "id": 1,
      "gameId": 1,
      "modId": 1001,
      "isAvailable": true,
      "displayName": "File 1",
      "fileName": "file1.zip",
      "releaseType": 1,
      "fileStatus": 4,
      "hashes": [
        {
          "algo": 1,
          "value": "d41d8cd98f00b204e9800998ecf8427e"
        }
      ],
      "fileDate": "2023-10-01T12:00:00Z",
      "fileLength": 1024,
      "downloadCount": 100,
      "fileSizeOnDisk": 2048,
      "downloadUrl": "https://example.com/file1.zip",
      "gameVersions": ["1.0.0"],
      "sortableGameVersions": [
        {
          "gameVersionName": "1.0.0",
          "gameVersion": "1.0.0",
          "gameVersionPadded": "0001.0000.0000",
          "gameVersionReleaseDate": "2023-10-01T12:00:00Z",
          "gameVersionTypeId": 1
        }
      ],
      "dependencies": [
        {
          "modId": 2001,
          "type": 3
        }
      ],
      "fingerprint": 1234567890
    },
    {
      "id": 2,
      "gameId": 1,
      "modId": 1001,
      "isAvailable": true,
      "displayName": "File 2",
      "fileName": "file2.zip",
      "releaseType": 2,
      "fileStatus": 4,
      "hashes": [
        {
          "algo": 2,
          "value": "e99a18c428cb38d5f260853678922e03"
        }
      ],
      "fileDate": "2023-10-02T12:00:00Z",
      "fileLength": 2048,
      "downloadCount": 200,
      "fileSizeOnDisk": 4096,
      "downloadUrl": "https://example.com/file2.zip",
      "gameVersions": ["1.1.0"],
      "sortableGameVersions": [
        {
          "gameVersionName": "1.1.0",
          "gameVersion": "1.1.0",
          "gameVersionPadded": "0001.0001.0000",
          "gameVersionReleaseDate": "2023-10-02T12:00:00Z",
          "gameVersionTypeId": 1
        }
      ],
      "dependencies": [
        {
          "modId": 2002,
          "type": 2
        }
      ],
      "fingerprint": 1234567891
    }
  ],
  "pagination": {
    "index": 0,
    "pageSize": 2,
    "resultCount": 2,
    "totalCount": 2
  }
}`

	t.Setenv("CURSEFORGE_API_KEY", "mock_curseforge_api_key")

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/mods/12345/files" {
			t.Errorf("Expected path '/v1/mods/12345/files', got '%s'", r.URL.Path)
		}

		if r.Header.Get("x-api-key") != "mock_curseforge_api_key" {
			t.Errorf("Expected x-api-key header to be 'mock_curseforge_api_key', got '%s'", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, mockResponse)
	}))
	defer mockServer.Close()

	// Call the function
	files, err := GetFilesForProject(context.Background(), 12345, NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, files)
	assert.Equal(t, 2, len(files))

	assert.Equal(t, 1, files[0].ID)
	assert.Equal(t, 1, files[0].GameID)
	assert.Equal(t, 1001, files[0].ProjectID)
	assert.True(t, files[0].IsAvailable)
	assert.Equal(t, "File 1", files[0].DisplayName)
	assert.Equal(t, "file1.zip", files[0].FileName)
	assert.Equal(t, Release, files[0].ReleaseType)
	assert.Equal(t, Approved, files[0].FileStatus)
	assert.Equal(t, 1, len(files[0].Hashes))
	assert.Equal(t, SHA1, files[0].Hashes[0].Algorithm)
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", files[0].Hashes[0].Hash)

	fileDate1, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, fileDate1, files[0].FileDate)

	assert.Equal(t, 1024, files[0].FileLength)
	assert.Equal(t, 100, files[0].DownloadCount)
	assert.Equal(t, 2048, files[0].FileSizeOnDisk)
	assert.Equal(t, "https://example.com/file1.zip", files[0].DownloadURL)
	assert.Equal(t, 1, len(files[0].GameVersions))
	assert.Equal(t, "1.0.0", files[0].GameVersions[0])
	assert.Equal(t, 1, len(files[0].SortableGameVersions))
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0000.0000", files[0].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate1, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate1, files[0].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[0].SortableGameVersions[0].GameVersionTypeID)
	assert.Equal(t, 1, len(files[0].Dependencies))
	assert.Equal(t, 2001, files[0].Dependencies[0].ProjectID)
	assert.Equal(t, RequiredDependency, files[0].Dependencies[0].Type)
	assert.Equal(t, 1234567890, files[0].Fingerprint)

	assert.Equal(t, 2, files[1].ID)
	assert.Equal(t, 1, files[1].GameID)
	assert.Equal(t, 1001, files[1].ProjectID)
	assert.True(t, files[1].IsAvailable)
	assert.Equal(t, "File 2", files[1].DisplayName)
	assert.Equal(t, "file2.zip", files[1].FileName)
	assert.Equal(t, Beta, files[1].ReleaseType)
	assert.Equal(t, Approved, files[1].FileStatus)
	assert.Equal(t, 1, len(files[1].Hashes))
	assert.Equal(t, MD5, files[1].Hashes[0].Algorithm)
	assert.Equal(t, "e99a18c428cb38d5f260853678922e03", files[1].Hashes[0].Hash)

	fileDate2, err := time.Parse(time.RFC3339, "2023-10-02T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, fileDate2, files[1].FileDate)

	assert.Equal(t, 2048, files[1].FileLength)
	assert.Equal(t, 200, files[1].DownloadCount)
	assert.Equal(t, 4096, files[1].FileSizeOnDisk)
	assert.Equal(t, "https://example.com/file2.zip", files[1].DownloadURL)
	assert.Equal(t, 1, len(files[1].GameVersions))
	assert.Equal(t, "1.1.0", files[1].GameVersions[0])
	assert.Equal(t, 1, len(files[1].SortableGameVersions))
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0001.0000", files[1].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate2, err := time.Parse(time.RFC3339, "2023-10-02T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate2, files[1].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[1].SortableGameVersions[0].GameVersionTypeID)
	assert.Equal(t, 1, len(files[1].Dependencies))
	assert.Equal(t, 2002, files[1].Dependencies[0].ProjectID)
	assert.Equal(t, OptionalDependency, files[1].Dependencies[0].Type)
	assert.Equal(t, 1234567891, files[1].Fingerprint)
}

func TestGetFilesForProjectWithPagination(t *testing.T) {
	// Define the mock response JSON with pagination
	mockResponsePage1 := `{
        "data": [
            {
                "id": 1,
                "gameId": 1,
                "modId": 1001,
                "isAvailable": true,
                "displayName": "File 1",
                "fileName": "file1.zip",
                "releaseType": 1,
                "fileStatus": 4,
                "hashes": [
                    {
                        "algo": 1,
                        "value": "d41d8cd98f00b204e9800998ecf8427e"
                    }
                ],
                "fileDate": "2023-10-01T12:00:00Z",
                "fileLength": 1024,
                "downloadCount": 100,
                "fileSizeOnDisk": 2048,
                "downloadUrl": "https://example.com/file1.zip",
                "gameVersions": ["1.0.0"],
                "sortableGameVersions": [
                    {
                        "gameVersionName": "1.0.0",
                        "gameVersion": "1.0.0",
                        "gameVersionPadded": "0001.0000.0000",
                        "gameVersionReleaseDate": "2023-10-01T12:00:00Z",
                        "gameVersionTypeId": 1
                    }
                ],
                "dependencies": [
                    {
                        "modId": 2001,
                        "type": 3
                    }
                ],
                "fingerprint": 1234567890
            }
        ],
        "pagination": {
            "index": 0,
            "pageSize": 1,
            "resultCount": 1,
            "totalCount": 2
        }
    }`

	mockResponsePage2 := `{
        "data": [
            {
                "id": 2,
                "gameId": 1,
                "modId": 1001,
                "isAvailable": true,
                "displayName": "File 2",
                "fileName": "file2.zip",
                "releaseType": 2,
                "fileStatus": 4,
                "hashes": [
                    {
                        "algo": 2,
                        "value": "e99a18c428cb38d5f260853678922e03"
                    }
                ],
                "fileDate": "2023-10-02T12:00:00Z",
                "fileLength": 2048,
                "downloadCount": 200,
                "fileSizeOnDisk": 4096,
                "downloadUrl": "https://example.com/file2.zip",
                "gameVersions": ["1.1.0"],
                "sortableGameVersions": [
                    {
                        "gameVersionName": "1.1.0",
                        "gameVersion": "1.1.0",
                        "gameVersionPadded": "0001.0001.0000",
                        "gameVersionReleaseDate": "2023-10-02T12:00:00Z",
                        "gameVersionTypeId": 1
                    }
                ],
                "dependencies": [
                    {
                        "modId": 2002,
                        "type": 2
                    }
                ],
                "fingerprint": 1234567891
            }
        ],
        "pagination": {
            "index": 1,
            "pageSize": 1,
            "resultCount": 1,
            "totalCount": 2
        }
    }`

	t.Setenv("CURSEFORGE_API_KEY", "mock_curseforge_api_key")

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/mods/12345/files" && r.URL.Query().Get("index") == "0" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			writeStringResponse(t, w, mockResponsePage1)
		} else if r.URL.Path == "/v1/mods/12345/files" && r.URL.Query().Get("index") == "1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			writeStringResponse(t, w, mockResponsePage2)
		} else {
			t.Errorf("Unexpected path or query: %s", r.URL.Path)
		}
	}))
	defer mockServer.Close()

	// Call the function
	files, err := GetFilesForProject(context.Background(), 12345, NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, files)
	assert.Equal(t, 2, len(files))

	// Assertions for the first file
	assert.Equal(t, 1, files[0].ID)
	assert.Equal(t, 1, files[0].GameID)
	assert.Equal(t, 1001, files[0].ProjectID)
	assert.True(t, files[0].IsAvailable)
	assert.Equal(t, "File 1", files[0].DisplayName)
	assert.Equal(t, "file1.zip", files[0].FileName)
	assert.Equal(t, Release, files[0].ReleaseType)
	assert.Equal(t, Approved, files[0].FileStatus)
	assert.Equal(t, 1, len(files[0].Hashes))
	assert.Equal(t, SHA1, files[0].Hashes[0].Algorithm)
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", files[0].Hashes[0].Hash)

	fileDate1, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, fileDate1, files[0].FileDate)

	assert.Equal(t, 1024, files[0].FileLength)
	assert.Equal(t, 100, files[0].DownloadCount)
	assert.Equal(t, 2048, files[0].FileSizeOnDisk)
	assert.Equal(t, "https://example.com/file1.zip", files[0].DownloadURL)
	assert.Equal(t, 1, len(files[0].GameVersions))
	assert.Equal(t, "1.0.0", files[0].GameVersions[0])
	assert.Equal(t, 1, len(files[0].SortableGameVersions))
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0000.0000", files[0].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate1, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate1, files[0].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[0].SortableGameVersions[0].GameVersionTypeID)
	assert.Equal(t, 1, len(files[0].Dependencies))
	assert.Equal(t, 2001, files[0].Dependencies[0].ProjectID)
	assert.Equal(t, RequiredDependency, files[0].Dependencies[0].Type)
	assert.Equal(t, 1234567890, files[0].Fingerprint)

	// Assertions for the second file
	assert.Equal(t, 2, files[1].ID)
	assert.Equal(t, 1, files[1].GameID)
	assert.Equal(t, 1001, files[1].ProjectID)
	assert.True(t, files[1].IsAvailable)
	assert.Equal(t, "File 2", files[1].DisplayName)
	assert.Equal(t, "file2.zip", files[1].FileName)
	assert.Equal(t, Beta, files[1].ReleaseType)
	assert.Equal(t, Approved, files[1].FileStatus)
	assert.Equal(t, 1, len(files[1].Hashes))
	assert.Equal(t, MD5, files[1].Hashes[0].Algorithm)
	assert.Equal(t, "e99a18c428cb38d5f260853678922e03", files[1].Hashes[0].Hash)

	fileDate2, err := time.Parse(time.RFC3339, "2023-10-02T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, fileDate2, files[1].FileDate)

	assert.Equal(t, 2048, files[1].FileLength)
	assert.Equal(t, 200, files[1].DownloadCount)
	assert.Equal(t, 4096, files[1].FileSizeOnDisk)
	assert.Equal(t, "https://example.com/file2.zip", files[1].DownloadURL)
	assert.Equal(t, 1, len(files[1].GameVersions))
	assert.Equal(t, "1.1.0", files[1].GameVersions[0])
	assert.Equal(t, 1, len(files[1].SortableGameVersions))
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0001.0000", files[1].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate2, err := time.Parse(time.RFC3339, "2023-10-02T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate2, files[1].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[1].SortableGameVersions[0].GameVersionTypeID)
	assert.Equal(t, 1, len(files[1].Dependencies))
	assert.Equal(t, 2002, files[1].Dependencies[0].ProjectID)
	assert.Equal(t, OptionalDependency, files[1].Dependencies[0].Type)
	assert.Equal(t, 1234567891, files[1].Fingerprint)
}

func TestGetFilesForProjectWhenProjectNotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	// Call the function
	project, err := GetFilesForProject(context.Background(), 12345, NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, &globalerrors.ProjectNotFoundError{
		ProjectID: "12345",
		Platform:  models.CURSEFORGE,
	})
	assert.Nil(t, project)
}

func TestGetFilesForProjectWhenProjectApiUnknownStatus(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer mockServer.Close()

	// Call the function
	project, err := GetFilesForProject(context.Background(), 12345, NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "unexpected status code: 418", pkgErrors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetFilesForProjectWhenProjectApiCorruptBody(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, `{`)
	}))
	defer mockServer.Close()

	// Call the function
	project, err := GetFilesForProject(context.Background(), 12345, NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "failed to decode response body: unexpected EOF", pkgErrors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetFilesForProjectWhenApiCallFails(t *testing.T) {
	// Call the function
	project, err := GetFilesForProject(context.Background(), 123456, NewClient(errorDoer{err: pkgErrors.New("request failed")}))

	// Assertions
	//assert.Error(t, err)
	assert.ErrorIs(t, err, &globalerrors.ProjectAPIError{
		ProjectID: "123456",
		Platform:  models.CURSEFORGE,
	})
	assert.Equal(t, "request failed", pkgErrors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetFilesForProjectWhenApiCallTimesOut(t *testing.T) {
	project, err := GetFilesForProject(context.Background(), 123457, NewClient(errorDoer{err: context.DeadlineExceeded}))

	assert.Error(t, err)
	var timeoutErr *httpclient.TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
	assert.Nil(t, project)
}

func TestGetFingerprintsMatchesWithOneExactMatch(t *testing.T) {
	mockResponse := `{
		"data": {
			"exactMatches": [
				{
					"id": 110,
					"file": {
						"id": 110,
            "modId": 111,
						"displayName": "string",
						"fileName": "string",
						"fileDate": "2019-08-24T14:15:22Z",
						"fileFingerprint": 1234
					}
				}
			],
			"unmatchedFingerprints": []
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/fingerprints/432" {
			t.Errorf("Expected path '/v1/fingerprints/432', got '%s'", r.URL.Path)
		}

		if r.Header.Get("x-api-key") != "mock_curseforge_api_key" {
			t.Errorf("Expected x-api-key header to be 'mock_curseforge_api_key', got '%s'", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, mockResponse)
	}))
	defer mockServer.Close()

	t.Setenv("CURSEFORGE_API_KEY", "mock_curseforge_api_key")
	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	fingerprints := []int{1234}

	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.NoError(t, err)
	assert.Len(t, result.Matches, 1)
	assert.Len(t, result.Unmatched, 0)
	assert.Equal(t, 110, result.Matches[0].ID)
	assert.Equal(t, 111, result.Matches[0].ProjectID)
	assert.Equal(t, "string", result.Matches[0].DisplayName)
	assert.Equal(t, 1234, result.Matches[0].Fingerprint)
}

func TestGetFingerprintsMatchesWithMultipleExactMatches(t *testing.T) {
	mockResponse := `{
		"data": {
			"exactMatches": [
				{
					"id": 0,
					"file": {
						"id": 20,
            "modId": 200,
						"displayName": "string1",
						"fileName": "string1",
						"fileDate": "2019-08-24T14:15:22Z",
						"fileFingerprint": 123456
					}
				},
				{
					"id": 1,
					"file": {
						"id": 21,
            "modId": 210,
						"displayName": "string2",
						"fileName": "string2",
						"fileDate": "2019-08-24T14:15:22Z",
						"fileFingerprint": 1234567
					}
				}
			],
			"unmatchedFingerprints": []
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, mockResponse)
	}))
	defer mockServer.Close()

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	fingerprints := []int{123456, 1234567}

	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.NoError(t, err)
	assert.Len(t, result.Matches, 2)
	assert.Len(t, result.Unmatched, 0)
	assert.Equal(t, 20, result.Matches[0].ID)
	assert.Equal(t, 200, result.Matches[0].ProjectID)
	assert.Equal(t, 21, result.Matches[1].ID)
	assert.Equal(t, 210, result.Matches[1].ProjectID)
}

func TestGetFingerprintsMatchesWithNoExactMatches(t *testing.T) {
	mockResponse := `{
		"data": {
			"exactMatches": [],
			"unmatchedFingerprints": [0, 1]
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, mockResponse)
	}))
	defer mockServer.Close()

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	fingerprints := []int{0, 1}

	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.NoError(t, err)
	assert.Len(t, result.Matches, 0)
	assert.Len(t, result.Unmatched, 2)
	assert.Equal(t, 0, result.Unmatched[0])
	assert.Equal(t, 1, result.Unmatched[1])
}

func TestGetFingerprintsMatches_AllowsObjectFingerprintFields(t *testing.T) {
	mockResponse := `{
		"data": {
			"exactMatches": [
				{
					"id": 0,
					"file": {
						"id": 20,
            "modId": 200,
						"displayName": "string1",
						"fileName": "string1",
						"fileDate": "2019-08-24T14:15:22Z",
						"fileFingerprint": 123456
					}
				}
			],
			"exactFingerprints": {"123456": true},
			"partialMatches": [],
			"partialMatchFingerprints": {"123456": true},
			"unmatchedFingerprints": null,
			"installedFingerprints": {"123456": true}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, mockResponse)
	}))
	defer mockServer.Close()

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	fingerprints := []int{123456}

	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.NoError(t, err)
	assert.Len(t, result.Matches, 1)
	assert.Equal(t, 20, result.Matches[0].ID)
	assert.Equal(t, 200, result.Matches[0].ProjectID)
	assert.Len(t, result.Unmatched, 0)
}

func TestDecodeUnmatchedFingerprints_AllowsMapKeyAny(t *testing.T) {
	raw := json.RawMessage(`{"123": {}, "456": 1}`)
	values, err := decodeUnmatchedFingerprints(raw)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []int{123, 456}, values)
}

func TestDecodeUnmatchedFingerprints_AllowsList(t *testing.T) {
	raw := json.RawMessage(`[1,2,3]`)
	values, err := decodeUnmatchedFingerprints(raw)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, values)
}

func TestDecodeUnmatchedFingerprints_AllowsNullAndEmpty(t *testing.T) {
	values, err := decodeUnmatchedFingerprints(nil)
	assert.NoError(t, err)
	assert.Len(t, values, 0)

	values, err = decodeUnmatchedFingerprints(json.RawMessage(`null`))
	assert.NoError(t, err)
	assert.Len(t, values, 0)
}

func TestDecodeUnmatchedFingerprints_InvalidMapKeyErrors(t *testing.T) {
	raw := json.RawMessage(`{"abc": true}`)
	values, err := decodeUnmatchedFingerprints(raw)
	assert.NoError(t, err)
	assert.Len(t, values, 0)
}

func TestDecodeUnmatchedFingerprints_UnsupportedTypeErrors(t *testing.T) {
	raw := json.RawMessage(`"nope"`)
	_, err := decodeUnmatchedFingerprints(raw)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "unsupported type")
}

func TestGetFingerprintsMatches_UnmatchedFingerprintsMapIsTolerated(t *testing.T) {
	mockResponse := `{
		"data": {
			"exactMatches": [],
			"partialMatches": [],
			"unmatchedFingerprints": {"123": true}
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, mockResponse)
	}))
	defer mockServer.Close()

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	fingerprints := []int{123456}

	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Matches, 0)
	assert.Equal(t, []int{123}, result.Unmatched)
}

func TestGetFingerprintsMatchesWithApiFailure(t *testing.T) {
	fingerprints := []int{0, 1}

	client := NewClient(errorDoer{err: pkgErrors.New("request failed")})
	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.ErrorContains(t, err, "request failed")
	assert.Nil(t, result)
}

func TestGetFingerprintsMatchesWithApiTimeout(t *testing.T) {
	fingerprints := []int{0, 1}

	client := NewClient(errorDoer{err: context.DeadlineExceeded})
	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.Error(t, err)
	var timeoutErr *httpclient.TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
	assert.Nil(t, result)
}

func TestGetFingerprintsMatchesWithNotFound(t *testing.T) {

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()
	fingerprints := []int{0, 1}

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.ErrorContains(t, err, "unexpected status code: 404")
	assert.Nil(t, result)
}

func TestGetFingerprintsMatchesWithUnexpectedStatusReturnsApiError(t *testing.T) {

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		writeStringResponse(t, w, `{"error":"forbidden"}`)
	}))
	defer mockServer.Close()
	fingerprints := []int{0, 1}

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.ErrorContains(t, err, "unexpected status code: 403")
	assert.Nil(t, result)
}

func TestGetFingerprintsMatches_UnmatchedFingerprintsUnsupportedTypeErrors(t *testing.T) {
	mockResponse := `{
		"data": {
			"exactMatches": [],
			"partialMatches": [],
			"unmatchedFingerprints": 123
		}
	}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, mockResponse)
	}))
	defer mockServer.Close()
	fingerprints := []int{0, 1}

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode unmatchedFingerprints")
	assert.Nil(t, result)
}

func TestGetFingerprintsMatchesWithCorruptedBody(t *testing.T) {

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, `{`)
	}))
	defer mockServer.Close()
	fingerprints := []int{0, 1}

	client := NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client()))
	result, err := GetFingerprintsMatches(context.Background(), fingerprints, client)
	assert.ErrorContains(t, err, "unexpected EOF")
	assert.Nil(t, result)
}

func TestGetFingerprintsMatchesReturnsErrorOnMarshalFailure(t *testing.T) {
	originalMarshal := marshalJSON
	marshalJSON = func(any) ([]byte, error) {
		return nil, stdErrors.New("marshal failed")
	}
	t.Cleanup(func() {
		marshalJSON = originalMarshal
	})

	result, err := GetFingerprintsMatches(context.Background(), []int{1}, NewClient(errorDoer{}))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetFingerprintsMatchesReturnsErrorOnRequestBuildFailure(t *testing.T) {
	originalRequest := newRequestWithContext
	newRequestWithContext = func(context.Context, string, string, io.Reader) (*http.Request, error) {
		return nil, stdErrors.New("request failed")
	}
	t.Cleanup(func() {
		newRequestWithContext = originalRequest
	})

	result, err := GetFingerprintsMatches(context.Background(), []int{1}, NewClient(errorDoer{}))
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetFingerprintsMatchesReturnsErrorOnResponseCloseFailure(t *testing.T) {
	closeErr := stdErrors.New("close failed")
	body := newCloseErrorBody(`{"data":{"exactMatches":[],"partialMatches":[],"unmatchedFingerprints":[]}}`, closeErr)
	client := NewClient(responseDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
			Header:     make(http.Header),
		},
	})

	result, err := GetFingerprintsMatches(context.Background(), []int{1}, client)
	assert.ErrorIs(t, err, closeErr)
	assert.NotNil(t, result)
}

func TestGetPaginatedFilesForProjectReturnsErrorOnRequestBuildFailure(t *testing.T) {
	originalRequest := newRequestWithContext
	newRequestWithContext = func(context.Context, string, string, io.Reader) (*http.Request, error) {
		return nil, stdErrors.New("request failed")
	}
	t.Cleanup(func() {
		newRequestWithContext = originalRequest
	})

	files, err := getPaginatedFilesForProject(context.Background(), 12345, errorDoer{}, 0)
	assert.Error(t, err)
	assert.Nil(t, files)
}

func TestGetPaginatedFilesForProjectReturnsErrorOnResponseCloseFailure(t *testing.T) {
	closeErr := stdErrors.New("close failed")
	body := newCloseErrorBody(`{"data":[],"pagination":{"index":0,"pageSize":50,"resultCount":0,"totalCount":0}}`, closeErr)
	client := responseDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
			Header:     make(http.Header),
		},
	}

	files, err := getPaginatedFilesForProject(context.Background(), 12345, client, 0)
	assert.ErrorIs(t, err, closeErr)
	assert.NotNil(t, files)
}

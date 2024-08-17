package curseforge

import (
	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
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

	err := os.Setenv("CURSEFORGE_API_KEY", "mock_curseforge_api_key")
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
		return
	}

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mods/12345/files" {
			t.Errorf("Expected path '/mods/12345/files', got '%s'", r.URL.Path)
		}

		if r.Header.Get("x-api-key") != "mock_curseforge_api_key" {
			t.Errorf("Expected x-api-key header to be 'mock_curseforge_api_key', got '%s'", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err1 := os.Setenv("CURSEFORGE_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() {
		os.Unsetenv("CURSEFORGE_API_URL")
		os.Unsetenv("CURSEFORGE_API_KEY")
	}()

	// Call the function
	files, err := GetFilesForProject(12345, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, files)
	assert.Equal(t, 2, len(files))

	assert.Equal(t, 1, files[0].Id)
	assert.Equal(t, 1, files[0].GameId)
	assert.Equal(t, 1001, files[0].ProjectId)
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
	assert.Equal(t, "https://example.com/file1.zip", files[0].DownloadUrl)
	assert.Equal(t, 1, len(files[0].GameVersions))
	assert.Equal(t, "1.0.0", files[0].GameVersions[0])
	assert.Equal(t, 1, len(files[0].SortableGameVersions))
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0000.0000", files[0].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate1, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate1, files[0].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[0].SortableGameVersions[0].GameVersionTypeId)
	assert.Equal(t, 1, len(files[0].Dependencies))
	assert.Equal(t, 2001, files[0].Dependencies[0].ProjectId)
	assert.Equal(t, RequiredDependency, files[0].Dependencies[0].Type)
	assert.Equal(t, 1234567890, files[0].Fingerprint)

	assert.Equal(t, 2, files[1].Id)
	assert.Equal(t, 1, files[1].GameId)
	assert.Equal(t, 1001, files[1].ProjectId)
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
	assert.Equal(t, "https://example.com/file2.zip", files[1].DownloadUrl)
	assert.Equal(t, 1, len(files[1].GameVersions))
	assert.Equal(t, "1.1.0", files[1].GameVersions[0])
	assert.Equal(t, 1, len(files[1].SortableGameVersions))
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0001.0000", files[1].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate2, err := time.Parse(time.RFC3339, "2023-10-02T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate2, files[1].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[1].SortableGameVersions[0].GameVersionTypeId)
	assert.Equal(t, 1, len(files[1].Dependencies))
	assert.Equal(t, 2002, files[1].Dependencies[0].ProjectId)
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

	err := os.Setenv("CURSEFORGE_API_KEY", "mock_curseforge_api_key")
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
		return
	}

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mods/12345/files" && r.URL.Query().Get("index") == "0" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponsePage1))
		} else if r.URL.Path == "/mods/12345/files" && r.URL.Query().Get("index") == "1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponsePage2))
		} else {
			t.Errorf("Unexpected path or query: %s", r.URL.Path)
		}
	}))
	defer mockServer.Close()

	err1 := os.Setenv("CURSEFORGE_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() {
		os.Unsetenv("CURSEFORGE_API_URL")
		os.Unsetenv("CURSEFORGE_API_KEY")
	}()

	// Call the function
	files, err := GetFilesForProject(12345, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, files)
	assert.Equal(t, 2, len(files))

	// Assertions for the first file
	assert.Equal(t, 1, files[0].Id)
	assert.Equal(t, 1, files[0].GameId)
	assert.Equal(t, 1001, files[0].ProjectId)
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
	assert.Equal(t, "https://example.com/file1.zip", files[0].DownloadUrl)
	assert.Equal(t, 1, len(files[0].GameVersions))
	assert.Equal(t, "1.0.0", files[0].GameVersions[0])
	assert.Equal(t, 1, len(files[0].SortableGameVersions))
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.0.0", files[0].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0000.0000", files[0].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate1, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate1, files[0].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[0].SortableGameVersions[0].GameVersionTypeId)
	assert.Equal(t, 1, len(files[0].Dependencies))
	assert.Equal(t, 2001, files[0].Dependencies[0].ProjectId)
	assert.Equal(t, RequiredDependency, files[0].Dependencies[0].Type)
	assert.Equal(t, 1234567890, files[0].Fingerprint)

	// Assertions for the second file
	assert.Equal(t, 2, files[1].Id)
	assert.Equal(t, 1, files[1].GameId)
	assert.Equal(t, 1001, files[1].ProjectId)
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
	assert.Equal(t, "https://example.com/file2.zip", files[1].DownloadUrl)
	assert.Equal(t, 1, len(files[1].GameVersions))
	assert.Equal(t, "1.1.0", files[1].GameVersions[0])
	assert.Equal(t, 1, len(files[1].SortableGameVersions))
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersionName)
	assert.Equal(t, "1.1.0", files[1].SortableGameVersions[0].GameVersion)
	assert.Equal(t, "0001.0001.0000", files[1].SortableGameVersions[0].GameVersionPadded)

	sortableGameVersionDate2, err := time.Parse(time.RFC3339, "2023-10-02T12:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, sortableGameVersionDate2, files[1].SortableGameVersions[0].GameVersionReleaseDate)

	assert.Equal(t, 1, files[1].SortableGameVersions[0].GameVersionTypeId)
	assert.Equal(t, 1, len(files[1].Dependencies))
	assert.Equal(t, 2002, files[1].Dependencies[0].ProjectId)
	assert.Equal(t, OptionalDependency, files[1].Dependencies[0].Type)
	assert.Equal(t, 1234567891, files[1].Fingerprint)
}

func TestGetFilesForProjectWhenProjectNotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	err1 := os.Setenv("CURSEFORGE_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("CURSEFORGE_API_URL") }()

	// Call the function
	project, err := GetFilesForProject(12345, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, &globalErrors.ProjectNotFoundError{
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

	err1 := os.Setenv("CURSEFORGE_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("CURSEFORGE_API_URL") }()

	// Call the function
	project, err := GetFilesForProject(12345, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "unexpected status code: 418", errors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetFilesForProjectWhenApiCallFails(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	err1 := os.Setenv("CURSEFORGE_API_URL", "invalid_url")
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("CURSEFORGE_API_URL") }()

	// Call the function
	project, err := GetFilesForProject(123456, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	//assert.Error(t, err)
	assert.ErrorIs(t, err, &globalErrors.ProjectApiError{
		ProjectID: "123456",
		Platform:  models.CURSEFORGE,
	})
	assert.Equal(t, "Get \"invalid_url/mods/123456/files?index=0\": unsupported protocol scheme \"\"", errors.Unwrap(err).Error())
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
		if r.URL.Path != "/fingerprints/432" {
			t.Errorf("Expected path '/fingerprints/432', got '%s'", r.URL.Path)
		}

		if r.Header.Get("x-api-key") != "mock_curseforge_api_key" {
			t.Errorf("Expected x-api-key header to be 'mock_curseforge_api_key', got '%s'", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err := os.Setenv("CURSEFORGE_API_URL", mockServer.URL)
	assert.NoError(t, err)
	defer os.Unsetenv("CURSEFORGE_API_URL")

	err = os.Setenv("CURSEFORGE_API_KEY", "mock_curseforge_api_key")
	assert.NoError(t, err)
	defer os.Unsetenv("CURSEFORGE_API_KEY")

	client := &Client{client: mockServer.Client()}
	fingerprints := []int{1234}

	result, err := GetFingerprintsMatches(fingerprints, client)
	assert.NoError(t, err)
	assert.Len(t, result.Matches, 1)
	assert.Len(t, result.Unmatched, 0)
	assert.Equal(t, 110, result.Matches[0].Id)
	assert.Equal(t, 111, result.Matches[0].ProjectId)
	assert.Equal(t, "string", result.Matches[0].DisplayName)
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
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err := os.Setenv("CURSEFORGE_API_URL", mockServer.URL)
	assert.NoError(t, err)
	defer os.Unsetenv("CURSEFORGE_API_URL")

	client := &Client{client: mockServer.Client()}
	fingerprints := []int{123456, 1234567}

	result, err := GetFingerprintsMatches(fingerprints, client)
	assert.NoError(t, err)
	assert.Len(t, result.Matches, 2)
	assert.Len(t, result.Unmatched, 0)
	assert.Equal(t, 20, result.Matches[0].Id)
	assert.Equal(t, 200, result.Matches[0].ProjectId)
	assert.Equal(t, 21, result.Matches[1].Id)
	assert.Equal(t, 210, result.Matches[1].ProjectId)
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
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err := os.Setenv("CURSEFORGE_API_URL", mockServer.URL)
	assert.NoError(t, err)
	defer os.Unsetenv("CURSEFORGE_API_URL")

	client := &Client{client: mockServer.Client()}
	fingerprints := []int{0, 1}

	result, err := GetFingerprintsMatches(fingerprints, client)
	assert.NoError(t, err)
	assert.Len(t, result.Matches, 0)
	assert.Len(t, result.Unmatched, 2)
	assert.Equal(t, 0, result.Unmatched[0])
	assert.Equal(t, 1, result.Unmatched[1])
}

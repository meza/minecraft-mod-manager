package curseforge

import (
	"context"
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

func TestGetProject(t *testing.T) {
	// Define the mock response JSON
	mockResponse := `{
  "data": {
    "id": 12345,
    "gameId": 67890,
    "name": "Example Project",
    "slug": "example-project",
    "links": {
      "websiteUrl": "https://example.com",
      "wikiUrl": "https://example.com/wiki",
      "issuesUrl": "https://example.com/issues",
      "sourceUrl": "https://example.com/source"
    },
    "summary": "This is an example project.",
    "downloadCount": 1000,
    "primaryCategoryId": 1,
    "categories": [
      {
        "id": 1,
        "gameId": 67890,
        "name": "Category 1",
        "slug": "category-1",
        "url": "https://example.com/category-1",
        "iconUrl": "https://example.com/icon1.png",
        "dateModified": "2023-10-01T12:00:00Z",
        "isClass": false,
        "classId": 0,
        "parentCategoryId": 0,
        "displayIndex": 1
      }
    ],
    "classId": 0,
    "authors": [
      {
        "id": 1,
        "name": "Author 1",
        "url": "https://example.com/author1"
      }
    ],
    "logo": {
      "id": 1,
      "projectId": 12345,
      "url": "https://example.com/logo.png",
      "title": "Example Logo",
      "description": "This is an example logo.",
      "thumbnailUrl": "https://example.com/logo_thumbnail.png"
    },
    "mainFileId": 1,
    "latestFiles": [
      {
        "id": 1,
        "gameId": 67890,
        "projectId": 12345,
        "isAvailable": true,
        "displayName": "Example File",
        "fileName": "example_file.zip",
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
        "downloadCount": 500,
        "fileSizeOnDisk": 2048,
        "downloadUrl": "https://example.com/download/example_file.zip",
        "gameVersions": [
          "1.0.0"
        ],
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
            "projectId": 2,
            "type": 3
          }
        ],
        "fingerprint": 1234567890
      }
    ],
    "dateCreated": "2023-10-01T12:00:00Z",
    "dateModified": "2023-10-01T12:00:00Z",
    "dateReleased": "2023-10-01T12:00:00Z",
    "gamePopularityRank": 1,
    "thumbsUpCount": 100,
    "rating": 5
  }
}`

	t.Setenv("CURSEFORGE_API_KEY", "mock_curseforge_api_key")

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/mods/12345" {
			t.Errorf("Expected path '/v1/mods/12345', got '%s'", r.URL.Path)
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
	project, err := GetProject(context.Background(), "12345", NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, 12345, project.ID)
	assert.Equal(t, "Example Project", project.Name)
	assert.Equal(t, "example-project", project.Slug)
	assert.Equal(t, "This is an example project.", project.Summary)
	assert.Equal(t, 1000, project.DownloadCount)
	assert.Equal(t, 1, project.PrimaryCategoryID)
	assert.Equal(t, 1, len(project.Categories))
	assert.Equal(t, "Category 1", project.Categories[0].Name)
	assert.Equal(t, "https://example.com", project.Links.WebsiteURL)
	assert.Equal(t, "https://example.com/wiki", project.Links.WikiURL)
	assert.Equal(t, "https://example.com/issues", project.Links.IssuesURL)
	assert.Equal(t, "https://example.com/source", project.Links.SourceURL)
	assert.Equal(t, 1, len(project.Authors))
	assert.Equal(t, "Author 1", project.Authors[0].Name)
	assert.Equal(t, "https://example.com/author1", project.Authors[0].URL)
	assert.Equal(t, "https://example.com/logo.png", project.Logo.URL)
	assert.Equal(t, "Example Logo", project.Logo.Title)
	assert.Equal(t, "This is an example logo.", project.Logo.Description)
	assert.Equal(t, "https://example.com/logo_thumbnail.png", project.Logo.ThumbnailURL)
	assert.Equal(t, 1, len(project.LatestFiles))
	assert.Equal(t, "Example File", project.LatestFiles[0].DisplayName)
	assert.Equal(t, "example_file.zip", project.LatestFiles[0].FileName)
	assert.Equal(t, Release, project.LatestFiles[0].ReleaseType)
	assert.Equal(t, Approved, project.LatestFiles[0].FileStatus)
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", project.LatestFiles[0].Hashes[0].Hash)
	assert.Equal(t, SHA1, project.LatestFiles[0].Hashes[0].Algorithm)
	assert.Equal(t, "1.0.0", project.LatestFiles[0].GameVersions[0])
	assert.Equal(t, 1234567890, project.LatestFiles[0].Fingerprint)

	dateCreated, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	dateModified, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)
	dateReleased, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	assert.NoError(t, err)

	assert.Equal(t, dateCreated, project.DateCreated)
	assert.Equal(t, dateModified, project.DateModified)
	assert.Equal(t, dateReleased, project.DateReleased)
	assert.Equal(t, 1, project.GamePopularityRank)
	assert.Equal(t, 100, project.ThumbsUpCount)
	assert.Equal(t, 5, project.Rating)
}

func TestGetProjectWhenProjectNotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	// Call the function
	project, err := GetProject(context.Background(), "AABBCCDD", NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, &globalerrors.ProjectNotFoundError{
		ProjectID: "AABBCCDD",
		Platform:  models.CURSEFORGE,
	})
	assert.Nil(t, project)
}

func TestGetProjectWhenProjectApiUnknownStatus(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer mockServer.Close()

	// Call the function
	project, err := GetProject(context.Background(), "AABBCCDD", NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "unexpected status code: 418", pkgErrors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetProjectWhenProjectApiCorruptedBody(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writeStringResponse(t, w, `{"data": {`)
	}))
	defer mockServer.Close()

	// Call the function
	project, err := GetProject(context.Background(), "AABBCCDD", NewClient(testutil.MustNewHostRewriteDoer(mockServer.URL, mockServer.Client())))

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "failed to decode response body: unexpected EOF", pkgErrors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetProjectWhenApiCallFails(t *testing.T) {
	// Call the function
	project, err := GetProject(context.Background(), "AABBCCDDEE", NewClient(errorDoer{err: pkgErrors.New("request failed")}))

	// Assertions
	//assert.Error(t, err)
	assert.ErrorIs(t, err, &globalerrors.ProjectAPIError{
		ProjectID: "AABBCCDDEE",
		Platform:  models.CURSEFORGE,
	})
	assert.Equal(t, "request failed", pkgErrors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetProjectWhenApiCallTimesOut(t *testing.T) {
	project, err := GetProject(context.Background(), "AABBCCDDEE", NewClient(errorDoer{err: context.DeadlineExceeded}))

	assert.Error(t, err)
	var timeoutErr *httpclient.TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
	assert.Nil(t, project)
}

func TestGetProjectReturnsErrorWhenRequestBuildFails(t *testing.T) {
	originalRequest := newRequestWithContext
	newRequestWithContext = func(context.Context, string, string, io.Reader) (*http.Request, error) {
		return nil, stdErrors.New("request failed")
	}
	t.Cleanup(func() {
		newRequestWithContext = originalRequest
	})

	project, err := GetProject(context.Background(), "AABBCCDD", NewClient(errorDoer{err: nil}))
	assert.Error(t, err)
	assert.Nil(t, project)
}

func TestGetProjectReturnsErrorWhenResponseCloseFails(t *testing.T) {
	closeErr := stdErrors.New("close failed")
	body := newCloseErrorBody(`{"data":{"id":12345,"name":"Example"}}`, closeErr)
	client := NewClient(responseDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
			Header:     make(http.Header),
		},
	})

	project, err := GetProject(context.Background(), "12345", client)
	assert.ErrorIs(t, err, closeErr)
	assert.NotNil(t, project)
}

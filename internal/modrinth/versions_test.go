package modrinth

import (
	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/pkg/errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionsForProject_SingleVersion(t *testing.T) {
	mockResponse := `[{
			"name": "Version 1.0.0",
			"version_number": "1.0.0",
			"changelog": "List of changes in this version: ...",
			"dependencies": [{
				"version_id": "IIJJKKLL",
				"project_id": "QQRRSSTT",
				"file_name": "sodium-fabric-mc1.19-0.4.2+build.16.jar",
				"dependency_type": "required"
			}],
			"game_versions": ["1.16.5", "1.17.1"],
			"version_type": "release",
			"loaders": ["fabric", "forge"],
			"featured": true,
			"status": "listed",
			"requested_status": "listed",
			"id": "IIJJKKLL",
			"project_id": "AABBCCDD",
			"author_id": "EEFFGGHH",
			"date_published": "2024-08-07T20:21:13.726918Z",
			"downloads": 0,
			"changelog_url": null,
			"files": [{
				"hashes": {
					"sha512": "93ecf5fe02914fb53d94aa3d28c1fb562e23985f8e4d48b9038422798618761fe208a31ca9b723667a4e05de0d91a3f86bcd8d018f6a686c39550e21b198d96f",
					"sha1": "c84dd4b3580c02b79958a0590afd5783d80ef504"
				},
				"url": "https://cdn.modrinth.com/data/AABBCCDD/versions/1.0.0/my_file.jar",
				"filename": "my_file.jar",
				"primary": false,
				"size": 1097270,
				"file_type": "required-resource-pack"
			}]
		}]`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/project/AABBCCDD/version" {
			t.Errorf("Expected path '/v2/project/AABBCCDD/version', got '%s'", r.URL.Path)
		}

		assert.Equal(t, `["1.16.5","1.17.1"]`, r.URL.Query().Get("game_versions"))
		assert.Equal(t, `["fabric","forge"]`, r.URL.Query().Get("loaders"))

		if r.Header.Get("Authorization") != "test-api-key" {
			t.Errorf("Expected Authorization header to be 'test-api-key', got '%s'", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	_ = os.Setenv("MODRINTH_API_KEY", "test-api-key")
	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	client := &Client{client: mockServer.Client()}
	lookup := &VersionLookup{
		ProjectId:    "AABBCCDD",
		Loaders:      []models.Loader{"fabric", "forge"},
		GameVersions: []string{"1.16.5", "1.17.1"},
	}

	versions, err := GetVersionsForProject(lookup, client)

	assert.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Equal(t, "Version 1.0.0", versions[0].Name)
	assert.Equal(t, "1.0.0", versions[0].VersionNumber)
	assert.Equal(t, "List of changes in this version: ...", versions[0].Changelog)
	assert.Contains(t, versions[0].GameVersions, "1.16.5")
	assert.Contains(t, versions[0].GameVersions, "1.17.1")
	assert.Equal(t, models.Release, versions[0].Type)
	assert.Equal(t, Listed, versions[0].Status)
	assert.Contains(t, versions[0].Loaders, models.FABRIC)
	assert.Contains(t, versions[0].Loaders, models.FORGE)
	assert.Equal(t, "IIJJKKLL", versions[0].VersionId)
	assert.Equal(t, "AABBCCDD", versions[0].ProjectId)

	expectedTime := time.Date(2024, 8, 7, 20, 21, 13, 726918000, time.UTC)

	assert.Equal(t, expectedTime, versions[0].DatePublished)
	assert.Len(t, versions[0].Files, 1)
	assert.Equal(t, "93ecf5fe02914fb53d94aa3d28c1fb562e23985f8e4d48b9038422798618761fe208a31ca9b723667a4e05de0d91a3f86bcd8d018f6a686c39550e21b198d96f", versions[0].Files[0].Hashes.Sha512)
	assert.Equal(t, "c84dd4b3580c02b79958a0590afd5783d80ef504", versions[0].Files[0].Hashes.Sha1)
	assert.Equal(t, "https://cdn.modrinth.com/data/AABBCCDD/versions/1.0.0/my_file.jar", versions[0].Files[0].Url)
	assert.Equal(t, "my_file.jar", versions[0].Files[0].FileName)
	assert.False(t, versions[0].Files[0].Primary)
	assert.Equal(t, int64(1097270), versions[0].Files[0].Size)
}

func TestGetVersionsForProject_MultipleVersions(t *testing.T) {
	mockResponse := `[{
			"name": "Version 1.0.0",
			"version_number": "1.0.0",
			"changelog": "List of changes in this version: ...",
			"dependencies": [{
				"version_id": "IIJJKKLL",
				"project_id": "QQRRSSTT",
				"file_name": "sodium-fabric-mc1.16-0.4.2+build.16.jar",
				"dependency_type": "required"
			}],
			"game_versions": ["1.16.5", "1.17.1"],
			"version_type": "release",
			"loaders": ["fabric", "forge"],
			"featured": true,
			"status": "archived",
			"requested_status": "listed",
			"id": "IIJJKKLL",
			"project_id": "AABBCCDD",
			"author_id": "EEFFGGHH",
			"date_published": "2024-08-08T20:21:13.726918Z",
			"downloads": 0,
			"changelog_url": null,
			"files": [{
				"hashes": {
					"sha512": "93ecf5fe02914fb53d94aa3d28c1fb562e23985f8e4d48b9038422798618761fe208a31ca9b723667a4e05de0d91a3f86bcd8d018f6a686c39550e21b198d96f",
					"sha1": "c84dd4b3580c02b79958a0590afd5783d80ef504"
				},
				"url": "https://cdn.modrinth.com/data/AABBCCDD/versions/1.0.0/my_file.jar",
				"filename": "my_file.jar",
				"primary": false,
				"size": 1097270,
				"file_type": "required-resource-pack"
			}]
		}, {
			"name": "Version 1.1.0",
			"version_number": "1.1.0",
			"changelog": "List of changes in this version: ...",
			"dependencies": [{
				"version_id": "MMNNOOPP",
				"project_id": "QQRRSSTT",
				"file_name": "sodium-fabric-mc1.19-0.4.2+build.16.jar",
				"dependency_type": "optional"
			}],
			"game_versions": ["1.16.5", "1.17.1"],
			"version_type": "release",
			"loaders": ["fabric", "forge"],
			"featured": true,
			"status": "unlisted",
			"requested_status": "listed",
			"id": "MMNNOOPP",
			"project_id": "AABBCCDD",
			"author_id": "EEFFGGHH",
			"date_published": "2024-08-09T20:21:13.726918Z",
			"downloads": 0,
			"changelog_url": null,
			"files": [{
				"hashes": {
					"sha512": "93ecf5fe02914fb53d94aa3d28c1fb562e23985f8e4d48b9038422798618761fe208a31ca9b723667a4e05de0d91a3f86bcd8d018f6a686c39550e21b198d96f",
					"sha1": "c84dd4b3580c02b79958a0590afd5783d80ef504"
				},
				"url": "https://cdn.modrinth.com/data/AABBCCDD/versions/1.1.0/my_file.jar",
				"filename": "my_file.jar",
				"primary": false,
				"size": 1097270,
				"file_type": "required-resource-pack"
			}]
		}]`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	client := &Client{client: mockServer.Client()}
	lookup := &VersionLookup{
		ProjectId:    "AABBCCDD",
		Loaders:      []models.Loader{models.FABRIC, models.FORGE},
		GameVersions: []string{"1.16.5", "1.17.1"},
	}

	versions, err := GetVersionsForProject(lookup, client)
	assert.NoError(t, err)
	assert.Len(t, versions, 2)
	assert.Equal(t, "Version 1.0.0", versions[0].Name)
	assert.Equal(t, "Version 1.1.0", versions[1].Name)
	assert.Equal(t, Archived, versions[0].Status)
	assert.Equal(t, Unlisted, versions[1].Status)
	assert.ObjectsAreEqualValues(versions[0].Dependencies, &VersionDependency{
		VersionId: "IIJJKKLL",
		ProjectId: "QQRRSSTT",
		FileName:  "sodium-fabric-mc1.16-0.4.2+build.16.jar",
		Type:      RequiredDependency,
	})
	assert.ObjectsAreEqualValues(versions[1].Dependencies, &VersionDependency{
		VersionId: "MMNNOOPP",
		ProjectId: "QQRRSSTT",
		FileName:  "sodium-fabric-mc1.19-0.4.2+build.16.jar",
		Type:      OptionalDependency,
	})
}

func TestGetVersionsForProject_NoVersions(t *testing.T) {
	mockResponse := `{"versions": []}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	client := &Client{client: mockServer.Client()}
	lookup := &VersionLookup{
		ProjectId:    "AABBCCDD",
		Loaders:      []models.Loader{models.FABRIC, models.FORGE},
		GameVersions: []string{"1.16.5", "1.17.1"},
	}

	versions, err := GetVersionsForProject(lookup, client)
	assert.NoError(t, err)
	assert.Len(t, versions, 0)
}

func TestGetVersionsForProjectWhenProjectNotFound(t *testing.T) {
	// Define the mock response JSON
	mockResponse := `{
    "error": "not_found",
    "description": "the requested route does not exist"
  }`

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	// Call the function
	lookup := &VersionLookup{
		ProjectId:    "AABBCCD1",
		Loaders:      []models.Loader{models.FORGE},
		GameVersions: []string{"1.19"},
	}
	project, err := GetVersionsForProject(lookup, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, &globalErrors.ProjectNotFoundError{
		ProjectID: "AABBCCD1",
		Platform:  models.MODRINTH,
	})
	assert.Errorf(t, err, "project not found: AABBCCD1")
	assert.Nil(t, project)
}

func TestGetVersionsForProjectWhenProjectApiUnknownStatus(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	lookup := &VersionLookup{
		ProjectId:    "AABBCCD2",
		Loaders:      []models.Loader{models.FABRIC},
		GameVersions: []string{"1.18.1"},
	}
	project, err := GetVersionsForProject(lookup, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "unexpected status code: 418", errors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetVersionsForProjectWhenApiCallFails(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", "invalid_url")
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	lookup := &VersionLookup{
		ProjectId:    "AABBCCD3",
		Loaders:      []models.Loader{models.QUILT},
		GameVersions: []string{"1.21.1"},
	}
	project, err := GetVersionsForProject(lookup, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	//assert.Error(t, err)
	assert.ErrorIs(t, err, &globalErrors.ProjectApiError{
		ProjectID: "AABBCCD3",
		Platform:  models.MODRINTH,
	})
	assert.Equal(t, "Get \"invalid_url/v2/project/AABBCCD3/version?game_versions=%5B%221.21.1%22%5D&loaders=%5B%22quilt%22%5D\": unsupported protocol scheme \"\"", errors.Unwrap(err).Error())
	assert.Nil(t, project)
}

// Hash search

func TestGetVersionForHash_SingleVersion(t *testing.T) {
	mockResponse := `{
			"name": "Version 1.0.0",
			"version_number": "1.0.0",
			"changelog": "List of changes in this version: ...",
			"dependencies": [{
				"version_id": "IIJJKKLL",
				"project_id": "QQRRSSTT",
				"file_name": "sodium-fabric-mc1.19-0.4.2+build.16.jar",
				"dependency_type": "required"
			}],
			"game_versions": ["1.16.5", "1.17.1"],
			"version_type": "release",
			"loaders": ["fabric", "forge"],
			"featured": true,
			"status": "listed",
			"requested_status": "listed",
			"id": "IIJJKKLL",
			"project_id": "AABBCCDD",
			"author_id": "EEFFGGHH",
			"date_published": "2024-08-07T20:21:13.726918Z",
			"downloads": 0,
			"changelog_url": null,
			"files": [{
				"hashes": {
					"sha512": "93ecf5fe02914fb53d94aa3d28c1fb562e23985f8e4d48b9038422798618761fe208a31ca9b723667a4e05de0d91a3f86bcd8d018f6a686c39550e21b198d96f",
					"sha1": "c84dd4b3580c02b79958a0590afd5783d80ef504"
				},
				"url": "https://cdn.modrinth.com/data/AABBCCDD/versions/1.0.0/my_file.jar",
				"filename": "my_file.jar",
				"primary": false,
				"size": 1097270,
				"file_type": "required-resource-pack"
			}]
		}`

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/version_file/c84dd4b3580c02b79958a0590afd5783d80ef504" {
			t.Errorf("Expected path '/v2/version_file/c84dd4b3580c02b79958a0590afd5783d80ef504', got '%s'", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "test-api-key" {
			t.Errorf("Expected Authorization header to be 'test-api-key', got '%s'", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	_ = os.Setenv("MODRINTH_API_KEY", "test-api-key")
	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	client := &Client{client: mockServer.Client()}
	lookup := &VersionHashLookup{
		algorithm: Sha1,
		hash:      "c84dd4b3580c02b79958a0590afd5783d80ef504",
	}

	version, err := GetVersionForHash(lookup, client)

	assert.NoError(t, err)
	assert.Equal(t, "Version 1.0.0", version.Name)
	assert.Equal(t, "1.0.0", version.VersionNumber)
	assert.Equal(t, "List of changes in this version: ...", version.Changelog)
	assert.Contains(t, version.GameVersions, "1.16.5")
	assert.Contains(t, version.GameVersions, "1.17.1")
	assert.Equal(t, models.Release, version.Type)
	assert.Equal(t, Listed, version.Status)
	assert.Contains(t, version.Loaders, models.FABRIC)
	assert.Contains(t, version.Loaders, models.FORGE)
	assert.Equal(t, "IIJJKKLL", version.VersionId)
	assert.Equal(t, "AABBCCDD", version.ProjectId)

	expectedTime := time.Date(2024, 8, 7, 20, 21, 13, 726918000, time.UTC)

	assert.Equal(t, expectedTime, version.DatePublished)
	assert.Len(t, version.Files, 1)
	assert.Equal(t, "93ecf5fe02914fb53d94aa3d28c1fb562e23985f8e4d48b9038422798618761fe208a31ca9b723667a4e05de0d91a3f86bcd8d018f6a686c39550e21b198d96f", version.Files[0].Hashes.Sha512)
	assert.Equal(t, "c84dd4b3580c02b79958a0590afd5783d80ef504", version.Files[0].Hashes.Sha1)
	assert.Equal(t, "https://cdn.modrinth.com/data/AABBCCDD/versions/1.0.0/my_file.jar", version.Files[0].Url)
	assert.Equal(t, "my_file.jar", version.Files[0].FileName)
	assert.False(t, version.Files[0].Primary)
	assert.Equal(t, int64(1097270), version.Files[0].Size)
}

func TestGetVersionForHashWhenProjectNotFound(t *testing.T) {
	// Define the mock response JSON
	mockResponse := `{
    "error": "not_found",
    "description": "the requested route does not exist"
  }`

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	// Call the function
	lookup := &VersionHashLookup{
		algorithm: Sha1,
		hash:      "c84dd4b3580c02b79958a0590afd5783d80ef504",
	}
	project, err := GetVersionForHash(lookup, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, &VersionNotFoundError{
		Lookup: *lookup,
	})
	assert.Errorf(t, err, "project not found: AABBCCD1")
	assert.Nil(t, project)
}

func TestGetVersionForHashWhenProjectApiUnknownStatus(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot)
	}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	lookup := &VersionHashLookup{
		algorithm: Sha1,
		hash:      "c84dd4b3580c02b79958a0590afd5783d80ef504",
	}
	project, err := GetVersionForHash(lookup, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "unexpected status code: 418", errors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetVersionForHashWhenApiCallFails(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", "invalid_url")
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	lookup := &VersionHashLookup{
		algorithm: Sha1,
		hash:      "c84dd4b3580c02b79958a0590afd5783d80ef504",
	}
	project, err := GetVersionForHash(lookup, &Client{
		client: mockServer.Client(),
	})

	// Assertions
	//assert.Error(t, err)
	assert.ErrorIs(t, err, &VersionApiError{
		Lookup: *lookup,
	})
	assert.Equal(t, "Get \"invalid_url/v2/version_file/c84dd4b3580c02b79958a0590afd5783d80ef504?algorithm=sha1\": unsupported protocol scheme \"\"", errors.Unwrap(err).Error())
	assert.Nil(t, project)
}

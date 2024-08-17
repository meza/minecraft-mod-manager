package modrinth

import (
	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetProject(t *testing.T) {
	// Define the mock response JSON
	mockResponse := `{
		"slug": "my_project",
		"title": "My Project",
		"description": "A short description",
		"categories": ["technology", "adventure", "fabric"],
		"client_side": "required",
		"server_side": "optional",
		"body": "A long body describing my project in detail",
		"status": "approved",
		"requested_status": "approved",
		"additional_categories": ["technology", "adventure", "fabric"],
		"issues_url": "https://github.com/my_user/my_project/issues",
		"source_url": "https://github.com/my_user/my_project",
		"wiki_url": "https://github.com/my_user/my_project/wiki",
		"discord_url": "https://discord.gg/AaBbCcDd",
		"donation_urls": [{"id": "patreon", "platform": "Patreon", "url": "https://www.patreon.com/my_user"}],
		"project_type": "mod",
		"downloads": 0,
		"icon_url": "https://cdn.com/data/AABBCCDD/b46513nd83hb4792a9a0e1fn28fgi6090c1842639.png",
		"color": 8703084,
		"thread_id": "TTUUVVWW",
		"monetization_status": "monetized",
		"id": "AABBCCDD",
		"team": "MMNNOOPP",
		"body_url": null,
		"moderator_message": null,
		"published": "string",
		"updated": "string",
		"approved": "string",
		"queued": "string",
		"followers": 0,
		"license": {"id": "LGPL-3.0-or-later", "name": "GNU Lesser General Public License v3 or later", "url": "string"},
		"versions": ["IIJJKKLL", "QQRRSSTT"],
		"game_versions": ["1.19", "1.19.1", "1.19.2", "1.19.3"],
		"loaders": ["forge", "fabric", "quilt"],
		"gallery": [{
			"url": "https://cdn.com/data/AABBCCDD/images/009b7d8d6e8bf04968a29421117c59b3efe2351a.png",
			"featured": true,
			"title": "My awesome screenshot!",
			"description": "This awesome screenshot shows all of the blocks in my mod!",
			"created": "string",
			"ordering": 0
		}]
	}`

	err := os.Setenv("MODRINTH_API_KEY", "mock_modrinth_api_key")
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
		return
	}

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/project/AABBCCDD" {
			t.Errorf("Expected path '/v2/project/AABBCCDD', got '%s'", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "mock_modrinth_api_key" {
			t.Errorf("Expected Authorization header to be 'mock_modrinth_api_key', got '%s'", r.Header.Get("Authorization"))
		}

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

	defer func() {
		os.Unsetenv("MODRINTH_API_URL")
		os.Unsetenv("MODRINTH_API_KEY")
	}()

	// Call the function
	project, err := GetProject("AABBCCDD", &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.Equal(t, "my_project", project.Slug)
	assert.Equal(t, "My Project", project.Title)
	assert.Equal(t, "A short description", project.Description)
	assert.Equal(t, []string{"technology", "adventure", "fabric"}, project.Categories)
	assert.Equal(t, Required, project.ClientSide)
	assert.Equal(t, Optional, project.ServerSide)
	assert.Equal(t, Approved, project.Status)
	assert.Equal(t, "AABBCCDD", project.Id)
	assert.Equal(t, Mod, project.Type)
	assert.Equal(t, []string{"1.19", "1.19.1", "1.19.2", "1.19.3"}, project.GameVersions)
	assert.Equal(t, []models.Loader{"forge", "fabric", "quilt"}, project.Loaders)
}

func TestGetProjectWhenProjectNotFound(t *testing.T) {
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
	project, err := GetProject("AABBCCDD", &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, &globalErrors.ProjectNotFoundError{
		ProjectID: "AABBCCDD",
		Platform:  models.MODRINTH,
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

	err1 := os.Setenv("MODRINTH_API_URL", mockServer.URL)
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	// Call the function
	project, err := GetProject("AABBCCDD", &Client{
		client: mockServer.Client(),
	})

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "unexpected status code: 418", errors.Unwrap(err).Error())
	assert.Nil(t, project)
}

func TestGetProjectWhenApiCallFails(t *testing.T) {

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	err1 := os.Setenv("MODRINTH_API_URL", "invalid_url")
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}

	defer func() { os.Unsetenv("MODRINTH_API_URL") }()

	// Call the function
	project, err := GetProject("AABBCCDDEE", &Client{
		client: mockServer.Client(),
	})

	// Assertions
	//assert.Error(t, err)
	assert.ErrorIs(t, err, &globalErrors.ProjectApiError{
		ProjectID: "AABBCCDDEE",
		Platform:  models.MODRINTH,
	})
	assert.Equal(t, "Get \"invalid_url/v2/project/AABBCCDDEE\": unsupported protocol scheme \"\"", errors.Unwrap(err).Error())
	assert.Nil(t, project)
}

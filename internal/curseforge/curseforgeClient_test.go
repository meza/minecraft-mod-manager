package curseforge

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"os"
	"testing"
)

// MockDoer is a mock implementation of the httpClient.Doer interface
type MockDoer struct {
	mock.Mock
}

func (m *MockDoer) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestClient_Do(t *testing.T) {
	// Mock the environment function
	err1 := os.Setenv("CURSEFORGE_API_KEY", "test-api-key")
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}
	defer func() { os.Unsetenv("CURSEFORGE_API_KEY") }()

	// Create a mock Doer
	mockDoer := new(MockDoer)
	mockDoer.On("Do", mock.Anything).Return(&http.Response{StatusCode: 200}, nil)

	client := &Client{client: mockDoer}

	req, err := http.NewRequest("GET", "https://api.curseforge.com/v1/mods/test-project-id", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify headers
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Equal(t, "test-api-key", req.Header.Get("x-api-key"))

	// Verify that the mock Doer was called with the correct request
	mockDoer.AssertCalled(t, "Do", req)
}

func TestBaseURLResolving(t *testing.T) {
	assert.Equal(t, "https://api.curseforge.com/v1", GetBaseUrl())
}

func TestBaseURLResolvingWithEnvironment(t *testing.T) {
	err := os.Setenv("CURSEFORGE_API_URL", "test-url")
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	defer func() { os.Unsetenv("CURSEFORGE_API_URL") }()
	assert.Equal(t, "test-url", GetBaseUrl())
}

func TestNewClient(t *testing.T) {
	mockDoer := new(MockDoer)
	client := NewClient(mockDoer)
	assert.NotNil(t, client)
}

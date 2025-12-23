package modrinth

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDoer is a mock implementation of the httpclient.Doer interface
type MockDoer struct {
	mock.Mock
}

func (m *MockDoer) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestClient_Do(t *testing.T) {
	// Mock the environment function
	err1 := os.Setenv("MODRINTH_API_KEY", "test-api-key")
	if err1 != nil {
		t.Fatalf("Failed to set environment variable: %v", err1)
		return
	}
	defer func() { os.Unsetenv("MODRINTH_API_KEY") }()

	// Create a mock Doer
	mockDoer := new(MockDoer)
	mockDoer.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil)

	client := &Client{client: mockDoer}

	req, err := http.NewRequest(http.MethodGet, "https://api.modrinth.com/v2/project/test-project-id", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	if resp.Body != nil {
		assert.NoError(t, resp.Body.Close())
	}

	// Verify headers
	assert.Equal(t, "github_com/meza/minecraft-mod-manager/REPL_VERSION", req.Header.Get("user-agent"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Equal(t, "test-api-key", req.Header.Get("Authorization"))

	// Verify that the mock Doer was called with the correct request
	mockDoer.AssertCalled(t, "Do", mock.MatchedBy(func(r *http.Request) bool {
		if r == nil {
			return false
		}
		return r.Method == req.Method &&
			r.URL.String() == req.URL.String() &&
			r.Header.Get("user-agent") == "github_com/meza/minecraft-mod-manager/REPL_VERSION" &&
			r.Header.Get("Accept") == "application/json" &&
			r.Header.Get("Authorization") == "test-api-key"
	}))
}

func TestBaseURLIsConstant(t *testing.T) {
	assert.Equal(t, "https://api.modrinth.com", GetBaseURL())
	t.Setenv("MODRINTH_API_URL", "https://example.com/v2")
	assert.Equal(t, "https://api.modrinth.com", GetBaseURL())
}

func TestNewClient(t *testing.T) {
	mockDoer := new(MockDoer)
	client := NewClient(mockDoer)
	assert.NotNil(t, client)
}

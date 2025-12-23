package curseforge

import (
	"io"
	"net/http"
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
	t.Setenv("CURSEFORGE_API_KEY", "test-api-key")

	// Create a mock Doer
	mockDoer := new(MockDoer)
	mockDoer.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil)

	client := &Client{client: mockDoer}

	req, err := http.NewRequest(http.MethodGet, "https://api.curseforge.com/v1/mods/test-project-id", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	if resp.Body != nil {
		assert.NoError(t, resp.Body.Close())
	}

	// Verify headers
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Equal(t, "test-api-key", req.Header.Get("x-api-key"))

	// Verify that the mock Doer was called with the correct request
	mockDoer.AssertCalled(t, "Do", mock.MatchedBy(func(r *http.Request) bool {
		if r == nil {
			return false
		}
		return r.Method == req.Method &&
			r.URL.String() == req.URL.String() &&
			r.Header.Get("Accept") == "application/json" &&
			r.Header.Get("x-api-key") == "test-api-key"
	}))
}

func TestBaseURLIsConstant(t *testing.T) {
	assert.Equal(t, "https://api.curseforge.com/v1", GetBaseURL())
	t.Setenv("CURSEFORGE_API_URL", "https://example.com")
	assert.Equal(t, "https://api.curseforge.com/v1", GetBaseURL())
}

func TestNewClient(t *testing.T) {
	mockDoer := new(MockDoer)
	client := NewClient(mockDoer)
	assert.NotNil(t, client)
}

package httpClient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type MockProgram struct {
	Sender
	sentMessages []tea.Msg
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (m *MockProgram) Send(msg tea.Msg) {
	m.sentMessages = append(m.sentMessages, msg)
}

func (m *MockProgram) SentMessages() []tea.Msg {
	return m.sentMessages
}

func TestDownloadFile(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		program := &MockProgram{}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("file content"))
		}))
		defer mockServer.Close()

		destinationFile := "testfile"

		err := DownloadFile(context.Background(), mockServer.URL, destinationFile, mockServer.Client(), program, fs)
		assert.NoError(t, err)

		// Verify the file content
		content, err := afero.ReadFile(fs, destinationFile)
		assert.NoError(t, err)
		assert.Equal(t, "file content", string(content))
		assert.Equal(t, 1, len(program.SentMessages()))

		// Verify the progress message
		_, ok := program.SentMessages()[0].(progressMsg)
		assert.True(t, ok)

	})

	t.Run("HTTP request error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer mockServer.Close()

		err := DownloadFile(context.Background(), "invalid-url", "testfile", mockServer.Client(), &MockProgram{}, afero.NewMemMapFs())
		assert.ErrorContains(t, err, "failed to download file")
	})

	t.Run("HTTP timeout error keeps i18n message", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		program := &MockProgram{}
		timeoutDoer := doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, &TimeoutError{Err: context.DeadlineExceeded}
		})

		err := DownloadFile(context.Background(), "https://example.com/file", "testfile", timeoutDoer, program, fs)
		assert.Error(t, err)
		assert.Equal(t, i18n.T("error.network_timeout"), err.Error())
	})

	t.Run("file creation error", func(t *testing.T) {
		memFs := afero.NewMemMapFs()
		fs := afero.NewReadOnlyFs(memFs)
		// Create a mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("file content"))
		}))

		err := DownloadFile(context.Background(), mockServer.URL, "/invalid/path/testfile", mockServer.Client(), &MockProgram{}, fs)
		assert.ErrorContains(t, err, "failed to create file")
	})

	t.Run("file write error", func(t *testing.T) {

		// Create a mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1")
		}))
		defer mockServer.Close()

		fs := afero.NewMemMapFs()

		program := &MockProgram{}
		err := DownloadFile(context.Background(), mockServer.URL, "test", mockServer.Client(), program, fs)
		assert.ErrorContains(t, err, "failed to write file")
	})
}

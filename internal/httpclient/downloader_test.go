package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (doer doerFunc) Do(req *http.Request) (*http.Response, error) {
	return doer(req)
}

func (program *MockProgram) Send(msg tea.Msg) {
	program.sentMessages = append(program.sentMessages, msg)
}

func (program *MockProgram) SentMessages() []tea.Msg {
	return program.sentMessages
}

type closeErrorBody struct {
	reader   *strings.Reader
	closeErr error
}

func newCloseErrorBody(payload string, closeErr error) *closeErrorBody {
	return &closeErrorBody{
		reader:   strings.NewReader(payload),
		closeErr: closeErr,
	}
}

func (body *closeErrorBody) Read(p []byte) (int, error) {
	return body.reader.Read(p)
}

func (body *closeErrorBody) Close() error {
	if body.closeErr != nil {
		return body.closeErr
	}
	return nil
}

type closeErrorFile struct {
	afero.File
	closeErr error
}

func (file closeErrorFile) Close() error {
	closeErr := file.File.Close()
	if closeErr != nil && file.closeErr != nil {
		return errors.Join(closeErr, file.closeErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if file.closeErr != nil {
		return file.closeErr
	}
	return nil
}

type closeErrorFs struct {
	afero.Fs
	closeErr error
}

func (filesystem closeErrorFs) Create(name string) (afero.File, error) {
	file, err := filesystem.Fs.Create(name)
	if err != nil {
		return nil, err
	}
	return closeErrorFile{File: file, closeErr: filesystem.closeErr}, nil
}

type removeErrorFs struct {
	afero.Fs
	failPath string
}

func (filesystem removeErrorFs) Remove(name string) error {
	if name == filesystem.failPath {
		return errors.New("remove failed")
	}
	return filesystem.Fs.Remove(name)
}

type readErrorBody struct {
	err error
}

func (body *readErrorBody) Read(_ []byte) (int, error) {
	return 0, body.err
}

func (body *readErrorBody) Close() error {
	return nil
}

func TestDownloadFile(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		program := &MockProgram{}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("file content")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
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

	t.Run("HTTP request build error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer mockServer.Close()

		err := DownloadFile(context.Background(), "http://[::1", "testfile", mockServer.Client(), &MockProgram{}, afero.NewMemMapFs())
		assert.ErrorContains(t, err, "failed to build download request")
	})

	t.Run("HTTP non-2xx response returns error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		program := &MockProgram{}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte("bad request")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

		err := DownloadFile(context.Background(), mockServer.URL, "testfile", mockServer.Client(), program, fs)
		assert.ErrorContains(t, err, "download request failed with status 400")
		exists, existsErr := afero.Exists(fs, "testfile")
		assert.NoError(t, existsErr)
		assert.False(t, exists)
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
			if _, err := w.Write([]byte("file content")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

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
		exists, existsErr := afero.Exists(fs, "test")
		assert.NoError(t, existsErr)
		assert.False(t, exists)
	})

	t.Run("response body close error returns error", func(t *testing.T) {
		bodyErr := errors.New("close failed")
		doer := doerFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       newCloseErrorBody("content", bodyErr),
			}, nil
		})

		err := DownloadFile(context.Background(), "https://example.com/file", "testfile", doer, &MockProgram{}, afero.NewMemMapFs())
		assert.ErrorIs(t, err, bodyErr)
	})

	t.Run("file close error returns error", func(t *testing.T) {
		baseFs := afero.NewMemMapFs()
		fs := closeErrorFs{Fs: baseFs, closeErr: errors.New("close failed")}
		doer := doerFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("content")),
			}, nil
		})

		err := DownloadFile(context.Background(), "https://example.com/file", "testfile", doer, &MockProgram{}, fs)
		assert.ErrorContains(t, err, "close failed")
	})

	t.Run("file write cleanup failure returns joined error", func(t *testing.T) {
		fs := removeErrorFs{Fs: afero.NewMemMapFs(), failPath: "test"}
		readErr := errors.New("read failed")
		doer := doerFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &readErrorBody{err: readErr},
			}, nil
		})
		program := &MockProgram{}

		err := DownloadFile(context.Background(), "https://example.com/file", "test", doer, program, fs)
		assert.ErrorContains(t, err, "failed to remove partial file")
	})
}

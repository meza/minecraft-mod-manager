package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type mockProgram struct {
	Sender
	sentMessages []tea.Msg
}

type doerFunc func(*http.Request) (*http.Response, error)

func (doer doerFunc) Do(req *http.Request) (*http.Response, error) {
	return doer(req)
}

type hostRewriteDoer struct {
	base *url.URL
	next Doer
}

func newHostRewriteDoer(serverURL string, next Doer) (*hostRewriteDoer, error) {
	if next == nil {
		return nil, errors.New("next doer is nil")
	}

	base, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	if base.Scheme == "" || base.Host == "" {
		return nil, errors.New("server url must include scheme and host")
	}
	return &hostRewriteDoer{base: base, next: next}, nil
}

func (doer *hostRewriteDoer) Do(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = doer.base.Scheme
	cloned.URL.Host = doer.base.Host
	cloned.Host = doer.base.Host
	return doer.next.Do(cloned)
}

func (program *mockProgram) Send(msg tea.Msg) {
	program.sentMessages = append(program.sentMessages, msg)
}

func (program *mockProgram) SentMessages() []tea.Msg {
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

type readCloseErrorBody struct {
	readErr  error
	closeErr error
}

func (body *readCloseErrorBody) Read(_ []byte) (int, error) {
	return 0, body.readErr
}

func (body *readCloseErrorBody) Close() error {
	return body.closeErr
}

func TestDownloadFile(t *testing.T) {
	allowedURL := func(path string) string {
		return "https://cdn.modrinth.com" + path
	}

	t.Run("successful download", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		program := &mockProgram{}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("file content")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

		doer, err := newHostRewriteDoer(mockServer.URL, mockServer.Client())
		assert.NoError(t, err)
		destinationFile := "testfile"

		err = DownloadFile(context.Background(), allowedURL("/testfile"), destinationFile, doer, program, fs)
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

	t.Run("nil program does not panic", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("file content")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

		doer, err := newHostRewriteDoer(mockServer.URL, mockServer.Client())
		assert.NoError(t, err)
		err = DownloadFile(context.Background(), allowedURL("/testfile"), "testfile", doer, nil, fs)
		assert.NoError(t, err)

		content, err := afero.ReadFile(fs, "testfile")
		assert.NoError(t, err)
		assert.Equal(t, "file content", string(content))
	})

	t.Run("invalid download URL returns validation error", func(t *testing.T) {
		err := DownloadFile(context.Background(), "invalid-url", "testfile", doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected request")
		}), &mockProgram{}, afero.NewMemMapFs())
		assert.Error(t, err)
		assert.Equal(t, i18n.T("error.download_url_invalid", i18n.Tvars{
			Data: &i18n.TData{"url": "invalid-url"},
		}), err.Error())
	})

	t.Run("insecure download URL returns validation error", func(t *testing.T) {
		insecureURL := "http://cdn.modrinth.com/file"

		err := DownloadFile(context.Background(), insecureURL, "testfile", doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected request")
		}), &mockProgram{}, afero.NewMemMapFs())
		assert.Error(t, err)
		assert.Equal(t, i18n.T("error.download_url_insecure", i18n.Tvars{
			Data: &i18n.TData{"url": insecureURL},
		}), err.Error())
	})

	t.Run("untrusted download host returns validation error", func(t *testing.T) {
		untrustedURL := "https://example.com/file"

		err := DownloadFile(context.Background(), untrustedURL, "testfile", doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected request")
		}), &mockProgram{}, afero.NewMemMapFs())
		assert.Error(t, err)
		assert.Equal(t, i18n.T("error.download_url_untrusted_host", i18n.Tvars{
			Data: &i18n.TData{"host": "example.com", "url": untrustedURL},
		}), err.Error())
	})

	t.Run("request build failure returns error", func(t *testing.T) {
		originalBuild := buildDownloadRequestFunc
		buildDownloadRequestFunc = func(context.Context, string) (*http.Request, func(), error) {
			return nil, func() {}, errors.New("request failed")
		}
		t.Cleanup(func() {
			buildDownloadRequestFunc = originalBuild
		})

		err := DownloadFile(context.Background(), allowedURL("/file"), "testfile", doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected request")
		}), &mockProgram{}, afero.NewMemMapFs())
		assert.ErrorContains(t, err, "failed to build download request")
	})

	t.Run("HTTP non-2xx response returns error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		program := &mockProgram{}

		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte("bad request")); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer mockServer.Close()

		doer, err := newHostRewriteDoer(mockServer.URL, mockServer.Client())
		assert.NoError(t, err)
		err = DownloadFile(context.Background(), allowedURL("/testfile"), "testfile", doer, program, fs)
		assert.ErrorContains(t, err, "download request failed with status 400")
		exists, existsErr := afero.Exists(fs, "testfile")
		assert.NoError(t, existsErr)
		assert.False(t, exists)
	})

	t.Run("HTTP timeout error keeps i18n message", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		program := &mockProgram{}
		timeoutDoer := doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, &TimeoutError{Err: context.DeadlineExceeded}
		})

		err := DownloadFile(context.Background(), allowedURL("/file"), "testfile", timeoutDoer, program, fs)
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

		doer, err := newHostRewriteDoer(mockServer.URL, mockServer.Client())
		assert.NoError(t, err)
		err = DownloadFile(context.Background(), allowedURL("/testfile"), "/invalid/path/testfile", doer, &mockProgram{}, fs)
		assert.ErrorContains(t, err, "failed to create file")
	})

	t.Run("file write error", func(t *testing.T) {
		// Create a mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1")
		}))
		defer mockServer.Close()

		fs := afero.NewMemMapFs()

		program := &mockProgram{}
		doer, err := newHostRewriteDoer(mockServer.URL, mockServer.Client())
		assert.NoError(t, err)
		err = DownloadFile(context.Background(), allowedURL("/test"), "test", doer, program, fs)
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

		err := DownloadFile(context.Background(), allowedURL("/file"), "testfile", doer, &mockProgram{}, afero.NewMemMapFs())
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

		err := DownloadFile(context.Background(), allowedURL("/file"), "testfile", doer, &mockProgram{}, fs)
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
		program := &mockProgram{}

		err := DownloadFile(context.Background(), allowedURL("/file"), "test", doer, program, fs)
		assert.ErrorContains(t, err, "failed to remove partial file")
	})

	t.Run("response close error ignored when write fails", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		readErr := errors.New("read failed")
		closeErr := errors.New("close failed")
		doer := doerFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &readCloseErrorBody{readErr: readErr, closeErr: closeErr},
			}, nil
		})
		program := &mockProgram{}

		err := DownloadFile(context.Background(), allowedURL("/file"), "test", doer, program, fs)
		assert.ErrorContains(t, err, "failed to write file")
		assert.ErrorIs(t, err, readErr)
		assert.NotErrorIs(t, err, closeErr)
		exists, existsErr := afero.Exists(fs, "test")
		assert.NoError(t, existsErr)
		assert.False(t, exists)
	})
}

type valueProgram struct{}

func (program valueProgram) Send(_ tea.Msg) {}

func TestSendProgressWithProgram(t *testing.T) {
	program := &mockProgram{}
	sendProgress(program, progressMsg(0.5))
	assert.Len(t, program.SentMessages(), 1)
	_, ok := program.SentMessages()[0].(progressMsg)
	assert.True(t, ok)
}

func TestSendProgressWithNilProgram(t *testing.T) {
	assert.NotPanics(t, func() {
		sendProgress(nil, progressMsg(0.5))
	})
}

func TestIsNilSenderHandlesTypedNil(t *testing.T) {
	var program *mockProgram
	var sender Sender = program
	assert.True(t, isNilSender(sender))
}

func TestIsNilSenderHandlesNonNil(t *testing.T) {
	assert.False(t, isNilSender(&mockProgram{}))
	assert.False(t, isNilSender(valueProgram{}))
}

func TestValidateDownloadURLAcceptsTrustedHost(t *testing.T) {
	parsed, err := validateDownloadURL("https://cdn.modrinth.com/data/file.jar")
	assert.NoError(t, err)
	assert.Equal(t, "cdn.modrinth.com", parsed.Hostname())
}

func TestValidateDownloadURLRejectsMissingHost(t *testing.T) {
	parsed, err := validateDownloadURL("https:///file.jar")
	assert.Error(t, err)
	assert.Nil(t, parsed)

	var invalidErr InvalidDownloadURLError
	assert.ErrorAs(t, err, &invalidErr)
}

func TestValidateDownloadURLRejectsEmptyHostname(t *testing.T) {
	parsed, err := validateDownloadURL("https://:443/file.jar")
	assert.Error(t, err)
	assert.Nil(t, parsed)

	var invalidErr InvalidDownloadURLError
	assert.ErrorAs(t, err, &invalidErr)
}

func TestValidateDownloadURLRejectsParseErrors(t *testing.T) {
	parsed, err := validateDownloadURL("http://[::1")
	assert.Error(t, err)
	assert.Nil(t, parsed)

	var invalidErr InvalidDownloadURLError
	assert.ErrorAs(t, err, &invalidErr)
}

func TestBuildDownloadRequestReturnsRequest(t *testing.T) {
	request, cancel, err := buildDownloadRequest(context.Background(), "https://cdn.modrinth.com/file.jar")
	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, http.MethodGet, request.Method)
	cancel()
}

func TestBuildDownloadRequestReturnsErrorOnInvalidURL(t *testing.T) {
	request, cancel, err := buildDownloadRequest(context.Background(), "http://[::1")
	assert.Error(t, err)
	assert.Nil(t, request)
	cancel()
}

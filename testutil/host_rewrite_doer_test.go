package testutil

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type recordingDoer struct {
	lastRequest *http.Request
}

func (d *recordingDoer) Do(req *http.Request) (*http.Response, error) {
	d.lastRequest = req
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

type errorDoer struct {
	err error
}

func (d errorDoer) Do(*http.Request) (*http.Response, error) {
	return nil, d.err
}

func TestNewHostRewriteDoer(t *testing.T) {
	t.Run("rejects nil next", func(t *testing.T) {
		_, err := NewHostRewriteDoer("https://example.com", nil)
		assert.ErrorContains(t, err, "next doer is nil")
	})

	t.Run("rejects invalid server URL", func(t *testing.T) {
		_, err := NewHostRewriteDoer(":", errorDoer{err: errors.New("unused")})
		assert.Error(t, err)
	})

	t.Run("rejects URL without host", func(t *testing.T) {
		_, err := NewHostRewriteDoer("https://", errorDoer{err: errors.New("unused")})
		assert.ErrorContains(t, err, "scheme and host")
	})

	t.Run("rewrites scheme and host", func(t *testing.T) {
		next := &recordingDoer{}
		doer, err := NewHostRewriteDoer("https://example.com:8443", next)
		if !assert.NoError(t, err) {
			return
		}

		req, err := http.NewRequest(http.MethodGet, "https://api.modrinth.com/v2/project/test", nil)
		if !assert.NoError(t, err) {
			return
		}

		resp, err := doer.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		if resp.Body != nil {
			assert.NoError(t, resp.Body.Close())
		}

		if assert.NotNil(t, next.lastRequest) {
			assert.Equal(t, "example.com:8443", next.lastRequest.URL.Host)
			assert.Equal(t, "https", next.lastRequest.URL.Scheme)
		}
	})
}

func TestMustNewHostRewriteDoer(t *testing.T) {
	t.Run("panics on invalid input", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = MustNewHostRewriteDoer(":", errorDoer{err: errors.New("unused")})
		})
	})

	t.Run("returns doer on valid input", func(t *testing.T) {
		next := &recordingDoer{}
		doer := MustNewHostRewriteDoer("https://example.com:8443", next)
		if assert.NotNil(t, doer) {
			req, err := http.NewRequest(http.MethodGet, "https://api.modrinth.com/v2/project/test", nil)
			if !assert.NoError(t, err) {
				return
			}
			resp, err := doer.Do(req)
			assert.NoError(t, err)
			if resp.Body != nil {
				assert.NoError(t, resp.Body.Close())
			}
		}
	})
}

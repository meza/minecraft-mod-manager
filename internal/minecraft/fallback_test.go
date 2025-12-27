package minecraft

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidReleaseVersionError(t *testing.T) {
	invalid := &InvalidReleaseVersionError{Version: "1.2.3"}
	assert.Equal(t, `minecraft version "1.2.3" is not a known release`, invalid.Error())

	matching := &InvalidReleaseVersionError{Version: "1.2.3"}
	other := &InvalidReleaseVersionError{Version: "1.2.4"}
	assert.True(t, invalid.Is(matching))
	assert.False(t, invalid.Is(other))
	assert.False(t, invalid.Is(errors.New("other")))
}

func TestNextPatchDownRejectsEmptyVersion(t *testing.T) {
	_, _, err := NextPatchDown(context.Background(), " ", doerFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("unexpected call")
	}))
	var invalid *InvalidReleaseVersionError
	assert.ErrorAs(t, err, &invalid)
}

func TestNextPatchDownRejectsMissingSeriesKey(t *testing.T) {
	ClearManifestCache()
	_, _, err := NextPatchDown(context.Background(), "26", doerFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"versions":[{"id":"26.1.1","type":"release"}]}`)),
		}, nil
	}))
	var invalid *InvalidReleaseVersionError
	assert.ErrorAs(t, err, &invalid)
}

func TestNextPatchDownReturnsNoFallback(t *testing.T) {
	ClearManifestCache()
	next, ok, err := NextPatchDown(context.Background(), "1.21.1", doerFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"versions":[{"id":"1.21.1","type":"release"},{"id":"1.20.2","type":"release"}]}`)),
		}, nil
	}))
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "1.21.1", next)
}

func TestNextPatchDownAllowsPatchToMinorFallback(t *testing.T) {
	ClearManifestCache()
	next, ok, err := NextPatchDown(context.Background(), "1.20.1", doerFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"versions":[{"id":"1.20.1","type":"release"},{"id":"1.20","type":"release"}]}`)),
		}, nil
	}))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "1.20", next)
}

func TestNextPatchDownSkipsInvalidReleaseEntries(t *testing.T) {
	ClearManifestCache()
	next, ok, err := NextPatchDown(context.Background(), "1.20.2", doerFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{"versions":[
        {"id":"1.20.2","type":"release"},
        {"id":"1","type":"release"},
        {"id":"1.20.1","type":"release"}
      ]}`)),
		}, nil
	}))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "1.20.1", next)
}

func TestNextPatchDownReturnsManifestError(t *testing.T) {
	ClearManifestCache()
	_, _, err := NextPatchDown(context.Background(), "1.20.1", doerFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("manifest failure")
	}))
	assert.ErrorContains(t, err, "manifest failure")
}

func TestNextPatchDownRejectsWhenNoReleaseVersions(t *testing.T) {
	ClearManifestCache()
	_, _, err := NextPatchDown(context.Background(), "1.20.1", doerFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"versions":[{"id":"26.1-snapshot-1","type":"snapshot"}]}`)),
		}, nil
	}))
	var invalid *InvalidReleaseVersionError
	assert.ErrorAs(t, err, &invalid)
}

func TestReleaseVersionsFromManifestNil(t *testing.T) {
	assert.Empty(t, releaseVersionsFromManifest(nil))
}

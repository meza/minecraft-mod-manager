package vcr

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

type failingTransport struct {
	closeErr error
}

func (transport failingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Request:    request,
		Body:       http.NoBody,
		Header:     http.Header{},
	}, nil
}

func (transport failingTransport) Close() error {
	return transport.closeErr
}

func TestLoadCassetteNil(t *testing.T) {
	LoadCassette(nil, "testdata/vcr/cassette.yaml")
}

func TestLiveHTTPBlockerRejectsNilRequest(t *testing.T) {
	response, err := liveHTTPBlocker{}.RoundTrip(nil)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.ErrorIs(t, err, errLiveHTTPBlocked)
}

func TestLiveHTTPBlockerRejectsRequests(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid", nil)
	require.NoError(t, err)

	response, err := liveHTTPBlocker{}.RoundTrip(request)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.ErrorIs(t, err, errLiveHTTPBlocked)
}

func TestLiveHTTPBlockerAllowsLocalhost(t *testing.T) {
	originalLiveTransport := liveTransport
	liveTransport = failingTransport{}
	t.Cleanup(func() { liveTransport = originalLiveTransport })

	request, err := http.NewRequest(http.MethodGet, "http://localhost/resource", nil)
	require.NoError(t, err)

	response, err := liveHTTPBlocker{}.RoundTrip(request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NoError(t, response.Body.Close())
}

func TestLiveHTTPBlockerRejectsLocalRequestWithoutLiveTransport(t *testing.T) {
	originalLiveTransport := liveTransport
	liveTransport = nil
	t.Cleanup(func() { liveTransport = originalLiveTransport })

	request, err := http.NewRequest(http.MethodGet, "http://localhost/resource", nil)
	require.NoError(t, err)

	response, err := liveHTTPBlocker{}.RoundTrip(request)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.ErrorIs(t, err, errLiveHTTPBlocked)
}

func TestIsLocalRequestNilURL(t *testing.T) {
	require.False(t, isLocalRequest(&http.Request{}))
}

func TestResolveCassettePathStripsYaml(t *testing.T) {
	cassette, err := resolveCassettePath("testdata/vcr/cassette.yaml")
	require.NoError(t, err)
	require.Equal(t, filepath.FromSlash("testdata/vcr/cassette"), cassette.basePath)
	require.False(t, cassette.record)
}

func TestResolveCassettePathStripsYml(t *testing.T) {
	cassette, err := resolveCassettePath("testdata/vcr/cassette.yml")
	require.NoError(t, err)
	require.Equal(t, filepath.FromSlash("testdata/vcr/cassette"), cassette.basePath)
}

func TestResolveCassettePathRejectsEmpty(t *testing.T) {
	_, err := resolveCassettePath(" ")
	require.Error(t, err)
}

func TestResolveCassettePathRejectsTrailingSeparator(t *testing.T) {
	_, err := resolveCassettePath("testdata/vcr/")
	require.Error(t, err)
}

func TestResolveCassettePathRejectsDot(t *testing.T) {
	_, err := resolveCassettePath(".")
	require.Error(t, err)
}

func TestResolveCassettePathUsesRecordFlag(t *testing.T) {
	t.Setenv(recordEnvVar, "true")

	cassette, err := resolveCassettePath("testdata/vcr/cassette.yaml")
	require.NoError(t, err)
	require.True(t, cassette.record)
}

func TestLoadCassetteRegistersCleanup(t *testing.T) {
	tempDir := t.TempDir()
	cassettePath := filepath.Join(tempDir, "cassette.yaml")
	originalDefaultTransport := http.DefaultTransport

	t.Run("apply", func(t *testing.T) {
		LoadCassette(t, cassettePath)
		require.NotEqual(t, originalDefaultTransport, http.DefaultTransport)
		require.NotNil(t, httpclient.NewRLClient(rate.NewLimiter(rate.Inf, 0)))
	})

	require.Equal(t, originalDefaultTransport, http.DefaultTransport)
	activeScenarioMu.Lock()
	require.Nil(t, activeScenario)
	activeScenarioMu.Unlock()
}

func TestLoadCassetteFailsWhenAlreadyActive(t *testing.T) {
	tempDir := t.TempDir()
	originalFatalf := applyFatalf
	originalScenario := activeScenario
	activeScenarioMu.Lock()
	activeScenario = &cassetteDetails{
		basePath: filepath.Join(tempDir, "active"),
		record:   false,
	}
	activeScenarioMu.Unlock()

	var fatalMessage string
	applyFatalf = func(test testing.TB, format string, args ...any) {
		fatalMessage = fmt.Sprintf(format, args...)
	}
	t.Cleanup(func() {
		applyFatalf = originalFatalf
		activeScenarioMu.Lock()
		activeScenario = originalScenario
		activeScenarioMu.Unlock()
	})

	LoadCassette(t, filepath.Join(tempDir, "next.yaml"))
	require.Contains(t, fatalMessage, "vcr already active")
}

func TestLoadCassetteReportsCassetteErrors(t *testing.T) {
	originalFatalf := applyFatalf
	var fatalMessage string
	applyFatalf = func(test testing.TB, format string, args ...any) {
		fatalMessage = fmt.Sprintf(format, args...)
	}
	t.Cleanup(func() { applyFatalf = originalFatalf })

	LoadCassette(t, " ")
	require.Contains(t, fatalMessage, "cassette path is empty")
}

func TestLoadCassetteReportsCloseErrors(t *testing.T) {
	tempDir := t.TempDir()
	originalNewTransport := newTransport
	originalReportCleanupError := reportCleanupError
	originalLiveTransport := liveTransport

	liveTransport = failingTransport{}
	newTransport = func(*cassetteDetails, http.RoundTripper) transportCloser {
		return failingTransport{closeErr: os.ErrInvalid}
	}

	var reportedErr error
	reportCleanupError = func(test testing.TB, err error) { reportedErr = err }

	t.Cleanup(func() {
		newTransport = originalNewTransport
		reportCleanupError = originalReportCleanupError
		liveTransport = originalLiveTransport
	})

	t.Run("apply", func(t *testing.T) {
		LoadCassette(t, filepath.Join(tempDir, "scenario.yaml"))
	})

	require.ErrorIs(t, reportedErr, os.ErrInvalid)
}

func TestLoadCassetteUsesCapturedLiveTransport(t *testing.T) {
	tempDir := t.TempDir()
	originalNewTransport := newTransport
	originalLiveTransport := liveTransport

	liveTransport = failingTransport{}
	var captured http.RoundTripper
	newTransport = func(_ *cassetteDetails, realTransport http.RoundTripper) transportCloser {
		captured = realTransport
		return failingTransport{}
	}

	t.Cleanup(func() {
		newTransport = originalNewTransport
		liveTransport = originalLiveTransport
	})

	LoadCassette(t, filepath.Join(tempDir, "scenario.yaml"))
	require.Equal(t, liveTransport, captured)
}

package vcr

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const recordEnvVar = "MMM_RECORD_HTTP"

type cassetteDetails struct {
	basePath string
	record   bool
}

type transportCloser interface {
	http.RoundTripper
	Close() error
}

var (
	activeScenarioMu sync.Mutex
	activeScenario   *cassetteDetails

	liveTransportMu sync.Mutex
	liveTransport   http.RoundTripper

	errLiveHTTPBlocked = errors.New("vcr: live HTTP disabled in tests; call vcr.LoadCassette")

	applyFatalf        = func(test testing.TB, format string, args ...any) { test.Fatalf(format, args...) }
	reportCleanupError = func(test testing.TB, err error) { test.Errorf("vcr transport close failed: %v", err) }
	newTransport       = func(cassette *cassetteDetails, realTransport http.RoundTripper) transportCloser {
		return newRecorderTransport(cassette, realTransport)
	}
)

func init() {
	captureLiveTransport()
	http.DefaultTransport = liveHTTPBlocker{}
}

type liveHTTPBlocker struct{}

func (liveHTTPBlocker) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil {
		return nil, errLiveHTTPBlocked
	}
	if isLocalRequest(request) {
		transport := liveTransportForLocal()
		if transport == nil {
			return nil, errLiveHTTPBlocked
		}
		return transport.RoundTrip(request)
	}
	return nil, fmt.Errorf("%w: %s %s", errLiveHTTPBlocked, request.Method, request.URL.String())
}

// LoadCassette activates VCR recording or replay for a test by swapping
// http.DefaultTransport to a recorder bound to the provided cassette path.
//
// Use it in tests that hit external APIs so responses replay deterministically.
//
// Contract:
// - The cassette path must include a file name (use filepath.Join); .yaml/.yml suffix is optional.
// - Replay is the default behavior; tests fail if the cassette is missing.
// - Recording only happens when MMM_RECORD_HTTP is truthy (use make vcr-record).
// - http.DefaultTransport is restored after the test finishes.
// - While the vcr package is imported, live external HTTP calls are blocked unless a cassette is active.
// - Localhost requests are allowed to keep httptest servers working.
//
// Concurrency:
// - Do not call LoadCassette concurrently. Only one cassette can be active at a time.
func LoadCassette(test testing.TB, cassettePath string) {
	if test == nil {
		return
	}

	cassette, err := resolveCassettePath(cassettePath)
	if err != nil {
		applyFatalf(test, "vcr setup failed: %v", err)
		return
	}

	activeScenarioMu.Lock()
	if activeScenario != nil {
		activeScenarioMu.Unlock()
		applyFatalf(test, "vcr already active for cassette %q", activeScenario.basePath)
		return
	}
	activeScenario = cassette
	activeScenarioMu.Unlock()

	transport := newTransport(cassette, captureLiveTransport())
	originalDefaultTransport := http.DefaultTransport
	http.DefaultTransport = transport

	test.Cleanup(func() {
		http.DefaultTransport = originalDefaultTransport
		if closeErr := transport.Close(); closeErr != nil {
			reportCleanupError(test, closeErr)
		}
		activeScenarioMu.Lock()
		activeScenario = nil
		activeScenarioMu.Unlock()
	})
}

func captureLiveTransport() http.RoundTripper {
	liveTransportMu.Lock()
	if liveTransport == nil {
		liveTransport = http.DefaultTransport
	}
	transport := liveTransport
	liveTransportMu.Unlock()
	return transport
}

func liveTransportForLocal() http.RoundTripper {
	liveTransportMu.Lock()
	transport := liveTransport
	liveTransportMu.Unlock()
	return transport
}

func isLocalRequest(request *http.Request) bool {
	if request.URL == nil {
		return false
	}
	switch request.URL.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func resolveCassettePath(cassettePath string) (*cassetteDetails, error) {
	trimmed := strings.TrimSpace(cassettePath)
	recordEnabled := isTruthy(os.Getenv(recordEnvVar))
	if trimmed == "" {
		return nil, errors.New("cassette path is empty")
	}
	if strings.HasSuffix(trimmed, "/") || strings.HasSuffix(trimmed, `\`) {
		return nil, errors.New("cassette path must include a file name")
	}
	normalized := filepath.Clean(filepath.FromSlash(trimmed))
	normalized = strings.TrimSuffix(normalized, ".yaml")
	normalized = strings.TrimSuffix(normalized, ".yml")
	if normalized == "." || normalized == string(filepath.Separator) || normalized == "" {
		return nil, fmt.Errorf("cassette path is invalid: %q", cassettePath)
	}

	return &cassetteDetails{
		basePath: normalized,
		record:   recordEnabled,
	}, nil
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

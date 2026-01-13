package vcr

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v2/recorder"
)

type stubTransport struct {
	callCount int
	body      string
	headers   http.Header
}

func (transport *stubTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.callCount++
	responseHeaders := http.Header{}
	for key, values := range transport.headers {
		responseHeaders[key] = values
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(transport.body)),
		Request:    request,
		Header:     responseHeaders,
	}, nil
}

func TestRecorderTransportRecordAndReplay(t *testing.T) {
	tempDir := t.TempDir()
	cassetteBase := filepath.Join(tempDir, "scenario")

	recordingTransport := &stubTransport{
		body: "ok",
		headers: http.Header{
			"X-Api-Key": []string{"secret"},
		},
	}
	recordScenario := &cassetteDetails{
		basePath: cassetteBase,
		record:   true,
	}

	recorderTransport := newRecorderTransport(recordScenario, recordingTransport)
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid/resource", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "secret")

	recordedResponse, err := recorderTransport.RoundTrip(request)
	require.NoError(t, err)
	bodyBytes, err := io.ReadAll(recordedResponse.Body)
	require.NoError(t, err)
	require.NoError(t, recordedResponse.Body.Close())
	require.Equal(t, "ok", string(bodyBytes))
	require.Equal(t, 1, recordingTransport.callCount)

	require.NoError(t, recorderTransport.Close())

	cassettePath := cassetteBase + ".yaml"
	cassetteContents, err := os.ReadFile(cassettePath)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(cassetteContents), "secret"))

	replayTransport := &stubTransport{body: "should-not-be-used"}
	replayScenario := &cassetteDetails{
		basePath: cassetteBase,
		record:   false,
	}
	replayRecorder := newRecorderTransport(replayScenario, replayTransport)
	replayedResponse, err := replayRecorder.RoundTrip(request)
	require.NoError(t, err)
	replayedBody, err := io.ReadAll(replayedResponse.Body)
	require.NoError(t, err)
	require.NoError(t, replayedResponse.Body.Close())
	require.Equal(t, "ok", string(replayedBody))
	require.Equal(t, 0, replayTransport.callCount)
	require.NoError(t, replayRecorder.Close())
}

func TestRecorderTransportReplayMissingCassetteFails(t *testing.T) {
	tempDir := t.TempDir()
	cassetteBase := filepath.Join(tempDir, "missing")

	transport := &stubTransport{body: "unused"}
	scenario := &cassetteDetails{
		basePath: cassetteBase,
		record:   false,
	}
	recorderTransport := newRecorderTransport(scenario, transport)
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid/missing", nil)
	require.NoError(t, err)

	response, err := recorderTransport.RoundTrip(request)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing")
	require.Equal(t, 0, transport.callCount)
}

func TestRecorderTransportCloseWithoutRecorder(t *testing.T) {
	scenario := &cassetteDetails{
		basePath: filepath.Join(t.TempDir(), "scenario"),
		record:   false,
	}
	recorderTransport := newRecorderTransport(scenario, &stubTransport{body: "unused"})
	require.NoError(t, recorderTransport.Close())
}

func TestRecorderTransportCloseNilReceiver(t *testing.T) {
	var recorderTransport *recorderTransport
	require.NoError(t, recorderTransport.Close())
}

func TestRecorderTransportNilReceiverFails(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid/nil", nil)
	require.NoError(t, err)

	var recorderTransport *recorderTransport
	response, err := recorderTransport.RoundTrip(request)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil")
}

func TestRecorderTransportRejectsNilRealTransport(t *testing.T) {
	scenario := &cassetteDetails{
		basePath: filepath.Join(t.TempDir(), "scenario"),
		record:   true,
	}
	recorderTransport := newRecorderTransport(scenario, nil)
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid/nil-transport", nil)
	require.NoError(t, err)

	response, err := recorderTransport.RoundTrip(request)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil real transport")
}

func TestRecorderTransportReportsRecorderInitError(t *testing.T) {
	originalNewRecorder := newRecorder
	newRecorder = func(string, recorder.Mode, http.RoundTripper) (*recorder.Recorder, error) {
		return nil, os.ErrInvalid
	}
	t.Cleanup(func() { newRecorder = originalNewRecorder })

	scenario := &cassetteDetails{
		basePath: filepath.Join(t.TempDir(), "scenario"),
		record:   true,
	}
	recorderTransport := newRecorderTransport(scenario, &stubTransport{body: "ok"})
	request, err := http.NewRequest(http.MethodGet, "https://example.invalid/error", nil)
	require.NoError(t, err)

	response, err := recorderTransport.RoundTrip(request)
	if response != nil {
		require.NoError(t, response.Body.Close())
	}
	require.Error(t, err)
}

func TestEnsureCassetteExistsReportsStatErrors(t *testing.T) {
	originalStatFile := statFile
	statFile = func(string) (os.FileInfo, error) {
		return nil, os.ErrPermission
	}
	t.Cleanup(func() { statFile = originalStatFile })

	err := ensureCassetteExists(filepath.Join(t.TempDir(), "cassette"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "check failed")
}

func TestRedactHeadersNoopOnNil(t *testing.T) {
	redactHeaders(nil)
}

func TestRedactHeadersKeepsNonSensitiveHeaders(t *testing.T) {
	headers := http.Header{"X-Trace-Id": []string{"abc123"}}
	redactHeaders(headers)
	require.Equal(t, http.Header{"X-Trace-Id": []string{"abc123"}}, headers)
}

func TestContainsHeader(t *testing.T) {
	sensitive := []string{"authorization", "x-api-key"}
	require.True(t, containsHeader(sensitive, "Authorization"))
	require.False(t, containsHeader(sensitive, "X-Other"))
}

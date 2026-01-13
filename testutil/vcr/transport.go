package vcr

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/dnaeon/go-vcr.v2/cassette"
	"gopkg.in/dnaeon/go-vcr.v2/recorder"
)

type recorderTransport struct {
	cassetteBase  string
	record        bool
	realTransport http.RoundTripper

	initOnce sync.Once
	initErr  error
	recorder *recorder.Recorder

	stopOnce sync.Once
	stopErr  error
}

var (
	statFile    = os.Stat
	newRecorder = recorder.NewAsMode
)

func newRecorderTransport(scenario *cassetteDetails, realTransport http.RoundTripper) *recorderTransport {
	return &recorderTransport{
		cassetteBase:  scenario.basePath,
		record:        scenario.record,
		realTransport: realTransport,
	}
}

func (transport *recorderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := transport.ensureRecorder(); err != nil {
		return nil, err
	}
	return transport.recorder.RoundTrip(request)
}

func (transport *recorderTransport) Close() error {
	if transport == nil {
		return nil
	}
	transport.stopOnce.Do(func() {
		if transport.recorder == nil {
			return
		}
		transport.stopErr = transport.recorder.Stop()
	})
	return transport.stopErr
}

func (transport *recorderTransport) ensureRecorder() error {
	if transport == nil {
		return errors.New("vcr transport is nil")
	}
	transport.initOnce.Do(func() {
		if transport.realTransport == nil {
			transport.initErr = errors.New("vcr transport has nil real transport")
			return
		}

		if !transport.record {
			if err := ensureCassetteExists(transport.cassetteBase); err != nil {
				transport.initErr = err
				return
			}
		}

		mode := recorder.ModeReplaying
		if transport.record {
			mode = recorder.ModeRecording
		}

		recorderInstance, err := newRecorder(transport.cassetteBase, mode, transport.realTransport)
		if err != nil {
			transport.initErr = err
			return
		}
		recorderInstance.SkipRequestLatency = true
		applyRedactions(recorderInstance)

		transport.recorder = recorderInstance
	})

	return transport.initErr
}

func ensureCassetteExists(cassetteBase string) error {
	cassettePath := cassetteBase + ".yaml"
	if _, err := statFile(cassettePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vcr cassette missing: %s", cassettePath)
		}
		return fmt.Errorf("vcr cassette check failed: %w", err)
	}
	return nil
}

func applyRedactions(recorderInstance *recorder.Recorder) {
	recorderInstance.AddSaveFilter(func(interaction *cassette.Interaction) error {
		redactHeaders(interaction.Request.Headers)
		redactHeaders(interaction.Response.Headers)
		return nil
	})
}

func redactHeaders(headers http.Header) {
	if headers == nil {
		return
	}
	sensitiveHeaders := []string{
		"authorization",
		"x-api-key",
	}
	for name := range headers {
		if containsHeader(sensitiveHeaders, name) {
			delete(headers, name)
		}
	}
}

func containsHeader(sensitiveHeaders []string, name string) bool {
	for _, candidate := range sensitiveHeaders {
		if strings.EqualFold(candidate, name) {
			return true
		}
	}
	return false
}

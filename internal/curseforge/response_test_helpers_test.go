package curseforge

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func writeJSONResponse(t *testing.T, writer http.ResponseWriter, payload any) {
	t.Helper()
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		t.Fatalf("write json response: %v", err)
	}
}

func writeStringResponse(t *testing.T, writer http.ResponseWriter, payload string) {
	t.Helper()
	if _, err := writer.Write([]byte(payload)); err != nil {
		t.Fatalf("write string response: %v", err)
	}
}

type responseDoer struct {
	response *http.Response
	err      error
}

func (d responseDoer) Do(_ *http.Request) (*http.Response, error) {
	return d.response, d.err
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

func (c *closeErrorBody) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *closeErrorBody) Close() error {
	if c.closeErr != nil {
		return c.closeErr
	}
	return nil
}

package curseforge

import (
	"net/http"

	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/perf"

	"go.opentelemetry.io/otel/attribute"
)

const baseURL = "https://api.curseforge.com/v1"

type Client struct {
	client httpclient.Doer
}

func NewClient(doer httpclient.Doer) *Client {
	return &Client{client: doer}
}

func (curseforgeClient *Client) Do(request *http.Request) (*http.Response, error) {
	ctx, span := perf.StartSpan(request.Context(), "api.curseforge.http.request", perf.WithAttributes(attribute.String("url", request.URL.String())))
	defer span.End()
	request.Header.Set("Accept", "application/json")
	request.Header["x-api-key"] = []string{environment.CurseforgeAPIKey()}

	return curseforgeClient.client.Do(request.WithContext(ctx))
}

func GetBaseURL() string {
	return baseURL
}

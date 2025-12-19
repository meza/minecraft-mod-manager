package curseforge

import (
	"net/http"

	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/perf"

	"go.opentelemetry.io/otel/attribute"
)

const baseURL = "https://api.curseforge.com/v1"

type Client struct {
	client httpClient.Doer
}

func NewClient(doer httpClient.Doer) *Client {
	return &Client{client: doer}
}

func (curseforgeClient *Client) Do(request *http.Request) (*http.Response, error) {
	ctx, span := perf.StartSpan(request.Context(), "api.curseforge.http.request", perf.WithAttributes(attribute.String("url", request.URL.String())))
	defer span.End()
	headers := map[string]string{
		"Accept":    "application/json",
		"x-api-key": environment.CurseforgeApiKey(),
	}

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	return curseforgeClient.client.Do(request.WithContext(ctx))
}

func GetBaseUrl() string {
	return baseURL
}

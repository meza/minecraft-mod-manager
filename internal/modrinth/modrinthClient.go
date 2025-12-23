// Package modrinth implements Modrinth API helpers and models.
package modrinth

import (
	"fmt"
	"net/http"

	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"go.opentelemetry.io/otel/attribute"
)

const baseURL = "https://api.modrinth.com"

type Client struct {
	client httpclient.Doer
}

func NewClient(doer httpclient.Doer) *Client {
	return &Client{client: doer}
}

func (modrinthClient *Client) Do(request *http.Request) (*http.Response, error) {
	ctx, span := perf.StartSpan(request.Context(), "api.modrinth.http.request", perf.WithAttributes(attribute.String("url", request.URL.String())))
	defer span.End()
	headers := map[string]string{
		"user-agent":    fmt.Sprintf("github_com/meza/minecraft-mod-manager/%s", environment.AppVersion()),
		"Accept":        "application/json",
		"Authorization": environment.ModrinthAPIKey(),
	}

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	return modrinthClient.client.Do(request.WithContext(ctx))
}

func GetBaseURL() string {
	return baseURL
}

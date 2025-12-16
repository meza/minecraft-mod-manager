package curseforge

import (
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"net/http"
	"os"
)

type Client struct {
	client httpClient.Doer
}

func NewClient(doer httpClient.Doer) *Client {
	return &Client{client: doer}
}

func (curseforgeClient *Client) Do(request *http.Request) (*http.Response, error) {
	region := perf.StartRegionWithDetails("curseforge-api-call", &perf.PerformanceDetails{
		"url": request.URL.String(),
	})
	defer region.End()
	headers := map[string]string{
		"Accept":    "application/json",
		"x-api-key": environment.CurseforgeApiKey(),
	}

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	return curseforgeClient.client.Do(request)
}

func GetBaseUrl() string {
	url, hasUrl := os.LookupEnv("CURSEFORGE_API_URL")
	if hasUrl {
		return url
	}

	return "https://api.curseforge.com/v1"
}

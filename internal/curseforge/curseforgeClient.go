package curseforge

import (
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/environment"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"net/http"
	"os"
)

type Client struct {
	client httpClient.Doer
}

func (curseforgeClient *Client) Do(request *http.Request) (*http.Response, error) {
	headers := map[string]string{
		"user-agent": fmt.Sprintf("github_com/meza/minecraft-mod-manager/%s", environment.AppVersion()),
		"Accept":     "application/json",
		"x-api-key":  environment.CurseforgeApiKey(),
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

package curseforge

import (
	"encoding/json"
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/pkg/errors"
	"net/http"
)

type getProjectResponse struct {
	Data Project `json:"data"`
}

func GetProject(projectId string, client httpClient.Doer) (*Project, error) {
	region := perf.StartRegionWithDetils("curseforge-getproject", &perf.PerformanceDetails{
		"projectId": projectId,
	})
	defer region.End()

	url := fmt.Sprintf("%s/mods/%s", GetBaseUrl(), projectId)
	request, _ := http.NewRequest(http.MethodGet, url, nil)

	response, err := client.Do(request)
	if err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(err, projectId, models.CURSEFORGE)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalErrors.ProjectNotFoundError{
			ProjectID: projectId,
			Platform:  models.CURSEFORGE,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), projectId, models.CURSEFORGE)
	}

	var projectResponse getProjectResponse
	err = json.NewDecoder(response.Body).Decode(&projectResponse)
	if err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Wrap(err, "failed to decode response body"), projectId, models.CURSEFORGE)
	}

	return &projectResponse.Data, nil
}

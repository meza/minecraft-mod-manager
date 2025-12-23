package curseforge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/pkg/errors"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
)

type getProjectResponse struct {
	Data Project `json:"data"`
}

func GetProject(ctx context.Context, projectId string, client httpClient.Doer) (project *Project, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.project.get", perf.WithAttributes(attribute.String("project_id", projectId)))
	defer span.End()

	url := fmt.Sprintf("%s/mods/%s", GetBaseUrl(), projectId)
	timeoutCtx, cancel := httpClient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		if httpClient.IsTimeoutError(err) {
			return nil, httpClient.WrapTimeoutError(err)
		}
		return nil, globalErrors.ProjectApiErrorWrap(err, projectId, models.CURSEFORGE)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

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

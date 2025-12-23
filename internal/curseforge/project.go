package curseforge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/pkg/errors"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
)

type getProjectResponse struct {
	Data Project `json:"data"`
}

func GetProject(ctx context.Context, projectID string, client httpclient.Doer) (project *Project, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.project.get", perf.WithAttributes(attribute.String("project_id", projectID)))
	defer span.End()

	url := fmt.Sprintf("%s/mods/%s", GetBaseURL(), projectID)
	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		if httpclient.IsTimeoutError(err) {
			return nil, httpclient.WrapTimeoutError(err)
		}
		return nil, globalerrors.ProjectAPIErrorWrap(err, projectID, models.CURSEFORGE)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalerrors.ProjectNotFoundError{
			ProjectID: projectID,
			Platform:  models.CURSEFORGE,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalerrors.ProjectAPIErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), projectID, models.CURSEFORGE)
	}

	var projectResponse getProjectResponse
	err = json.NewDecoder(response.Body).Decode(&projectResponse)
	if err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(errors.Wrap(err, "failed to decode response body"), projectID, models.CURSEFORGE)
	}

	return &projectResponse.Data, nil
}

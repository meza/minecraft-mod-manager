package curseforge

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/meza/minecraft-mod-manager/internal/globalerrors"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/urlbuilder"
	"github.com/pkg/errors"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
)

type getProjectResponse struct {
	Data Project `json:"data"`
}

// GetProject fetches a CurseForge project by ID for downstream selection logic.
// It does no caching and returns the raw project payload as returned by the API.
//
// It returns ProjectNotFoundError when the project does not exist and ProjectAPIError
// when the request fails, the status is non-200, or the response cannot be decoded.
//
// Example:
//
//	project, err := curseforge.GetProject(ctx, "1234", client)
func GetProject(ctx context.Context, projectID string, client httpclient.Doer) (project *Project, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.curseforge.project.get", perf.WithAttributes(attribute.String("project_id", projectID)))
	defer span.End()

	requestURL, err := buildProjectURL(projectID)
	if err != nil {
		return nil, err
	}
	timeoutCtx, cancel := httpclient.WithMetadataTimeout(ctx)
	defer cancel()
	request, err := newRequestWithContext(timeoutCtx, http.MethodGet, requestURL.String(), nil)
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
		return nil, globalerrors.ProjectAPIErrorWrap(httpclient.NewResponseError(response), projectID, models.CURSEFORGE)
	}

	var projectResponse getProjectResponse
	err = json.NewDecoder(response.Body).Decode(&projectResponse)
	if err != nil {
		return nil, globalerrors.ProjectAPIErrorWrap(errors.Wrap(err, "failed to decode response body"), projectID, models.CURSEFORGE)
	}

	return &projectResponse.Data, nil
}

func buildProjectURL(projectID string) (*url.URL, error) {
	baseURL, err := parseURL(GetBaseURL())
	if err != nil {
		return nil, err
	}
	requestURL := urlbuilder.JoinEscapedPath(baseURL, "mods", projectID)
	return requestURL, nil
}

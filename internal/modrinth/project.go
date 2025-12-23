package modrinth

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

type ProjectStatus string
type ProjectType string
type ProjectEnvironment string

const (
	Approved ProjectStatus = "approved"
	Rejected ProjectStatus = "rejected"
	Pending  ProjectStatus = "pending"
)

const (
	Mod          ProjectType = "mod"
	Modpack      ProjectType = "modpack"
	ResourcePack ProjectType = "resourcepack"
	Datapack     ProjectType = "datapack"
	Shader       ProjectType = "shader"
)

const (
	Required    ProjectEnvironment = "required"
	Optional    ProjectEnvironment = "optional"
	Unsupported ProjectEnvironment = "unsupported"
)

type Project struct {
	Id           string             `json:"id"`
	Title        string             `json:"title"`
	Slug         string             `json:"slug"`
	Description  string             `json:"description"`
	Categories   []string           `json:"categories"`
	ClientSide   ProjectEnvironment `json:"client_side"`
	ServerSide   ProjectEnvironment `json:"server_side"`
	Status       ProjectStatus      `json:"status"`
	Type         ProjectType        `json:"project_type"`
	GameVersions []string           `json:"game_versions"`
	Loaders      []models.Loader    `json:"loaders"`
}

func GetProject(ctx context.Context, projectId string, client httpClient.Doer) (project *Project, returnErr error) {
	ctx, span := perf.StartSpan(ctx, "api.modrinth.project.get", perf.WithAttributes(attribute.String("project_id", projectId)))
	defer span.End()

	url := fmt.Sprintf("%s/v2/project/%s", GetBaseUrl(), projectId)

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
		return nil, globalErrors.ProjectApiErrorWrap(err, projectId, models.MODRINTH)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && returnErr == nil {
			returnErr = closeErr
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalErrors.ProjectNotFoundError{
			ProjectID: projectId,
			Platform:  models.MODRINTH,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), projectId, models.MODRINTH)
	}

	project = &Project{}
	if err := json.NewDecoder(response.Body).Decode(project); err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Wrap(err, "failed to decode response body"), projectId, models.MODRINTH)
	}
	return project, nil

}

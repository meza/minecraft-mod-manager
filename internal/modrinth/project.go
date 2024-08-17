package modrinth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/globalErrors"
	"github.com/meza/minecraft-mod-manager/internal/httpClient"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/pkg/errors"
	"net/http"
	"runtime/trace"
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

func GetProject(projectId string, client httpClient.Doer) (*Project, error) {
	ctx := context.WithValue(context.Background(), "projectId", projectId)
	region := trace.StartRegion(ctx, "modrinth-getproject")
	defer region.End()

	url := fmt.Sprintf("%s/v2/project/%s", GetBaseUrl(), projectId)

	request, _ := http.NewRequest(http.MethodGet, url, nil)

	response, err := client.Do(request)
	if err != nil {
		return nil, globalErrors.ProjectApiErrorWrap(err, projectId, models.MODRINTH)
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, &globalErrors.ProjectNotFoundError{
			ProjectID: projectId,
			Platform:  models.MODRINTH,
		}
	}

	if response.StatusCode != http.StatusOK {
		return nil, globalErrors.ProjectApiErrorWrap(errors.Errorf("unexpected status code: %d", response.StatusCode), projectId, models.MODRINTH)
	}

	result := &Project{}
	json.NewDecoder(response.Body).Decode(result)
	defer response.Body.Close()
	return result, nil

}

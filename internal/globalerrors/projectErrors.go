// Package globalerrors defines shared cross-platform error types.
package globalerrors

import (
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

type ProjectNotFoundError struct {
	ProjectID string
	Platform  models.Platform
}

func (projectError *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("Project not found on %s: %s", projectError.Platform, projectError.ProjectID)
}

func (projectError *ProjectNotFoundError) Is(target error) bool {
	t, ok := target.(*ProjectNotFoundError)
	if !ok {
		return false
	}
	return projectError.ProjectID == t.ProjectID && projectError.Platform == t.Platform
}

//

type ProjectAPIError struct {
	ProjectID string
	Platform  models.Platform
	Err       error
}

func (apiError *ProjectAPIError) Error() string {
	return fmt.Sprintf("Project cannot be fetched due to an api error on %s: %s", apiError.Platform, apiError.ProjectID)
}

func (apiError *ProjectAPIError) Is(target error) bool {
	t, ok := target.(*ProjectAPIError)
	if !ok {
		return false
	}
	return apiError.ProjectID == t.ProjectID && apiError.Platform == t.Platform
}

func (apiError *ProjectAPIError) Unwrap() error {
	return apiError.Err
}

func ProjectAPIErrorWrap(err error, projectID string, platform models.Platform) error {
	return &ProjectAPIError{
		ProjectID: projectID,
		Platform:  platform,
		Err:       err,
	}
}

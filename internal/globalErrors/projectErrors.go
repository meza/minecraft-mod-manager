package globalErrors

import (
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

type ProjectNotFoundError struct {
	ProjectID string
	Platform  models.Platform
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("Project not found on %s: %s", e.Platform, e.ProjectID)
}

func (e *ProjectNotFoundError) Is(target error) bool {
	t, ok := target.(*ProjectNotFoundError)
	if !ok {
		return false
	}
	return e.ProjectID == t.ProjectID && e.Platform == t.Platform
}

//

type ProjectApiError struct {
	ProjectID string
	Platform  models.Platform
	Err       error
}

func (e *ProjectApiError) Error() string {
	return fmt.Sprintf("Project cannot be fetched due to an api error on %s: %s", e.Platform, e.ProjectID)
}

func (e *ProjectApiError) Is(target error) bool {
	t, ok := target.(*ProjectApiError)
	if !ok {
		return false
	}
	return e.ProjectID == t.ProjectID && e.Platform == t.Platform
}

func (e *ProjectApiError) Unwrap() error {
	return e.Err
}

func ProjectApiErrorWrap(err error, projectId string, platform models.Platform) error {
	return &ProjectApiError{
		ProjectID: projectId,
		Platform:  platform,
		Err:       err,
	}
}

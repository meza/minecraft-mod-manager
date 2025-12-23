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

type ProjectAPIError struct {
	ProjectID string
	Platform  models.Platform
	Err       error
}

func (e *ProjectAPIError) Error() string {
	return fmt.Sprintf("Project cannot be fetched due to an api error on %s: %s", e.Platform, e.ProjectID)
}

func (e *ProjectAPIError) Is(target error) bool {
	t, ok := target.(*ProjectAPIError)
	if !ok {
		return false
	}
	return e.ProjectID == t.ProjectID && e.Platform == t.Platform
}

func (e *ProjectAPIError) Unwrap() error {
	return e.Err
}

func ProjectAPIErrorWrap(err error, projectID string, platform models.Platform) error {
	return &ProjectAPIError{
		ProjectID: projectID,
		Platform:  platform,
		Err:       err,
	}
}

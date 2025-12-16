package platform

import (
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

type UnknownPlatformError struct {
	Platform string
}

func (e *UnknownPlatformError) Error() string {
	return fmt.Sprintf("unknown platform: %s", e.Platform)
}

type ModNotFoundError struct {
	Platform  models.Platform
	ProjectID string
}

func (e *ModNotFoundError) Error() string {
	return fmt.Sprintf("mod not found on %s: %s", e.Platform, e.ProjectID)
}

type NoCompatibleFileError struct {
	Platform  models.Platform
	ProjectID string
}

func (e *NoCompatibleFileError) Error() string {
	return fmt.Sprintf("no compatible file found on %s for %s", e.Platform, e.ProjectID)
}

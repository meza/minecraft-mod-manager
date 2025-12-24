package platform

import (
	"fmt"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

type UnknownPlatformError struct {
	Platform string
}

func (platformError *UnknownPlatformError) Error() string {
	return fmt.Sprintf("unknown platform: %s", platformError.Platform)
}

type ModNotFoundError struct {
	Platform  models.Platform
	ProjectID string
}

func (platformError *ModNotFoundError) Error() string {
	return fmt.Sprintf("mod not found on %s: %s", platformError.Platform, platformError.ProjectID)
}

type NoCompatibleFileError struct {
	Platform  models.Platform
	ProjectID string
}

func (platformError *NoCompatibleFileError) Error() string {
	return fmt.Sprintf("no compatible file found on %s for %s", platformError.Platform, platformError.ProjectID)
}

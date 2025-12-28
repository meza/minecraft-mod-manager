package models

import "fmt"

// ModNotFoundError indicates a project ID that is not present on a platform.
type ModNotFoundError struct {
	Platform  Platform
	ProjectID string
}

func (platformError *ModNotFoundError) Error() string {
	return fmt.Sprintf("mod not found on %s: %s", platformError.Platform, platformError.ProjectID)
}

// NoCompatibleFileError indicates that no file matches the selection constraints.
type NoCompatibleFileError struct {
	Platform  Platform
	ProjectID string
}

func (platformError *NoCompatibleFileError) Error() string {
	return fmt.Sprintf("no compatible file found on %s for %s", platformError.Platform, platformError.ProjectID)
}

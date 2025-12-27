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

type ModNotFoundError = models.ModNotFoundError

type NoCompatibleFileError = models.NoCompatibleFileError

//go:build windows

package config

import (
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMetadataModsFolderPathKeepsWindowsDriveAbsolute(t *testing.T) {
	meta := NewMetadata(`C:\home\user\modlist.json`)
	cfg := models.ModsJSON{ModsFolder: `C:\var\mc\mods`}
	assert.Equal(t, `C:\var\mc\mods`, meta.ModsFolderPath(cfg))
}

package config

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMetadataLockPathUsesConfigBasename(t *testing.T) {
	meta := NewMetadata(filepath.FromSlash("/home/user/modlist.json"))
	assert.Equal(t, filepath.FromSlash("/home/user/modlist-lock.json"), meta.LockPath())
}

func TestMetadataModsFolderPathResolvesRelativeAgainstConfigDir(t *testing.T) {
	meta := NewMetadata(filepath.FromSlash("/home/user/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	assert.Equal(t, filepath.FromSlash("/home/user/mods"), meta.ModsFolderPath(cfg))
}

func TestMetadataModsFolderPathKeepsAbsolute(t *testing.T) {
	meta := NewMetadata(filepath.FromSlash("/home/user/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: filepath.FromSlash("/var/mc/mods")}
	assert.Equal(t, filepath.FromSlash("/var/mc/mods"), meta.ModsFolderPath(cfg))
}

func TestMetadataModsFolderPathKeepsRootedForwardSlash(t *testing.T) {
	meta := NewMetadata(filepath.FromSlash("/home/user/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "/var/mc/mods"}
	assert.Equal(t, "/var/mc/mods", meta.ModsFolderPath(cfg))
}

func TestMetadataModsFolderPathKeepsWindowsDriveAbsolute(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only coverage for drive-letter absolute paths")
	}

	meta := NewMetadata(`C:\home\user\modlist.json`)
	cfg := models.ModsJSON{ModsFolder: `C:\var\mc\mods`}
	assert.Equal(t, `C:\var\mc\mods`, meta.ModsFolderPath(cfg))
}

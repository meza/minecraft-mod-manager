package config

import (
	"path/filepath"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

type Metadata struct {
	ConfigPath string
}

func NewMetadata(configPath string) Metadata {
	return Metadata{ConfigPath: configPath}
}

func (metadata Metadata) Dir() string {
	return filepath.Dir(filepath.FromSlash(metadata.ConfigPath))
}

func (metadata Metadata) LockPath() string {
	base := filepath.Base(metadata.ConfigPath)
	baseNoExt := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(metadata.Dir(), baseNoExt+"-lock.json")
}

func (metadata Metadata) ModsFolderPath(config models.ModsJSON) string {
	if isAbsoluteOrRootedPath(config.ModsFolder) {
		return config.ModsFolder
	}
	return filepath.Join(metadata.Dir(), config.ModsFolder)
}

func isAbsoluteOrRootedPath(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\")
}

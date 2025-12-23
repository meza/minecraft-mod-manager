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

func (m Metadata) Dir() string {
	return filepath.Dir(filepath.FromSlash(m.ConfigPath))
}

func (m Metadata) LockPath() string {
	base := filepath.Base(m.ConfigPath)
	baseNoExt := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(m.Dir(), baseNoExt+"-lock.json")
}

func (m Metadata) ModsFolderPath(config models.ModsJSON) string {
	if isAbsoluteOrRootedPath(config.ModsFolder) {
		return config.ModsFolder
	}
	return filepath.Join(m.Dir(), config.ModsFolder)
}

func isAbsoluteOrRootedPath(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\")
}

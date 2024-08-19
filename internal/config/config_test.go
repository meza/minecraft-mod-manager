package config

import (
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func home() string {
	home, _ := os.UserHomeDir()
	return home
}

func TestGetModsFolder(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		config     models.ModsJson
		expected   string
	}{
		{
			name:       "Absolute Mods Folder",
			configPath: filepath.FromSlash("/home/user/.minecraft/modlist.json"),
			config: models.ModsJson{
				ModsFolder: filepath.Join(home(), "mods"),
			},
			expected: filepath.Join(home(), "mods"),
		},
		{
			name:       "Relative Mods Folder",
			configPath: filepath.FromSlash("/home/user/.minecraft/modlist.json"),
			config: models.ModsJson{
				ModsFolder: "./mods",
			},
			expected: filepath.FromSlash("/home/user/.minecraft/mods"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetModsFolder(tt.configPath, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureConfigurationNotExistingInQuiet(t *testing.T) {
	fs := afero.NewMemMapFs()

	_, err := EnsureConfiguration(filepath.FromSlash("/modlist.json"), true, fs)

	var cf *ConfigFileNotFoundException
	assert.ErrorAs(t, err, &cf)

}

func TestEnsureConfigurationReadIssues(t *testing.T) {
	fs := afero.NewMemMapFs()
	configFile := "modlist"
	_, err := EnsureConfiguration(configFile, true, fs)

	assert.ErrorContains(t, err, "failed to read configuration file")
	assert.ErrorContains(t, err, "file does not exist")

}

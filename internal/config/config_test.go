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

func TestEnsureConfiguration(t *testing.T) {
	t.Run("Happy Path", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = afero.WriteFile(fs, "/modlist.json", []byte(`{"modsFolder": "./mods"}`), 0644)
		config, err := GetConfiguration(filepath.FromSlash("/modlist.json"), true, fs)

		assert.NoError(t, err)
		assert.Equal(t, "./mods", config.ModsFolder)
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = afero.WriteFile(fs, "/modlist.json", []byte("malformed json"), 0644)
		_, err := GetConfiguration(filepath.FromSlash("/modlist.json"), true, fs)

		assert.ErrorContains(t, err, "invalid character")
	})

	t.Run("Configuration read issue", func(t *testing.T) {
		configFile := filepath.Join(t.TempDir(), "config")
		_ = os.Mkdir(configFile, 0755)
		_, err := GetConfiguration(configFile, true)

		defer os.Remove(configFile)

		assert.ErrorContains(t, err, "failed to read configuration file")
	})

	t.Run("Configuration file not found", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_, err := GetConfiguration(filepath.FromSlash("/modlist.json"), true, fs)

		var cf *ConfigFileNotFoundException
		assert.ErrorAs(t, err, &cf)
	})

}

func TestInitializeConfigFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	config, err := InitializeConfigFile(filepath.FromSlash("/modlist.json"), fs)

	assert.NoError(t, err)
	assert.Equal(t, "mods", config.ModsFolder)

	data, err := afero.ReadFile(fs, filepath.FromSlash("/modlist.json"))
	assert.NoError(t, err)

	assert.JSONEq(t, `{"loader":"forge","gameVersion":"1.21.1","defaultAllowedReleaseTypes":["release","beta"],"modsFolder":"mods","mods":[]}`, string(data))

	t.Run("File Unwritable", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "config")
		fs := afero.NewReadOnlyFs(afero.NewOsFs())
		_, err := InitializeConfigFile(file, fs)

		assert.ErrorContains(t, err, "operation not permitted")
	})

}

func TestEnsureLockFile(t *testing.T) {
	t.Run("Happy Path", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = afero.WriteFile(fs, "/modlist.lock", []byte{}, 0644)
		_, err := EnsureLockFile(filepath.FromSlash("/modlist.json"), fs)

		assert.NoError(t, err)
	})

	t.Run("Lock file not found", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		empty, err := EnsureLockFile(filepath.FromSlash("/modlist.json"), fs)

		assert.NoError(t, err)
		assert.Equal(t, []models.ModInstall(nil), empty)

		data, err := afero.ReadFile(fs, filepath.FromSlash("/modlist-lock.json"))
		assert.NoError(t, err)
		assert.JSONEq(t, "[]", string(data))

	})

	t.Run("Lock file unwritable", func(t *testing.T) {
		fs := afero.NewReadOnlyFs(afero.NewOsFs())
		_, err := EnsureLockFile(filepath.FromSlash("/modlist.json"), fs)

		assert.ErrorContains(t, err, "operation not permitted")
	})

	t.Run("Lock file read error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = afero.WriteFile(fs, filepath.FromSlash("/modlist-lock.json"), []byte("malformed json"), 0644)
		_, err := EnsureLockFile(filepath.FromSlash("/modlist.json"), fs)

		assert.ErrorContains(t, err, "invalid character")
	})
}

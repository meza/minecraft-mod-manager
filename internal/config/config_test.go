package config

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type doerFunc func(request *http.Request) (*http.Response, error)

func (d doerFunc) Do(request *http.Request) (*http.Response, error) {
	return d(request)
}

func latestMinecraftVersionClient(version string) doerFunc {
	return func(request *http.Request) (*http.Response, error) {
		body := io.NopCloser(strings.NewReader(`{"latest":{"release":"` + version + `","snapshot":"x"},"versions":[]}`))
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	}
}

func TestReadConfigMissingFileReturnsNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	_, err := ReadConfig(fs, meta)
	var notFound *ConfigFileNotFoundException
	assert.ErrorAs(t, err, &notFound)
}

func TestReadConfigMalformedJSONReturnsInvalid(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))
	_ = afero.WriteFile(fs, meta.ConfigPath, []byte("not json"), 0644)

	_, err := ReadConfig(fs, meta)
	var invalid *ConfigFileInvalidError
	assert.ErrorAs(t, err, &invalid)
}

func TestInitConfigUsesLatestMinecraftVersionAndReadConfigRoundTrip(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	minecraft.ClearManifestCache()
	created, err := InitConfig(fs, meta, latestMinecraftVersionClient("9.9.9"))
	assert.NoError(t, err)
	assert.Equal(t, "9.9.9", created.GameVersion)

	read, err := ReadConfig(fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, created, read)
}

func TestWriteConfigOverwrites(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	minecraft.ClearManifestCache()
	created, err := InitConfig(fs, meta, latestMinecraftVersionClient("1.21.1"))
	assert.NoError(t, err)

	created.GameVersion = "1.20.1"
	err = WriteConfig(fs, meta, created)
	assert.NoError(t, err)

	read, err := ReadConfig(fs, meta)
	assert.NoError(t, err)
	assert.Equal(t, "1.20.1", read.GameVersion)
}

func TestReadConfigReturnsErrorWhenPathIsDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))
	assert.NoError(t, fs.Mkdir(meta.ConfigPath, 0755))

	_, err := ReadConfig(fs, meta)
	assert.Error(t, err)
}

func TestReadConfigReturnsReadErrorWhenPathIsDirectoryOnOsFs(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "modlist.json")
	assert.NoError(t, afero.NewOsFs().MkdirAll(configDir, 0755))
	meta := NewMetadata(configDir)

	_, err := ReadConfig(afero.NewOsFs(), meta)
	assert.Error(t, err)
}

func TestWriteConfigReturnsErrorWhenPathIsDirectory(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	err := WriteConfig(fs, meta, models.ModsJson{Loader: models.FABRIC})
	assert.Error(t, err)
}

func TestInitConfigReturnsErrorWhenPathIsDirectory(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	minecraft.ClearManifestCache()
	_, err := InitConfig(fs, meta, latestMinecraftVersionClient("1.21.1"))
	assert.Error(t, err)
}

func TestInitConfigErrorsWhenLatestMinecraftVersionErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := NewMetadata(filepath.FromSlash("/modlist.json"))

	minecraftClient := doerFunc(func(request *http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})

	minecraft.ClearManifestCache()
	_, err := InitConfig(fs, meta, minecraftClient)
	assert.Error(t, err)
	exists, existsErr := afero.Exists(fs, meta.ConfigPath)
	assert.NoError(t, existsErr)
	assert.False(t, exists)
}

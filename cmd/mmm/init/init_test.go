package init

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type doerFunc func(request *http.Request) (*http.Response, error)

func (d doerFunc) Do(request *http.Request) (*http.Response, error) {
	return d(request)
}

func manifestDoer(versions []string) doerFunc {
	return func(request *http.Request) (*http.Response, error) {
		if len(versions) == 0 {
			versions = []string{"1.0.0"}
		}

		items := make([]string, 0, len(versions))
		for _, v := range versions {
			items = append(items, `{"id":"`+v+`"}`)
		}
		body := `{"latest":{"release":"` + versions[0] + `"},"versions":[` + strings.Join(items, ",") + `]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

type fakePrompter struct {
	overwrite  bool
	confirmErr error
	newPath    string
	newPathErr error
}

func (p fakePrompter) ConfirmOverwrite(configPath string) (bool, error) {
	return p.overwrite, p.confirmErr
}

func (p fakePrompter) RequestNewConfigPath(configPath string) (string, error) {
	return p.newPath, p.newPathErr
}

func TestInitWithDeps(t *testing.T) {
	t.Run("writes config and empty lock", func(t *testing.T) {
		minecraft.ClearManifestCache()
		fs := afero.NewMemMapFs()
		meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
		assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

		_, err := initWithDeps(initOptions{
			ConfigPath:   meta.ConfigPath,
			Loader:       models.FABRIC,
			GameVersion:  "1.21.1",
			ReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
			ModsFolder:   "mods",
		}, initDeps{
			fs:              fs,
			minecraftClient: manifestDoer([]string{"1.21.1"}),
		})
		assert.NoError(t, err)

		cfg, err := config.ReadConfig(fs, meta)
		assert.NoError(t, err)
		assert.Equal(t, models.FABRIC, cfg.Loader)
		assert.Equal(t, "1.21.1", cfg.GameVersion)
		assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, cfg.DefaultAllowedReleaseTypes)
		assert.Equal(t, "mods", cfg.ModsFolder)
		assert.Empty(t, cfg.Mods)

		lock, err := config.ReadLock(fs, meta)
		assert.NoError(t, err)
		assert.Empty(t, lock)
	})

	t.Run("missing required flags returns error", func(t *testing.T) {
		minecraft.ClearManifestCache()
		_, err := initWithDeps(initOptions{}, initDeps{fs: afero.NewMemMapFs(), minecraftClient: manifestDoer([]string{"1.21.1"})})
		assert.ErrorContains(t, err, "init requires flag")
	})

	t.Run("missing mods folder returns error", func(t *testing.T) {
		minecraft.ClearManifestCache()
		fs := afero.NewMemMapFs()

		_, err := initWithDeps(initOptions{
			ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
			Loader:       models.FABRIC,
			GameVersion:  "1.21.1",
			ReleaseTypes: []models.ReleaseType{models.Release},
			ModsFolder:   "mods",
		}, initDeps{fs: fs, minecraftClient: manifestDoer([]string{"1.21.1"})})
		assert.ErrorContains(t, err, "mods folder does not exist")
	})

	t.Run("invalid minecraft version returns error", func(t *testing.T) {
		minecraft.ClearManifestCache()
		fs := afero.NewMemMapFs()
		assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

		_, err := initWithDeps(initOptions{
			ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
			Loader:       models.FABRIC,
			GameVersion:  "1.21.9",
			ReleaseTypes: []models.ReleaseType{models.Release},
			ModsFolder:   "mods",
		}, initDeps{fs: fs, minecraftClient: manifestDoer([]string{"1.21.1"})})
		assert.ErrorContains(t, err, "invalid minecraft version")
	})

	t.Run("config exists with --quiet returns error", func(t *testing.T) {
		minecraft.ClearManifestCache()
		fs := afero.NewMemMapFs()
		assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
		assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/modlist.json"), []byte(`{"existing":true}`), 0644))

		_, err := initWithDeps(initOptions{
			ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
			Quiet:        true,
			Loader:       models.FABRIC,
			GameVersion:  "1.21.1",
			ReleaseTypes: []models.ReleaseType{models.Release},
			ModsFolder:   "mods",
		}, initDeps{fs: fs, minecraftClient: manifestDoer([]string{"1.21.1"}), prompter: fakePrompter{overwrite: true}})
		assert.ErrorContains(t, err, "already exists")
	})

	t.Run("config exists and overwrite replaces config", func(t *testing.T) {
		minecraft.ClearManifestCache()
		fs := afero.NewMemMapFs()
		assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
		assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/modlist.json"), []byte(`{"loader":"forge"}`), 0644))

		meta, err := initWithDeps(initOptions{
			ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
			Loader:       models.FABRIC,
			GameVersion:  "1.21.1",
			ReleaseTypes: []models.ReleaseType{models.Release},
			ModsFolder:   "mods",
		}, initDeps{
			fs:              fs,
			minecraftClient: manifestDoer([]string{"1.21.1"}),
			prompter:        fakePrompter{overwrite: true},
		})
		assert.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("/cfg/modlist.json"), meta.ConfigPath)

		cfg, err := config.ReadConfig(fs, config.NewMetadata(meta.ConfigPath))
		assert.NoError(t, err)
		assert.Equal(t, models.FABRIC, cfg.Loader)
	})

	t.Run("config exists and choose new path writes to new file", func(t *testing.T) {
		minecraft.ClearManifestCache()
		fs := afero.NewMemMapFs()
		assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
		assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/modlist.json"), []byte(`{"loader":"forge"}`), 0644))

		meta, err := initWithDeps(initOptions{
			ConfigPath:   filepath.FromSlash("/cfg/modlist.json"),
			Loader:       models.FABRIC,
			GameVersion:  "1.21.1",
			ReleaseTypes: []models.ReleaseType{models.Release},
			ModsFolder:   "mods",
		}, initDeps{
			fs:              fs,
			minecraftClient: manifestDoer([]string{"1.21.1"}),
			prompter:        fakePrompter{overwrite: false, newPath: filepath.FromSlash("/cfg/alt.json")},
		})
		assert.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("/cfg/alt.json"), meta.ConfigPath)

		originalBytes, err := afero.ReadFile(fs, filepath.FromSlash("/cfg/modlist.json"))
		assert.NoError(t, err)
		assert.Contains(t, string(originalBytes), `"loader":"forge"`)

		newCfg, err := config.ReadConfig(fs, config.NewMetadata(filepath.FromSlash("/cfg/alt.json")))
		assert.NoError(t, err)
		assert.Equal(t, models.FABRIC, newCfg.Loader)
	})
}

func TestTerminalPrompter(t *testing.T) {
	t.Run("confirm overwrite yes", func(t *testing.T) {
		var out bytes.Buffer
		p := terminalPrompter{
			in:  strings.NewReader("y\n"),
			out: &out,
		}

		ok, err := p.ConfirmOverwrite("modlist.json")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Contains(t, out.String(), "Overwrite?")
	})

	t.Run("confirm overwrite EOF returns error", func(t *testing.T) {
		var out bytes.Buffer
		p := terminalPrompter{
			in:  strings.NewReader(""),
			out: &out,
		}

		ok, err := p.ConfirmOverwrite("modlist.json")
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestLoaderFlag(t *testing.T) {
	t.Run("accepts valid loader", func(t *testing.T) {
		var flag loaderFlag
		err := flag.Set("fabric")
		assert.NoError(t, err)
		assert.Equal(t, models.FABRIC, flag.value)
	})

	t.Run("rejects invalid loader", func(t *testing.T) {
		var flag loaderFlag
		err := flag.Set("nope")
		assert.ErrorContains(t, err, "invalid loader")
		assert.Empty(t, flag.value)
	})
}

func TestParseReleaseTypes(t *testing.T) {
	t.Run("parses list", func(t *testing.T) {
		actual, err := parseReleaseTypes([]string{"release", "beta"})
		assert.NoError(t, err)
		assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, actual)
	})

	t.Run("rejects invalid release type", func(t *testing.T) {
		_, err := parseReleaseTypes([]string{"release", "nope"})
		assert.ErrorContains(t, err, "invalid release type")
	})

	t.Run("rejects empty list", func(t *testing.T) {
		_, err := parseReleaseTypes([]string{""})
		assert.ErrorContains(t, err, "release types cannot be empty")
	})
}

package interaction

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUnmanagedNoticeReturnsEmptyWhenNoFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))

	notice, err := BuildUnmanagedNotice(fs, meta, cfg, nil, view.ColorDisabled)
	require.NoError(t, err)
	assert.Empty(t, notice.Files)
	assert.Equal(t, "", notice.Message)
}

func TestBuildUnmanagedNoticeIncludesFilesAndMessage(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	notice, err := BuildUnmanagedNotice(fs, meta, cfg, nil, view.ColorDisabled)
	require.NoError(t, err)
	require.Len(t, notice.Files, 1)
	assert.Contains(t, notice.Message, "unmanaged.jar")
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (fs statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(fs.failPath) {
		return nil, fs.err
	}
	return fs.Fs.Stat(name)
}

func TestBuildUnmanagedNoticeIfModsFolderMissingReturnsEmpty(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	notice, err := BuildUnmanagedNoticeIfModsFolderExists(fs, meta, cfg, nil, view.ColorDisabled)
	require.NoError(t, err)
	assert.Empty(t, notice.Files)
	assert.Empty(t, notice.Message)
}

func TestBuildUnmanagedNoticeIfModsFolderMissingReturnsErrorForOtherFailures(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, baseFs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(baseFs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	failPath := filepath.Join(meta.Dir(), ".mmmignore")
	wrapped := statErrorFs{Fs: baseFs, failPath: failPath, err: errors.New("stat failed")}

	_, err := BuildUnmanagedNoticeIfModsFolderExists(wrapped, meta, cfg, nil, view.ColorDisabled)
	assert.Error(t, err)
}

func TestBuildUnmanagedNoticeIfModsFolderExistsReturnsNotice(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}

	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	notice, err := BuildUnmanagedNoticeIfModsFolderExists(fs, meta, cfg, nil, view.ColorDisabled)
	require.NoError(t, err)
	assert.Len(t, notice.Files, 1)
	assert.Contains(t, notice.Message, "unmanaged.jar")
}

func TestRenderUnmanagedNoticeReturnsEmptyForNoFiles(t *testing.T) {
	output := RenderUnmanagedNotice(nil, view.ColorDisabled)
	assert.Equal(t, "", output)
}

func TestRenderUnmanagedNoticeReturnsEmptyOnWriteErrors(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})

	for failAt := 1; failAt <= 7; failAt++ {
		callCount := 0
		view.WriteString = func(io.Writer, string) error {
			callCount++
			if callCount == failAt {
				return errors.New("write failed")
			}
			return nil
		}

		output := RenderUnmanagedNotice([]string{"one.jar"}, view.ColorDisabled)
		assert.Equal(t, "", output)
	}
}

func TestRenderUnmanagedNoticeReturnsEmptyOnSeparatorWriteError(t *testing.T) {
	originalWriteString := view.WriteString
	t.Cleanup(func() {
		view.WriteString = originalWriteString
	})

	callCount := 0
	view.WriteString = func(io.Writer, string) error {
		callCount++
		if callCount == 4 {
			return errors.New("write failed")
		}
		return nil
	}

	output := RenderUnmanagedNotice([]string{"one.jar", "two.jar"}, view.ColorDisabled)
	assert.Equal(t, "", output)
}

func TestRenderUnmanagedNoticeWithColorEnabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)

	output := RenderUnmanagedNotice([]string{"one.jar"}, view.ColorEnabled)
	assert.Contains(t, output, "cmd.list.unmanaged.cta")
}

func TestRenderUnmanagedNoticeIncludesMultipleFiles(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	output := RenderUnmanagedNotice([]string{"one.jar", "two.jar"}, view.ColorDisabled)
	assert.Contains(t, output, "one.jar")
	assert.Contains(t, output, "two.jar")
}

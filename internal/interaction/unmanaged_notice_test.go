package interaction

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestLogUnmanagedNoticeReturnsErrorWhenOutputMissing(t *testing.T) {
	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Message: "missing output",
	})

	assert.Error(t, err)
}

func TestLogUnmanagedNoticeUsesErrorOutput(t *testing.T) {
	var out bytes.Buffer
	var errBuffer bytes.Buffer
	outWriter := output.New(&out, &errBuffer, false)

	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Output:   outWriter,
		Message:  "unmanaged error",
		UseError: true,
	})

	assert.NoError(t, err)
	assert.Empty(t, out.String())
	assert.Equal(t, "unmanaged error\n", errBuffer.String())
}

func TestLogUnmanagedNoticeUsesLogOutput(t *testing.T) {
	var out bytes.Buffer
	var errBuffer bytes.Buffer
	outWriter := output.New(&out, &errBuffer, false)

	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Output:     outWriter,
		Message:    "unmanaged log",
		UseError:   false,
		Visibility: output.LogForce,
	})

	assert.NoError(t, err)
	assert.Equal(t, "unmanaged log\n", out.String())
	assert.Empty(t, errBuffer.String())
}

func TestLogUnmanagedNoticeDefaultsToLogForce(t *testing.T) {
	var out bytes.Buffer
	var errBuffer bytes.Buffer
	outWriter := output.New(&out, &errBuffer, true)

	err := LogUnmanagedNotice(UnmanagedNoticeOptions{
		Output:  outWriter,
		Message: "forced output",
	})

	assert.NoError(t, err)
	assert.Equal(t, "forced output\n", out.String())
	assert.Empty(t, errBuffer.String())
}

func TestEmitUnmanagedNoticeSkipsEmptyMessage(t *testing.T) {
	called := false
	err := EmitUnmanagedNotice("   ", func([]string) error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.False(t, called)
}

func TestEmitUnmanagedNoticeReturnsErrorWhenWriterMissing(t *testing.T) {
	err := EmitUnmanagedNotice("notice", nil)
	assert.Error(t, err)
}

func TestEmitUnmanagedNoticeWritesMessage(t *testing.T) {
	var lines []string
	err := EmitUnmanagedNotice("notice", func(written []string) error {
		lines = append(lines, written...)
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"notice"}, lines)
}

func TestRequireNoUnmanagedFilesReturnsErrWhenUnmanagedFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	var lines []string
	notice, err := RequireNoUnmanagedFiles(UnmanagedGateInput{
		Fs:                     fs,
		Meta:                   meta,
		Config:                 cfg,
		Lock:                   []models.ModInstall{},
		ColorMode:              view.ColorDisabled,
		Write:                  func(written []string) error { lines = append(lines, written...); return nil },
		AllowMissingModsFolder: true,
	})

	assert.ErrorIs(t, err, ErrUnmanagedFiles)
	assert.NotEmpty(t, notice.Files)
	assert.NotEmpty(t, lines)
}

func TestRequireNoUnmanagedFilesSkipsMissingModsFolderWhenAllowed(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	var lines []string
	notice, err := RequireNoUnmanagedFiles(UnmanagedGateInput{
		Fs:                     fs,
		Meta:                   meta,
		Config:                 cfg,
		Lock:                   []models.ModInstall{},
		ColorMode:              view.ColorDisabled,
		Write:                  func(written []string) error { lines = append(lines, written...); return nil },
		AllowMissingModsFolder: true,
	})

	assert.NoError(t, err)
	assert.Empty(t, notice.Files)
	assert.Empty(t, lines)
}

func TestRequireNoUnmanagedFilesReturnsErrorWhenWriterMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "unmanaged.jar"), []byte("data"), 0644))

	_, err := RequireNoUnmanagedFiles(UnmanagedGateInput{
		Fs:                     fs,
		Meta:                   meta,
		Config:                 cfg,
		Lock:                   []models.ModInstall{},
		ColorMode:              view.ColorDisabled,
		Write:                  nil,
		AllowMissingModsFolder: true,
	})

	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrUnmanagedFiles))
}

func TestRequireNoUnmanagedFilesReturnsErrorWhenModsFolderMissingAndStrict(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	cfg := models.ModsJSON{
		Loader:      models.FABRIC,
		GameVersion: "1.20.1",
		ModsFolder:  "mods",
	}

	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, []models.ModInstall{}))

	_, err := RequireNoUnmanagedFiles(UnmanagedGateInput{
		Fs:                     fs,
		Meta:                   meta,
		Config:                 cfg,
		Lock:                   []models.ModInstall{},
		ColorMode:              view.ColorDisabled,
		Write:                  func([]string) error { return nil },
		AllowMissingModsFolder: false,
	})

	assert.Error(t, err)
}

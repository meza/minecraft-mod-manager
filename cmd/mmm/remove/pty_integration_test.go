//go:build !windows

package remove

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func TestRemoveCommandInteractivePTYIncludesSummary(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restoreColors := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.Ascii })
	t.Cleanup(restoreColors)

	originalRunRemoveProgram := runRemoveProgram
	runRemoveProgram = defaultRunRemoveProgram
	t.Cleanup(func() { runRemoveProgram = originalRunRemoveProgram })

	master, slave, err := pty.Open()
	require.NoError(t, err)
	t.Cleanup(func() { closePTY(t, master) })
	t.Cleanup(func() { closePTY(t, slave) })

	require.NoError(t, pty.Setsize(master, &pty.Winsize{Cols: 120, Rows: 40}))

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")
	meta := config.NewMetadata(configPath)
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods: []models.Mod{
			{ID: "mod-a", Name: "Mod A", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"},
	}

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("mod"), 0644))

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(slave)
	cmd.SetOut(slave)
	cmd.SetErr(slave)
	cmd.SetArgs([]string{"--config", configPath, "mod-a"})

	var output bytes.Buffer
	readDone := make(chan struct{})
	readErr := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&output, master)
		readErr <- copyErr
		close(readDone)
	}()

	execErr := cmd.Execute()
	require.NoError(t, execErr)
	closePTY(t, slave)

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for PTY output")
	}
	require.NoError(t, normalizePTYReadError(<-readErr))

	normalized := stripControlSequences(output.String())
	require.Contains(t, normalized, "cmd.remove.summary.success")
}

func stripControlSequences(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	value = stripOSCSequences(value)
	value = stripCSISequences(value)

	return value
}

func closePTY(t *testing.T, file *os.File) {
	if file == nil {
		return
	}
	if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		require.NoError(t, err)
	}
}

func normalizePTYReadError(err error) error {
	if err == nil || errors.Is(err, syscall.EIO) {
		return nil
	}
	return err
}

var oscSequence = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

func stripOSCSequences(value string) string {
	return oscSequence.ReplaceAllString(value, "")
}

var csiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripCSISequences(value string) string {
	return csiSequence.ReplaceAllString(value, "")
}

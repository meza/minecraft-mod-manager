package remove

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/testutil/terminal"
	terminalpty "github.com/meza/minecraft-mod-manager/testutil/terminal/pty"
)

func TestRemoveCommandInteractivePTYIncludesSummary(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunRemoveProgram := runRemoveProgram
	runRemoveProgram = defaultRunRemoveProgram
	t.Cleanup(func() { runRemoveProgram = originalRunRemoveProgram })

	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: 40}))
	require.NotNil(t, session)

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
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"--config", configPath, "--force", "mod-a"})

	execErr := cmd.Execute()
	require.NoError(t, execErr)
	session.WaitForOutputAndClose(t, func(output []byte) bool {
		return strings.Contains(string(output), "cmd.remove.summary.success")
	}, terminalpty.WithWaitDuration(2*time.Second))

	normalized := terminal.NormalizeOutput(session.OutputString(), terminal.NormalizeOptions{StripControlSequences: true})
	require.Contains(t, normalized, "cmd.remove.summary.success")
}

func TestRemoveCommandInteractivePTYConfirmSnapshotShortHeight(t *testing.T) {
	runRemovePTYConfirmSnapshot(t, 25)
}

func TestRemoveCommandInteractivePTYConfirmSnapshotTallHeight(t *testing.T) {
	runRemovePTYConfirmSnapshot(t, tallSnapshotRows())
}

func TestRemoveCommandInteractivePTYSuccessSnapshotShortHeight(t *testing.T) {
	runRemovePTYSuccessSnapshot(t, 25)
}

func TestRemoveCommandInteractivePTYSuccessSnapshotTallHeight(t *testing.T) {
	runRemovePTYSuccessSnapshot(t, tallSnapshotRows())
}

func TestRemoveCommandInteractivePTYSuccessLongListShowsHeaderWhenShort(t *testing.T) {
	terminal.ApplyFixtures(t)

	originalRunRemoveProgram := runRemoveProgram
	runRemoveProgram = defaultRunRemoveProgram
	t.Cleanup(func() { runRemoveProgram = originalRunRemoveProgram })

	rows := uint16(12)
	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	configPath := writeRemoveLongListFixture(t, 30)

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"--config", configPath, "--force", "mod-*"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	waitForRemoveOutput(t, session, "cmd.remove.summary.success")

	select {
	case err := <-execErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remove success")
	}
	require.NoError(t, session.Close())

	normalized := terminal.NormalizeOutput(trimToLastFrame(session.OutputString()), terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	require.Contains(t, normalized, "cmd.remove.header.result")
}

func runRemovePTYConfirmSnapshot(t *testing.T, rows uint16) {
	t.Setenv("LANG", "en_GB.UTF-8")
	terminal.ApplyFixtures(t)

	originalRunRemoveProgram := runRemoveProgram
	runRemoveProgram = defaultRunRemoveProgram
	t.Cleanup(func() { runRemoveProgram = originalRunRemoveProgram })

	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	configPath := writeRemoveFixture(t)

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"--config", configPath, "mod-a"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	waitForRemoveOutput(t, session, "cmd.remove.confirm.question")
	_, writeErr := session.SendInput([]byte("\r"))
	require.NoError(t, writeErr)

	waitForRemoveOutput(t, session, "cmd.remove.cancelled")

	select {
	case err := <-execErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remove cancel")
	}
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeRemovePromptSnapshot(session.OutputString()))
}

func runRemovePTYSuccessSnapshot(t *testing.T, rows uint16) {
	t.Setenv("LANG", "en_GB.UTF-8")
	terminal.ApplyFixtures(t)

	originalRunRemoveProgram := runRemoveProgram
	runRemoveProgram = defaultRunRemoveProgram
	t.Cleanup(func() { runRemoveProgram = originalRunRemoveProgram })

	session := terminalpty.NewSession(t, terminalpty.WithSize(terminal.Size{Columns: 120, Rows: int(rows)}))
	require.NotNil(t, session)

	configPath := writeRemoveFixture(t)

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(session.Input())
	cmd.SetOut(session.Output())
	cmd.SetErr(session.Output())
	cmd.SetArgs([]string{"--config", configPath, "mod-a"})

	execErr := make(chan error, 1)
	go func() {
		execErr <- cmd.Execute()
	}()

	waitForRemoveOutput(t, session, "cmd.remove.confirm.question")
	yesShort := i18n.T("cmd.init.prompt.option.yes.short", nil)
	_, writeErr := session.SendInput([]byte(yesShort + "\r"))
	require.NoError(t, writeErr)

	waitForRemoveOutput(t, session, "cmd.remove.summary.success")

	select {
	case err := <-execErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remove success")
	}
	require.NoError(t, session.Close())

	snaps.MatchSnapshot(t, normalizeRemoveSuccessSnapshot(session.OutputString()))
}

func writeRemoveFixture(t *testing.T) string {
	t.Helper()

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
			{ID: "mod-b", Name: "Mod B", Type: models.MODRINTH},
		},
	}
	lock := []models.ModInstall{
		{ID: "mod-a", Type: models.MODRINTH, FileName: "mod-a.jar"},
		{ID: "mod-b", Type: models.MODRINTH, FileName: "mod-b.jar"},
	}

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-a.jar"), []byte("mod"), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), "mod-b.jar"), []byte("mod"), 0644))

	return configPath
}

func writeRemoveLongListFixture(t *testing.T, count int) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "modlist.json")
	meta := config.NewMetadata(configPath)
	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       make([]models.Mod, 0, count),
	}
	lock := make([]models.ModInstall, 0, count)
	for index := 0; index < count; index++ {
		modID := fmt.Sprintf("mod-%02d", index+1)
		modName := fmt.Sprintf("Mod %02d", index+1)
		cfg.Mods = append(cfg.Mods, models.Mod{ID: modID, Name: modName, Type: models.MODRINTH})
		lock = append(lock, models.ModInstall{ID: modID, Type: models.MODRINTH, FileName: modID + ".jar"})
	}

	fs := afero.NewOsFs()
	require.NoError(t, fs.MkdirAll(meta.Dir(), 0755))
	require.NoError(t, fs.MkdirAll(meta.ModsFolderPath(cfg), 0755))
	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, config.WriteLock(context.Background(), fs, meta, lock))
	for _, entry := range lock {
		require.NoError(t, afero.WriteFile(fs, filepath.Join(meta.ModsFolderPath(cfg), entry.FileName), []byte("mod"), 0644))
	}

	return configPath
}

func waitForRemoveOutput(t *testing.T, session *terminalpty.Session, needle string) {
	t.Helper()

	session.WaitForOutput(t, func(output []byte) bool {
		normalized := terminal.NormalizeOutput(string(output), terminal.NormalizeOptions{
			StripControlSequences: true,
		})
		return strings.Contains(normalized, needle)
	}, terminalpty.WithWaitDuration(2*time.Second))
}

func normalizeRemovePromptSnapshot(value string) string {
	normalized := terminal.NormalizeOutput(trimToLastFrame(value), terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	if headerIndex := strings.LastIndex(normalized, "cmd.remove.header.confirm"); headerIndex >= 0 {
		return strings.TrimSpace(normalized[headerIndex:])
	}
	return normalized
}

func normalizeRemoveSuccessSnapshot(value string) string {
	normalized := terminal.NormalizeOutput(trimToLastFrame(value), terminal.NormalizeOptions{
		StripControlSequences:  true,
		TrimTrailingWhitespace: true,
		TrimTrailingEmptyLines: true,
		TrimSpace:              true,
	})
	return strings.TrimSpace(extractRemoveSuccessSummary(normalized))
}

func extractRemoveSuccessSummary(normalized string) string {
	if headerIndex := strings.LastIndex(normalized, "cmd.remove.header.confirm"); headerIndex >= 0 {
		normalized = strings.TrimSpace(normalized[headerIndex:])
	}
	return reorderRemoveResultHeader(collapseDuplicateLines(normalized))
}

func trimToLastFrame(value string) string {
	indices := cursorHomeSequence.FindAllStringIndex(value, -1)
	if len(indices) == 0 {
		return value
	}
	return value[indices[len(indices)-1][0]:]
}

func tallSnapshotRows() uint16 {
	if runtime.GOOS == "windows" {
		return 40
	}
	return 80
}

var cursorHomeSequence = regexp.MustCompile(`\x1b\[[0-9;]*H`)

func collapseDuplicateLines(value string) string {
	lines := strings.Split(value, "\n")
	output := make([]string, 0, len(lines))
	lastLine := ""
	for _, line := range lines {
		if strings.Contains(line, "cmd.init.prompt.confirm.suffix .yes.short") {
			continue
		}
		if strings.Contains(line, "cmd.remove.header.removing") {
			continue
		}
		if line == lastLine {
			continue
		}
		output = append(output, line)
		lastLine = line
	}
	return strings.Join(output, "\n")
}

func reorderRemoveResultHeader(value string) string {
	lines := strings.Split(value, "\n")
	headerIndex := -1
	firstItemIndex := -1
	for lineIndex, line := range lines {
		if line == "cmd.remove.header.result" {
			headerIndex = lineIndex
		}
		if firstItemIndex == -1 && (strings.HasPrefix(line, "\u2705") || strings.HasPrefix(line, "\u274c")) {
			firstItemIndex = lineIndex
		}
	}
	if headerIndex == -1 || firstItemIndex == -1 || headerIndex < firstItemIndex {
		return value
	}
	header := lines[headerIndex]
	lines = append(lines[:headerIndex], lines[headerIndex+1:]...)
	if headerIndex < firstItemIndex {
		firstItemIndex--
	}
	lines = append(lines[:firstItemIndex], append([]string{header}, lines[firstItemIndex:]...)...)
	return strings.Join(lines, "\n")
}

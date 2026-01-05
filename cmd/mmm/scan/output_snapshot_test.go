package scan

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

func TestScanInitCanceledOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	restoreTerminal := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	originalRunner := runInteractiveInit
	runInteractiveInit = func(context.Context, *cobra.Command, initCmd.InteractiveInitDeps, initCmd.InteractiveInitOptions) error {
		return initCmd.ErrInitCanceled
	}
	t.Cleanup(func() {
		runInteractiveInit = originalRunner
	})

	configPath := filepath.FromSlash("./modlist.json")

	out := &fakeTerminalWriter{}
	errOut := &fakeTerminalWriter{}
	in := &fakeTerminalReader{}
	_, err := in.WriteString("cmd.init.prompt.option.yes.short\n")
	assert.NoError(t, err)

	cmd := Command()
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetArgs([]string{"--config", configPath})

	err = cmd.Execute()
	assert.NoError(t, err)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		normalizeSnapshotOutput(strings.TrimSpace(out.String())),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func normalizeSnapshotOutput(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}

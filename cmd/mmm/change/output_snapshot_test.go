package change

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/tui"
)

func TestChangeCommandPlainOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	restore := tui.SetIsTerminalFuncForTesting(func(int) bool { return true })
	defer restore()

	outputWriter := &terminalWriter{}
	errorWriter := &bytes.Buffer{}
	cmd := commandWithRunner(func(_ context.Context, _ *cobra.Command, _ changeOptions, deps changeDeps) (changeResult, error) {
		return changeResult{}, deps.output.Log("hello", output.LogForce)
	})

	cmd.Flags().String("config", "./modlist.json", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")

	cmd.SetIn(fdReader{fd: 1})
	cmd.SetOut(outputWriter)
	cmd.SetErr(errorWriter)
	cmd.SetArgs([]string{"--config", "./modlist.json"})

	assert.NoError(t, cmd.Execute())

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		strings.TrimSpace(outputWriter.String()),
		strings.TrimSpace(errorWriter.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

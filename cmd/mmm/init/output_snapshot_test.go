package init

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/config"
)

func TestInitCommandRuntimeErrorOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return errors.New("boom")
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	err := cmd.Execute()
	assert.ErrorContains(t, err, "boom")

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		strings.TrimSpace(out.String()),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

func TestInitCommandCanceledOutputSnapshot(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	cmd := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return ErrInitCanceled
	})
	addPersistentFlagsForTesting(cmd)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	err := cmd.Execute()
	assert.NoError(t, err)

	snapshot := fmt.Sprintf(
		"stdout:\n%s\nstderr:\n%s",
		strings.TrimSpace(out.String()),
		strings.TrimSpace(errOut.String()),
	)
	snaps.MatchSnapshot(t, snapshot)
}

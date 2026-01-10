package update

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunUpdateCommandReturnsFlagError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	setCommandOutputForTesting(cmd)

	assert.Error(t, runUpdateCommand(cmd))
}

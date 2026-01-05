package change

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestNewChangeDepsUsesDedicatedDownloadClient(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	clients := platform.DefaultClients(nil)
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Clients: clients,
	})

	deps := newChangeDeps(common)
	assert.NotNil(t, deps.downloadClient)
	assert.NotEqual(t, deps.clients.Modrinth, deps.downloadClient)

	downloadClient, ok := deps.downloadClient.(*httpclient.RLHTTPClient)
	assert.True(t, ok)
	assert.Same(t, common.Limiter, downloadClient.RateLimiter)
}

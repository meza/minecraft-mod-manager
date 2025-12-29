package cmddeps

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestNewCommonDeps_UsesDefaults(t *testing.T) {
	t.Parallel()

	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&outBuffer)
	cmd.SetErr(&errBuffer)

	deps := NewCommonDeps(cmd, CommonDepsOptions{Quiet: false, Debug: false})

	require.NotNil(t, deps.FS)
	require.NotNil(t, deps.Output)
	require.NotNil(t, deps.Logger)
	require.NotNil(t, deps.Clients.Modrinth)
	require.NotNil(t, deps.Clients.Curseforge)
	require.NotNil(t, deps.MinecraftClient)
	require.NotNil(t, deps.Limiter)

	require.NoError(t, deps.Output.Log("hello", output.LogQuiet))
	require.NoError(t, deps.Output.Error("oops"))
	assert.Contains(t, outBuffer.String(), "hello")
	assert.Contains(t, errBuffer.String(), "oops")
}

func TestNewCommonDeps_RespectsOverrides(t *testing.T) {
	t.Parallel()

	filesystem := afero.NewMemMapFs()
	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	customOutput := output.New(&outBuffer, &errBuffer, true)
	customLogger := logger.New(&outBuffer, &errBuffer, true, true)
	customLimiter := rate.NewLimiter(rate.Every(time.Second), 2)
	customClients := platform.Clients{
		Modrinth:   http.DefaultClient,
		Curseforge: http.DefaultClient,
	}
	customMinecraftClient := http.DefaultClient

	deps := NewCommonDeps(nil, CommonDepsOptions{
		FS:              filesystem,
		Output:          customOutput,
		Logger:          customLogger,
		Clients:         customClients,
		MinecraftClient: customMinecraftClient,
		Limiter:         customLimiter,
	})

	assert.Same(t, filesystem, deps.FS)
	assert.Same(t, customOutput, deps.Output)
	assert.Same(t, customLogger, deps.Logger)
	assert.Same(t, customClients.Modrinth, deps.Clients.Modrinth)
	assert.Same(t, customClients.Curseforge, deps.Clients.Curseforge)
	assert.Same(t, customMinecraftClient, deps.MinecraftClient)
	assert.Same(t, customLimiter, deps.Limiter)
}

func TestNewCommonDeps_UsesProvidedWritersWhenCmdNil(t *testing.T) {
	t.Parallel()

	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer

	deps := NewCommonDeps(nil, CommonDepsOptions{
		OutWriter: &outBuffer,
		ErrWriter: &errBuffer,
		Quiet:     false,
		Debug:     false,
	})

	require.NoError(t, deps.Output.Log("hello", output.LogQuiet))
	require.NoError(t, deps.Output.Error("oops"))
	assert.Contains(t, outBuffer.String(), "hello")
	assert.Contains(t, errBuffer.String(), "oops")
}

// Package cmddeps provides shared command dependencies with consistent defaults.
package cmddeps

import (
	"io"

	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/platform"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

// CommonDeps holds shared command dependencies that should be wired consistently.
// Use it to build per-command deps without reimplementing default setup.
type CommonDeps struct {
	FS              afero.Fs
	Output          *output.Output
	Logger          *logger.Logger
	Clients         platform.Clients
	MinecraftClient httpclient.Doer
	Limiter         *rate.Limiter
}

// CommonDepsOptions lets callers override defaults for CommonDeps.
// Zero values are replaced by standard defaults.
type CommonDepsOptions struct {
	FS              afero.Fs
	Output          *output.Output
	Logger          *logger.Logger
	Clients         platform.Clients
	MinecraftClient httpclient.Doer
	Limiter         *rate.Limiter
	OutWriter       io.Writer
	ErrWriter       io.Writer
	Quiet           bool
	Debug           bool
}

// NewCommonDeps constructs CommonDeps using provided options and command I/O.
// It ensures all fields are initialized to non-nil defaults.
func NewCommonDeps(cmd *cobra.Command, options CommonDepsOptions) CommonDeps {
	outWriter := options.OutWriter
	if outWriter == nil {
		outWriter = writerOrDiscard(cmd, (*cobra.Command).OutOrStdout)
	}

	errWriter := options.ErrWriter
	if errWriter == nil {
		errWriter = writerOrDiscard(cmd, (*cobra.Command).ErrOrStderr)
	}

	if options.Output == nil {
		options.Output = output.New(outWriter, errWriter, options.Quiet && !options.Debug)
	}
	if options.Logger == nil {
		options.Logger = logger.New(outWriter, errWriter, false, options.Debug)
	}

	filesystem := options.FS
	if filesystem == nil {
		filesystem = afero.NewOsFs()
	}

	limiter := options.Limiter
	if limiter == nil {
		limiter = httpclient.DefaultLimiter()
	}

	clients := options.Clients
	if clients.Modrinth == nil && clients.Curseforge == nil {
		clients = platform.DefaultClients(limiter)
	}

	minecraftClient := options.MinecraftClient
	if minecraftClient == nil {
		minecraftClient = httpclient.NewRLClient(limiter)
	}

	return CommonDeps{
		FS:              filesystem,
		Output:          options.Output,
		Logger:          options.Logger,
		Clients:         clients,
		MinecraftClient: minecraftClient,
		Limiter:         limiter,
	}
}

func writerOrDiscard(cmd *cobra.Command, selector func(*cobra.Command) io.Writer) io.Writer {
	if cmd == nil {
		return io.Discard
	}
	return selector(cmd)
}

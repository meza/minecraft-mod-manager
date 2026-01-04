package add

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

func TestResolveRemoteModReturnsLoggerError(t *testing.T) {
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true)
	inputs := addResolveInputs{
		ctx: context.Background(),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		opts: addOptions{},
		deps: addDeps{
			logger: log,
		},
		platformValue: models.MODRINTH,
		projectID:     "abc",
	}

	_, err := resolveRemoteMod(context.Background(), inputs)
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsLogFailureErrorOnFetch(t *testing.T) {
	writeErr := errors.New("write failed")
	log := logger.New(errorWriter{err: writeErr}, errorWriter{err: writeErr}, false, true)
	inputs := addResolveInputs{
		ctx: context.Background(),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		opts: addOptions{},
		deps: addDeps{
			logger: log,
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, errors.New("fetch failed")
			},
		},
		platformValue: models.MODRINTH,
		projectID:     "abc",
	}

	_, err := resolveRemoteMod(context.Background(), inputs)
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsLogFailureErrorAfterFetch(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := &errAfterWriter{remaining: 1, err: writeErr}
	log := logger.New(writer, writer, false, true)
	inputs := addResolveInputs{
		ctx: context.Background(),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		opts: addOptions{},
		deps: addDeps{
			logger: log,
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, errors.New("fetch failed")
			},
		},
		platformValue: models.MODRINTH,
		projectID:     "abc",
	}

	_, err := resolveRemoteMod(context.Background(), inputs)
	assert.ErrorIs(t, err, writeErr)
}

func TestResolveRemoteModReturnsRemoteMod(t *testing.T) {
	inputs := addResolveInputs{
		ctx: context.Background(),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		opts: addOptions{},
		deps: addDeps{
			logger: logger.New(io.Discard, io.Discard, false, true),
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{Name: "Example"}, nil
			},
		},
		platformValue: models.MODRINTH,
		projectID:     "abc",
	}

	resolved, err := resolveRemoteMod(context.Background(), inputs)
	assert.NoError(t, err)
	assert.Equal(t, "Example", resolved.remoteMod.Name)
	assert.Equal(t, models.MODRINTH, resolved.platform)
}

func TestResolveRemoteModReturnsFetchError(t *testing.T) {
	fetchErr := errors.New("fetch failed")
	inputs := addResolveInputs{
		ctx: context.Background(),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		opts: addOptions{},
		deps: addDeps{
			logger: logger.New(io.Discard, io.Discard, false, true),
			fetchMod: func(context.Context, models.Platform, string, platform.FetchOptions, platform.Clients) (platform.RemoteMod, error) {
				return platform.RemoteMod{}, fetchErr
			},
		},
		platformValue: models.MODRINTH,
		projectID:     "abc",
	}

	_, err := resolveRemoteMod(context.Background(), inputs)
	assert.ErrorIs(t, err, fetchErr)
}

func TestHandleResolveFailureNonInteractiveNotFound(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(output)
	cmd.SetErr(output)

	runState := addRunState{
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeNonTTY,
	}
	deps := addDeps{
		logger: logger.New(output, output, false, true),
	}

	outcome := handleResolveFailure(cmd, runState, deps, models.MODRINTH, "abc", &platform.ModNotFoundError{
		Platform:  models.MODRINTH,
		ProjectID: "abc",
	})
	assert.False(t, outcome.recovered)
	assert.True(t, clierrors.IsHandled(outcome.err))
	assert.Contains(t, output.String(), "cmd.add.error.not_found_unattended")
	assert.Contains(t, output.String(), "cmd.add.error.not_found_hint")
}

func TestHandleResolveFailureInteractiveReturnsRecovery(t *testing.T) {
	runState := addRunState{
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeInteractive,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	deps := addDeps{
		clients: platform.DefaultClients(rate.NewLimiter(rate.Inf, 0)),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return recoveryFlowModel{
				selectedPlatform: models.CURSEFORGE,
				selectedProject:  "def",
			}, nil
		},
	}

	outcome := handleResolveFailure(cmd, runState, deps, models.MODRINTH, "abc", &platform.ModNotFoundError{
		Platform:  models.MODRINTH,
		ProjectID: "abc",
	})
	assert.NoError(t, outcome.err)
	assert.True(t, outcome.recovered)
	assert.Equal(t, models.CURSEFORGE, outcome.platformValue)
	assert.Equal(t, "def", outcome.projectID)
}

func TestHandleRecoveryPromptMissingRunner(t *testing.T) {
	runState := addRunState{
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeInteractive,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome := handleRecoveryPrompt(cmd, runState, addDeps{}, recoveryReasonNotFound, models.MODRINTH, "abc", errors.New("boom"))
	assert.False(t, outcome.recovered)
	assert.True(t, clierrors.IsHandled(outcome.err))
}

func TestHandleRecoveryPromptUnsupportedReasonNonInteractive(t *testing.T) {
	runState := addRunState{
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeNonTTY,
	}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	outcome := handleRecoveryPrompt(cmd, runState, addDeps{}, recoveryReason(99), models.MODRINTH, "abc", errors.New("boom"))
	assert.False(t, outcome.recovered)
	assert.Error(t, outcome.err)
}

func TestWriteUnattendedResolveFailureNoCompatible(t *testing.T) {
	t.Setenv("MMM_TEST", "true")

	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	runState := addRunState{
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeNonTTY,
	}

	err := writeUnattendedResolveFailure(cmd, runState, addDeps{}, recoveryReasonNoCompatible, models.MODRINTH, "abc")
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "cmd.add.error.no_compatible")
}

func TestWriteUnattendedResolveFailureUnsupportedReason(t *testing.T) {
	output := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	runState := addRunState{
		meta: config.NewMetadata("modlist.json"),
		cfg: models.ModsJSON{
			Loader:      models.FABRIC,
			GameVersion: "1.20.1",
		},
		mode: interaction.ExecutionModeNonTTY,
	}

	err := writeUnattendedResolveFailure(cmd, runState, addDeps{}, recoveryReason(99), models.MODRINTH, "abc")
	assert.Error(t, err)
}

func TestExecutionModeName(t *testing.T) {
	assert.Equal(t, "interactive", interaction.ExecutionModeInteractive.String())
	assert.Equal(t, "unattended", interaction.ExecutionModeUnattended.String())
	assert.Equal(t, "non_tty", interaction.ExecutionModeNonTTY.String())
	assert.Equal(t, "unknown", interaction.ExecutionMode(99).String())
}

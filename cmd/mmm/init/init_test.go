package init

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/minecraft"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/privacy"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	"github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type doerFunc func(request *http.Request) (*http.Response, error)

func (doer doerFunc) Do(request *http.Request) (*http.Response, error) {
	return doer(request)
}

func manifestDoer(versions []string) doerFunc {
	return func(request *http.Request) (*http.Response, error) {
		if len(versions) == 0 {
			versions = []string{"1.0.0"}
		}

		items := make([]string, 0, len(versions))
		for _, v := range versions {
			items = append(items, `{"id":"`+v+`"}`)
		}
		body := `{"latest":{"release":"` + versions[0] + `"},"versions":[` + strings.Join(items, ",") + `]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

func TestGameVersionPlaceholderUsesLatest(t *testing.T) {
	minecraft.ClearManifestCache()

	model := NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.11"}), "")

	assert.Contains(t, model.input.View(), "1.21.11")
	assert.GreaterOrEqual(t, model.input.Width, len("1.21.11"))
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type errorAfterWriter struct {
	err     error
	allowed int
	writes  int
}

func (writer *errorAfterWriter) Write(value []byte) (int, error) {
	writer.writes++
	if writer.writes > writer.allowed {
		return 0, writer.err
	}
	return len(value), nil
}

func TestInitWithDeps(t *testing.T) {
	t.Run("writes config and empty lock", func(t *testing.T) {
		minecraft.ClearManifestCache()
		fs := afero.NewMemMapFs()
		meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
		assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

		err := initWithDeps(context.Background(), initOptions{
			ConfigPath:   meta.ConfigPath,
			Loader:       models.FABRIC,
			GameVersion:  "1.21.1",
			ReleaseTypes: []models.ReleaseType{models.Release, models.Beta},
			ModsFolder:   "mods",
		}, initDeps{
			output:          output.New(io.Discard, io.Discard, true),
			fs:              fs,
			minecraftClient: manifestDoer([]string{"1.21.1"}),
		})
		assert.NoError(t, err)

		cfg, err := config.ReadConfig(context.Background(), fs, meta)
		assert.NoError(t, err)
		assert.Equal(t, models.FABRIC, cfg.Loader)
		assert.Equal(t, "1.21.1", cfg.GameVersion)
		assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, cfg.DefaultAllowedReleaseTypes)
		assert.Equal(t, "mods", cfg.ModsFolder)
		assert.Empty(t, cfg.Mods)

		lock, err := config.ReadLock(context.Background(), fs, meta)
		assert.NoError(t, err)
		assert.Empty(t, lock)
	})

	t.Run("missing required flags returns error", func(t *testing.T) {
		minecraft.ClearManifestCache()
		err := initWithDeps(context.Background(), initOptions{}, initDeps{
			output:          output.New(io.Discard, io.Discard, true),
			fs:              afero.NewMemMapFs(),
			minecraftClient: manifestDoer([]string{"1.21.1"}),
		})
		assert.ErrorContains(t, err, "init requires flag")
	})
}

func TestCommandWithRunnerHandlesCompletionRegistrationError(t *testing.T) {
	originalRegister := registerFlagCompletion
	t.Cleanup(func() {
		registerFlagCompletion = originalRegister
	})
	registerFlagCompletion = func(cmd *cobra.Command, flagName string, completionFunc cobra.CompletionFunc) error {
		return errors.New("completion failed")
	}

	cmd := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	})
	assert.NotNil(t, cmd)
}

func TestCommandWithRunnerHandlesCompletionWriteError(t *testing.T) {
	originalRegister := registerFlagCompletion
	originalWarnWriter := completionWarnWriter
	t.Cleanup(func() {
		registerFlagCompletion = originalRegister
		completionWarnWriter = originalWarnWriter
	})
	registerFlagCompletion = func(cmd *cobra.Command, flagName string, completionFunc cobra.CompletionFunc) error {
		return errors.New("completion failed")
	}
	completionWarnWriter = errorWriter{err: errors.New("write failed")}

	cmd := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	})
	assert.NotNil(t, cmd)
}

func TestCommandWithRunnerHandlesSecondCompletionWriteError(t *testing.T) {
	originalRegister := registerFlagCompletion
	originalWarnWriter := completionWarnWriter
	t.Cleanup(func() {
		registerFlagCompletion = originalRegister
		completionWarnWriter = originalWarnWriter
	})
	callCount := 0
	registerFlagCompletion = func(cmd *cobra.Command, flagName string, completionFunc cobra.CompletionFunc) error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return errors.New("completion failed")
	}
	completionWarnWriter = errorWriter{err: errors.New("write failed")}

	cmd := commandWithRunner(func(context.Context, *cobra.Command, initOptions, initDeps, config.Metadata) error {
		return nil
	})
	assert.NotNil(t, cmd)
}

type fakeTTYReader struct {
	*bytes.Reader
}

func (fakeTTYReader) Fd() uintptr {
	return 0
}

type fakeTTYWriter struct {
	bytes.Buffer
}

func (fakeTTYWriter) Fd() uintptr {
	return 1
}

func TestRunInitCommandRecordsTelemetryUsingFinalOptions(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	var payloads []telemetry.CommandTelemetry
	deps := initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		telemetry: func(payload telemetry.CommandTelemetry) {
			payloads = append(payloads, payload)
		},
	}

	err := runInitCommand(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "latest",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, deps, meta)
	assert.NoError(t, err)

	if assert.Len(t, payloads, 1) {
		payload := payloads[0]
		assert.Equal(t, "init", payload.Command)
		assert.True(t, payload.Success)
		assert.Equal(t, 0, payload.ExitCode)

		args := payload.Arguments
		assert.Equal(t, models.FABRIC, args["loader"])
		assert.Equal(t, "1.21.1", args["gameVersion"])
		assert.Equal(t, []string{"release"}, args["releaseTypes"])
		assert.Equal(t, "mods", args["modsFolder"])
	}
}

func TestBuildTelemetryPayloadRedactsModsFolderUsername(t *testing.T) {
	privacy.ResetForTesting()
	t.Setenv("USER", "alice")

	var modsFolder string
	var expected string
	switch runtime.GOOS {
	case "windows":
		modsFolder = `C:\Users\alice\mods`
		expected = `C:\Users\<user>\mods`
	case "darwin":
		modsFolder = "/Users/alice/mods"
		expected = "/Users/<user>/mods"
	default:
		modsFolder = "/home/alice/mods"
		expected = "/home/<user>/mods"
	}

	payload := buildTelemetryPayload(initOptions{
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   modsFolder,
	}, false, nil)

	args := payload.Arguments
	assert.Equal(t, expected, args["modsFolder"])
}

func TestRunInitCommandDoesNotMarkInteractiveWhenInteractiveFlowWasNotLaunched(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	in := fakeTTYReader{Reader: bytes.NewReader(nil)}
	out := &fakeTTYWriter{}

	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(out)

	var payloads []telemetry.CommandTelemetry
	deps := initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		telemetry: func(payload telemetry.CommandTelemetry) {
			payloads = append(payloads, payload)
		},
	}

	err := runInitCommand(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, deps, meta)
	assert.NoError(t, err)

	if assert.Len(t, payloads, 1) {
		assert.False(t, payloads[0].Interactive)
	}
}

func TestRunInitCommandSwallowsCanceledError(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	in := fakeTTYReader{Reader: bytes.NewReader(nil)}
	out := &fakeTTYWriter{}

	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(out)

	var payloads []telemetry.CommandTelemetry
	deps := initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return CommandModel{state: stateLoader}, nil
		},
		telemetry: func(payload telemetry.CommandTelemetry) {
			payloads = append(payloads, payload)
		},
	}

	err := runInitCommand(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, deps, meta)
	assert.NoError(t, err)

	exists, existsErr := afero.Exists(fs, meta.ConfigPath)
	assert.NoError(t, existsErr)
	assert.False(t, exists)

	if assert.Len(t, payloads, 1) {
		assert.True(t, payloads[0].Success)
	}
}

func TestRunInitUnattendedLatestUnavailableReportsError(t *testing.T) {
	minecraft.ClearManifestCache()

	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "latest",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: doerFunc(func(_ *http.Request) (*http.Response, error) { return nil, errors.New("offline") }),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Contains(t, outBuffer.String(), "Could not resolve latest Minecraft version.")
}

func TestRunInitUnattendedMissingRequiredReportsError(t *testing.T) {
	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		Unattended: true,
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              afero.NewMemMapFs(),
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")))
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Contains(t, outBuffer.String(), "Missing required values for init in unattended mode.")
}

func TestRunInitUnattendedConfigExistsReportsError(t *testing.T) {
	minecraft.ClearManifestCache()

	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	assert.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte(`{"existing":true}`), 0644))

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Contains(t, outBuffer.String(), "Configuration file already exists")
}

func TestRunInitUnattendedInvalidGameVersionReportsError(t *testing.T) {
	minecraft.ClearManifestCache()

	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.12",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Contains(t, outBuffer.String(), "Invalid Minecraft version 1.21.12.")
}

func TestValidateUnattendedInputsGameVersionUnavailableReportsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	minecraft.ClearManifestCache()

	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	_, err := validateUnattendedInputs(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: doerFunc(func(_ *http.Request) (*http.Response, error) { return nil, errors.New("offline") }),
	}, meta)
	assert.Error(t, err)
	var outputErr *unattendedOutputError
	if assert.ErrorAs(t, err, &outputErr) {
		assert.Equal(t, "cmd.init.error.game-version.unavailable", outputErr.messageKey)
		assert.Empty(t, outputErr.hintKey)
	}
}

func TestMissingRequiredUnattended(t *testing.T) {
	assert.True(t, missingRequiredUnattended(initOptions{}))

	assert.True(t, missingRequiredUnattended(initOptions{
		Loader: models.FABRIC,
	}))

	assert.True(t, missingRequiredUnattended(initOptions{
		GameVersion: "1.21.1",
	}))

	assert.False(t, missingRequiredUnattended(initOptions{
		Loader:      models.FABRIC,
		GameVersion: "1.21.1",
	}))
}

func TestValidateModsFolderUnattended(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	statErrFs := statErrorFs{Fs: fs, err: errors.New("stat failed")}
	err := validateModsFolderUnattended(statErrFs, meta, "mods")
	assert.ErrorContains(t, err, "stat failed")

	err = validateModsFolderUnattended(fs, meta, "")
	var emptyErr *modsFolderValidationError
	assert.ErrorAs(t, err, &emptyErr)
	assert.Equal(t, modsFolderEmpty, emptyErr.Kind())

	err = validateModsFolderUnattended(fs, meta, "mods")
	var missingErr *modsFolderValidationError
	assert.ErrorAs(t, err, &missingErr)
	assert.Equal(t, modsFolderMissing, missingErr.Kind())
	assert.Equal(t, filepath.FromSlash("/cfg/mods"), missingErr.Error())

	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg"), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/mods"), []byte("file"), 0644))
	err = validateModsFolderUnattended(fs, meta, "mods")
	var notDirErr *modsFolderValidationError
	assert.ErrorAs(t, err, &notDirErr)
	assert.Equal(t, modsFolderNotDirectory, notDirErr.Kind())
	assert.Equal(t, filepath.FromSlash("/cfg/mods"), notDirErr.Error())

	assert.NoError(t, fs.Remove(filepath.FromSlash("/cfg/mods")))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	info, statErr := fs.Stat(filepath.FromSlash("/cfg/mods"))
	if statErr != nil {
		t.Fatalf("stat failed: %v", statErr)
	}
	sequenceFs := &statSequenceFs{Fs: fs, err: errors.New("stat second call"), info: info}
	err = validateModsFolderUnattended(sequenceFs, meta, "mods")
	assert.ErrorContains(t, err, "stat second call")

	err = validateModsFolderUnattended(fs, meta, "mods")
	assert.NoError(t, err)
}

func TestRunInitUnattendedMissingModsFolderReportsError(t *testing.T) {
	minecraft.ClearManifestCache()

	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods-does-not-exist",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Contains(t, outBuffer.String(), "Mods folder does not exist")
}

func TestRunInitUnattendedEmptyModsFolderReportsError(t *testing.T) {
	minecraft.ClearManifestCache()

	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   " ",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Contains(t, outBuffer.String(), "Mods folder cannot be empty")
}

func TestRunInitUnattendedModsFolderNotDirectoryReportsError(t *testing.T) {
	minecraft.ClearManifestCache()

	outBuffer := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg"), 0755))
	assert.NoError(t, afero.WriteFile(fs, filepath.FromSlash("/cfg/mods"), []byte("file"), 0644))

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Contains(t, outBuffer.String(), "Mods folder path is not a directory")
}

func TestRunInitInteractiveSuccessUsesInteractiveFlow(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	in := fakeTTYReader{Reader: bytes.NewReader(nil)}
	out := &fakeTTYWriter{}

	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(out)

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return CommandModel{
				state: done,
				result: initOptions{
					ConfigPath:   meta.ConfigPath,
					Loader:       models.FABRIC,
					GameVersion:  "1.21.1",
					ReleaseTypes: []models.ReleaseType{models.Release},
					ModsFolder:   "mods",
					Provided: providedFlags{
						Loader:       true,
						GameVersion:  true,
						ReleaseTypes: true,
						ModsFolder:   true,
					},
				},
			}, nil
		},
	}, meta)
	assert.NoError(t, err)
	assert.True(t, didUseInteractiveFlow)
}

func TestRunInitWithDefaultsSkipsPromptsWhenValuesProvided(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	in := fakeTTYReader{Reader: bytes.NewReader(nil)}
	out := &fakeTTYWriter{}

	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(out)

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   false,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(outputLinesModel); ok {
				return model, nil
			}
			return nil, errors.New("unexpected interactive run")
		},
	}, meta)
	assert.NoError(t, err)
	assert.False(t, didUseInteractiveFlow)

	exists, existsErr := afero.Exists(fs, meta.ConfigPath)
	assert.NoError(t, existsErr)
	assert.True(t, exists)
}

func TestRunInitQuietSkipsSuccessOutput(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	in := fakeTTYReader{Reader: bytes.NewReader(nil)}
	outBuffer := &bytes.Buffer{}

	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(outBuffer)
	cmd.SetErr(io.Discard)

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Quiet:        true,
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("unexpected output run")
		},
	}, meta)
	assert.NoError(t, err)
	assert.False(t, didUseInteractiveFlow)
	assert.Empty(t, outBuffer.String())
}

func TestRunInitConfigExistsCheckErrorReturns(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(&fakeTTYWriter{})

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	fs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "latest",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
}

func TestRunInitUnattendedConfigExistsCheckErrorReturns(t *testing.T) {
	minecraft.ClearManifestCache()

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	fs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Unattended:   true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.Error(t, err)
	assert.False(t, didUseInteractiveFlow)
}

func TestValidateUnattendedInputsConfigExistsWithForceSucceeds(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	assert.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte(`{"existing":true}`), 0644))

	cmd := &cobra.Command{}
	cmd.SetErr(io.Discard)

	_, err := validateUnattendedInputs(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Force:        true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.NoError(t, err)
}

func TestValidateUnattendedInputsConfigExistsCheckErrorReturns(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	minecraft.ClearManifestCache()

	cmd := &cobra.Command{}
	cmd.SetErr(io.Discard)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	fs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}

	_, err := validateUnattendedInputs(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.ErrorContains(t, err, "cmd.init.error.config-file.check")
}

func TestUnattendedOutputErrorMessage(t *testing.T) {
	err := &unattendedOutputError{messageKey: "cmd.init.error.unattended.missing_required"}
	assert.Equal(t, "init unattended error", err.Error())
}

func TestOutputLineCmdReturnsErrorForNilWriter(t *testing.T) {
	cmd := outputLineCmd(nil, "line")
	msg := cmd()
	typed, ok := msg.(outputLineErrorMsg)
	assert.True(t, ok)
	assert.ErrorContains(t, typed.err, "output writer is nil")
}

func TestOutputLineCmdReturnsErrorForWriteFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := outputLineCmd(errorWriter{err: writeErr}, "line")
	msg := cmd()
	typed, ok := msg.(outputLineErrorMsg)
	assert.True(t, ok)
	assert.ErrorIs(t, typed.err, writeErr)
}

func TestMarkModsFolderProvided(t *testing.T) {
	options := initOptions{
		ModsFolder: "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
		},
	}
	updated := markModsFolderProvided(executionModeUnattended, options)
	assert.False(t, updated.Provided.ModsFolder)

	options.Provided.ModsFolder = true
	updated = markModsFolderProvided(executionModeInteractive, options)
	assert.True(t, updated.Provided.ModsFolder)

	options = initOptions{
		ModsFolder: "mods",
		Provided: providedFlags{
			GameVersion:  true,
			ReleaseTypes: true,
		},
	}
	updated = markModsFolderProvided(executionModeInteractive, options)
	assert.False(t, updated.Provided.ModsFolder)

	options = initOptions{
		ModsFolder: "",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
		},
	}
	updated = markModsFolderProvided(executionModeInteractive, options)
	assert.False(t, updated.Provided.ModsFolder)

	options = initOptions{
		ModsFolder: "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
		},
	}
	updated = markModsFolderProvided(executionModeInteractive, options)
	assert.True(t, updated.Provided.ModsFolder)
}

func TestResolveExecutionMode(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(&fakeTTYWriter{})

	mode := resolveExecutionMode(initOptions{Unattended: true}, cmd)
	assert.Equal(t, executionModeUnattended, mode)

	mode = resolveExecutionMode(initOptions{}, cmd)
	assert.Equal(t, executionModeInteractive, mode)

	cmd.SetIn(bytes.NewReader(nil))
	mode = resolveExecutionMode(initOptions{}, cmd)
	assert.Equal(t, executionModeNonTTY, mode)
}

func TestShouldRunWithoutPrompt(t *testing.T) {
	options := initOptions{
		Loader:      models.FABRIC,
		GameVersion: "1.21.1",
		ReleaseTypes: []models.ReleaseType{
			models.Release,
		},
		ModsFolder: "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}
	assert.True(t, shouldRunWithoutPrompt(configMissing, options))
	assert.False(t, shouldRunWithoutPrompt(configPresent, options))

	options.Force = true
	assert.True(t, shouldRunWithoutPrompt(configPresent, options))

	options = initOptions{}
	assert.False(t, shouldRunWithoutPrompt(configMissing, options))
}

func TestHasAllRequiredInputs(t *testing.T) {
	assert.False(t, hasAllRequiredInputs(initOptions{}))

	assert.False(t, hasAllRequiredInputs(initOptions{
		GameVersion: "1.21.1",
		ReleaseTypes: []models.ReleaseType{
			models.Release,
		},
		ModsFolder: "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}))

	assert.False(t, hasAllRequiredInputs(initOptions{
		Loader: models.FABRIC,
		ReleaseTypes: []models.ReleaseType{
			models.Release,
		},
		ModsFolder: "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}))

	assert.False(t, hasAllRequiredInputs(initOptions{
		Loader:      models.FABRIC,
		GameVersion: "1.21.1",
		ModsFolder:  "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}))

	assert.False(t, hasAllRequiredInputs(initOptions{
		Loader:      models.FABRIC,
		GameVersion: "1.21.1",
		ReleaseTypes: []models.ReleaseType{
			models.Release,
		},
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}))

	assert.True(t, hasAllRequiredInputs(initOptions{
		Loader:      models.FABRIC,
		GameVersion: "1.21.1",
		ReleaseTypes: []models.ReleaseType{
			models.Release,
		},
		ModsFolder: "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}))
}

func TestRunInitWithoutPromptReturnsOutputError(t *testing.T) {
	writeErr := errors.New("write failed")
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err := runInitWithoutPrompt(context.Background(), cmd, initOptions{}, initDeps{
		fs:              afero.NewMemMapFs(),
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return outputLinesModel{err: writeErr}, nil
		},
	}, config.NewMetadata(filepath.FromSlash("/cfg/modlist.json")))
	assert.ErrorIs(t, err, writeErr)
}

func TestRunInitWithoutPromptReturnsInitError(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	writeErr := errors.New("mkdir failed")
	_, err := runInitWithoutPrompt(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		fs:              mkdirErrorFs{Fs: fs, failPath: meta.Dir(), err: writeErr},
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.ErrorIs(t, err, writeErr)
}

func TestRunInitWithoutPromptReturnsSuccessOutputError(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	writeErr := errors.New("write failed")
	_, err := runInitWithoutPrompt(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return outputLinesModel{err: writeErr}, nil
		},
	}, meta)
	assert.ErrorIs(t, err, writeErr)
}

func TestRunInitInteractiveReturnsOutputError(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(&fakeTTYWriter{})

	writeErr := errors.New("write failed")
	callCount := 0
	_, _, err := runInitInteractive(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			callCount++
			if callCount == 1 {
				return CommandModel{
					state: done,
					result: initOptions{
						ConfigPath:   meta.ConfigPath,
						Loader:       models.FABRIC,
						GameVersion:  "1.21.1",
						ReleaseTypes: []models.ReleaseType{models.Release},
						ModsFolder:   "mods",
						Provided: providedFlags{
							Loader:       true,
							GameVersion:  true,
							ReleaseTypes: true,
							ModsFolder:   true,
						},
					},
				}, nil
			}
			return outputLinesModel{err: writeErr}, nil
		},
	}, meta, false)
	assert.ErrorIs(t, err, writeErr)
	assert.Equal(t, 2, callCount)
}

func TestRunInitInteractiveReturnsInitError(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(&fakeTTYWriter{})

	writeErr := errors.New("mkdir failed")
	_, _, err := runInitInteractive(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		fs:              mkdirErrorFs{Fs: fs, failPath: meta.Dir(), err: writeErr},
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return CommandModel{
				state: done,
				result: initOptions{
					ConfigPath:   meta.ConfigPath,
					Loader:       models.FABRIC,
					GameVersion:  "1.21.1",
					ReleaseTypes: []models.ReleaseType{models.Release},
					ModsFolder:   "mods",
					Provided: providedFlags{
						Loader:       true,
						GameVersion:  true,
						ReleaseTypes: true,
						ModsFolder:   true,
					},
				},
			}, nil
		},
	}, meta, false)
	assert.ErrorIs(t, err, writeErr)
}

func TestRunInitInteractiveReturnsLaunchError(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(&fakeTTYWriter{})

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	_, _, err := runInitInteractive(context.Background(), cmd, initOptions{}, initDeps{
		fs:              afero.NewMemMapFs(),
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("boom")
		},
	}, meta, false)
	assert.ErrorContains(t, err, "boom")
}

func TestRunInitInteractiveReturnsCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(&fakeTTYWriter{})

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	_, _, err := runInitInteractive(context.Background(), cmd, initOptions{}, initDeps{
		fs:              afero.NewMemMapFs(),
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return CommandModel{state: stateLoader}, nil
		},
	}, meta, false)
	assert.ErrorIs(t, err, ErrInitCanceled)
}

func TestRunOutputLinesReturnsRunTeaError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)

	writeErr := errors.New("run error")
	err := runOutputLines(cmd, initDeps{
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, writeErr
		},
	}, []string{"line"})
	assert.ErrorIs(t, err, writeErr)
}

func TestWriteUnattendedOutputStylesWhenColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreProfile)

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewReader(nil))
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(io.Discard)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	err := writeUnattendedOutput(cmd, initDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			outputModel, ok := model.(outputLinesModel)
			assert.True(t, ok)

			icon := "!!"
			if view.SupportsUnicode() {
				icon = "\u203c\ufe0f"
			}
			expectedHeadline := view.ErrorStyle.Render(messageWithIcon(
				icon,
				i18n.T("cmd.init.error.unattended.config_exists", &i18n.Tvars{
					Data: &i18n.TData{"configPath": meta.ConfigPath},
				}),
			))
			expectedHint := view.CtaStyle.Render(
				i18n.T("cmd.init.error.unattended.config_exists_hint", nil),
			)
			assert.Equal(t, []string{expectedHeadline, expectedHint}, outputModel.lines)
			return model, nil
		},
	}, &unattendedOutputError{
		messageKey: "cmd.init.error.unattended.config_exists",
		messageVars: &i18n.Tvars{
			Data: &i18n.TData{"configPath": meta.ConfigPath},
		},
		hintKey: "cmd.init.error.unattended.config_exists_hint",
	})
	assert.NoError(t, err)
}

func TestWriteInitSuccessStylesCtaWhenColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreProfile)

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewReader(nil))
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(io.Discard)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	options := initOptions{
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}

	err := writeInitSuccess(cmd, initDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			outputModel, ok := model.(outputLinesModel)
			assert.True(t, ok)
			expectedHeadline := i18n.T("cmd.init.success", &i18n.Tvars{
				Data: &i18n.TData{
					"configPath":  meta.ConfigPath,
					"loader":      options.Loader.String(),
					"gameVersion": options.GameVersion,
				},
			})
			expectedHint := view.CtaStyle.Render(
				i18n.T("cmd.init.success.next_steps", nil),
			)
			assert.Equal(t, []string{expectedHeadline, expectedHint}, outputModel.lines)
			return model, nil
		},
	}, options, meta)
	assert.NoError(t, err)
}

func TestRunInitWithForceSkipsInteractive(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))
	assert.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte(`{"existing":true}`), 0644))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(&fakeTTYWriter{})

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Force:        true,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			if _, ok := model.(outputLinesModel); ok {
				return model, nil
			}
			return nil, errors.New("unexpected interactive run")
		},
	}, meta)
	assert.NoError(t, err)
	assert.False(t, didUseInteractiveFlow)
}

func TestRunInitInteractiveUsesUpdatedConfigPathInSuccess(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	cmd.SetErr(io.Discard)

	options := initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}
	newConfigPath := filepath.FromSlash("/cfg/alt/modlist.json")
	expectedMeta := config.NewMetadata(newConfigPath)

	_, didUseInteractiveFlow, err := runInitInteractive(context.Background(), cmd, options, initDeps{
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			switch typed := model.(type) {
			case CommandModel, *CommandModel:
				updated := options
				updated.ConfigPath = newConfigPath
				return CommandModel{state: done, result: updated}, nil
			case outputLinesModel:
				assert.Contains(t, typed.lines[0], expectedMeta.ConfigPath)
				return typed, nil
			default:
				return nil, errors.New("unexpected model")
			}
		},
	}, meta, false)
	assert.NoError(t, err)
	assert.True(t, didUseInteractiveFlow)
}

func TestConfigFileExistsReturnsError(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := statErrorFs{Fs: afero.NewMemMapFs(), err: errors.New("stat failed")}

	exists, err := configFileExists(fs, filepath.FromSlash("/cfg/modlist.json"))
	assert.False(t, exists)
	assert.ErrorContains(t, err, "cmd.init.error.config-file.check")
}

func TestColorModeForWriter(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTerminal)
	restoreProfile := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreProfile)

	assert.Equal(t, view.ColorDisabled, colorModeForWriter(nil))

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(&fakeTTYWriter{})
	assert.Equal(t, view.ColorEnabled, colorModeForWriter(cmd))

	cmd.SetIn(bytes.NewReader(nil))
	cmd.SetOut(&fakeTTYWriter{})
	assert.Equal(t, view.ColorEnabled, colorModeForWriter(cmd))

	cmd.SetIn(&fakeTTYReader{Reader: bytes.NewReader(nil)})
	cmd.SetOut(io.Discard)
	assert.Equal(t, view.ColorDisabled, colorModeForWriter(cmd))
}

func TestRunInitInteractiveErrorPropagates(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	in := fakeTTYReader{Reader: bytes.NewReader(nil)}
	out := &fakeTTYWriter{}

	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(out)

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("boom")
		},
	}, meta)
	assert.Error(t, err)
	assert.True(t, didUseInteractiveFlow)
}

func TestRunInitUnattendedSkipsInteractiveFlow(t *testing.T) {
	minecraft.ClearManifestCache()
	restoreTTY := view.SetIsTerminalFuncForTesting(func(int) bool { return true })
	t.Cleanup(restoreTTY)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	in := fakeTTYReader{Reader: bytes.NewReader(nil)}
	out := &fakeTTYWriter{}

	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(out)

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Unattended:   true,
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.NoError(t, err)
	assert.False(t, didUseInteractiveFlow)
}

func TestRunInitNonTTYBehavesLikeUnattended(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.FromSlash("/cfg/mods"), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, didUseInteractiveFlow, err := runInit(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta)
	assert.NoError(t, err)
	assert.False(t, didUseInteractiveFlow)

	exists, existsErr := afero.Exists(fs, meta.ConfigPath)
	assert.NoError(t, existsErr)
	assert.True(t, exists)
}

func TestRunInteractiveInitWithLaunchFlagUsesDefaultProgram(t *testing.T) {
	minecraft.ClearManifestCache()

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("\x03"))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	_, launched, err := runInteractiveInitWithLaunchFlag(context.Background(), cmd, initOptions{}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              afero.NewMemMapFs(),
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea:          defaultRunTea,
	}, meta, false)
	assert.Error(t, err)
	assert.True(t, launched)
}

func TestRunInteractiveInitWithLaunchFlagUsesRunTeaSuccess(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("\r"))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	updated, launched, err := runInteractiveInitWithLaunchFlag(context.Background(), cmd, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
		runTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return CommandModel{
				state: done,
				result: initOptions{
					ConfigPath:   meta.ConfigPath,
					Loader:       models.FABRIC,
					GameVersion:  "1.21.1",
					ReleaseTypes: []models.ReleaseType{models.Release},
					ModsFolder:   "mods",
					Provided: providedFlags{
						Loader:       true,
						GameVersion:  true,
						ReleaseTypes: true,
						ModsFolder:   true,
					},
				},
			}, nil
		},
	}, meta, false)

	assert.NoError(t, err)
	assert.True(t, launched)
	assert.Equal(t, "mods", updated.ModsFolder)
	assert.True(t, updated.Provided.ModsFolder)
}

func TestLoaderFlag(t *testing.T) {
	t.Run("accepts valid loader", func(t *testing.T) {
		var flag loaderFlag
		err := flag.Set("fabric")
		assert.NoError(t, err)
		assert.Equal(t, models.FABRIC, flag.value)
	})

	t.Run("rejects invalid loader", func(t *testing.T) {
		var flag loaderFlag
		err := flag.Set("nope")
		assert.ErrorContains(t, err, "invalid loader")
		assert.Empty(t, flag.value)
	})
}

func TestParseReleaseTypes(t *testing.T) {
	t.Run("parses list", func(t *testing.T) {
		actual, err := parseReleaseTypes([]string{"release", "beta"})
		assert.NoError(t, err)
		assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, actual)
	})

	t.Run("rejects invalid release type", func(t *testing.T) {
		_, err := parseReleaseTypes([]string{"release", "nope"})
		assert.ErrorContains(t, err, "invalid release type")
	})

	t.Run("rejects empty list", func(t *testing.T) {
		_, err := parseReleaseTypes([]string{""})
		assert.ErrorContains(t, err, "release types cannot be empty")
	})
}

func TestGameVersionModelRejectsOfflineEntry(t *testing.T) {
	minecraft.ClearManifestCache()

	offlineDoer := doerFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("offline")
	})

	model := NewGameVersionModel(context.Background(), offlineDoer, "")
	model.input.SetValue("1.2.3")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Empty(t, updated.Value)
	assert.Nil(t, cmd)
	assert.NotNil(t, updated.error)
}

func TestGameVersionModelUsesPlaceholderWhenEmpty(t *testing.T) {
	minecraft.ClearManifestCache()

	model := NewGameVersionModel(context.Background(), manifestDoer([]string{"1.21.1"}), "")
	model.input.SetValue("")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "1.21.1", updated.Value)

	msg := cmd()
	assert.IsType(t, GameVersionSelectedMessage{}, msg)
	assert.Equal(t, "1.21.1", msg.(GameVersionSelectedMessage).GameVersion)
}

func TestReleaseTypesModelRequiresSelection(t *testing.T) {
	model := NewReleaseTypesModel([]models.ReleaseType{})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.ErrorContains(t, updated.error, "release types cannot be empty")

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeySpace})
	assert.NotEmpty(t, updated.values())

	finished, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)

	msg := cmd().(ReleaseTypesSelectedMessage)
	assert.Equal(t, finished.Value, msg.ReleaseTypes)
}

func TestCommandModelProgression(t *testing.T) {
	minecraft.ClearManifestCache()

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		ModsFolder:   "mods",
		ReleaseTypes: []models.ReleaseType{models.Release},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)

	current := *model

	next, _ := current.Update(LoaderSelectedMessage{Loader: models.FABRIC})
	current = next.(CommandModel)

	next, _ = current.Update(GameVersionSelectedMessage{GameVersion: "1.21.1"})
	current = next.(CommandModel)

	next, _ = current.Update(ReleaseTypesSelectedMessage{ReleaseTypes: []models.ReleaseType{models.Release, models.Beta}})
	current = next.(CommandModel)

	finalModel, cmd := current.Update(ModsFolderSelectedMessage{ModsFolder: "mods"})
	assert.Equal(t, stateConfirmWrite, finalModel.(CommandModel).state)
	assert.Nil(t, cmd)

	finalModel, cmd = finalModel.Update(ConfirmWriteSelectedMessage{Confirmed: true})
	assert.Equal(t, done, finalModel.(CommandModel).state)
	assert.NotNil(t, cmd)

	result := finalModel.(CommandModel).result
	assert.Equal(t, models.FABRIC, result.Loader)
	assert.Equal(t, "1.21.1", result.GameVersion)
	assert.Equal(t, []models.ReleaseType{models.Release, models.Beta}, result.ReleaseTypes)
	assert.Equal(t, "mods", result.ModsFolder)
}

func TestModsFolderModelUsesPlaceholderWhenEmpty(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModsFolderModel(modsFolderModelInput{modsFolder: "mods", meta: meta, fs: fs, prefill: false})
	model.input.SetValue("")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "mods", updated.Value)

	msg := cmd()
	assert.IsType(t, ModsFolderSelectedMessage{}, msg)
	assert.Equal(t, "mods", msg.(ModsFolderSelectedMessage).ModsFolder)
}

func TestViewHidesProvidedQuestions(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata(filepath.FromSlash("/cfg/modlist.json"))
	assert.NoError(t, fs.MkdirAll(filepath.Dir(meta.ConfigPath), 0755))
	assert.NoError(t, fs.MkdirAll(meta.ModsFolderPath(models.ModsJSON{ModsFolder: "mods"}), 0755))

	model := NewModel(context.Background(), nil, initOptions{
		ConfigPath:   meta.ConfigPath,
		Loader:       models.FABRIC,
		GameVersion:  "1.21.1",
		ReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:   "mods",
		Provided: providedFlags{
			Loader:       true,
			GameVersion:  true,
			ReleaseTypes: true,
			ModsFolder:   true,
		},
	}, initDeps{
		output:          output.New(io.Discard, io.Discard, true),
		fs:              fs,
		minecraftClient: manifestDoer([]string{"1.21.1"}),
	}, meta, false)

	view := model.View()
	assert.Contains(t, view, "cmd.init.prompt.confirm-write.question")
}

func TestNormalizeGameVersion(t *testing.T) {
	t.Run("leaves explicit version untouched", func(t *testing.T) {
		minecraft.ClearManifestCache()
		opts, err := normalizeGameVersion(context.Background(), initOptions{
			GameVersion: "1.21.1",
		}, initDeps{
			output:          output.New(io.Discard, io.Discard, true),
			minecraftClient: manifestDoer([]string{"1.21.1"}),
		}, gameVersionInteractive)
		assert.NoError(t, err)
		assert.Equal(t, "1.21.1", opts.GameVersion)
	})

	t.Run("resolves latest when provided flag set to latest", func(t *testing.T) {
		minecraft.ClearManifestCache()
		opts, err := normalizeGameVersion(context.Background(), initOptions{
			GameVersion: "latest",
			Provided:    providedFlags{GameVersion: true},
		}, initDeps{
			output:          output.New(io.Discard, io.Discard, true),
			minecraftClient: manifestDoer([]string{"2.0.0"}),
		}, gameVersionUnattended)
		assert.NoError(t, err)
		assert.Equal(t, "2.0.0", opts.GameVersion)
	})

	t.Run("defaults to asking when default latest cannot resolve", func(t *testing.T) {
		minecraft.ClearManifestCache()
		opts, err := normalizeGameVersion(context.Background(), initOptions{
			GameVersion: "latest",
			Provided:    providedFlags{GameVersion: false},
		}, initDeps{minecraftClient: doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("offline")
		}),
			output: output.New(io.Discard, io.Discard, true),
		}, gameVersionInteractive)
		assert.NoError(t, err)
		assert.Equal(t, "", opts.GameVersion)
	})

	t.Run("clears provided latest in interactive mode", func(t *testing.T) {
		minecraft.ClearManifestCache()
		opts, err := normalizeGameVersion(context.Background(), initOptions{
			GameVersion: "latest",
			Provided:    providedFlags{GameVersion: true},
		}, initDeps{minecraftClient: doerFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("offline")
		}),
			output: output.New(io.Discard, io.Discard, true),
		}, gameVersionInteractive)
		assert.NoError(t, err)
		assert.Equal(t, "", opts.GameVersion)
		assert.False(t, opts.Provided.GameVersion)
	})
}

func TestOutputLinesModelUpdateAndView(t *testing.T) {
	model := outputLinesModel{lines: []string{"line one"}}
	cmd := model.Init()
	assert.NotNil(t, cmd)

	updated, updateCmd := model.Update(nil)
	assert.Equal(t, model, updated)
	assert.Nil(t, updateCmd)
	assert.Equal(t, "line one", model.View())

	updated, updateCmd = model.Update(outputLineErrorMsg{err: errors.New("write failed")})
	assert.ErrorContains(t, updated.(outputLinesModel).err, "write failed")
	assert.NotNil(t, updateCmd)
}

func TestOutputLinesModelInitWithEmptyLinesQuits(t *testing.T) {
	model := outputLinesModel{lines: []string{}}
	cmd := model.Init()
	assert.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
}

func TestOutputLinesModelError(t *testing.T) {
	primaryErr := errors.New("boom")
	assert.ErrorIs(t, outputLinesModelError(outputLinesModel{err: primaryErr}), primaryErr)

	otherErr := errors.New("another")
	assert.ErrorIs(t, outputLinesModelError(&outputLinesModel{err: otherErr}), otherErr)
	assert.NoError(t, outputLinesModelError(CommandModel{}))
}

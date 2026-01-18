package remove

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	initCmd "github.com/meza/minecraft-mod-manager/cmd/mmm/init"
	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type dummyModel struct{}

func (dummyModel) Init() tea.Cmd                       { return nil }
func (dummyModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return dummyModel{}, nil }
func (dummyModel) View() string                        { return "" }

type fakeTerminalWriter struct {
	bytes.Buffer
}

func (writer *fakeTerminalWriter) Fd() uintptr {
	return 1
}

type fakeTerminalReader struct {
	*strings.Reader
}

func (reader *fakeTerminalReader) Fd() uintptr {
	return 0
}

type openErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem openErrorFs) Open(name string) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Open(name)
}

type writeFailureFs struct {
	afero.Fs
	failRenamePath string
	failWritePath  string
	renameErr      error
	writeErr       error
}

func (filesystem writeFailureFs) Rename(oldname, newname string) error {
	if filepath.Clean(newname) == filepath.Clean(filesystem.failRenamePath) {
		return filesystem.renameErr
	}
	return filesystem.Fs.Rename(oldname, newname)
}

func (filesystem writeFailureFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failWritePath) && (flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE)) != 0 {
		return nil, filesystem.writeErr
	}
	return filesystem.Fs.OpenFile(name, flag, perm)
}

func TestBuildConfirmPromptIncludesQuestion(t *testing.T) {
	text := buildConfirmPrompt("Question?", "y", "n")
	assert.Contains(t, text, "Question?")
}

func TestBuildConfirmPromptWithoutFormat(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	text := buildConfirmPrompt("Question?", "y", "n")
	assert.Contains(t, text, "Question?")
}

func TestConfirmPromptModelAcceptsDefault(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	_ = model.Init()
	updated, cmd, _ := model.handleEnterKey()
	model = updated
	assert.Equal(t, model.noOption.short, model.value)
	msg := cmd()
	confirmMsg, ok := msg.(confirmSelectedMessage)
	require.True(t, ok)
	assert.False(t, confirmMsg.confirmed)
}

func TestConfirmPromptModelInvalidChoiceSetsError(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.input.SetValue("nope")
	updated, _, _ := model.handleEnterKey()
	assert.Error(t, updated.error)
}

func TestConfirmPromptModelAcceptsYesAndNo(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.input.SetValue(model.yesOption.short)
	updated, cmd, _ := model.handleEnterKey()
	msg := cmd()
	confirmMsg, ok := msg.(confirmSelectedMessage)
	require.True(t, ok)
	assert.True(t, confirmMsg.confirmed)
	assert.Equal(t, model.yesOption.short, updated.value)

	model = newConfirmPromptModel("Question?")
	model.input.SetValue(model.noOption.short)
	updated, cmd, _ = model.handleEnterKey()
	msg = cmd()
	confirmMsg, ok = msg.(confirmSelectedMessage)
	require.True(t, ok)
	assert.False(t, confirmMsg.confirmed)
	assert.Equal(t, model.noOption.short, updated.value)
}

func TestConfirmPromptModelUpdateSetsAnswer(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotEmpty(t, updated.value)
}

func TestConfirmPromptModelUpdateClearsError(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.error = errors.New("oops")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	assert.NoError(t, updated.error)
}

func TestConfirmPromptModelViewShowsAnsweredValue(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.value = model.noOption.short
	assert.Contains(t, model.View(), model.noOption.short)
}

func TestConfirmPromptModelViewShowsError(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	model.error = errors.New("bad")
	viewText := model.View()
	assert.Contains(t, viewText, "bad")
}

func TestConfirmPromptModelMatchesOption(t *testing.T) {
	model := newConfirmPromptModel("Question?")
	assert.True(t, model.matchesOption(model.yesOption.short, model.yesOption))
	assert.True(t, model.matchesOption(model.yesOption.label, model.yesOption))
	assert.False(t, model.matchesOption("nope", model.yesOption))
}

func TestConfigInitModelCancel(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	typed := updated.(configInitModel)
	assert.True(t, typed.canceled)
}

func TestConfigInitModelConfirmSelected(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	updated, _ := model.Update(confirmSelectedMessage{confirmed: true})
	typed := updated.(configInitModel)
	assert.True(t, typed.confirmed)
}

func TestConfigInitModelUpdateFallsThroughToPrompt(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	typed := updated.(configInitModel)
	assert.False(t, typed.canceled)
}

func TestConfigInitModelViewShowsHeadline(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	assert.Contains(t, model.View(), "headline")
}

func TestConfigInitModelViewWithoutHeadline(t *testing.T) {
	model := newConfigInitModel("", "question")
	assert.Contains(t, model.View(), "question")
}

func TestConfigInitModelInit(t *testing.T) {
	model := newConfigInitModel("headline", "question")
	assert.Nil(t, model.Init())
}

func TestConfigInitResultHandlesUnexpectedModel(t *testing.T) {
	_, _, err := configInitResult(dummyModel{})
	assert.Error(t, err)
}

func TestConfigInitResultHandlesModelPointer(t *testing.T) {
	confirmed, canceled, err := configInitResult(&configInitModel{confirmed: true})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRemoveConfirmModelCancel(t *testing.T) {
	model := newRemoveConfirmModel("list", "question")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	typed := updated.(removeConfirmModel)
	assert.True(t, typed.canceled)
}

func TestRemoveConfirmModelConfirmSelected(t *testing.T) {
	model := newRemoveConfirmModel("list", "question")
	updated, _ := model.Update(confirmSelectedMessage{confirmed: true})
	typed := updated.(removeConfirmModel)
	assert.True(t, typed.confirmed)
}

func TestRemoveConfirmModelInitReturnsNil(t *testing.T) {
	model := newRemoveConfirmModel("list", "question")
	assert.Nil(t, model.Init())
}

func TestRemoveConfirmModelUpdateFallsThroughToPrompt(t *testing.T) {
	model := newRemoveConfirmModel("list", "question")
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	typed := updated.(removeConfirmModel)
	assert.False(t, typed.canceled)
}

func TestRemoveConfirmModelViewUsesList(t *testing.T) {
	model := newRemoveConfirmModel("list", "question")
	assert.Contains(t, model.View(), "list")
}

func TestRemoveConfirmModelViewWithoutListUsesPrompt(t *testing.T) {
	model := newRemoveConfirmModel("", "question")
	assert.Contains(t, model.View(), "question")
}

func TestRunRemoveConfirmPromptUsesRunTea(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return removeConfirmModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runRemoveConfirmPrompt(cmd, deps, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunRemoveConfirmPromptUsesFallbackRunner(t *testing.T) {
	original := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return removeConfirmModel{confirmed: true}, nil
	}
	t.Cleanup(func() { runTeaProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	confirmed, canceled, err := runRemoveConfirmPrompt(cmd, removeDeps{}, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunRemoveConfirmPromptRunTeaErrorReturnsError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("run failed")
		},
	}

	_, _, err := runRemoveConfirmPrompt(cmd, deps, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	assert.Error(t, err)
}

func TestRemoveConfirmResultUnexpectedModel(t *testing.T) {
	_, _, err := removeConfirmResult(dummyModel{})
	assert.Error(t, err)
}

func TestRemoveConfirmResultHandlesPointer(t *testing.T) {
	confirmed, canceled, err := removeConfirmResult(&removeConfirmModel{confirmed: true})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestConfirmRemoveNonTTYRequiresForce(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	confirmed, err := confirmRemove(cmd, removeDeps{}, removeOptions{}, interaction.ExecutionModeNonTTY, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	assert.False(t, confirmed)
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "force")
}

func TestConfirmRemoveUnattendedSkipsPrompt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	confirmed, err := confirmRemove(cmd, removeDeps{}, removeOptions{}, interaction.ExecutionModeUnattended, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	require.NoError(t, err)
	assert.True(t, confirmed)
}

func TestConfirmRemoveForceSkipsPrompt(t *testing.T) {
	called := false
	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			called = true
			return removeConfirmModel{confirmed: false}, nil
		},
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	confirmed, err := confirmRemove(cmd, deps, removeOptions{Force: true}, interaction.ExecutionModeNonTTY, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, called)
}

func TestConfirmRemoveInteractivePromptRunsWhenQuiet(t *testing.T) {
	called := false
	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			called = true
			return removeConfirmModel{confirmed: true}, nil
		},
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	confirmed, err := confirmRemove(cmd, deps, removeOptions{Quiet: true}, interaction.ExecutionModeInteractive, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.True(t, called)
}

func TestConfirmRemoveInteractiveDeclined(t *testing.T) {
	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return removeConfirmModel{confirmed: false}, nil
		},
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	confirmed, err := confirmRemove(cmd, deps, removeOptions{}, interaction.ExecutionModeInteractive, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	require.NoError(t, err)
	assert.False(t, confirmed)
}

func TestConfirmRemoveInteractiveCanceled(t *testing.T) {
	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return removeConfirmModel{canceled: true}, nil
		},
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	confirmed, err := confirmRemove(cmd, deps, removeOptions{}, interaction.ExecutionModeInteractive, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	require.NoError(t, err)
	assert.False(t, confirmed)
}

func TestConfirmRemovePromptError(t *testing.T) {
	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("prompt failed")
		},
	}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	confirmed, err := confirmRemove(cmd, deps, removeOptions{}, interaction.ExecutionModeInteractive, view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}})
	assert.False(t, confirmed)
	assert.Error(t, err)
}

func TestHandleRemoveForceRequiredOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("run failed")
		},
	}

	err := handleRemoveForceRequired(cmd, deps, view.ColorDisabled)
	assert.Error(t, err)
}

func TestRenderRemoveCanceledLine(t *testing.T) {
	assert.Contains(t, renderRemoveCanceledLine(), "Remove")
}

func TestRenderRemoveForceRequiredLineColorized(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	line := renderRemoveForceRequiredLine(view.ColorEnabled)
	assert.Contains(t, line, "Remove")
}

func TestRunRemoveWithMatchesCancelledOutputsMessage(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeInteractive,
	}

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			switch typed := model.(type) {
			case removeConfirmModel:
				return removeConfirmModel{confirmed: false}, nil
			case view.OutputLinesModel:
				for _, line := range typed.Lines {
					if _, err := fmt.Fprintln(typed.Output, line); err != nil {
						return typed, err
					}
				}
				return typed, nil
			default:
				return model, nil
			}
		},
	}

	matches := []removeMatch{{mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	removed, _, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{}, deps, runState, matches)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
	assert.Contains(t, out.String(), "Remove canceled")
}

func TestRunRemoveWithMatchesCancelOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeInteractive,
	}

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			switch typed := model.(type) {
			case removeConfirmModel:
				return removeConfirmModel{confirmed: false}, nil
			case view.OutputLinesModel:
				for _, line := range typed.Lines {
					if _, err := fmt.Fprintln(typed.Output, line); err != nil {
						return typed, err
					}
				}
				return typed, nil
			default:
				return model, nil
			}
		},
	}

	matches := []removeMatch{{mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	_, _, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{}, deps, runState, matches)
	assert.Error(t, err)
}

func TestRunConfigInitPromptUsesRunTea(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	deps := removeDeps{
		fs:     afero.NewMemMapFs(),
		logger: logger.New(io.Discard, io.Discard, false, false),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, config.NewMetadata("modlist.json"))
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunConfigInitPromptHandlesRunTeaError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	deps := removeDeps{
		fs:     afero.NewMemMapFs(),
		logger: logger.New(io.Discard, io.Discard, false, false),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("run failed")
		},
	}

	_, _, err := runConfigInitPrompt(cmd, deps, config.NewMetadata("modlist.json"))
	assert.Error(t, err)
}

func TestRunConfigInitPromptUsesDefaultRunner(t *testing.T) {
	original := runTeaProgram
	runTeaProgram = func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		return configInitModel{confirmed: true}, nil
	}
	t.Cleanup(func() { runTeaProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	deps := removeDeps{
		fs:     afero.NewMemMapFs(),
		logger: logger.New(io.Discard, io.Discard, false, false),
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, config.NewMetadata("modlist.json"))
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRunConfigInitPromptWithColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	confirmed, canceled, err := runConfigInitPrompt(cmd, deps, config.NewMetadata("modlist.json"))
	require.NoError(t, err)
	assert.True(t, confirmed)
	assert.False(t, canceled)
}

func TestRemoveOptionsFromFlagsMissingFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "")
	_, err := removeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)

	cmd = &cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Bool("unattended", false, "")
	_, err = removeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)

	cmd = &cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	_, err = removeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)

	cmd = &cobra.Command{}
	cmd.Flags().String("config", "", "")
	cmd.Flags().Bool("unattended", false, "")
	cmd.Flags().Bool("quiet", false, "")
	cmd.Flags().Bool("debug", false, "")
	_, err = removeOptionsFromFlags(cmd, nil)
	assert.Error(t, err)
}

func TestRemoveOptionsFromFlagsSuccess(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "modlist.json", "")
	cmd.Flags().Bool("unattended", true, "")
	cmd.Flags().Bool("quiet", true, "")
	cmd.Flags().Bool("debug", true, "")
	cmd.Flags().Bool("force", true, "")

	opts, err := removeOptionsFromFlags(cmd, []string{"mod-a"})
	require.NoError(t, err)
	assert.Equal(t, "modlist.json", opts.ConfigPath)
	assert.True(t, opts.Unattended)
	assert.True(t, opts.Quiet)
	assert.True(t, opts.Debug)
	assert.True(t, opts.Force)
}

func TestDefaultRemoveDepsRunInitUsesStub(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetIn(strings.NewReader(""))

	called := false
	original := runInteractiveInit
	runInteractiveInit = func(ctx context.Context, cmd *cobra.Command, deps initCmd.InteractiveInitDeps, options initCmd.InteractiveInitOptions) error {
		called = true
		return nil
	}
	t.Cleanup(func() { runInteractiveInit = original })

	deps := defaultRemoveDeps(cmd, removeOptions{})
	require.NotNil(t, deps.runInit)
	assert.NoError(t, deps.runInit(context.Background(), cmd, initRequest{configPath: "modlist.json"}))
	assert.True(t, called)
}

func TestHandleRemoveFailureOutputsError(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{
		fs:     afero.NewMemMapFs(),
		logger: logger.New(io.Discard, io.Discard, false, false),
		runTea: defaultRunTea,
	}

	err := handleRemoveFailure(cmd, deps, errors.New("boom"))
	assert.Error(t, err)
	assert.Contains(t, out.String(), i18n.T("cmd.remove.error.failed", &i18n.Tvars{Data: &i18n.TData{"reason": "boom"}}))
}

func TestRemoveModelInitReturnsQuitWhenMissingSender(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, nil, nil, func(context.Context, removeExecSender) removeExecutionOutcome {
		return removeExecutionOutcome{}
	})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestRemoveModelInitStartsExecutionWhenBound(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, nil, nil, func(context.Context, removeExecSender) removeExecutionOutcome {
		return removeExecutionOutcome{}
	})
	model.bindSender(func(tea.Msg) {})

	cmd := model.Init()
	msg := cmd()
	assert.NotNil(t, msg)
}

func TestRemoveModelUpdateHandlesSpinnerTick(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, nil, nil, func(context.Context, removeExecSender) removeExecutionOutcome {
		return removeExecutionOutcome{}
	})
	updated, _ := model.Update(spinner.TickMsg{})
	assert.NotNil(t, updated)
}

func TestRemoveModelUpdateHandlesSuccessAndFailure(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}, Status: removeItemPending}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveModel(context.Background(), view.ColorDisabled, items, indexByKey, nil)

	updated, _ := model.Update(removeItemSuccessMsg{key: "modrinth:sodium"})
	typed := updated.(*removeModel)
	assert.Equal(t, removeItemSuccess, typed.items[0].Status)
	assert.Equal(t, "", typed.items[0].FailureReason)

	updated, _ = typed.Update(removeItemFailureMsg{key: "modrinth:sodium", reason: "boom"})
	typed = updated.(*removeModel)
	assert.Equal(t, removeItemFailed, typed.items[0].Status)
	assert.Equal(t, "boom", typed.items[0].FailureReason)
}

func TestRemoveModelUpdateIgnoresUnknownKey(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}, Status: removeItemPending}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveModel(context.Background(), view.ColorDisabled, items, indexByKey, nil)

	updated, _ := model.Update(removeItemSuccessMsg{key: "modrinth:missing"})
	typed := updated.(*removeModel)
	assert.Equal(t, removeItemPending, typed.items[0].Status)
}

func TestRemoveModelUpdateIgnoresUnknownMessage(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, nil, nil, nil)
	updated, _ := model.Update(struct{}{})
	assert.NotNil(t, updated)
}

func TestRemoveModelUpdateHandlesExecutionFinished(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}, Status: removeItemSuccess}}
	model := newRemoveModel(context.Background(), view.ColorDisabled, items, map[string]int{"modrinth:sodium": 0}, nil)

	updated, cmd := model.Update(removeExecutionFinishedMsg{outcome: removeExecutionOutcome{items: items}})
	typed := updated.(*removeModel)
	assert.True(t, typed.done)
	assert.Len(t, typed.items, 1)
	msg := cmd()
	_, ok := msg.(removeFinalizeMsg)
	assert.True(t, ok)

	_, cmd = model.Update(msg)
	assert.NotNil(t, cmd)
	quitMsg := cmd()
	_, ok = quitMsg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestRemoveModelViewFinalSuccessIncludesSummary(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod:    models.Mod{Name: "Sodium", ID: "sodium"},
		Status: removeItemSuccess,
	}}, map[string]int{"modrinth:sodium": 0}, nil)
	model.done = true
	model.outcome = removeExecutionOutcome{errType: removeExecutionErrorNone}

	viewText := model.View()
	assert.Contains(t, viewText, "Remove complete.")
}

func TestRemoveModelViewFinalWriteFailureIncludesSummary(t *testing.T) {
	model := newRemoveModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod:    models.Mod{Name: "Sodium", ID: "sodium"},
		Status: removeItemSuccess,
	}}, map[string]int{"modrinth:sodium": 0}, nil)
	model.done = true
	model.outcome = removeExecutionOutcome{err: errors.New("boom"), errType: removeExecutionErrorWriteLock}

	viewText := model.View()
	assert.Contains(t, viewText, "Remove failed")
}

func TestRenderRemoveRunningSectionIncludesHeader(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}, Status: removeItemPending}}
	spin := view.NewSpinner()
	viewText := renderRemoveRunningSection(view.ColorDisabled, items, &spin)
	assert.Contains(t, viewText, "Removing mods:")
}

func TestShouldUseTranscriptReturnsTrueForNilCmd(t *testing.T) {
	assert.True(t, shouldUseTranscript(nil))
}

func TestDefaultRunRemoveProgramRunsModel(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveModel(context.Background(), view.ColorDisabled, items, indexByKey, func(context.Context, removeExecSender) removeExecutionOutcome {
		return removeExecutionOutcome{items: items}
	})

	result, err := defaultRunRemoveProgram(model, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	require.NoError(t, err)
	outcome, outcomeErr := removeOutcomeFromModel(result)
	require.NoError(t, outcomeErr)
	assert.Len(t, outcome.items, 1)
}

func TestDefaultRunRemoveTranscriptProgramRunsModel(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, io.Discard, func(context.Context, removeExecSender) removeExecutionOutcome {
		return removeExecutionOutcome{items: items}
	})

	result, err := defaultRunRemoveTranscriptProgram(model, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	require.NoError(t, err)
	outcome, outcomeErr := removeOutcomeFromModel(result)
	require.NoError(t, outcomeErr)
	assert.Len(t, outcome.items, 1)
}

func TestRunRemoveInteractiveOutputUsesOutcome(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	execInput := removeExecutionInput{items: items, deps: removeDeps{}}

	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = removeExecutionOutcome{items: items}
		return model, nil
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	count, err := runRemoveInteractiveOutput(context.Background(), cmd, execInput, view.ColorDisabled, indexByKey)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRunRemoveInteractiveOutputHandlesProgramError(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	execInput := removeExecutionInput{items: items, deps: removeDeps{}}

	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("program failed")
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	_, err := runRemoveInteractiveOutput(context.Background(), cmd, execInput, view.ColorDisabled, indexByKey)
	assert.Error(t, err)
}

func TestRunRemoveInteractiveOutputHandlesOutcomeError(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	execInput := removeExecutionInput{items: items, deps: removeDeps{}}

	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = removeExecutionOutcome{items: items, err: errors.New("failed")}
		return model, nil
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	_, err := runRemoveInteractiveOutput(context.Background(), cmd, execInput, view.ColorDisabled, indexByKey)
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunRemoveTranscriptOutputHandlesProgramError(t *testing.T) {
	items := []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	execInput := removeExecutionInput{items: items, deps: removeDeps{}}

	original := runRemoveTranscriptProgram
	runRemoveTranscriptProgram = func(model *removeTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("program failed")
	}
	t.Cleanup(func() { runRemoveTranscriptProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	_, err := runRemoveTranscriptOutput(context.Background(), cmd, execInput, view.ColorDisabled, indexByKey)
	assert.Error(t, err)
}

func TestOutputProgramOptionsHandlesNilCmd(t *testing.T) {
	options := outputProgramOptions(nil, nil)
	assert.NotEmpty(t, options)
}

func TestOutputProgramOptionsUsesCommandOutput(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	options := outputProgramOptions(cmd, nil)
	assert.NotEmpty(t, options)
}

func TestColorModeForWriterEnabledWhenTerminalAndColor(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	writer := &fakeTerminalWriter{}
	cmd := &cobra.Command{}
	cmd.SetOut(writer)

	assert.Equal(t, view.ColorEnabled, colorModeForWriter(cmd))
}

func TestUncertaintyIconUsesUnicodeWhenAvailable(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restoreUnicode)

	assert.Equal(t, "\u2754", uncertaintyIcon(view.ColorDisabled))
}

func TestHandleRemoveNoMatchesQuiet(t *testing.T) {
	cmd := &cobra.Command{}
	err := handleRemoveNoMatches(cmd, removeDeps{}, removeOptions{Quiet: true})
	assert.NoError(t, err)
}

func TestRunOutputLinesUsesDefaultRunner(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	deps := removeDeps{}
	assert.NoError(t, runOutputLines(cmd, deps, io.Discard, []string{"line"}))
}

func TestHandleRemoveNoMatchesOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	err := handleRemoveNoMatches(cmd, removeDeps{runTea: defaultRunTea}, removeOptions{})
	assert.Error(t, err)
}

func TestRunRemoveQuietOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	execInput := removeExecutionInput{
		items: []removeItem{{
			Mod:          models.Mod{Name: "Sodium", ID: "sodium"},
			HasLockEntry: true,
			LockFileName: "",
		}},
		deps: removeDeps{runTea: defaultRunTea},
	}

	_, err := runRemoveQuiet(context.Background(), cmd, execInput, view.ColorDisabled)
	assert.Error(t, err)
}

func TestRenderRemoveQuietFailureWriteError(t *testing.T) {
	outcome := removeExecutionOutcome{err: errors.New("boom"), errType: removeExecutionErrorWriteLock}
	lines := renderRemoveQuietFailure(view.ColorDisabled, outcome)
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "Remove failed")
}

func TestRenderRemoveQuietFailureUnknownWithoutErrorReturnsNil(t *testing.T) {
	lines := renderRemoveQuietFailure(view.ColorDisabled, removeExecutionOutcome{errType: removeExecutionErrorUnknown})
	assert.Nil(t, lines)
}

func TestRenderRemoveQuietFailureNoneReturnsNil(t *testing.T) {
	lines := renderRemoveQuietFailure(view.ColorDisabled, removeExecutionOutcome{errType: removeExecutionErrorNone})
	assert.Nil(t, lines)
}

func TestRenderRemoveQuietDeleteFailureNoFailedItems(t *testing.T) {
	lines := renderRemoveQuietDeleteFailure(view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}, Status: removeItemSuccess}})
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "Remove incomplete")
}

func TestRenderRemoveFailureSummaryWithColor(t *testing.T) {
	summary := renderRemoveFailureSummary(view.ColorEnabled)
	assert.Contains(t, summary, "Remove incomplete")
}

func TestRenderRemoveFailureLineWithColor(t *testing.T) {
	line := renderRemoveFailureLine(view.ColorEnabled, errors.New("boom"))
	assert.Contains(t, line, "Remove failed")
}

func TestWriteConfigMissingOutputWithColor(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(""))

	deps := removeDeps{runTea: defaultRunTea}
	assert.NoError(t, writeConfigMissingOutput(cmd, deps, config.NewMetadata("modlist.json")))
	assert.Contains(t, out.String(), "No configuration file found")
}

func TestWriteConfigMissingOutputColorEnabled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile { return termenv.TrueColor })
	t.Cleanup(restoreColor)

	var out fakeTerminalWriter
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})

	deps := removeDeps{runTea: defaultRunTea}
	assert.NoError(t, writeConfigMissingOutput(cmd, deps, config.NewMetadata("modlist.json")))
	assert.Contains(t, out.String(), "No configuration file found")
}

func TestHandleRemoveFailureReturnsOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetIn(strings.NewReader(""))
	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return view.OutputLinesModel{Err: errors.New("write failed")}, nil
		},
	}
	err := handleRemoveFailure(cmd, deps, errors.New("boom"))
	assert.Error(t, err)
}

func TestRunRemoveInteractiveReturnsOutcomeError(t *testing.T) {
	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return dummyModel{}, nil
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	_, err := runRemoveInteractive(context.Background(), cmd, removeExecutionInput{}, view.ColorDisabled, map[string]int{})
	assert.Error(t, err)
}

func TestRunRemoveWithMatchesUsesInteractiveOutput(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		for index := range model.items {
			item := model.items[index]
			item.Status = removeItemSuccess
			model.items[index] = item
		}
		model.outcome = removeExecutionOutcome{items: model.items}
		return model, nil
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeInteractive,
	}
	matches := []removeMatch{{mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return removeConfirmModel{confirmed: true}, nil
		},
	}

	removed, interactive, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{}, deps, runState, matches)
	require.NoError(t, err)
	assert.True(t, interactive)
	assert.Equal(t, 1, removed)
}

func TestRunRemoveWithMatchesNoMatchesOutputsMessage(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeNonTTY,
	}

	removed, interactive, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{}, removeDeps{runTea: defaultRunTea}, runState, nil)
	require.NoError(t, err)
	assert.False(t, interactive)
	assert.Equal(t, 0, removed)
	assert.Contains(t, out.String(), "No matching mods found")
}

func TestRunRemoveWithMatchesNoMatchesOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeNonTTY,
	}

	_, _, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{}, removeDeps{runTea: defaultRunTea}, runState, nil)
	assert.Error(t, err)
}

func TestRunRemoveReturnsErrorOnInvalidConfig(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, afero.WriteFile(fs, meta.ConfigPath, []byte("{"), 0644))

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	deps := removeDeps{fs: fs, runTea: defaultRunTea}
	_, _, err := runRemove(context.Background(), cmd, removeOptions{ConfigPath: meta.ConfigPath, Lookups: []string{"sod*"}}, deps)
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
}

func TestRunRemoveReturnsErrorOnInvalidPattern(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
		Mods:                       []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}},
	}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	deps := removeDeps{fs: fs, runTea: defaultRunTea}
	_, _, err := runRemove(context.Background(), cmd, removeOptions{ConfigPath: meta.ConfigPath, Lookups: []string{"["}}, deps)
	assert.Error(t, err)
}

func TestRunRemoveStopsWhenPromptDeclined(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false}, nil
		},
	}

	removed, interactive, err := runRemove(context.Background(), cmd, removeOptions{ConfigPath: "modlist.json", Lookups: []string{"sod*"}}, deps)
	require.NoError(t, err)
	assert.True(t, interactive)
	assert.Equal(t, 0, removed)
}

func TestRunRemoveWithMatchesTranscriptError(t *testing.T) {
	original := runRemoveTranscriptProgram
	runRemoveTranscriptProgram = func(model *removeTranscriptModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("program failed")
	}
	t.Cleanup(func() { runRemoveTranscriptProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeNonTTY,
	}
	matches := []removeMatch{{mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}

	_, _, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{Force: true}, removeDeps{}, runState, matches)
	assert.Error(t, err)
}

func TestRunRemoveWithMatchesQuietOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		lock: []models.ModInstall{{
			Type:     models.MODRINTH,
			ID:       "sodium",
			FileName: "",
		}},
		mode: interaction.ExecutionModeNonTTY,
	}
	matches := []removeMatch{{
		mod:          models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		hasLockEntry: true,
		lockFileName: "",
	}}

	_, _, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{Quiet: true, Force: true}, removeDeps{runTea: defaultRunTea}, runState, matches)
	assert.Error(t, err)
}

func TestRunRemoveWithMatchesInteractiveError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("program failed")
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	runState := removeRunState{
		meta: config.NewMetadata("modlist.json"),
		mode: interaction.ExecutionModeInteractive,
	}
	matches := []removeMatch{{mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return removeConfirmModel{confirmed: true}, nil
		},
	}

	_, _, err := runRemoveWithMatches(context.Background(), cmd, removeOptions{}, deps, runState, matches)
	assert.Error(t, err)
}

func TestRunRemoveInteractiveReturnsProgramError(t *testing.T) {
	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		return nil, errors.New("program failed")
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	_, err := runRemoveInteractive(context.Background(), cmd, removeExecutionInput{}, view.ColorDisabled, map[string]int{})
	assert.Error(t, err)
}

func TestRunRemoveInteractiveSuccess(t *testing.T) {
	original := runRemoveProgram
	runRemoveProgram = func(model *removeModel, _ ...tea.ProgramOption) (tea.Model, error) {
		model.outcome = model.execRunner(context.Background(), removeExecSender{})
		return model, nil
	}
	t.Cleanup(func() { runRemoveProgram = original })

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)

	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, Name: "Sodium", ID: "sodium"}}}
	execInput := removeExecutionInput{
		meta:  config.NewMetadata("modlist.json"),
		cfg:   models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}},
		items: items,
		deps:  removeDeps{fs: afero.NewMemMapFs()},
	}

	outcome, err := runRemoveInteractive(context.Background(), cmd, execInput, view.ColorDisabled, map[string]int{"modrinth:sodium": 0})
	require.NoError(t, err)
	assert.Len(t, outcome.items, 1)
}

func TestHandleMissingRemoveConfigPromptDeclined(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: false}, nil
		},
	}

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	updated, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	require.NoError(t, err)
	assert.False(t, updated.shouldContinue)
}

func TestHandleMissingRemoveConfigPromptCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{canceled: true}, nil
		},
	}

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	updated, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	require.NoError(t, err)
	assert.False(t, updated.shouldContinue)
}

func TestHandleMissingRemoveConfigUnattendedReturnsPromptError(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	_, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{Unattended: true}, removeDeps{runTea: defaultRunTea}, runState)
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
	assert.Contains(t, out.String(), "No configuration file found")
}

func TestHandleMissingRemoveConfigPromptOutputError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(errorWriter{err: errors.New("write failed")})

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	_, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{Unattended: true}, removeDeps{runTea: defaultRunTea}, runState)
	assert.Error(t, err)
}

func TestHandleMissingRemoveConfigMissingInitRunner(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
	}

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	_, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	assert.Error(t, err)
}

func TestHandleMissingRemoveConfigPromptError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return nil, errors.New("prompt failed")
		},
	}

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	_, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	assert.Error(t, err)
}

func TestHandleMissingRemoveConfigInitCanceled(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return initCmd.ErrInitCanceled
		},
	}

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	updated, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	require.NoError(t, err)
	assert.False(t, updated.shouldContinue)
}

func TestHandleMissingRemoveConfigInitError(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return errors.New("init failed")
		},
	}

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	_, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	assert.Error(t, err)
}

func TestHandleMissingRemoveConfigSuccessAfterInit(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		fs: fs,
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(ctx context.Context, _ *cobra.Command, request initRequest) error {
			cfg := models.ModsJSON{
				Loader:                     models.FABRIC,
				GameVersion:                "1.20.1",
				DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
				ModsFolder:                 "mods",
			}
			return config.WriteConfig(ctx, fs, meta, cfg)
		},
	}

	runState := removeRunState{meta: meta, shouldContinue: true}
	updated, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	require.NoError(t, err)
	assert.True(t, updated.shouldContinue)
}

func TestHandleMissingRemoveConfigInitLoadFailure(t *testing.T) {
	restoreTerminal := view.SetIsTerminalFuncForTesting(func(_ int) bool { return true })
	t.Cleanup(restoreTerminal)

	cmd := &cobra.Command{}
	cmd.SetIn(&fakeTerminalReader{Reader: strings.NewReader("")})
	cmd.SetOut(&fakeTerminalWriter{})

	deps := removeDeps{
		fs: afero.NewMemMapFs(),
		runTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return configInitModel{confirmed: true}, nil
		},
		runInit: func(context.Context, *cobra.Command, initRequest) error {
			return nil
		},
	}

	runState := removeRunState{meta: config.NewMetadata("modlist.json"), shouldContinue: true}
	_, err := handleMissingRemoveConfig(context.Background(), cmd, removeOptions{}, deps, runState)
	assert.Error(t, err)
	assert.True(t, clierrors.IsHandled(err))
}

func TestRemoveSingleItemWithoutLockEntrySucceeds(t *testing.T) {
	result := removeSingleItem(removeExecutionInput{}, removeItem{Mod: models.Mod{Name: "Sodium", ID: "sodium"}})
	assert.Equal(t, removeItemSuccess, result.Status)
}

func TestRemoveLockEntryNoMatch(t *testing.T) {
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "sodium"}}
	updated := removeLockEntry(lock, models.Mod{Type: models.MODRINTH, ID: "other"})
	assert.Len(t, updated, 1)
}

func TestRemoveConfigEntryNoMatch(t *testing.T) {
	cfg := models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium"}}}
	updated := removeConfigEntry(cfg, models.Mod{Type: models.MODRINTH, ID: "other"})
	assert.Len(t, updated.Mods, 1)
}

func TestConfigIndexForMatch(t *testing.T) {
	cfg := models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium"}}}
	assert.Equal(t, 0, configIndexFor(models.Mod{Type: models.MODRINTH, ID: "sodium"}, cfg.Mods))
}

func TestReadRemoveConfigDoesNotCreateLockWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))

	_, lock, err := readRemoveConfig(context.Background(), removeDeps{fs: fs}, meta)
	require.NoError(t, err)
	assert.Empty(t, lock)

	exists, err := afero.Exists(fs, meta.LockPath())
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestReadRemoveConfigReturnsLockReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, config.WriteConfig(context.Background(), fs, meta, cfg))
	require.NoError(t, afero.WriteFile(fs, meta.LockPath(), []byte("{"), 0644))

	_, _, err := readRemoveConfig(context.Background(), removeDeps{fs: fs}, meta)
	assert.Error(t, err)
}

func TestReadRemoveConfigReturnsLockCheckError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")

	cfg := models.ModsJSON{
		Loader:                     models.FABRIC,
		GameVersion:                "1.20.1",
		DefaultAllowedReleaseTypes: []models.ReleaseType{models.Release},
		ModsFolder:                 "mods",
	}

	require.NoError(t, config.WriteConfig(context.Background(), base, meta, cfg))

	fs := statErrorFs{Fs: base, failPath: meta.LockPath(), err: errors.New("stat failed")}
	_, _, err := readRemoveConfig(context.Background(), removeDeps{fs: fs}, meta)
	assert.Error(t, err)
}

func TestBuildRemoveItemsIncludesLockEntry(t *testing.T) {
	matches := []removeMatch{{
		mod:          models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		hasLockEntry: true,
		lockFileName: "sodium.jar",
	}}

	items := buildRemoveItems(matches)
	require.Len(t, items, 1)
	assert.True(t, items[0].HasLockEntry)
	assert.Equal(t, "sodium.jar", items[0].LockFileName)
}

func TestBuildRemoveItemsSortsByNameThenIDWhenPlatformMatches(t *testing.T) {
	matches := []removeMatch{
		{mod: models.Mod{Type: models.MODRINTH, ID: "b", Name: "Same"}},
		{mod: models.Mod{Type: models.MODRINTH, ID: "a", Name: "Same"}},
	}

	items := buildRemoveItems(matches)
	require.Len(t, items, 2)
	assert.Equal(t, "a", items[0].Mod.ID)
	assert.Equal(t, "b", items[1].Mod.ID)
}

func TestColorModeForWriterNilCmd(t *testing.T) {
	assert.Equal(t, view.ColorDisabled, colorModeForWriter(nil))
}

func TestRemoveOutcomeFromModelReturnsErrorOnUnexpectedType(t *testing.T) {
	_, err := removeOutcomeFromModel(dummyModel{})
	assert.Error(t, err)
}

func TestRemoveTranscriptModelInitReturnsQuitWhenMissingSender(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, nil, nil, io.Discard, func(context.Context, removeExecSender) removeExecutionOutcome {
		return removeExecutionOutcome{}
	})

	cmd := model.Init()
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestRemoveTranscriptModelInitStartsExecutionWhenBound(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, nil, nil, io.Discard, func(context.Context, removeExecSender) removeExecutionOutcome {
		return removeExecutionOutcome{}
	})
	model.bindSender(func(tea.Msg) {})

	cmd := model.Init()
	msg := cmd()
	assert.NotNil(t, msg)
}

func TestRemoveTranscriptModelUpdateSuccessOutputsLine(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeItemSuccessMsg{key: "modrinth:sodium"})
	assert.Nil(t, cmd)
	assert.Contains(t, out.String(), "V Sodium (sodium)")
}

func TestRemoveTranscriptModelUpdateSuccessIgnoresTerminal(t *testing.T) {
	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}, Status: removeItemSuccess}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeItemSuccessMsg{key: "modrinth:sodium"})
	assert.Nil(t, cmd)
	assert.Empty(t, out.String())
}

func TestRemoveTranscriptModelUpdateFailureOutputsLine(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeItemFailureMsg{key: "modrinth:sodium", reason: "boom"})
	assert.Nil(t, cmd)
	assert.Contains(t, out.String(), "X Sodium (sodium)")
}

func TestRemoveTranscriptModelUpdateSuccessHandlesOutputError(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
	}}, map[string]int{"modrinth:sodium": 0}, errorWriter{err: errors.New("write failed")}, nil)

	updated, cmd := model.Update(removeItemSuccessMsg{key: "modrinth:sodium"})
	assert.NotNil(t, cmd)
	assert.Equal(t, removeExecutionErrorUnknown, updated.(*removeTranscriptModel).outcome.errType)
}

func TestRemoveTranscriptModelUpdateSuccessHandlesNilOutput(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
	}}, map[string]int{"modrinth:sodium": 0}, nil, nil)

	updated, cmd := model.Update(removeItemSuccessMsg{key: "modrinth:sodium"})
	assert.NotNil(t, cmd)
	assert.Equal(t, removeExecutionErrorUnknown, updated.(*removeTranscriptModel).outcome.errType)
}

func TestRemoveTranscriptModelUpdateFailureHandlesOutputError(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, []removeItem{{
		Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
	}}, map[string]int{"modrinth:sodium": 0}, errorWriter{err: errors.New("write failed")}, nil)

	updated, cmd := model.Update(removeItemFailureMsg{key: "modrinth:sodium", reason: "boom"})
	assert.NotNil(t, cmd)
	assert.Equal(t, removeExecutionErrorUnknown, updated.(*removeTranscriptModel).outcome.errType)
}

func TestRemoveTranscriptModelOutputsCompletionOrder(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	items := []removeItem{
		{Mod: models.Mod{Type: models.MODRINTH, ID: "first", Name: "First"}},
		{Mod: models.Mod{Type: models.MODRINTH, ID: "second", Name: "Second"}},
	}
	indexByKey := map[string]int{
		"modrinth:first":  0,
		"modrinth:second": 1,
	}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeItemSuccessMsg{key: "modrinth:second"})
	assert.Nil(t, cmd)
	_, cmd = model.Update(removeItemSuccessMsg{key: "modrinth:first"})
	assert.Nil(t, cmd)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "Second (second)")
	assert.Contains(t, lines[1], "First (first)")
}

func TestRemoveTranscriptModelUpdateFailureIgnoresTerminal(t *testing.T) {
	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}, Status: removeItemFailed}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeItemFailureMsg{key: "modrinth:sodium", reason: "boom"})
	assert.Nil(t, cmd)
	assert.Empty(t, out.String())
}

func TestRemoveTranscriptModelUpdateMissingKey(t *testing.T) {
	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeItemFailureMsg{key: "modrinth:missing", reason: "boom"})
	assert.Nil(t, cmd)
	assert.Empty(t, out.String())
}

func TestRemoveTranscriptModelUpdateIgnoresUnknownMessage(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, nil, nil, io.Discard, nil)

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	assert.NotNil(t, updated)
	assert.Nil(t, cmd)
}

func TestRemoveTranscriptModelUpdateExecutionFinishedOutputsSummary(t *testing.T) {
	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}, Status: removeItemSuccess}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeExecutionFinishedMsg{outcome: removeExecutionOutcome{items: items}})
	require.NotNil(t, cmd)
	assert.Equal(t, 1, len(model.outcome.items))
}

func TestRemoveTranscriptModelSummaryLinesWithoutItems(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, nil, nil, io.Discard, nil)
	model.outcome = removeExecutionOutcome{errType: removeExecutionErrorNone}

	lines := model.summaryLines()
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "Remove complete")
}

func TestRemoveTranscriptModelSummaryLinesWithoutSummary(t *testing.T) {
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, nil, nil, io.Discard, nil)
	model.outcome = removeExecutionOutcome{errType: removeExecutionErrorUnknown}

	lines := model.summaryLines()
	assert.Empty(t, lines)
}

func TestMissingTranscriptLinesSkipsTerminalItems(t *testing.T) {
	lines := missingTranscriptLines(view.ColorDisabled, []removeItem{{
		Mod:    models.Mod{Name: "Sodium", ID: "sodium"},
		Status: removeItemSuccess,
	}}, []removeItem{{
		Mod:    models.Mod{Name: "Sodium", ID: "sodium"},
		Status: removeItemSuccess,
	}})
	assert.Empty(t, lines)
}

func TestMissingTranscriptLinesIncludesPendingItems(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	lines := missingTranscriptLines(view.ColorDisabled, []removeItem{{
		Mod:    models.Mod{Name: "Sodium", ID: "sodium"},
		Status: removeItemPending,
	}}, []removeItem{{
		Mod:           models.Mod{Name: "Sodium", ID: "sodium"},
		Status:        removeItemFailed,
		FailureReason: "boom",
	}})
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "X Sodium (sodium)")
}

func TestMissingTranscriptLinesHandlesEmptyFinalItems(t *testing.T) {
	lines := missingTranscriptLines(view.ColorDisabled, []removeItem{{Mod: models.Mod{Name: "Sodium", ID: "sodium"}}}, nil)
	assert.Nil(t, lines)
}

func TestMissingTranscriptLinesUsesFinalItemsWhenMissingCurrent(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	lines := missingTranscriptLines(view.ColorDisabled, nil, []removeItem{{
		Mod:    models.Mod{Name: "Sodium", ID: "sodium"},
		Status: removeItemSuccess,
	}})
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "V Sodium (sodium)")
}

func TestRemoveTranscriptModelIgnoresUpdatesAfterFinish(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeExecutionFinishedMsg{outcome: removeExecutionOutcome{
		items:   []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}, Status: removeItemSuccess}},
		errType: removeExecutionErrorNone,
	}})
	require.NotNil(t, cmd)
	_ = cmd()

	_, cmd = model.Update(removeItemFailureMsg{key: "modrinth:sodium", reason: "boom"})
	assert.Nil(t, cmd)
}

func TestSummaryLinesCmdEmptyReturnsNil(t *testing.T) {
	cmd := summaryLinesCmd(io.Discard, nil)
	assert.Nil(t, cmd)
}

func TestReadExistingFileReturnsMissingWhenAbsent(t *testing.T) {
	fs := afero.NewMemMapFs()
	data, exists, err := readExistingFile(fs, "missing.json")
	require.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, data)
}

func TestReadExistingFileReturnsErrorOnStat(t *testing.T) {
	fs := statErrorFs{Fs: afero.NewMemMapFs(), failPath: "missing.json", err: errors.New("stat failed")}
	_, _, err := readExistingFile(fs, "missing.json")
	assert.Error(t, err)
}

func TestReadExistingFileReturnsErrorOnRead(t *testing.T) {
	base := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(base, "modlist.json", []byte("ok"), 0644))
	fs := openErrorFs{Fs: base, failPath: "modlist.json", err: errors.New("read failed")}
	_, _, err := readExistingFile(fs, "modlist.json")
	assert.Error(t, err)
}

func TestReadOriginalRemoveFilesCapturesMissingLock(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "modlist.json", []byte("{}"), 0644))

	original, err := readOriginalRemoveFiles(fs, config.NewMetadata("modlist.json"))
	require.NoError(t, err)
	assert.True(t, original.configExists)
	assert.False(t, original.lockExists)
}

func TestRemoveFileIfExistsRemovesWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "modlist.json", []byte("ok"), 0644))

	err := removeFileIfExists(fs, "modlist.json")
	require.NoError(t, err)

	exists, err := afero.Exists(fs, "modlist.json")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRemoveFileIfExistsReturnsErrorOnRemove(t *testing.T) {
	base := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(base, "modlist.json", []byte("ok"), 0644))
	fs := removeErrorFs{Fs: base, failPath: "modlist.json", err: errors.New("remove failed")}

	err := removeFileIfExists(fs, "modlist.json")
	assert.Error(t, err)
}

func TestRestoreExistingFileWrites(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := restoreExistingFile(fs, "modlist.json", []byte("ok"))
	require.NoError(t, err)

	data, err := afero.ReadFile(fs, "modlist.json")
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))
}

func TestApplyRemoveUpdatesReturnsErrorWhenOriginalFilesReadFails(t *testing.T) {
	fs := statErrorFs{Fs: afero.NewMemMapFs(), failPath: "modlist.json", err: errors.New("stat failed")}
	meta := config.NewMetadata("modlist.json")
	input := removeExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}},
		lock: []models.ModInstall{{Type: models.MODRINTH, ID: "sodium"}},
		deps: removeDeps{fs: fs},
	}

	outcome := applyRemoveUpdates(context.Background(), input, []removeItem{{
		Mod:    models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		Status: removeItemSuccess,
	}})
	assert.Error(t, outcome.err)
	assert.Equal(t, removeExecutionErrorUnknown, outcome.errType)
}

func TestApplyRemoveUpdatesReturnsJoinedErrorWhenConfigRestoreFails(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), base, meta, models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}))
	require.NoError(t, config.WriteLock(context.Background(), base, meta, []models.ModInstall{{Type: models.MODRINTH, ID: "sodium"}}))

	fs := writeFailureFs{
		Fs:             base,
		failRenamePath: meta.ConfigPath,
		failWritePath:  meta.ConfigPath,
		renameErr:      errors.New("rename failed"),
		writeErr:       errors.New("write failed"),
	}

	input := removeExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}},
		lock: []models.ModInstall{{Type: models.MODRINTH, ID: "sodium"}},
		deps: removeDeps{fs: fs},
	}

	outcome := applyRemoveUpdates(context.Background(), input, []removeItem{{
		Mod:    models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		Status: removeItemSuccess,
	}})
	assert.Error(t, outcome.err)
	assert.Equal(t, removeExecutionErrorWriteConfig, outcome.errType)
}

func TestApplyRemoveUpdatesReturnsNoErrorWhenNoItemsRemoved(t *testing.T) {
	input := removeExecutionInput{
		meta: config.NewMetadata("modlist.json"),
		cfg:  models.ModsJSON{},
		lock: []models.ModInstall{},
		deps: removeDeps{fs: afero.NewMemMapFs()},
	}

	outcome := applyRemoveUpdates(context.Background(), input, []removeItem{{
		Mod:    models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		Status: removeItemFailed,
	}})
	assert.Nil(t, outcome.err)
}

func TestReadOriginalRemoveFilesReturnsErrorOnLockRead(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, afero.WriteFile(base, meta.ConfigPath, []byte("{}"), 0644))
	require.NoError(t, afero.WriteFile(base, meta.LockPath(), []byte("[]"), 0644))

	fs := openErrorFs{Fs: base, failPath: meta.LockPath(), err: errors.New("read failed")}
	_, err := readOriginalRemoveFiles(fs, meta)
	assert.Error(t, err)
}

func TestRestoreOriginalConfigRemovesWhenMissing(t *testing.T) {
	base := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(base, "modlist.json", []byte("ok"), 0644))
	original := removeOriginalFiles{configExists: false}

	err := restoreOriginalConfig(base, config.NewMetadata("modlist.json"), original)
	require.NoError(t, err)

	exists, err := afero.Exists(base, "modlist.json")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRestoreOriginalConfigReturnsRemoveError(t *testing.T) {
	base := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(base, "modlist.json", []byte("ok"), 0644))
	fs := removeErrorFs{Fs: base, failPath: "modlist.json", err: errors.New("remove failed")}

	err := restoreOriginalConfig(fs, config.NewMetadata("modlist.json"), removeOriginalFiles{configExists: false})
	assert.Error(t, err)
}

func TestRestoreOriginalConfigWritesWhenExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	original := removeOriginalFiles{configExists: true, configBytes: []byte("ok")}

	err := restoreOriginalConfig(fs, config.NewMetadata("modlist.json"), original)
	require.NoError(t, err)

	data, err := afero.ReadFile(fs, "modlist.json")
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))
}

func TestRestoreOriginalFilesReturnsErrorWhenConfigRemoveFails(t *testing.T) {
	base := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(base, "modlist.json", []byte("ok"), 0644))
	fs := removeErrorFs{Fs: base, failPath: "modlist.json", err: errors.New("remove failed")}

	err := restoreOriginalFiles(fs, config.NewMetadata("modlist.json"), removeOriginalFiles{configExists: false})
	assert.Error(t, err)
}

func TestRestoreOriginalFilesWritesLockWhenExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	original := removeOriginalFiles{lockExists: true, lockBytes: []byte("[]")}

	err := restoreOriginalFiles(fs, meta, original)
	require.NoError(t, err)

	data, err := afero.ReadFile(fs, meta.LockPath())
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data))
}

func TestRestoreOriginalFilesRemovesLockWhenMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, afero.WriteFile(fs, meta.LockPath(), []byte("[]"), 0644))
	original := removeOriginalFiles{configExists: true, configBytes: []byte("{}"), lockExists: false}

	err := restoreOriginalFiles(fs, meta, original)
	require.NoError(t, err)

	exists, err := afero.Exists(fs, meta.LockPath())
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestApplyRemoveUpdatesReturnsJoinedErrorWhenLockRestoreFails(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, config.WriteConfig(context.Background(), base, meta, models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}))
	require.NoError(t, config.WriteLock(context.Background(), base, meta, []models.ModInstall{{Type: models.MODRINTH, ID: "sodium"}}))

	fs := writeFailureFs{
		Fs:             base,
		failRenamePath: meta.LockPath(),
		failWritePath:  meta.ConfigPath,
		renameErr:      errors.New("rename failed"),
		writeErr:       errors.New("write failed"),
	}

	input := removeExecutionInput{
		meta: meta,
		cfg:  models.ModsJSON{Mods: []models.Mod{{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}},
		lock: []models.ModInstall{{Type: models.MODRINTH, ID: "sodium"}},
		deps: removeDeps{fs: fs},
	}

	outcome := applyRemoveUpdates(context.Background(), input, []removeItem{{
		Mod:    models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"},
		Status: removeItemSuccess,
	}})
	assert.Error(t, outcome.err)
	assert.Equal(t, removeExecutionErrorWriteLock, outcome.errType)
}

func TestRemoveTranscriptModelIgnoresSuccessAfterFinish(t *testing.T) {
	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restoreUnicode)

	var out bytes.Buffer
	items := []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}}}
	indexByKey := map[string]int{"modrinth:sodium": 0}
	model := newRemoveTranscriptModel(context.Background(), view.ColorDisabled, items, indexByKey, &out, nil)

	_, cmd := model.Update(removeExecutionFinishedMsg{outcome: removeExecutionOutcome{
		items:   []removeItem{{Mod: models.Mod{Type: models.MODRINTH, ID: "sodium", Name: "Sodium"}, Status: removeItemSuccess}},
		errType: removeExecutionErrorNone,
	}})
	require.NotNil(t, cmd)
	_ = cmd()

	_, cmd = model.Update(removeItemSuccessMsg{key: "modrinth:sodium"})
	assert.Nil(t, cmd)
}

type statErrorFs struct {
	afero.Fs
	failPath string
	err      error
}

func (filesystem statErrorFs) Stat(name string) (os.FileInfo, error) {
	if filepath.Clean(name) == filepath.Clean(filesystem.failPath) {
		return nil, filesystem.err
	}
	return filesystem.Fs.Stat(name)
}

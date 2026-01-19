package locksync

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

type quitModel struct{}

func (quitModel) Init() tea.Cmd                       { return tea.Quit }
func (quitModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return quitModel{}, tea.Quit }
func (quitModel) View() string                        { return "" }

type stubModel struct{}

func (stubModel) Init() tea.Cmd                       { return nil }
func (stubModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return stubModel{}, nil }
func (stubModel) View() string                        { return "" }

func TestRegisterPolicyFlagsRegistersFlags(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterPolicyFlags(flags)
	assert.NotNil(t, flags.Lookup(FlagAdd))
	assert.NotNil(t, flags.Lookup(FlagDelete))
	assert.NotNil(t, flags.Lookup(FlagIgnore))
	assert.NotNil(t, flags.Lookup(FlagSkip))
}

func TestPolicyFlagsFromFlagsDefaultsWhenMissing(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	values, err := PolicyFlagsFromFlags(flags)
	require.NoError(t, err)
	assert.Equal(t, PolicyFlags{}, values)
}

func TestPolicyFlagsFromFlagsReturnsErrorOnWrongType(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String(FlagAdd, "", "")
	_, err := PolicyFlagsFromFlags(flags)
	assert.Error(t, err)
}

func TestPolicyFlagsFromFlagsReturnsErrorOnDeleteFlagType(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool(FlagAdd, false, "")
	flags.String(FlagDelete, "", "")
	_, err := PolicyFlagsFromFlags(flags)
	assert.Error(t, err)
}

func TestPolicyFlagsFromFlagsReturnsErrorOnIgnoreFlagType(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool(FlagAdd, false, "")
	flags.Bool(FlagDelete, false, "")
	flags.String(FlagIgnore, "", "")
	_, err := PolicyFlagsFromFlags(flags)
	assert.Error(t, err)
}

func TestPolicyFlagsFromFlagsReturnsErrorOnSkipFlagType(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool(FlagAdd, false, "")
	flags.Bool(FlagDelete, false, "")
	flags.Bool(FlagIgnore, false, "")
	flags.String(FlagSkip, "", "")
	_, err := PolicyFlagsFromFlags(flags)
	assert.Error(t, err)
}

func TestPolicyFlagsFromFlagsReadsValues(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterPolicyFlags(flags)
	require.NoError(t, flags.Set(FlagAdd, "true"))
	require.NoError(t, flags.Set(FlagDelete, "true"))
	require.NoError(t, flags.Set(FlagIgnore, "true"))
	require.NoError(t, flags.Set(FlagSkip, "true"))

	values, err := PolicyFlagsFromFlags(flags)
	require.NoError(t, err)
	assert.Equal(t, PolicyFlags{
		Add:    true,
		Delete: true,
		Ignore: true,
		Skip:   true,
	}, values)
}

func TestResolveRunTeaUsesDefaultWhenMissing(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	runTea := resolveRunTea(nil)
	_, err := runTea(quitModel{}, tea.WithInput(bytes.NewBuffer(nil)), tea.WithOutput(io.Discard))
	require.NoError(t, err)
}

func TestResolveRunTeaUsesProvidedRunner(t *testing.T) {
	called := false
	custom := func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
		called = true
		return model, nil
	}
	runTea := resolveRunTea(custom)
	_, err := runTea(quitModel{}, tea.WithInput(bytes.NewBuffer(nil)), tea.WithOutput(io.Discard))
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDecideLockSyncPolicyPromptCanceledReturnsNoContinue(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata("modlist.json")
	input := GateInput{
		RunTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return lockSyncPromptModel{canceled: true}, nil
		},
		Meta:      meta,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       io.Discard,
	}
	policy, shouldContinue, err := decideLockSyncPolicy(input, []extraLockEntry{}, PolicySkip, promptModeEnabled)
	require.NoError(t, err)
	assert.False(t, shouldContinue)
	assert.Equal(t, PolicyUnknown, policy)
}

func TestDecideLockSyncPolicyPromptSelectsPolicy(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata("modlist.json")
	input := GateInput{
		RunTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return lockSyncPromptModel{policy: PolicyDelete}, nil
		},
		Meta:      meta,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       io.Discard,
	}
	policy, shouldContinue, err := decideLockSyncPolicy(input, []extraLockEntry{}, PolicySkip, promptModeEnabled)
	require.NoError(t, err)
	assert.True(t, shouldContinue)
	assert.Equal(t, PolicyDelete, policy)
}

func TestDecideLockSyncPolicyReturnsPolicyWhenPromptDisabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata("modlist.json")
	input := GateInput{
		RunTea: func(model tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			return model, nil
		},
		Meta:      meta,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       io.Discard,
	}
	policy, shouldContinue, err := decideLockSyncPolicy(input, []extraLockEntry{}, PolicySkip, promptModeDisabled)
	require.NoError(t, err)
	assert.True(t, shouldContinue)
	assert.Equal(t, PolicySkip, policy)
}

func TestDecideLockSyncPolicyReturnsErrorOnPromptFailure(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata("modlist.json")
	expectedErr := errors.New("prompt failed")
	input := GateInput{
		RunTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, expectedErr
		},
		Meta:      meta,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       io.Discard,
	}
	_, _, err := decideLockSyncPolicy(input, []extraLockEntry{}, PolicySkip, promptModeEnabled)
	assert.ErrorIs(t, err, expectedErr)
}

func TestApplyLockSyncPolicyUnknownReturnsError(t *testing.T) {
	outcome := GateOutcome{Policy: PolicyUnknown}
	_, err := applyLockSyncPolicy(GateInput{}, outcome, nil)
	assert.Error(t, err)
}

func TestApplyLockSyncPolicyAddNoChangeReturnsUnapplied(t *testing.T) {
	cfg := models.ModsJSON{
		Mods: []models.Mod{{
			Type: models.MODRINTH,
			ID:   "alpha",
			Name: "Alpha",
		}},
	}
	extras := []extraLockEntry{{
		Install:     models.ModInstall{Type: models.MODRINTH, ID: "alpha", Name: "Alpha"},
		DisplayName: "Alpha",
	}}
	outcome, err := applyLockSyncPolicy(GateInput{}, GateOutcome{Policy: PolicyAdd, Config: cfg}, extras)
	require.NoError(t, err)
	assert.False(t, outcome.Applied)
	assert.Equal(t, cfg, outcome.Config)
}

func TestApplyLockSyncPolicyAddWriteFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, base.MkdirAll(meta.Dir(), 0o755))
	readonly := afero.NewReadOnlyFs(base)

	cfg := models.ModsJSON{ModsFolder: "mods"}
	extras := []extraLockEntry{{
		Install:     models.ModInstall{Type: models.MODRINTH, ID: "alpha", Name: "Alpha"},
		DisplayName: "Alpha",
	}}
	_, err := applyLockSyncPolicy(GateInput{
		Ctx:  context.Background(),
		Fs:   readonly,
		Meta: meta,
	}, GateOutcome{Policy: PolicyAdd, Config: cfg}, extras)
	assert.Error(t, err)
}

func TestApplyLockSyncPolicyDeleteWriteFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, base.MkdirAll(meta.Dir(), 0o755))
	readonly := afero.NewReadOnlyFs(base)

	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "alpha", FileName: "alpha.jar"}}
	extras := []extraLockEntry{{
		Install:     lock[0],
		DisplayName: "alpha",
		FileStatus:  fileStatusMissing,
	}}
	_, err := applyLockSyncPolicy(GateInput{
		Ctx:  context.Background(),
		Fs:   readonly,
		Meta: meta,
	}, GateOutcome{Policy: PolicyDelete, Config: cfg, Lock: lock}, extras)
	assert.Error(t, err)
}

func TestApplyLockSyncPolicyIgnoreWriteFailureReturnsError(t *testing.T) {
	base := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	require.NoError(t, base.MkdirAll(meta.Dir(), 0o755))

	fs := lockWriteErrorFs{
		Fs:            base,
		failSubstring: "modlist-lock.json",
		err:           errors.New("write failed"),
	}

	cfg := models.ModsJSON{ModsFolder: "mods"}
	lock := []models.ModInstall{{Type: models.MODRINTH, ID: "alpha", FileName: "alpha.jar"}}
	extras := []extraLockEntry{{
		Install:            lock[0],
		DisplayName:        "alpha",
		NormalizedFileName: "alpha.jar",
		FileStatus:         fileStatusPresent,
	}}
	_, err := applyLockSyncPolicy(GateInput{
		Ctx:  context.Background(),
		Fs:   fs,
		Meta: meta,
	}, GateOutcome{Policy: PolicyIgnore, Config: cfg, Lock: lock}, extras)
	assert.ErrorIs(t, err, fs.err)
}

func TestSortExtraLockEntriesSortsByNamePlatformAndID(t *testing.T) {
	entries := []extraLockEntry{
		{DisplayName: "Beta", Install: models.ModInstall{Type: models.CURSEFORGE, ID: "b"}},
		{DisplayName: "Alpha", Install: models.ModInstall{Type: models.MODRINTH, ID: "b"}},
		{DisplayName: "Alpha", Install: models.ModInstall{Type: models.MODRINTH, ID: "a"}},
	}
	sorted := sortExtraLockEntries(entries)
	require.Len(t, sorted, 3)
	assert.Equal(t, "Alpha", sorted[0].DisplayName)
	assert.Equal(t, "a", sorted[0].Install.ID)
	assert.Equal(t, "Alpha", sorted[1].DisplayName)
	assert.Equal(t, "b", sorted[1].Install.ID)
	assert.Equal(t, "Beta", sorted[2].DisplayName)
}

func TestSortExtraLockEntriesSortsByPlatformWhenNamesEqual(t *testing.T) {
	entries := []extraLockEntry{
		{DisplayName: "Alpha", Install: models.ModInstall{Type: models.MODRINTH, ID: "b"}},
		{DisplayName: "Alpha", Install: models.ModInstall{Type: models.CURSEFORGE, ID: "a"}},
	}
	sorted := sortExtraLockEntries(entries)
	require.Len(t, sorted, 2)
	assert.Equal(t, models.CURSEFORGE, sorted[0].Install.Type)
	assert.Equal(t, models.MODRINTH, sorted[1].Install.Type)
}

func TestDeleteExtraLockFilesSkipsMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	err := deleteExtraLockFiles(fs, meta, cfg, []extraLockEntry{{
		Install:    models.ModInstall{FileName: "missing.jar"},
		FileStatus: fileStatusMissing,
	}})
	require.NoError(t, err)
}

func TestDeleteExtraLockFilesRemovesPresentFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	modsDir := filepath.Join(meta.Dir(), "mods")
	require.NoError(t, fs.MkdirAll(modsDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modsDir, "alpha.jar"), []byte("ok"), 0o644))

	err := deleteExtraLockFiles(fs, meta, cfg, []extraLockEntry{{
		Install:            models.ModInstall{FileName: "alpha.jar"},
		NormalizedFileName: "alpha.jar",
		FileStatus:         fileStatusPresent,
	}})
	require.NoError(t, err)

	exists, existsErr := afero.Exists(fs, filepath.Join(modsDir, "alpha.jar"))
	require.NoError(t, existsErr)
	assert.False(t, exists)
}

func TestDeleteExtraLockFilesReturnsErrorOnResolveFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "locksync-resolve")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("cleanup temp dir: %v", err)
		}
	})

	base := afero.NewOsFs()
	meta := config.NewMetadata(filepath.Join(tempDir, "modlist.json"))
	cfg := models.ModsJSON{ModsFolder: "mods"}
	modsDir := filepath.Join(meta.Dir(), "mods")
	require.NoError(t, base.MkdirAll(modsDir, 0o755))
	require.NoError(t, afero.WriteFile(base, filepath.Join(modsDir, "alpha.jar"), []byte("ok"), 0o644))

	fs := resolveErrorFs{Fs: base, err: errors.New("resolve failed")}
	deleteErr := deleteExtraLockFiles(fs, meta, cfg, []extraLockEntry{{
		Install:            models.ModInstall{FileName: "alpha.jar"},
		NormalizedFileName: "alpha.jar",
		FileStatus:         fileStatusPresent,
	}})
	assert.ErrorIs(t, deleteErr, fs.err)
}

func TestRemoveExtraLockEntriesNoExtrasReturnsOriginal(t *testing.T) {
	lock := []models.ModInstall{{ID: "alpha", Type: models.MODRINTH}}
	remaining := removeExtraLockEntries(lock, nil)
	require.Len(t, remaining, 1)
	assert.Equal(t, "alpha", remaining[0].ID)
}

func TestLockSyncPromptResultPointerModel(t *testing.T) {
	outcome, err := lockSyncPromptResult(&lockSyncPromptModel{policy: PolicySkip})
	require.NoError(t, err)
	assert.Equal(t, PolicySkip, outcome.policy)
}

func TestLockSyncPromptResultUnexpectedModelReturnsError(t *testing.T) {
	_, err := lockSyncPromptResult(stubModel{})
	assert.Error(t, err)
}

func TestLockSyncPromptModelInitReturnsNil(t *testing.T) {
	model := newLockSyncPromptModel("list")
	assert.Nil(t, model.Init())
}

func TestLockSyncPolicyItemFilterValue(t *testing.T) {
	item := lockSyncPolicyItem{label: "label"}
	assert.Equal(t, "", item.FilterValue())
}

func TestLockSyncPolicyDelegateUpdateReturnsNil(t *testing.T) {
	delegate := lockSyncPolicyDelegate{}
	assert.Nil(t, delegate.Update(nil, &list.Model{}))
}

func TestLockSyncPolicyDelegateRenderWritesSelection(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	items := []list.Item{
		lockSyncPolicyItem{label: "Alpha"},
		lockSyncPolicyItem{label: "Beta"},
	}
	model := list.New(items, lockSyncPolicyDelegate{}, 80, 5)
	model.Select(0)

	var selected bytes.Buffer
	lockSyncPolicyDelegate{}.Render(&selected, model, 0, items[0])
	assert.Contains(t, selected.String(), "Alpha")

	var unselected bytes.Buffer
	lockSyncPolicyDelegate{}.Render(&unselected, model, 1, items[1])
	assert.Contains(t, unselected.String(), "Beta")
}

func TestLockSyncPolicyDelegateRenderSkipsUnknownItem(t *testing.T) {
	var output bytes.Buffer
	model := list.New([]list.Item{stubListItem{}}, lockSyncPolicyDelegate{}, 80, 5)
	lockSyncPolicyDelegate{}.Render(&output, model, 0, stubListItem{})
	assert.Equal(t, "", output.String())
}

func TestLockSyncPolicyDelegateRenderReturnsOnSelectedWriteError(t *testing.T) {
	items := []list.Item{
		lockSyncPolicyItem{label: "Alpha"},
	}
	model := list.New(items, lockSyncPolicyDelegate{}, 80, 5)
	model.Select(0)
	assert.NotPanics(t, func() {
		lockSyncPolicyDelegate{}.Render(errorWriter{err: errors.New("write failed")}, model, 0, items[0])
	})
}

func TestLockSyncPolicyDelegateRenderReturnsOnUnselectedWriteError(t *testing.T) {
	items := []list.Item{
		lockSyncPolicyItem{label: "Alpha"},
		lockSyncPolicyItem{label: "Beta"},
	}
	model := list.New(items, lockSyncPolicyDelegate{}, 80, 5)
	model.Select(0)
	assert.NotPanics(t, func() {
		lockSyncPolicyDelegate{}.Render(errorWriter{err: errors.New("write failed")}, model, 1, items[1])
	})
}

func TestLockSyncPointerRespectsUnicodeSupport(t *testing.T) {
	restore := view.SetUnicodeSupportFuncForTesting(func() bool { return true })
	t.Cleanup(restore)
	assert.Equal(t, "\u276F ", lockSyncPointer())

	restore = view.SetUnicodeSupportFuncForTesting(func() bool { return false })
	t.Cleanup(restore)
	assert.Equal(t, "> ", lockSyncPointer())
}

func TestLockSyncAnswerLineReturnsQuestionWhenUnknown(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	line := lockSyncAnswerLine(PolicyUnknown)
	assert.Contains(t, line, "cmd.lock_sync.prompt")
}

func TestLockSyncAnswerLineIncludesAnswerWhenKnown(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	line := lockSyncAnswerLine(PolicySkip)
	assert.Contains(t, line, "cmd.lock_sync.answer.skip")
}

func TestDeleteExtraLockFilesReturnsErrorOnInvalid(t *testing.T) {
	fs := afero.NewMemMapFs()
	meta := config.NewMetadata("modlist.json")
	cfg := models.ModsJSON{ModsFolder: "mods"}
	err := deleteExtraLockFiles(fs, meta, cfg, []extraLockEntry{{
		Install:     models.ModInstall{FileName: "invalid.jar"},
		DisplayName: "Invalid",
		FileStatus:  fileStatusInvalid,
	}})
	assert.Error(t, err)
}

func TestDecideLockSyncPolicyIgnoresSummaryRunnerWhenPromptDisabled(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	meta := config.NewMetadata("modlist.json")
	expectedErr := errors.New("summary failed")
	input := GateInput{
		RunTea: func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
			return nil, expectedErr
		},
		Meta:      meta,
		ColorMode: view.ColorDisabled,
		In:        bytes.NewBuffer(nil),
		Out:       io.Discard,
	}
	_, _, err := decideLockSyncPolicy(input, []extraLockEntry{}, PolicySkip, promptModeDisabled)
	assert.NoError(t, err)
}

func TestResolvePolicySelectsFlagsForceAndMode(t *testing.T) {
	cases := []struct {
		name       string
		flags      PolicyFlags
		force      forceMode
		mode       interaction.ExecutionMode
		wantPolicy Policy
		wantPrompt promptMode
		wantErr    bool
	}{
		{
			name:       "single flag add",
			flags:      PolicyFlags{Add: true},
			force:      forceModeUnset,
			mode:       interaction.ExecutionModeNonTTY,
			wantPolicy: PolicyAdd,
			wantPrompt: promptModeDisabled,
		},
		{
			name:       "single flag ignore",
			flags:      PolicyFlags{Ignore: true},
			force:      forceModeUnset,
			mode:       interaction.ExecutionModeNonTTY,
			wantPolicy: PolicyIgnore,
			wantPrompt: promptModeDisabled,
		},
		{
			name:       "single flag skip",
			flags:      PolicyFlags{Skip: true},
			force:      forceModeUnset,
			mode:       interaction.ExecutionModeNonTTY,
			wantPolicy: PolicySkip,
			wantPrompt: promptModeDisabled,
		},
		{
			name:       "force selects skip",
			flags:      PolicyFlags{},
			force:      forceModeEnabled,
			mode:       interaction.ExecutionModeNonTTY,
			wantPolicy: PolicySkip,
			wantPrompt: promptModeDisabled,
		},
		{
			name:       "interactive prompts",
			flags:      PolicyFlags{},
			force:      forceModeUnset,
			mode:       interaction.ExecutionModeInteractive,
			wantPolicy: PolicyUnknown,
			wantPrompt: promptModeEnabled,
		},
		{
			name:       "non-interactive defaults add",
			flags:      PolicyFlags{},
			force:      forceModeUnset,
			mode:       interaction.ExecutionModeNonTTY,
			wantPolicy: PolicyAdd,
			wantPrompt: promptModeDisabled,
		},
		{
			name:    "multiple flags error",
			flags:   PolicyFlags{Add: true, Delete: true},
			force:   forceModeUnset,
			mode:    interaction.ExecutionModeNonTTY,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, promptState, err := resolvePolicy(tc.flags, tc.force, tc.mode)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantPolicy, policy)
			assert.Equal(t, tc.wantPrompt, promptState)
		})
	}
}

func TestLockSyncEntrySuffixPresentUsesFilename(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	entry := extraLockEntry{
		Install: models.ModInstall{
			Type:     models.MODRINTH,
			ID:       "alpha",
			Name:     "Alpha",
			FileName: "mods/alpha.jar",
		},
		FileStatus: fileStatusPresent,
	}
	line := lockSyncEntrySuffix(entry)
	assert.Contains(t, line, "cmd.lock_sync.entry.present_suffix")
	assert.Contains(t, line, "file:mods/alpha.jar")
}

func TestLockSyncEntrySuffixMissingUsesFilename(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	entry := extraLockEntry{
		Install: models.ModInstall{
			Type:     models.MODRINTH,
			ID:       "alpha",
			Name:     "Alpha",
			FileName: "mods/alpha.jar",
		},
		FileStatus: fileStatusMissing,
	}
	line := lockSyncEntrySuffix(entry)
	assert.Contains(t, line, "cmd.lock_sync.entry.missing_suffix")
	assert.Contains(t, line, "file:mods/alpha.jar")
}

func TestLockSyncEntrySuffixInvalidUsesFilename(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	entry := extraLockEntry{
		Install: models.ModInstall{
			Type:     models.MODRINTH,
			ID:       "alpha",
			Name:     "Alpha",
			FileName: "../alpha.jar",
		},
		FileStatus: fileStatusInvalid,
	}
	line := lockSyncEntrySuffix(entry)
	assert.Contains(t, line, "cmd.lock_sync.entry.invalid_suffix")
	assert.Contains(t, line, "file:../alpha.jar")
}

func TestLockSyncEntrySuffixUnknownReturnsEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	entry := extraLockEntry{
		Install: models.ModInstall{
			Type:     models.MODRINTH,
			ID:       "alpha",
			Name:     "Alpha",
			FileName: "alpha.jar",
		},
		FileStatus: fileStatus(99),
	}
	assert.Equal(t, "", lockSyncEntrySuffix(entry))
}

func TestLockSyncResolutionLinesIncludeContinueAndDivider(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := lockSyncResolutionLines("install", view.ColorDisabled, PolicyAdd)
	require.Len(t, lines, 3)
	assert.Contains(t, lines[0], "cmd.lock_sync.result.add")
	assert.Contains(t, lines[1], "cmd.lock_sync.continue")
	assert.Equal(t, strings.Repeat("-", 61), lines[2])
}

func TestLockSyncResolutionLinesOmitContinueWhenCommandEmpty(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := lockSyncResolutionLines("", view.ColorDisabled, PolicyAdd)
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "cmd.lock_sync.result.add")
	assert.Equal(t, strings.Repeat("-", 61), lines[1])
}

func TestLockSyncResolutionLinesUnknownPolicyReturnsNil(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	lines := lockSyncResolutionLines("install", view.ColorDisabled, PolicyUnknown)
	assert.Nil(t, lines)
}

func TestLockSyncResultLineHandlesPolicies(t *testing.T) {
	t.Setenv("MMM_TEST", "true")
	cases := []struct {
		name   string
		policy Policy
		expect string
	}{
		{name: "add", policy: PolicyAdd, expect: "cmd.lock_sync.result.add"},
		{name: "delete", policy: PolicyDelete, expect: "cmd.lock_sync.result.delete"},
		{name: "ignore", policy: PolicyIgnore, expect: "cmd.lock_sync.result.ignore"},
		{name: "skip", policy: PolicySkip, expect: "cmd.common.no_changes"},
		{name: "unknown", policy: PolicyUnknown, expect: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line := lockSyncResultLine(view.ColorDisabled, tc.policy)
			if tc.expect == "" {
				assert.Equal(t, "", line)
				return
			}
			assert.Contains(t, line, tc.expect)
		})
	}
}

type stubListItem struct{}

func (stubListItem) FilterValue() string { return "" }

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type resolveErrorFs struct {
	afero.Fs
	err error
}

func (fs resolveErrorFs) ReadlinkIfPossible(string) (string, error) {
	return "", fs.err
}

func (fs resolveErrorFs) LstatIfPossible(string) (os.FileInfo, bool, error) {
	return nil, false, fs.err
}

type lockWriteErrorFs struct {
	afero.Fs
	failSubstring string
	err           error
}

func (fs lockWriteErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if strings.Contains(filepath.Base(name), fs.failSubstring) {
		return nil, fs.err
	}
	return fs.Fs.OpenFile(name, flag, perm)
}

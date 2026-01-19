// Package locksync coordinates sync decisions when the lock file contains entries missing from config.
package locksync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/interaction"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modfilename"
	"github.com/meza/minecraft-mod-manager/internal/modpath"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

const (
	FlagAdd    = "lock-sync-add"
	FlagDelete = "lock-sync-delete"
	FlagIgnore = "lock-sync-ignore"
	FlagSkip   = "lock-sync-skip"
)

type Policy int

const (
	PolicyUnknown Policy = iota
	PolicyAdd
	PolicyDelete
	PolicyIgnore
	PolicySkip
)

type forceMode int

const (
	forceModeUnset forceMode = iota
	forceModeEnabled
)

type promptMode int

const (
	promptModeDisabled promptMode = iota
	promptModeEnabled
)

type PolicyFlags struct {
	Add    bool
	Delete bool
	Ignore bool
	Skip   bool
}

func RegisterPolicyFlags(flags *pflag.FlagSet) {
	flags.Bool(FlagAdd, false, "Add lockfile-only mods to the config when syncing")
	flags.Bool(FlagDelete, false, "Delete lockfile-only mods from disk and remove their lock entries when syncing")
	flags.Bool(FlagIgnore, false, "Ignore lockfile-only mods by adding them to .mmmignore and removing their lock entries")
	flags.Bool(FlagSkip, false, "Do nothing when lockfile-only mods are detected")
}

func PolicyFlagsFromFlags(flags *pflag.FlagSet) (PolicyFlags, error) {
	add, err := optionalFlagBool(flags, FlagAdd)
	if err != nil {
		return PolicyFlags{}, err
	}
	deleteFlag, err := optionalFlagBool(flags, FlagDelete)
	if err != nil {
		return PolicyFlags{}, err
	}
	ignore, err := optionalFlagBool(flags, FlagIgnore)
	if err != nil {
		return PolicyFlags{}, err
	}
	skip, err := optionalFlagBool(flags, FlagSkip)
	if err != nil {
		return PolicyFlags{}, err
	}
	return PolicyFlags{
		Add:    add,
		Delete: deleteFlag,
		Ignore: ignore,
		Skip:   skip,
	}, nil
}

func optionalFlagBool(flags *pflag.FlagSet, name string) (bool, error) {
	if flags.Lookup(name) == nil {
		return false, nil
	}
	return flags.GetBool(name)
}

type GateInput struct {
	Ctx         context.Context
	Fs          afero.Fs
	Meta        config.Metadata
	Config      models.ModsJSON
	Lock        []models.ModInstall
	Mode        interaction.ExecutionMode
	CommandName string
	ColorMode   view.ColorMode
	In          io.Reader
	Out         io.Writer
	RunTea      func(tea.Model, ...tea.ProgramOption) (tea.Model, error)
	PolicyFlags PolicyFlags
	Force       bool
}

type GateOutcome struct {
	Config         models.ModsJSON
	Lock           []models.ModInstall
	ShouldContinue bool
	Applied        bool
	Policy         Policy
}

func resolveRunTea(runTea func(tea.Model, ...tea.ProgramOption) (tea.Model, error)) func(tea.Model, ...tea.ProgramOption) (tea.Model, error) {
	if runTea != nil {
		return runTea
	}
	return defaultRunTea
}

func defaultRunTea(model tea.Model, options ...tea.ProgramOption) (tea.Model, error) {
	return tea.NewProgram(model, options...).Run()
}

func lockSyncOutputProgramOptions(out io.Writer) []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithInput(nil),
		tea.WithOutput(out),
		tea.WithoutRenderer(),
	}
}

func RunLockSyncGate(input GateInput) (GateOutcome, error) {
	resolvedInput := input
	resolvedInput.RunTea = resolveRunTea(input.RunTea)

	outcome := GateOutcome{
		Config:         input.Config,
		Lock:           input.Lock,
		ShouldContinue: true,
	}

	extras, err := buildExtraLockEntries(input.Fs, input.Meta, input.Config, input.Lock)
	if err != nil {
		return outcome, err
	}
	if len(extras) == 0 {
		return outcome, nil
	}

	forceModeValue := forceModeUnset
	if input.Force {
		forceModeValue = forceModeEnabled
	}
	policy, promptState, err := resolvePolicy(input.PolicyFlags, forceModeValue, input.Mode)
	if err != nil {
		return outcome, err
	}

	sortedExtras := sortExtraLockEntries(extras)
	policy, shouldContinue, err := decideLockSyncPolicy(resolvedInput, sortedExtras, policy, promptState)
	if err != nil {
		return outcome, err
	}
	if !shouldContinue {
		return GateOutcome{
			Config:         input.Config,
			Lock:           input.Lock,
			ShouldContinue: false,
		}, nil
	}

	outcome.Policy = policy
	updatedOutcome, err := applyLockSyncPolicy(resolvedInput, outcome, sortedExtras)
	if err != nil {
		return updatedOutcome, err
	}
	if outputErr := writeLockSyncGateOutput(resolvedInput, updatedOutcome.Policy, sortedExtras, promptState); outputErr != nil {
		return updatedOutcome, outputErr
	}
	return updatedOutcome, nil
}

func decideLockSyncPolicy(input GateInput, extras []extraLockEntry, policy Policy, promptMode promptMode) (Policy, bool, error) {
	if promptMode == promptModeEnabled {
		promptOutcome, promptErr := runLockSyncPrompt(lockSyncPromptInput{
			listView: renderLockSyncList(extras, input.ColorMode),
			runTea:   input.RunTea,
			in:       input.In,
			out:      input.Out,
		})
		if promptErr != nil {
			return PolicyUnknown, false, promptErr
		}
		if promptOutcome.canceled {
			return PolicyUnknown, false, nil
		}
		return promptOutcome.policy, true, nil
	}

	return policy, true, nil
}

func writeLockSyncGateOutput(input GateInput, policy Policy, extras []extraLockEntry, promptState promptMode) error {
	if promptState == promptModeEnabled {
		return writeLockSyncResolution(input, policy)
	}
	return writeLockSyncSummary(lockSyncSummaryInput{
		runTea:    input.RunTea,
		in:        input.In,
		out:       input.Out,
		colorMode: input.ColorMode,
		meta:      input.Meta,
		extras:    extras,
		policy:    policy,
		command:   input.CommandName,
	})
}

func applyLockSyncPolicy(input GateInput, outcome GateOutcome, extras []extraLockEntry) (GateOutcome, error) {
	switch outcome.Policy {
	case PolicyAdd:
		updatedConfig, changed := addLockEntriesToConfig(outcome.Config, extras)
		if changed {
			if err := config.WriteConfig(input.Ctx, input.Fs, input.Meta, updatedConfig); err != nil {
				return outcome, err
			}
			outcome.Config = updatedConfig
			outcome.Applied = true
		}
		return outcome, nil
	case PolicyDelete:
		if err := deleteExtraLockFiles(input.Fs, input.Meta, outcome.Config, extras); err != nil {
			return outcome, err
		}
		updatedLock := removeExtraLockEntries(outcome.Lock, extras)
		if err := config.WriteLock(input.Ctx, input.Fs, input.Meta, updatedLock); err != nil {
			return outcome, err
		}
		outcome.Lock = updatedLock
		outcome.Applied = true
		return outcome, nil
	case PolicyIgnore:
		if err := appendIgnoreEntries(input.Fs, input.Meta, extras); err != nil {
			return outcome, err
		}
		updatedLock := removeExtraLockEntries(outcome.Lock, extras)
		if err := config.WriteLock(input.Ctx, input.Fs, input.Meta, updatedLock); err != nil {
			return outcome, err
		}
		outcome.Lock = updatedLock
		outcome.Applied = true
		return outcome, nil
	case PolicySkip:
		return outcome, nil
	default:
		return outcome, errors.New("unexpected lock sync policy")
	}
}

func resolvePolicy(flags PolicyFlags, force forceMode, mode interaction.ExecutionMode) (Policy, promptMode, error) {
	selected := []Policy{}
	if flags.Add {
		selected = append(selected, PolicyAdd)
	}
	if flags.Delete {
		selected = append(selected, PolicyDelete)
	}
	if flags.Ignore {
		selected = append(selected, PolicyIgnore)
	}
	if flags.Skip {
		selected = append(selected, PolicySkip)
	}

	if len(selected) > 1 {
		return PolicyUnknown, promptModeDisabled, errors.New(i18n.T("cmd.lock_sync.error.policy_multiple", nil))
	}
	if len(selected) == 1 {
		return selected[0], promptModeDisabled, nil
	}
	if force == forceModeEnabled {
		return PolicySkip, promptModeDisabled, nil
	}
	if mode == interaction.ExecutionModeInteractive {
		return PolicyUnknown, promptModeEnabled, nil
	}
	return PolicyAdd, promptModeDisabled, nil
}

func writeLockSyncResolution(input GateInput, policy Policy) error {
	if input.RunTea == nil {
		return errors.New("missing lock sync summary runner")
	}
	lines := lockSyncResolutionLines(input.CommandName, input.ColorMode, policy)
	if len(lines) == 0 {
		return nil
	}
	model := view.OutputLinesModel{
		Lines:     []string{strings.Join(lines, "\n")},
		Output:    input.Out,
		Separator: view.SectionSeparatorParagraph,
	}
	return view.RunOutputLines(input.RunTea, model, lockSyncOutputProgramOptions(input.Out)...)
}

type fileStatus int

const (
	fileStatusMissing fileStatus = iota
	fileStatusPresent
	fileStatusInvalid
)

type extraLockEntry struct {
	Install            models.ModInstall
	DisplayName        string
	NormalizedFileName string
	FileStatus         fileStatus
}

func buildExtraLockEntries(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall) ([]extraLockEntry, error) {
	configKeys := make(map[string]struct{}, len(cfg.Mods))
	for _, mod := range cfg.Mods {
		configKeys[lockKey(mod.Type, mod.ID)] = struct{}{}
	}

	extras := make([]extraLockEntry, 0)
	for _, install := range lock {
		if _, ok := configKeys[lockKey(install.Type, install.ID)]; ok {
			continue
		}
		entry := extraLockEntry{
			Install:     install,
			DisplayName: lockDisplayName(install),
		}
		normalizedFileName, err := modfilename.Normalize(install.FileName)
		if err != nil {
			entry.FileStatus = fileStatusInvalid
			extras = append(extras, entry)
			continue
		}
		entry.NormalizedFileName = normalizedFileName
		path := filepath.Join(meta.ModsFolderPath(cfg), normalizedFileName)
		exists, err := afero.Exists(fs, path)
		if err != nil {
			return nil, err
		}
		if exists {
			entry.FileStatus = fileStatusPresent
		} else {
			entry.FileStatus = fileStatusMissing
		}
		extras = append(extras, entry)
	}

	return extras, nil
}

func sortExtraLockEntries(entries []extraLockEntry) []extraLockEntry {
	sorted := append([]extraLockEntry{}, entries...)
	sort.SliceStable(sorted, func(leftIndex int, rightIndex int) bool {
		left := strings.ToLower(sorted[leftIndex].DisplayName)
		right := strings.ToLower(sorted[rightIndex].DisplayName)
		if left != right {
			return left < right
		}
		leftPlatform := sorted[leftIndex].Install.Type
		rightPlatform := sorted[rightIndex].Install.Type
		if leftPlatform != rightPlatform {
			return leftPlatform < rightPlatform
		}
		return strings.ToLower(sorted[leftIndex].Install.ID) < strings.ToLower(sorted[rightIndex].Install.ID)
	})
	return sorted
}

func lockKey(platform models.Platform, id string) string {
	return fmt.Sprintf("%s:%s", platform, id)
}

func lockDisplayName(install models.ModInstall) string {
	name := strings.TrimSpace(install.Name)
	if name == "" {
		return strings.TrimSpace(install.ID)
	}
	return name
}

func addLockEntriesToConfig(cfg models.ModsJSON, extras []extraLockEntry) (models.ModsJSON, bool) {
	updated := cfg
	existing := make(map[string]struct{}, len(cfg.Mods))
	for _, mod := range cfg.Mods {
		existing[lockKey(mod.Type, mod.ID)] = struct{}{}
	}

	changed := false
	for _, entry := range extras {
		key := lockKey(entry.Install.Type, entry.Install.ID)
		if _, ok := existing[key]; ok {
			continue
		}
		updated.Mods = append(updated.Mods, models.Mod{
			Type: entry.Install.Type,
			ID:   entry.Install.ID,
			Name: lockDisplayName(entry.Install),
		})
		existing[key] = struct{}{}
		changed = true
	}
	return updated, changed
}

func deleteExtraLockFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, extras []extraLockEntry) error {
	modsRoot := meta.ModsFolderPath(cfg)
	for _, entry := range extras {
		switch entry.FileStatus {
		case fileStatusInvalid:
			return errors.New(i18n.T("cmd.lock_sync.error.invalid_filename_lock", &i18n.Tvars{
				Data: &i18n.TData{
					"name": entry.DisplayName,
					"file": modfilename.Display(entry.Install.FileName),
				},
			}))
		case fileStatusMissing:
			continue
		case fileStatusPresent:
			destination := filepath.Join(modsRoot, entry.NormalizedFileName)
			resolved, err := modpath.ResolveWritablePath(fs, modsRoot, destination)
			if err != nil {
				return err
			}
			if err := removeFileForce(fs, resolved); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendIgnoreEntries(fs afero.Fs, meta config.Metadata, extras []extraLockEntry) error {
	files := make([]string, 0)
	for _, entry := range extras {
		switch entry.FileStatus {
		case fileStatusInvalid:
			return errors.New(i18n.T("cmd.lock_sync.error.invalid_filename_lock", &i18n.Tvars{
				Data: &i18n.TData{
					"name": entry.DisplayName,
					"file": modfilename.Display(entry.Install.FileName),
				},
			}))
		case fileStatusPresent:
			files = append(files, entry.NormalizedFileName)
		case fileStatusMissing:
			continue
		}
	}
	if len(files) == 0 {
		return nil
	}
	return mmmignore.AppendPatterns(fs, meta.Dir(), files)
}

func removeExtraLockEntries(lock []models.ModInstall, extras []extraLockEntry) []models.ModInstall {
	extraKeys := make(map[string]struct{}, len(extras))
	for _, entry := range extras {
		extraKeys[lockKey(entry.Install.Type, entry.Install.ID)] = struct{}{}
	}

	remaining := make([]models.ModInstall, 0, len(lock))
	for _, install := range lock {
		if _, ok := extraKeys[lockKey(install.Type, install.ID)]; ok {
			continue
		}
		remaining = append(remaining, install)
	}
	return remaining
}

func removeFileForce(fs afero.Fs, path string) error {
	if err := fs.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

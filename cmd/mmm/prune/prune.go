package prune

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/cmddeps"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/output"
	"github.com/meza/minecraft-mod-manager/internal/perf"
	"github.com/meza/minecraft-mod-manager/internal/telemetry"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
)

type pruneOptions struct {
	ConfigPath string
	Unattended bool
	Quiet      bool
	Debug      bool
	Force      bool
}

type pruneDeps struct {
	fs        afero.Fs
	logger    *logger.Logger
	output    *output.Output
	telemetry func(telemetry.CommandTelemetry)
}

var errPromptDisabled = errors.New("prune aborted: prompt disabled")
var absPath = filepath.Abs

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: i18n.T("cmd.prune.short", nil),
		RunE:  runPruneCommand,
	}

	cmd.Flags().BoolP("force", "f", false, i18n.T("cmd.prune.flag.force", nil))

	return cmd
}

func runPruneCommand(cmd *cobra.Command, _ []string) error {
	ctx, span := perf.StartSpan(cmd.Context(), "app.command.prune")

	options, err := pruneOptionsFromFlags(cmd)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		span.End()
		return err
	}

	deps := defaultPruneDeps(cmd, options)
	deletedCount, runErr := runPrune(ctx, cmd, options, deps)
	span.SetAttributes(attribute.Bool("success", runErr == nil))
	span.End()

	recordPruneTelemetry(deps.telemetry, options, deletedCount, runErr)
	applyPruneCommandErrorPolicy(cmd, runErr)
	return runErr
}

func applyPruneCommandErrorPolicy(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	if clierrors.IsHandled(err) {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
}

func pruneOptionsFromFlags(cmd *cobra.Command) (pruneOptions, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return pruneOptions{}, err
	}
	unattended, err := cmd.Flags().GetBool("unattended")
	if err != nil {
		return pruneOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return pruneOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return pruneOptions{}, err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return pruneOptions{}, err
	}

	return pruneOptions{
		ConfigPath: configPath,
		Unattended: unattended,
		Quiet:      quiet,
		Debug:      debug,
		Force:      force,
	}, nil
}

func defaultPruneDeps(cmd *cobra.Command, options pruneOptions) pruneDeps {
	common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
		Quiet: options.Quiet,
		Debug: options.Debug,
	})
	return pruneDeps{
		fs:        common.FS,
		logger:    common.Logger,
		output:    common.Output,
		telemetry: telemetry.RecordCommand,
	}
}

func recordPruneTelemetry(record func(telemetry.CommandTelemetry), options pruneOptions, deletedCount int, err error) {
	payload := telemetry.CommandTelemetry{
		Command:     "prune",
		Success:     err == nil,
		Error:       err,
		ExitCode:    0,
		Interactive: false,
		Arguments: map[string]interface{}{
			"force": options.Force,
		},
	}
	if err != nil {
		payload.ExitCode = 1
	} else {
		payload.Extra = map[string]interface{}{
			"deletedCount": deletedCount,
		}
	}
	record(payload)
}

func runPrune(ctx context.Context, cmd *cobra.Command, options pruneOptions, deps pruneDeps) (int, error) {
	meta := config.NewMetadata(options.ConfigPath)

	cfg, err := config.ReadConfig(ctx, deps.fs, meta)
	if err != nil {
		return 0, err
	}

	lock, err := readLockRequired(ctx, deps.fs, meta)
	if err != nil {
		return 0, handleLockReadError(err, deps.output)
	}

	unmanagedFiles, err := listUnmanagedFiles(deps.fs, meta, cfg, lock)
	if err != nil {
		return 0, err
	}

	if len(unmanagedFiles) == 0 {
		if outputErr := deps.output.Log(i18n.T("cmd.prune.no_unmanaged", nil), output.LogQuiet); outputErr != nil {
			return 0, outputErr
		}
		return 0, nil
	}

	colorMode := tui.ColorDisabled
	if tui.IsTerminalWriter(cmd.OutOrStdout()) {
		colorMode = tui.ColorEnabled
	}

	shouldDelete, err := shouldDeleteUnmanaged(cmd, options, deps, colorMode, unmanagedFiles)
	if err != nil {
		return 0, err
	}
	if !shouldDelete {
		return 0, nil
	}

	return deleteUnmanagedFiles(deps, unmanagedFiles)
}

func shouldDeleteUnmanaged(cmd *cobra.Command, options pruneOptions, deps pruneDeps, colorMode tui.ColorMode, unmanagedFiles []string) (bool, error) {
	if options.Force {
		return true, nil
	}

	if options.Unattended {
		if outputErr := deps.output.Error(i18n.T("cmd.prune.error.prompt_disabled", nil)); outputErr != nil {
			return false, outputErr
		}
		if outputErr := printUnmanagedList(deps.output, colorMode, unmanagedFiles); outputErr != nil {
			return false, outputErr
		}
		return false, nil
	}

	if !tui.SupportsPrompting(cmd.InOrStdin(), cmd.OutOrStdout()) {
		return false, reportPromptDisabled(deps.output)
	}
	if outputErr := printUnmanagedList(deps.output, colorMode, unmanagedFiles); outputErr != nil {
		return false, outputErr
	}
	confirmed, confirmErr := confirmDeletion(cmd.InOrStdin(), cmd.OutOrStdout(), colorMode)
	if confirmErr != nil {
		return false, confirmErr
	}
	return confirmed, nil
}

type lockMissingError struct {
	message string
}

func (err *lockMissingError) Error() string {
	return err.message
}

func readLockRequired(ctx context.Context, fs afero.Fs, meta config.Metadata) ([]models.ModInstall, error) {
	lock, err := config.ReadLock(ctx, fs, meta)
	if err == nil {
		return lock, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, &lockMissingError{
			message: i18n.T("cmd.prune.error.lock_missing", &i18n.Tvars{
				Data: &i18n.TData{
					"lock_path":       meta.LockPath(),
					"install_command": "mmm install",
				},
			}),
		}
	}
	return nil, err
}

func handleLockReadError(err error, out *output.Output) error {
	var lockMissing *lockMissingError
	if !errors.As(err, &lockMissing) {
		return err
	}
	if outputErr := out.Error(lockMissing.Error()); outputErr != nil {
		return outputErr
	}
	return clierrors.MarkHandled(err)
}

func listUnmanagedFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall) ([]string, error) {
	candidates, err := listJarFiles(fs, meta, cfg)
	if err != nil {
		return nil, err
	}

	unmanaged := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if fileIsManaged(candidate, lock) {
			continue
		}
		unmanaged = append(unmanaged, candidate)
	}

	return unmanaged, nil
}

func listJarFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON) ([]string, error) {
	modsFolder := meta.ModsFolderPath(cfg)
	modsFolderAbs, err := absPath(modsFolder)
	if err != nil {
		return nil, err
	}

	allEntries, err := afero.ReadDir(fs, modsFolderAbs)
	if err != nil {
		return nil, err
	}

	candidates := make([]string, 0, len(allEntries))
	for _, entry := range allEntries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(modsFolderAbs, entry.Name()))
	}

	patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if mmmignore.IsIgnored(modsFolderAbs, candidate, patterns) {
			continue
		}
		filtered = append(filtered, candidate)
	}

	return filtered, nil
}

func fileIsManaged(path string, lock []models.ModInstall) bool {
	fileName := filepath.Base(path)
	for _, install := range lock {
		if install.FileName == fileName {
			return true
		}
	}
	return false
}

func printUnmanagedList(out *output.Output, colorMode tui.ColorMode, unmanaged []string) error {
	icon := tui.ErrorIcon(colorMode)
	for _, filePath := range unmanaged {
		entry := i18n.T("cmd.prune.unmanaged.entry", &i18n.Tvars{
			Data: &i18n.TData{
				"file": filePath,
			},
		})
		if err := out.Log(fmt.Sprintf("%s %s", icon, entry), output.LogForce); err != nil {
			return err
		}
	}
	return nil
}

func confirmDeletion(in io.Reader, out io.Writer, colorMode tui.ColorMode) (bool, error) {
	questionPrefix := "?"
	if colorMode.Enabled() {
		questionPrefix = tui.QuestionStyle.Render(questionPrefix)
	}
	return tui.RunConfirmPrompt(in, out, tui.ConfirmPrompt{
		Prefix:   questionPrefix,
		Question: i18n.T("cmd.prune.confirm", nil),
	})
}

func reportPromptDisabled(out *output.Output) error {
	if err := out.Error(i18n.T("cmd.prune.error.prompt_disabled", nil)); err != nil {
		return err
	}
	if err := out.Error(i18n.T("cmd.prune.error.prompt_disabled_hint", &i18n.Tvars{
		Data: &i18n.TData{
			"force_command": "mmm prune --force",
		},
	})); err != nil {
		return err
	}
	return clierrors.MarkHandled(errPromptDisabled)
}

func deleteUnmanagedFiles(deps pruneDeps, unmanaged []string) (int, error) {
	deletedCount := 0
	for _, filePath := range unmanaged {
		if err := removeFileForce(deps.fs, filePath); err != nil {
			return deletedCount, err
		}
		deletedCount++
		if err := deps.output.Log(i18n.T("cmd.prune.deleted", &i18n.Tvars{
			Data: &i18n.TData{
				"file": filePath,
			},
		}), output.LogQuiet); err != nil {
			return deletedCount, err
		}
	}

	return deletedCount, nil
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

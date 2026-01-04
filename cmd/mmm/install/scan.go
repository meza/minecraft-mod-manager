package install

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/logger"
	"github.com/meza/minecraft-mod-manager/internal/mmmignore"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"github.com/meza/minecraft-mod-manager/internal/output"
	tui "github.com/meza/minecraft-mod-manager/internal/view"
	"github.com/spf13/afero"
)

func preflightInstall(ctx context.Context, meta config.Metadata, cfg models.ModsJSON, lock []models.ModInstall, deps installDeps, colorize bool) (scanReportOutcome, error) {
	preflight, err := preflightUnknownFiles(preflightInputs{
		ctx:      ctx,
		meta:     meta,
		cfg:      cfg,
		lock:     lock,
		deps:     deps,
		colorize: colorize,
	})
	if err != nil {
		return scanReportOutcome{}, err
	}
	if preflight.unresolved {
		if outputErr := deps.output.Error(i18n.T("cmd.install.error.unresolved", nil)); outputErr != nil {
			return scanReportOutcome{}, outputErr
		}
		return scanReportOutcome{}, clierrors.MarkHandled(errUnresolvedFiles)
	}
	return preflight, nil
}

func preflightUnknownFiles(input preflightInputs) (scanReportOutcome, error) {
	files, err := listModFiles(input.deps.fs, input.meta, input.cfg)
	if err != nil {
		return scanReportOutcome{}, err
	}

	nonManaged := make([]string, 0)
	for _, file := range files {
		if !fileIsManaged(file, input.lock) {
			nonManaged = append(nonManaged, file)
		}
	}

	if len(nonManaged) == 0 {
		return scanReportOutcome{}, nil
	}
	scanned, err := scanFiles(input.ctx, nonManaged, input.deps)
	if err != nil {
		handled, handleErr := handlePreflightScanFailure(input, err)
		if handled {
			return scanReportOutcome{unresolved: true}, nil
		}
		return scanReportOutcome{}, handleErr
	}
	return reportScanResults(scanReportInputs{
		scanned:  scanned,
		cfg:      input.cfg,
		lock:     input.lock,
		deps:     input.deps,
		colorize: input.colorize,
	})
}

func handlePreflightScanFailure(input preflightInputs, scanErr error) (bool, error) {
	var lookupFailure *platformLookupFailure
	if errors.As(scanErr, &lookupFailure) {
		colorMode := tui.ColorDisabled
		if input.colorize {
			colorMode = tui.ColorEnabled
		}
		if err := logPlatformLookupFailure(input.deps.output, input.deps.logger, lookupFailure, colorMode); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, scanErr
}

func scanFiles(ctx context.Context, files []string, deps installDeps) ([]scannedFile, error) {
	candidates, err := buildScanCandidates(files, deps)
	if err != nil {
		return nil, err
	}
	if err := applyCurseforgeHits(ctx, candidates.results, candidates.fingerprintToIndices, candidates.fingerprints, deps); err != nil {
		return nil, newPlatformLookupFailure(models.CURSEFORGE, files, err)
	}
	if err := applyModrinthHits(ctx, candidates.results, deps); err != nil {
		return nil, err
	}
	for i := range candidates.results {
		candidates.results[i].Hits = sortHitsPreferModrinth(candidates.results[i].Hits)
	}
	return candidates.results, nil
}

func (failure *platformLookupFailure) Error() string {
	return failure.Reason
}

func newPlatformLookupFailure(platformValue models.Platform, files []string, err error) *platformLookupFailure {
	if err == nil {
		return &platformLookupFailure{
			Platform: platformValue,
			Files:    files,
			Reason: i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
				Data: &i18n.TData{"platform": string(platformValue)},
			}),
		}
	}

	summary, _ := clierrors.SummarizePlatformError(err, platformValue)
	return &platformLookupFailure{
		Platform:     platformValue,
		Files:        files,
		Reason:       summary.Reason,
		DebugDetails: platformErrorDetails(err),
	}
}

func platformErrorDetails(err error) string {
	if err == nil {
		return ""
	}

	responseDetails := ""
	if responseErr, ok := httpclient.ExtractResponseError(err); ok {
		responseDetails = responseErr.Error()
	}

	var fingerprintAPIError *curseforge.FingerprintAPIError
	if errors.As(err, &fingerprintAPIError) {
		if responseDetails != "" {
			return fmt.Sprintf("fingerprints=%v; %s", fingerprintAPIError.Lookup, responseDetails)
		}
		return fmt.Sprintf("fingerprints=%v; %s", fingerprintAPIError.Lookup, fingerprintAPIError.Err)
	}

	if responseDetails != "" {
		return responseDetails
	}
	return err.Error()
}

func logPlatformLookupFailure(out *output.Output, log *logger.Logger, failure *platformLookupFailure, colorMode tui.ColorMode) error {
	if failure == nil || out == nil {
		return nil
	}
	if strings.TrimSpace(failure.DebugDetails) != "" {
		if log == nil {
			return errors.New("missing logger for platform debug output")
		}
		if err := log.Debug(i18n.T("cmd.install.debug.platform_error", &i18n.Tvars{
			Data: &i18n.TData{
				"platform": string(failure.Platform),
				"details":  failure.DebugDetails,
			},
		})); err != nil {
			return err
		}
	}
	for _, filePath := range failure.Files {
		if err := out.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.install.unsure.platform_error", &i18n.Tvars{
			Data: &i18n.TData{
				"file":     filepath.Base(filePath),
				"platform": string(failure.Platform),
				"reason":   failure.Reason,
			},
		})), output.LogForce); err != nil {
			return err
		}
	}
	if strings.TrimSpace(failure.DebugDetails) != "" {
		if err := out.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.install.unsure.platform_error_details", &i18n.Tvars{
			Data: &i18n.TData{
				"platform": string(failure.Platform),
				"details":  failure.DebugDetails,
			},
		})), output.LogForce); err != nil {
			return err
		}
	}
	return nil
}

func buildScanCandidates(files []string, deps installDeps) (scanCandidates, error) {
	results := make([]scannedFile, 0, len(files))
	fingerprints := make([]uint32, 0, len(files))
	fingerprintToIndices := make(map[uint32][]int, len(files))

	for index, filePath := range files {
		sha, err := sha1ForFile(deps.fs, filePath)
		if err != nil {
			return scanCandidates{}, err
		}
		fingerprint := deps.curseforgeFingerprint(filePath)
		fingerprints = append(fingerprints, fingerprint)
		fingerprintToIndices[fingerprint] = append(fingerprintToIndices[fingerprint], index)
		results = append(results, scannedFile{
			Path: filePath,
			Sha1: sha,
		})
	}
	sort.Slice(fingerprints, func(leftIndex int, rightIndex int) bool {
		return fingerprints[leftIndex] < fingerprints[rightIndex]
	})
	return scanCandidates{
		results:              results,
		fingerprints:         fingerprints,
		fingerprintToIndices: fingerprintToIndices,
	}, nil
}

func applyCurseforgeHits(ctx context.Context, results []scannedFile, fingerprintToIndices map[uint32][]int, fingerprints []uint32, deps installDeps) error {
	curseforgeByFingerprint, err := curseforgeMatchesByFingerprint(ctx, fingerprints, deps)
	if err != nil {
		return err
	}
	for fingerprint, hit := range curseforgeByFingerprint {
		indices, ok := fingerprintToIndices[fingerprint]
		if !ok || len(indices) == 0 {
			continue
		}
		for _, index := range indices {
			results[index].Hits = append(results[index].Hits, hit)
		}
	}
	return nil
}

func applyModrinthHits(ctx context.Context, results []scannedFile, deps installDeps) error {
	for i := range results {
		version, err := deps.modrinthVersionForSha(ctx, results[i].Sha1, deps.clients.Modrinth)
		if err != nil {
			var notFound *modrinth.VersionNotFoundError
			if errors.As(err, &notFound) {
				continue
			}
			return newPlatformLookupFailure(models.MODRINTH, []string{results[i].Path}, err)
		}
		name, err := deps.modrinthProjectTitle(ctx, version.ProjectID, deps.clients.Modrinth)
		if err != nil {
			return newPlatformLookupFailure(models.MODRINTH, []string{results[i].Path}, err)
		}
		results[i].Hits = append(results[i].Hits, scanHit{
			Platform: models.MODRINTH,
			Project:  version.ProjectID,
			Name:     name,
		})
	}
	return nil
}

func sortHitsPreferModrinth(hits []scanHit) []scanHit {
	if len(hits) <= 1 {
		return hits
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Platform == models.MODRINTH && hits[j].Platform != models.MODRINTH {
			return true
		}
		if hits[i].Platform != models.MODRINTH && hits[j].Platform == models.MODRINTH {
			return false
		}
		return hits[i].Platform < hits[j].Platform
	})

	return hits
}

func curseforgeMatchesByFingerprint(ctx context.Context, fingerprints []uint32, deps installDeps) (map[uint32]scanHit, error) {
	unique := uniqueUint32s(fingerprints)
	if len(unique) == 0 {
		return map[uint32]scanHit{}, nil
	}
	result, err := deps.curseforgeFingerprintMatch(ctx, unique, deps.clients.Curseforge)
	if err != nil {
		return nil, err
	}
	matches := make(map[uint32]scanHit, len(result.Matches))
	for _, file := range result.Matches {
		projectID := fmt.Sprintf("%d", file.ProjectID)
		name, err := deps.curseforgeProjectName(ctx, projectID, deps.clients.Curseforge)
		if err != nil {
			return nil, err
		}
		matches[file.Fingerprint] = scanHit{
			Platform: models.CURSEFORGE,
			Project:  projectID,
			Name:     name,
		}
	}
	return matches, nil
}

func reportScanResults(input scanReportInputs) (scanReportOutcome, error) {
	outcome := scanReportOutcome{}
	colorMode := tui.ColorDisabled
	if input.colorize {
		colorMode = tui.ColorEnabled
	}

	for _, item := range input.scanned {
		itemOutcome, err := reportScanResult(input, item, colorMode)
		if err != nil {
			return scanReportOutcome{}, err
		}
		outcome.unmanagedFound = outcome.unmanagedFound || itemOutcome.unmanagedFound
		outcome.unresolved = outcome.unresolved || itemOutcome.unresolved
	}

	return outcome, nil
}

func reportScanResult(input scanReportInputs, item scannedFile, colorMode tui.ColorMode) (scanReportOutcome, error) {
	if len(item.Hits) == 0 {
		return scanReportOutcome{}, nil
	}

	matchedModIndex := findConfiguredModIndex(input.cfg, item.Hits)
	if matchedModIndex < 0 {
		name := item.Hits[0].Name
		if input.colorize {
			name = tui.TitleStyle.Bold(true).Render(name)
		}
		if err := input.deps.output.Log(tui.SuccessIcon(colorMode)+i18n.T("cmd.install.unmanaged.found", &i18n.Tvars{
			Data: &i18n.TData{"name": name},
		}), output.LogForce); err != nil {
			return scanReportOutcome{}, err
		}
		return scanReportOutcome{unmanagedFound: true}, nil
	}

	mod := input.cfg.Mods[matchedModIndex]
	lockIndex := models.LockIndexForMod(mod, input.lock)
	if lockIndex < 0 {
		if err := input.deps.output.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.install.unsure.lock_missing", &i18n.Tvars{
			Data: &i18n.TData{"name": item.Hits[0].Name},
		})), output.LogForce); err != nil {
			return scanReportOutcome{}, err
		}
		return scanReportOutcome{unresolved: true}, nil
	}

	if !strings.EqualFold(input.lock[lockIndex].Hash, item.Sha1) {
		if err := input.deps.output.Log(messageWithIcon(tui.ErrorIcon(colorMode), i18n.T("cmd.install.unsure.hash_mismatch", &i18n.Tvars{
			Data: &i18n.TData{"name": item.Hits[0].Name},
		})), output.LogForce); err != nil {
			return scanReportOutcome{}, err
		}
		return scanReportOutcome{unresolved: true}, nil
	}

	return scanReportOutcome{}, nil
}

func listModFiles(fs afero.Fs, meta config.Metadata, cfg models.ModsJSON) ([]string, error) {
	all, err := afero.ReadDir(fs, meta.ModsFolderPath(cfg))
	if err != nil {
		return nil, err
	}

	candidates := make([]string, 0, len(all))
	for _, entry := range all {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}
		candidates = append(candidates, filepath.Join(meta.ModsFolderPath(cfg), entry.Name()))
	}

	patterns, err := mmmignore.ListPatterns(fs, meta.Dir())
	if err != nil {
		return nil, err
	}

	filtered := make([]string, 0, len(candidates))
	for _, path := range candidates {
		if mmmignore.IsIgnored(meta.ModsFolderPath(cfg), path, patterns) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered, nil
}

func sha1ForFile(fs afero.Fs, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return "", errors.Join(err, closeErr)
		}
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

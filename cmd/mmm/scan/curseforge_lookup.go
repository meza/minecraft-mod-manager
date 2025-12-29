package scan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func lookupCurseforge(ctx context.Context, candidates []scanCandidate, deps scanDeps) (platformLookupOutcome, error) {
	outcome := platformLookupOutcome{
		matches: make([]scanMatch, 0),
		unsure:  make(map[string]error),
	}

	fingerprintIndex := buildCurseforgeFingerprintIndex(candidates, deps)

	unique := uniqueUint32s(fingerprintIndex.fingerprints)
	if len(unique) == 0 {
		outcome.misses = candidates
		return outcome, nil
	}

	result, err := deps.curseforgeFingerprintMatch(ctx, unique, deps.clients.Curseforge)
	if err != nil {
		summary := summarizePlatformFailure(err, models.CURSEFORGE)
		if logErr := logPlatformDebug(deps.logger, models.CURSEFORGE, summary.DebugDetails); logErr != nil {
			return platformLookupOutcome{}, logErr
		}
		allowFallback := allowPlatformFallback(err)
		misses := []scanCandidate(nil)
		if allowFallback {
			misses = candidates
		}
		return platformLookupOutcome{
			matches: nil,
			misses:  misses,
			unsure:  buildCurseforgeErrors(candidates, platformUnsureReason(models.CURSEFORGE, summary.Reason)),
		}, nil
	}

	fallbackEligible, err := addCurseforgeMatchesWithCache(ctx, candidates, fingerprintIndex, result.Matches, deps, &outcome.matches, outcome.unsure)
	if err != nil {
		return platformLookupOutcome{}, err
	}
	outcome.misses = curseforgeMisses(candidates, outcome.matches, outcome.unsure, fallbackEligible)
	return outcome, nil
}

type curseforgeFingerprintIndex struct {
	fingerprints         []uint32
	fingerprintToIndices map[uint32][]int
}

func buildCurseforgeFingerprintIndex(candidates []scanCandidate, deps scanDeps) curseforgeFingerprintIndex {
	fingerprints := make([]uint32, 0, len(candidates))
	fingerprintToIndices := make(map[uint32][]int, len(candidates))

	for i, candidate := range candidates {
		fingerprint := deps.curseforgeFingerprint(candidate.Path)
		fingerprints = append(fingerprints, fingerprint)
		fingerprintToIndices[fingerprint] = append(fingerprintToIndices[fingerprint], i)
	}

	return curseforgeFingerprintIndex{
		fingerprints:         fingerprints,
		fingerprintToIndices: fingerprintToIndices,
	}
}

func buildCurseforgeErrors(candidates []scanCandidate, reason string) map[string]error {
	unsure := make(map[string]error, len(candidates))
	for _, candidate := range candidates {
		unsure[candidate.Path] = errors.New(reason)
	}
	return unsure
}

func addCurseforgeMatchesWithCache(
	ctx context.Context,
	candidates []scanCandidate,
	fingerprintIndex curseforgeFingerprintIndex,
	matches []curseforge.File,
	deps scanDeps,
	scanMatches *[]scanMatch,
	unsure map[string]error,
) (map[string]bool, error) {
	nameCache := make(map[string]string)
	var nameMu sync.Mutex
	fallbackEligible := make(map[string]bool)

	if err := addCurseforgeMatches(ctx, curseforgeMatchContext{
		candidates:           candidates,
		fingerprintToIndices: fingerprintIndex.fingerprintToIndices,
		matches:              matches,
		deps:                 deps,
		nameCache:            nameCache,
		nameMu:               &nameMu,
		scanMatches:          scanMatches,
		unsure:               unsure,
		fallbackEligible:     fallbackEligible,
	}); err != nil {
		return nil, err
	}
	return fallbackEligible, nil
}

type curseforgeMatchContext struct {
	candidates           []scanCandidate
	fingerprintToIndices map[uint32][]int
	matches              []curseforge.File
	deps                 scanDeps
	nameCache            map[string]string
	nameMu               *sync.Mutex
	scanMatches          *[]scanMatch
	unsure               map[string]error
	fallbackEligible     map[string]bool
}

func addCurseforgeMatches(ctx context.Context, matchContext curseforgeMatchContext) error {
	for _, file := range matchContext.matches {
		if err := applyCurseforgeMatch(ctx, matchContext, file); err != nil {
			return err
		}
	}
	return nil
}

func applyCurseforgeMatch(ctx context.Context, matchContext curseforgeMatchContext, file curseforge.File) error {
	indices := matchContext.fingerprintToIndices[file.Fingerprint]
	if len(indices) == 0 {
		return nil
	}

	projectID := fmt.Sprintf("%d", file.ProjectID)
	name, err := cachedCurseforgeProjectName(ctx, projectID, matchContext.deps, matchContext.nameCache, matchContext.nameMu)
	if err != nil {
		return recordCurseforgeUnsure(matchContext, indices, err)
	}

	if strings.TrimSpace(file.DownloadURL) == "" {
		return recordCurseforgeUnsure(matchContext, indices, errors.New("curseforge match missing download url"))
	}

	published := file.FileDate.Format(time.RFC3339)
	for _, index := range indices {
		*matchContext.scanMatches = append(*matchContext.scanMatches, scanMatch{
			Path:        matchContext.candidates[index].Path,
			Platform:    models.CURSEFORGE,
			ProjectID:   projectID,
			Name:        name,
			FileName:    matchContext.candidates[index].FileName,
			Hash:        matchContext.candidates[index].Sha1,
			ReleaseDate: published,
			DownloadURL: file.DownloadURL,
		})
	}
	return nil
}

func recordCurseforgeUnsure(matchContext curseforgeMatchContext, indices []int, err error) error {
	summary := summarizePlatformFailure(err, models.CURSEFORGE)
	if logErr := logPlatformDebug(matchContext.deps.logger, models.CURSEFORGE, summary.DebugDetails); logErr != nil {
		return logErr
	}
	reason := platformUnsureReason(models.CURSEFORGE, summary.Reason)
	allowFallback := allowPlatformFallback(err)
	for _, index := range indices {
		path := matchContext.candidates[index].Path
		matchContext.unsure[path] = errors.New(reason)
		if allowFallback {
			matchContext.fallbackEligible[path] = true
		}
	}
	return nil
}

func cachedCurseforgeProjectName(ctx context.Context, projectID string, deps scanDeps, nameCache map[string]string, nameMu *sync.Mutex) (string, error) {
	nameMu.Lock()
	name, ok := nameCache[projectID]
	nameMu.Unlock()
	if ok {
		return name, nil
	}

	name, err := deps.curseforgeProjectName(ctx, projectID, deps.clients.Curseforge)
	if err != nil {
		return "", err
	}

	nameMu.Lock()
	nameCache[projectID] = name
	nameMu.Unlock()
	return name, nil
}

func curseforgeMisses(candidates []scanCandidate, matches []scanMatch, unsure map[string]error, fallbackEligible map[string]bool) []scanCandidate {
	matched := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		matched[match.Path] = struct{}{}
	}

	misses := make([]scanCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := matched[candidate.Path]; ok {
			continue
		}
		if _, ok := unsure[candidate.Path]; ok {
			if fallbackEligible != nil && fallbackEligible[candidate.Path] {
				misses = append(misses, candidate)
			}
			continue
		}
		misses = append(misses, candidate)
	}
	return misses
}

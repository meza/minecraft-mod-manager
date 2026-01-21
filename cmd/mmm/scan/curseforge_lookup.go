package scan

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/curseforge"
	"github.com/meza/minecraft-mod-manager/internal/httpclient"
	"github.com/meza/minecraft-mod-manager/internal/models"
)

func lookupCurseforge(ctx context.Context, candidates []scanCandidate, deps scanDeps) (platformLookupOutcome, error) {
	return lookupCurseforgeWithObserver(ctx, candidates, deps, nil)
}

func lookupCurseforgeWithObserver(
	ctx context.Context,
	candidates []scanCandidate,
	deps scanDeps,
	state *scanExecState,
) (platformLookupOutcome, error) {
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

	requestCtx := ctx
	if state != nil {
		requestCtx = httpclient.WithRequestStartHook(ctx, func(_ *http.Request) {
			markScanCandidatesActive(state, candidates)
		})
	}
	result, err := deps.curseforgeFingerprintMatch(requestCtx, unique, deps.clients.Curseforge)
	if err != nil {
		return handleCurseforgeFingerprintError(err, candidates, deps, state)
	}

	fallbackEligible, err := addCurseforgeMatchesWithCache(candidates, fingerprintIndex, result.Matches, deps, &outcome.matches, outcome.unsure, state)
	if err != nil {
		return platformLookupOutcome{}, err
	}
	outcome.misses = curseforgeMisses(candidates, outcome.matches, outcome.unsure, fallbackEligible)
	if state != nil {
		markScanCandidatesPending(state, outcome.misses)
	}
	return outcome, nil
}

func handleCurseforgeFingerprintError(
	err error,
	candidates []scanCandidate,
	deps scanDeps,
	state *scanExecState,
) (platformLookupOutcome, error) {
	summary := summarizePlatformFailure(err, models.CURSEFORGE)
	if logErr := logPlatformDebug(deps.logger, models.CURSEFORGE, summary.DebugDetails); logErr != nil {
		return platformLookupOutcome{}, logErr
	}
	allowFallback := allowPlatformFallback(err)
	misses := []scanCandidate(nil)
	if allowFallback {
		if state != nil {
			markScanCandidatesPending(state, candidates)
		}
		misses = candidates
	} else if state != nil {
		markScanCandidatesUnsure(state, candidates)
	}
	return platformLookupOutcome{
		matches: nil,
		misses:  misses,
		unsure:  buildCurseforgeErrors(candidates, platformUnsureReason(models.CURSEFORGE, summary.Reason)),
	}, nil
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
	candidates []scanCandidate,
	fingerprintIndex curseforgeFingerprintIndex,
	matches []curseforge.File,
	deps scanDeps,
	scanMatches *[]scanMatch,
	unsure map[string]error,
	state *scanExecState,
) (map[string]bool, error) {
	fallbackEligible := make(map[string]bool)

	if err := addCurseforgeMatches(curseforgeMatchContext{
		candidates:           candidates,
		fingerprintToIndices: fingerprintIndex.fingerprintToIndices,
		matches:              matches,
		deps:                 deps,
		scanMatches:          scanMatches,
		unsure:               unsure,
		fallbackEligible:     fallbackEligible,
		state:                state,
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
	scanMatches          *[]scanMatch
	unsure               map[string]error
	fallbackEligible     map[string]bool
	state                *scanExecState
}

func addCurseforgeMatches(matchContext curseforgeMatchContext) error {
	for _, file := range matchContext.matches {
		if err := applyCurseforgeMatch(matchContext, file); err != nil {
			return err
		}
	}
	return nil
}

func applyCurseforgeMatch(matchContext curseforgeMatchContext, file curseforge.File) error {
	indices := matchContext.fingerprintToIndices[file.Fingerprint]
	if len(indices) == 0 {
		return nil
	}

	projectID := fmt.Sprintf("%d", file.ProjectID)
	name := strings.TrimSpace(file.DisplayName)
	if name == "" {
		name = strings.TrimSpace(file.FileName)
	}
	if name == "" {
		name = projectID
	}

	if strings.TrimSpace(file.DownloadURL) == "" {
		return recordCurseforgeUnsure(matchContext, indices, errors.New("curseforge match missing download url"))
	}

	published := file.FileDate.Format(time.RFC3339)
	for _, index := range indices {
		match := scanMatch{
			Path:        matchContext.candidates[index].Path,
			Platform:    models.CURSEFORGE,
			ProjectID:   projectID,
			Name:        name,
			FileName:    matchContext.candidates[index].FileName,
			Hash:        matchContext.candidates[index].Sha1,
			ReleaseDate: published,
			DownloadURL: file.DownloadURL,
		}
		*matchContext.scanMatches = append(*matchContext.scanMatches, match)
		if matchContext.state != nil {
			matchContext.state.updateStatus(match.FileName, scanItemStatusRecognized, match)
		}
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
	status := scanItemStatusUnsure
	if allowFallback {
		status = scanItemStatusPending
	}
	for _, index := range indices {
		path := matchContext.candidates[index].Path
		matchContext.unsure[path] = errors.New(reason)
		if allowFallback {
			matchContext.fallbackEligible[path] = true
		}
		if matchContext.state != nil {
			matchContext.state.updateStatus(matchContext.candidates[index].FileName, status, scanMatch{})
		}
	}
	return nil
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

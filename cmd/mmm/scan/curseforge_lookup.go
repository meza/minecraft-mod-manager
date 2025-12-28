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

func lookupCurseforge(ctx context.Context, candidates []scanCandidate, deps scanDeps) ([]scanMatch, []scanCandidate, map[string]error) {
	matches := make([]scanMatch, 0)
	unsure := make(map[string]error)

	fingerprintIndex := buildCurseforgeFingerprintIndex(candidates, deps)

	unique := uniqueUint32s(fingerprintIndex.fingerprints)
	if len(unique) == 0 {
		return nil, candidates, unsure
	}

	result, err := deps.curseforgeFingerprintMatch(ctx, unique, deps.clients.Curseforge)
	if err != nil {
		summary := summarizePlatformFailure(err, models.CURSEFORGE)
		logPlatformDebug(deps.logger, models.CURSEFORGE, summary.DebugDetails)
		return nil, nil, buildCurseforgeErrors(candidates, platformUnsureReason(models.CURSEFORGE, summary.Reason))
	}

	addCurseforgeMatchesWithCache(ctx, candidates, fingerprintIndex, result.Matches, deps, &matches, unsure)
	misses := curseforgeMisses(candidates, matches, unsure)
	return matches, misses, unsure
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
) {
	nameCache := make(map[string]string)
	var nameMu sync.Mutex

	addCurseforgeMatches(ctx, curseforgeMatchContext{
		candidates:           candidates,
		fingerprintToIndices: fingerprintIndex.fingerprintToIndices,
		matches:              matches,
		deps:                 deps,
		nameCache:            nameCache,
		nameMu:               &nameMu,
		scanMatches:          scanMatches,
		unsure:               unsure,
	})
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
}

func addCurseforgeMatches(ctx context.Context, matchContext curseforgeMatchContext) {
	for _, file := range matchContext.matches {
		indices := matchContext.fingerprintToIndices[file.Fingerprint]
		if len(indices) == 0 {
			continue
		}

		projectID := fmt.Sprintf("%d", file.ProjectID)
		name, err := cachedCurseforgeProjectName(ctx, projectID, matchContext.deps, matchContext.nameCache, matchContext.nameMu)
		if err != nil {
			summary := summarizePlatformFailure(err, models.CURSEFORGE)
			logPlatformDebug(matchContext.deps.logger, models.CURSEFORGE, summary.DebugDetails)
			reason := platformUnsureReason(models.CURSEFORGE, summary.Reason)
			for _, index := range indices {
				matchContext.unsure[matchContext.candidates[index].Path] = errors.New(reason)
			}
			continue
		}

		if strings.TrimSpace(file.DownloadURL) == "" {
			summary := summarizePlatformFailure(errors.New("curseforge match missing download url"), models.CURSEFORGE)
			logPlatformDebug(matchContext.deps.logger, models.CURSEFORGE, summary.DebugDetails)
			reason := platformUnsureReason(models.CURSEFORGE, summary.Reason)
			for _, index := range indices {
				matchContext.unsure[matchContext.candidates[index].Path] = errors.New(reason)
			}
			continue
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
	}
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

func curseforgeMisses(candidates []scanCandidate, matches []scanMatch, unsure map[string]error) []scanCandidate {
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
			continue
		}
		misses = append(misses, candidate)
	}
	return misses
}

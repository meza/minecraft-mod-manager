package scan

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"golang.org/x/sync/errgroup"
)

func lookupModrinth(ctx context.Context, candidates []scanCandidate, deps scanDeps) (platformLookupOutcome, error) {
	results, err := runModrinthLookups(ctx, candidates, deps)
	if err != nil {
		if isContextCancellation(err) {
			return contextCanceledOutcome(candidates, err), nil
		}
		return platformLookupOutcome{
			matches: nil,
			misses:  candidates,
			unsure:  nil,
		}, err
	}

	return splitModrinthResults(candidates, results), nil
}

func contextCanceledOutcome(candidates []scanCandidate, err error) platformLookupOutcome {
	unsure := make(map[string]error, len(candidates))
	for _, candidate := range candidates {
		unsure[candidate.Path] = err
	}
	return platformLookupOutcome{
		matches: nil,
		misses:  candidates,
		unsure:  unsure,
	}
}

func isContextCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func runModrinthLookups(ctx context.Context, candidates []scanCandidate, deps scanDeps) ([]modrinthLookupResult, error) {
	results := make([]modrinthLookupResult, len(candidates))

	group, groupCtx := errgroup.WithContext(ctx)

	for candidateIndex := range candidates {
		candidateIndex := candidateIndex
		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}
			result, lookupErr := lookupModrinthCandidate(groupCtx, candidates[candidateIndex], deps)
			if lookupErr != nil {
				return lookupErr
			}
			results[candidateIndex] = result
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

type modrinthLookupResult struct {
	match         *scanMatch
	err           error
	miss          bool
	allowFallback bool
}

func lookupModrinthCandidate(ctx context.Context, candidate scanCandidate, deps scanDeps) (modrinthLookupResult, error) {
	version, err := deps.modrinthVersionForSha(ctx, candidate.Sha1, deps.clients.Modrinth)
	if err != nil {
		var notFound *modrinth.VersionNotFoundError
		if errors.As(err, &notFound) {
			return modrinthLookupResult{miss: true}, nil
		}
		return modrinthLookupFailure(err, deps)
	}

	projectID := version.ProjectID
	name := strings.TrimSpace(version.Name)
	if name == "" {
		name = strings.TrimSpace(version.VersionNumber)
	}
	if name == "" {
		name = projectID
	}

	info, err := modrinthDownloadDetails(version)
	if err != nil {
		return modrinthLookupFailure(err, deps)
	}

	return modrinthLookupResult{match: &scanMatch{
		Path:        candidate.Path,
		Platform:    models.MODRINTH,
		ProjectID:   projectID,
		Name:        name,
		FileName:    candidate.FileName,
		Hash:        candidate.Sha1,
		ReleaseDate: info.publishedAt,
		DownloadURL: info.downloadURL,
	}}, nil
}

func modrinthLookupFailure(err error, deps scanDeps) (modrinthLookupResult, error) {
	summary := summarizePlatformFailure(err, models.MODRINTH)
	if logErr := logPlatformDebug(deps.logger, models.MODRINTH, summary.DebugDetails); logErr != nil {
		return modrinthLookupResult{}, logErr
	}
	return modrinthLookupResult{
		err:           errors.New(platformUnsureReason(models.MODRINTH, summary.Reason)),
		allowFallback: allowPlatformFallback(err),
	}, nil
}

func splitModrinthResults(candidates []scanCandidate, results []modrinthLookupResult) platformLookupOutcome {
	matches := make([]scanMatch, 0, len(candidates))
	misses := make([]scanCandidate, 0, len(candidates))
	unsure := make(map[string]error)

	for i, res := range results {
		candidate := candidates[i]
		if res.match != nil {
			matches = append(matches, *res.match)
			continue
		}
		if res.miss {
			misses = append(misses, candidate)
			continue
		}
		if res.err != nil {
			unsure[candidate.Path] = res.err
			if res.allowFallback {
				misses = append(misses, candidate)
			}
		}
	}

	return platformLookupOutcome{
		matches: matches,
		misses:  misses,
		unsure:  unsure,
	}
}

type modrinthDownloadInfo struct {
	downloadURL string
	publishedAt string
}

func modrinthDownloadDetails(version *modrinth.Version) (modrinthDownloadInfo, error) {
	if version == nil {
		return modrinthDownloadInfo{}, errors.New("modrinth version is nil")
	}
	if len(version.Files) == 0 {
		return modrinthDownloadInfo{}, errors.New("modrinth version has no files")
	}

	chosen := version.Files[0]
	for _, file := range version.Files {
		if file.Primary {
			chosen = file
			break
		}
	}

	if strings.TrimSpace(chosen.URL) == "" {
		return modrinthDownloadInfo{}, errors.New("modrinth file missing url")
	}

	return modrinthDownloadInfo{downloadURL: chosen.URL, publishedAt: version.DatePublished.Format(time.RFC3339)}, nil
}

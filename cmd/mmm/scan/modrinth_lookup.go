package scan

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modrinth"
	"golang.org/x/sync/errgroup"
)

func lookupModrinth(ctx context.Context, candidates []scanCandidate, deps scanDeps) ([]scanMatch, []scanCandidate, map[string]error) {
	results, err := runModrinthLookups(ctx, candidates, deps)
	if err != nil {
		return nil, candidates, matchErrorsForCandidates(candidates, err)
	}

	return splitModrinthResults(candidates, results)
}

func runModrinthLookups(ctx context.Context, candidates []scanCandidate, deps scanDeps) ([]modrinthLookupResult, error) {
	results := make([]modrinthLookupResult, len(candidates))
	titleCache := newModrinthTitleCache()

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(4)

	for i := range candidates {
		i := i
		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}
			results[i] = lookupModrinthCandidate(groupCtx, candidates[i], deps, titleCache)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

type modrinthLookupResult struct {
	match *scanMatch
	err   error
	miss  bool
}

func lookupModrinthCandidate(ctx context.Context, candidate scanCandidate, deps scanDeps, titleCache *modrinthTitleCache) modrinthLookupResult {
	version, err := deps.modrinthVersionForSha(ctx, candidate.Sha1, deps.clients.Modrinth)
	if err != nil {
		var notFound *modrinth.VersionNotFoundError
		if errors.As(err, &notFound) {
			return modrinthLookupResult{miss: true}
		}
		summary := summarizePlatformFailure(err, models.MODRINTH)
		logPlatformDebug(deps.logger, models.MODRINTH, summary.DebugDetails)
		return modrinthLookupResult{err: errors.New(platformUnsureReason(models.MODRINTH, summary.Reason))}
	}

	projectID := version.ProjectID
	name, err := cachedModrinthTitle(ctx, projectID, deps, titleCache)
	if err != nil {
		summary := summarizePlatformFailure(err, models.MODRINTH)
		logPlatformDebug(deps.logger, models.MODRINTH, summary.DebugDetails)
		return modrinthLookupResult{err: errors.New(platformUnsureReason(models.MODRINTH, summary.Reason))}
	}

	info, err := modrinthDownloadDetails(version)
	if err != nil {
		summary := summarizePlatformFailure(err, models.MODRINTH)
		logPlatformDebug(deps.logger, models.MODRINTH, summary.DebugDetails)
		return modrinthLookupResult{err: errors.New(platformUnsureReason(models.MODRINTH, summary.Reason))}
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
	}}
}

type modrinthTitleFetch struct {
	title string
	err   error
	ready chan struct{}
}

type modrinthTitleCache struct {
	mu       sync.Mutex
	titles   map[string]string
	inflight map[string]*modrinthTitleFetch
}

func newModrinthTitleCache() *modrinthTitleCache {
	return &modrinthTitleCache{
		titles:   make(map[string]string),
		inflight: make(map[string]*modrinthTitleFetch),
	}
}

func (cache *modrinthTitleCache) get(ctx context.Context, projectID string, deps scanDeps) (string, error) {
	cache.mu.Lock()
	if title, ok := cache.titles[projectID]; ok {
		cache.mu.Unlock()
		return title, nil
	}
	if pending, ok := cache.inflight[projectID]; ok {
		cache.mu.Unlock()
		select {
		case <-pending.ready:
			cache.mu.Lock()
			title := pending.title
			err := pending.err
			cache.mu.Unlock()
			return title, err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	pending := &modrinthTitleFetch{ready: make(chan struct{})}
	cache.inflight[projectID] = pending
	cache.mu.Unlock()

	title, err := deps.modrinthProjectTitle(ctx, projectID, deps.clients.Modrinth)

	cache.mu.Lock()
	if err == nil {
		cache.titles[projectID] = title
	}
	pending.title = title
	pending.err = err
	delete(cache.inflight, projectID)
	close(pending.ready)
	cache.mu.Unlock()
	return title, err
}

func cachedModrinthTitle(ctx context.Context, projectID string, deps scanDeps, titleCache *modrinthTitleCache) (string, error) {
	return titleCache.get(ctx, projectID, deps)
}

func matchErrorsForCandidates(candidates []scanCandidate, err error) map[string]error {
	unsure := make(map[string]error, len(candidates))
	for _, candidate := range candidates {
		unsure[candidate.Path] = err
	}
	return unsure
}

func splitModrinthResults(candidates []scanCandidate, results []modrinthLookupResult) ([]scanMatch, []scanCandidate, map[string]error) {
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
		}
	}

	return matches, misses, unsure
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

package scan

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"

	"github.com/meza/minecraft-mod-manager/internal/config"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/modsetup"
)

type scanExecutionInput struct {
	meta             config.Metadata
	cfg              models.ModsJSON
	lock             []models.ModInstall
	setupCoordinator *modsetup.SetupCoordinator
	candidates       []scanCandidate
	preferPlatform   models.Platform
	deps             scanDeps
}

type scanExecutionOutcome struct {
	items   []scanItem
	matches []scanMatch
	unknown []string
	unsure  []scanUnsure
	added   []scanMatch
	err     error
}

type scanExecSender struct {
	send func(tea.Msg)
}

func (sender scanExecSender) Send(msg tea.Msg) {
	if sender.send == nil {
		return
	}
	sender.send(msg)
}

type scanExecState struct {
	items      []scanItem
	indexByKey map[string]int
	sender     scanExecSender
}

func newScanExecState(items []scanItem, indexByKey map[string]int, sender scanExecSender) *scanExecState {
	return &scanExecState{items: items, indexByKey: indexByKey, sender: sender}
}

func (state *scanExecState) updateStatus(key string, status scanItemStatus, match scanMatch) {
	index, ok := state.indexByKey[key]
	if !ok || index < 0 || index >= len(state.items) {
		return
	}
	item := state.items[index]
	if isTerminalScanStatus(item.Status) {
		return
	}
	item.Status = status
	if status == scanItemStatusRecognized {
		item.Match = match
	}
	state.items[index] = item
	state.sender.Send(scanItemUpdateMsg{key: key, status: status, match: match})
}

func buildScanItems(candidates []scanCandidate) ([]scanItem, map[string]int) {
	items := make([]scanItem, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, scanItem{FileName: candidate.FileName, Status: scanItemStatusPending})
	}
	sortScanItemsByFile(items)
	return items, scanIndexByFile(items)
}

func runScanExecution(ctx context.Context, input scanExecutionInput, sender scanExecSender) scanExecutionOutcome {
	candidates := cloneScanCandidates(input.candidates)
	sortScanCandidatesByFile(candidates)

	items, indexByKey := buildScanItems(candidates)
	state := newScanExecState(items, indexByKey, sender)

	preferredOutcome, err := lookupPlatformWithUpdates(ctx, input.preferPlatform, candidates, input.deps, state)
	if err != nil {
		return scanExecutionOutcome{items: state.items, err: err}
	}

	fallbackPlatform := alternatePlatform(input.preferPlatform)
	fallbackOutcome, err := lookupPlatformWithUpdates(ctx, fallbackPlatform, preferredOutcome.misses, input.deps, state)
	if err != nil {
		return scanExecutionOutcome{items: state.items, err: err}
	}

	matches := append(preferredOutcome.matches, fallbackOutcome.matches...)
	unknown, unsure := finalizeScanResults(state, preferredOutcome.unsure, fallbackOutcome.unsure, fallbackOutcome.misses, matches)

	return scanExecutionOutcome{
		items:   state.items,
		matches: matches,
		unknown: unknown,
		unsure:  unsure,
	}
}

func cloneScanCandidates(candidates []scanCandidate) []scanCandidate {
	cloned := make([]scanCandidate, len(candidates))
	copy(cloned, candidates)
	return cloned
}

func lookupPlatformWithUpdates(
	ctx context.Context,
	platformValue models.Platform,
	candidates []scanCandidate,
	deps scanDeps,
	state *scanExecState,
) (platformLookupOutcome, error) {
	switch platformValue {
	case models.MODRINTH:
		return lookupModrinthWithUpdates(ctx, candidates, deps, state)
	case models.CURSEFORGE:
		return lookupCurseforgeWithUpdates(ctx, candidates, deps, state)
	default:
		return platformLookupOutcome{matches: nil, misses: candidates, unsure: map[string]error{}}, nil
	}
}

func lookupModrinthWithUpdates(
	ctx context.Context,
	candidates []scanCandidate,
	deps scanDeps,
	state *scanExecState,
) (platformLookupOutcome, error) {
	matches := make([]scanMatch, 0, len(candidates))
	misses := make([]scanCandidate, 0, len(candidates))
	unsure := make(map[string]error)

	var mutex sync.Mutex
	titleCache := newModrinthTitleCache()

	markScanCandidatesActive(state, candidates)
	group, groupCtx := errgroup.WithContext(ctx)
	for index := range candidates {
		index := index
		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}
			mutex.Lock()
			state.updateStatus(candidates[index].FileName, scanItemStatusScanning, scanMatch{})
			mutex.Unlock()
			result, lookupErr := lookupModrinthCandidate(groupCtx, candidates[index], deps, titleCache)
			if lookupErr != nil {
				return lookupErr
			}

			mutex.Lock()
			applyModrinthLookupResult(state, candidates[index], result, &matches, &misses, unsure)
			mutex.Unlock()

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return platformLookupOutcome{}, err
	}

	return finalizeModrinthLookupOutcome(matches, misses, unsure), nil
}

func applyModrinthLookupResult(
	state *scanExecState,
	candidate scanCandidate,
	result modrinthLookupResult,
	matches *[]scanMatch,
	misses *[]scanCandidate,
	unsure map[string]error,
) {
	switch {
	case result.match != nil:
		*matches = append(*matches, *result.match)
		state.updateStatus(candidate.FileName, scanItemStatusRecognized, *result.match)
	case result.miss:
		*misses = append(*misses, candidate)
		state.updateStatus(candidate.FileName, scanItemStatusScanning, scanMatch{})
	case result.err != nil:
		unsure[candidate.Path] = result.err
		if result.allowFallback {
			*misses = append(*misses, candidate)
			state.updateStatus(candidate.FileName, scanItemStatusScanning, scanMatch{})
		} else {
			state.updateStatus(candidate.FileName, scanItemStatusUnsure, scanMatch{})
		}
	}
}

func finalizeModrinthLookupOutcome(matches []scanMatch, misses []scanCandidate, unsure map[string]error) platformLookupOutcome {
	sortScanCandidatesByFile(misses)
	return platformLookupOutcome{matches: matches, misses: misses, unsure: unsure}
}

func lookupCurseforgeWithUpdates(
	ctx context.Context,
	candidates []scanCandidate,
	deps scanDeps,
	state *scanExecState,
) (platformLookupOutcome, error) {
	markScanCandidatesActive(state, candidates)
	outcome, err := lookupCurseforge(ctx, candidates, deps)
	if err != nil {
		return platformLookupOutcome{}, err
	}

	sortedMatches := cloneScanMatches(outcome.matches)
	sortScanMatchesByFile(sortedMatches)
	sortedMisses := cloneScanCandidates(outcome.misses)
	sortScanCandidatesByFile(sortedMisses)

	missesByPath := make(map[string]struct{}, len(outcome.misses))
	for _, miss := range outcome.misses {
		missesByPath[miss.Path] = struct{}{}
	}

	for _, match := range sortedMatches {
		state.updateStatus(match.FileName, scanItemStatusRecognized, match)
	}
	for _, miss := range sortedMisses {
		state.updateStatus(miss.FileName, scanItemStatusScanning, scanMatch{})
	}
	for _, candidate := range candidates {
		outcomeErr, ok := outcome.unsure[candidate.Path]
		if !ok {
			continue
		}
		if _, ok := missesByPath[candidate.Path]; ok {
			continue
		}
		state.updateStatus(candidate.FileName, scanItemStatusUnsure, scanMatch{})
		outcome.unsure[candidate.Path] = outcomeErr
	}

	return platformLookupOutcome{matches: outcome.matches, misses: outcome.misses, unsure: outcome.unsure}, nil
}

func finalizeScanResults(
	state *scanExecState,
	preferredUnsure map[string]error,
	fallbackUnsure map[string]error,
	fallbackMisses []scanCandidate,
	matches []scanMatch,
) ([]string, []scanUnsure) {
	finalUnsure := mergeUnsure(preferredUnsure, fallbackUnsure, matches)
	unknown := make([]string, 0, len(fallbackMisses))

	seenMisses := make(map[string]struct{}, len(fallbackMisses))
	for _, miss := range fallbackMisses {
		if _, seen := seenMisses[miss.Path]; seen {
			continue
		}
		seenMisses[miss.Path] = struct{}{}

		if _, ok := finalUnsure[miss.Path]; ok {
			state.updateStatus(miss.FileName, scanItemStatusUnsure, scanMatch{})
			continue
		}
		unknown = append(unknown, miss.FileName)
		state.updateStatus(miss.FileName, scanItemStatusUnknown, scanMatch{})
	}

	unsure := make([]scanUnsure, 0, len(finalUnsure))
	for path, outcomeErr := range finalUnsure {
		fileName := filepath.Base(path)
		unsure = append(unsure, scanUnsure{Path: fileName, Error: outcomeErr})
	}

	sort.Strings(unknown)
	sort.SliceStable(unsure, func(index int, other int) bool {
		return strings.ToLower(unsure[index].Path) < strings.ToLower(unsure[other].Path)
	})

	return unknown, unsure
}

func markScanCandidatesActive(state *scanExecState, candidates []scanCandidate) {
	if state == nil || len(candidates) == 0 {
		return
	}
	ordered := cloneScanCandidates(candidates)
	sortScanCandidatesByFile(ordered)
	for _, candidate := range ordered {
		state.updateStatus(candidate.FileName, scanItemStatusScanning, scanMatch{})
	}
}

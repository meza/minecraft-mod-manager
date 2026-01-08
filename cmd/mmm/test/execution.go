package test

import (
	"context"
	"errors"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"

	"github.com/meza/minecraft-mod-manager/internal/clierrors"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/platform"
)

type testExecSender struct {
	send func(tea.Msg)
}

func (sender testExecSender) Send(msg tea.Msg) {
	if sender.send == nil {
		return
	}
	sender.send(msg)
}

type testCheckCandidate struct {
	index int
	mod   models.Mod
}

type testCheckOutcome struct {
	item      testItem
	logEvents []logEvent
}

func runTestExecution(ctx context.Context, input testExecutionInput, sender testExecSender) testExecutionOutcome {
	if err := ctx.Err(); err != nil {
		return testExecutionOutcome{err: err}
	}

	items := cloneTestItems(input.items)
	candidates := make([]testCheckCandidate, 0, len(items))
	for index, item := range items {
		candidates = append(candidates, testCheckCandidate{
			index: index,
			mod:   item.Mod,
		})
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(defaultTestMaxConcurrency)
	var itemsMutex sync.Mutex

	for _, candidate := range candidates {
		candidate := candidate
		group.Go(func() error {
			outcome := checkMod(groupCtx, input.cfg, candidate, input.targetVersion, input.deps)

			if logErr := logEvents(input.deps, outcome.logEvents); logErr != nil {
				return logErr
			}

			itemsMutex.Lock()
			if candidate.index >= 0 && candidate.index < len(items) {
				items[candidate.index] = outcome.item
			}
			itemsMutex.Unlock()

			sender.Send(testItemUpdateMsg{
				key:    testModKey(candidate.mod),
				status: outcome.item.Status,
				reason: outcome.item.Reason,
			})

			if err := groupCtx.Err(); err != nil {
				return err
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return testExecutionOutcome{err: err}
	}

	return testExecutionOutcome{items: items}
}

func logEvents(deps testDeps, events []logEvent) error {
	for _, event := range events {
		switch event.Kind {
		case logEventKindDebug, logEventKindError:
			if deps.logger == nil {
				continue
			}
			if err := deps.logger.Debug(event.Message); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

func checkMod(
	ctx context.Context,
	cfg models.ModsJSON,
	candidate testCheckCandidate,
	targetVersion string,
	deps testDeps,
) testCheckOutcome {
	mod := candidate.mod
	outcome := testCheckOutcome{
		item: testItem{
			Mod:    mod,
			Status: testItemStatusSupported,
		},
	}

	outcome.logEvents = append(outcome.logEvents, buildCheckModLogEvent(mod, targetVersion))
	fetchOpts := buildFetchOptions(cfg, mod, targetVersion)

	_, fetchErr := deps.fetchMod(ctx, mod.Type, mod.ID, fetchOpts, deps.clients)
	if fetchErr == nil {
		return outcome
	}
	applyFetchFailure(&outcome, fetchErr, cfg, mod, targetVersion, fetchOpts)
	return outcome
}

func isUnsupportedFetchError(fetchErr error) bool {
	var notFound *platform.ModNotFoundError
	var noFile *platform.NoCompatibleFileError
	return errors.As(fetchErr, &notFound) || errors.As(fetchErr, &noFile)
}

func buildCheckModLogEvent(mod models.Mod, targetVersion string) logEvent {
	return logEvent{
		Kind: logEventKindDebug,
		Message: i18n.T("cmd.test.debug.checking", &i18n.Tvars{
			Data: &i18n.TData{
				"name":     mod.Name,
				"platform": string(mod.Type),
				"version":  targetVersion,
			},
		}),
	}
}

func buildFetchOptions(cfg models.ModsJSON, mod models.Mod, targetVersion string) platform.FetchOptions {
	fetchOpts := platform.FetchOptions{
		AllowedReleaseTypes: models.EffectiveAllowedReleaseTypes(mod, cfg),
		GameVersion:         targetVersion,
		Loader:              cfg.Loader,
		AllowFallback:       mod.AllowVersionFallback != nil && *mod.AllowVersionFallback,
	}

	if mod.Version != nil && strings.TrimSpace(*mod.Version) != "" {
		fetchOpts.FixedVersion = *mod.Version
	}
	return fetchOpts
}

func applyFetchFailure(
	outcome *testCheckOutcome,
	fetchErr error,
	cfg models.ModsJSON,
	mod models.Mod,
	targetVersion string,
	fetchOpts platform.FetchOptions,
) {
	outcome.logEvents = append(outcome.logEvents, fetchFailureUserEvent(fetchErr, mod))
	if detailEvent, ok := fetchFailureDetailEvent(fetchErr, mod); ok {
		outcome.logEvents = append(outcome.logEvents, detailEvent)
	}
	if debugEvent, ok := fetchFailureDebugEvent(fetchErr, mod, cfg, targetVersion, fetchOpts); ok {
		outcome.logEvents = append(outcome.logEvents, debugEvent)
	}

	if isUnsupportedFetchError(fetchErr) {
		outcome.item.Status = testItemStatusUnsupported
		return
	}

	outcome.item.Status = testItemStatusInconclusive
	outcome.item.Reason = fetchFailureReason(fetchErr, mod.Type)
}

func fetchFailureReason(fetchErr error, platformName models.Platform) string {
	summary, ok := clierrors.SummarizePlatformError(fetchErr, platformName)
	if ok && strings.TrimSpace(summary.Reason) != "" {
		return summary.Reason
	}
	return i18n.T("cmd.platform.error.reason.unknown", &i18n.Tvars{
		Data: &i18n.TData{"platform": string(platformName)},
	})
}

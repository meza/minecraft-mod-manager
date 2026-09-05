//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/meza/minecraft-mod-manager/testutil/bdd"
)

const (
	actionSucceededOutcome = "action-succeeded"
	actionFailedOutcome    = "action-failed"
)

type scenarioEnvironment struct {
	workspace string
	binary    string
	driver    terminalDriver
}

type actorAction struct {
	run func() error
}

func (action actorAction) Execute(_ *bdd.Actor) bdd.Outcome {
	if err := action.run(); err != nil {
		return bdd.NewOutcome(actionFailedOutcome, err.Error())
	}
	return bdd.NewOutcome(actionSucceededOutcome, "")
}

func (action actorAction) Copy() bdd.Action {
	return actorAction{run: action.run}
}

func (state *scenarioState) initializeTerminalSteps(scenario *godog.ScenarioContext) {
	scenario.Step(`^(\w+) uses MMM in an empty workspace$`, state.actorUsesMMMInEmptyWorkspace)
	scenario.Step(`^(\w+) starts interactive initialization$`, state.actorStartsInteractiveInitialization)
	scenario.Step(`^(\w+) should see the i18n key "([^"]+)"$`, state.actorShouldSeeI18nKey)
	scenario.Step(`^(\w+) cancels initialization$`, state.actorCancelsInitialization)
	scenario.Step(`^(\w+) should observe a successful exit$`, state.actorShouldObserveSuccessfulExit)
	scenario.Step(`^(\w+) should find no configuration in the workspace$`, state.actorShouldFindNoConfiguration)
}

func (state *scenarioState) actorUsesMMMInEmptyWorkspace(actorLabel string) error {
	workspace, err := os.MkdirTemp("", "mmm-e2e-")
	if err != nil {
		return fmt.Errorf("create isolated E2E workspace: %w", err)
	}

	commandDirectory, err := os.Getwd()
	if err != nil {
		_ = os.RemoveAll(workspace)
		return fmt.Errorf("resolve E2E command directory: %w", err)
	}
	binary, err := resolveE2EBinary(commandDirectory)
	if err != nil {
		_ = os.RemoveAll(workspace)
		return err
	}
	driver, err := newTUITestClient(commandDirectory)
	if err != nil {
		_ = os.RemoveAll(workspace)
		return err
	}

	state.environment = &scenarioEnvironment{
		workspace: workspace,
		binary:    binary,
		driver:    driver,
	}
	state.actors.Context().SetSubject(state.environment)
	return state.actorIsAnActor(actorLabel)
}

func resolveE2EBinary(commandDirectory string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("MMM_E2E_BINARY")); override != "" {
		return filepath.Abs(override)
	}

	name := "mmm"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary, err := filepath.Abs(filepath.Join(commandDirectory, "..", "build", "e2e", name))
	if err != nil {
		return "", fmt.Errorf("resolve E2E MMM binary: %w", err)
	}
	if info, statErr := os.Stat(binary); statErr != nil || info.IsDir() {
		return "", fmt.Errorf("E2E MMM binary not found at %q; run make e2e-build first", binary)
	}
	return binary, nil
}

func (state *scenarioState) actorStartsInteractiveInitialization(actorLabel string) error {
	return state.runActorAction(actorLabel, func() error {
		environment, err := state.requireEnvironment()
		if err != nil {
			return err
		}
		if err := environment.driver.Run(tuiTestLaunch{
			Program:          environment.binary,
			Arguments:        []string{"init"},
			WorkingDirectory: environment.workspace,
			Environment: []string{
				"MMM_TEST=1",
				"MMM_DISABLE_TELEMETRY=1",
				"MODRINTH_API_KEY=",
				"CURSEFORGE_API_KEY=",
				"POSTHOG_API_KEY=",
			},
			Columns: 120,
			Rows:    25,
		}); err != nil {
			return err
		}
		if err := environment.driver.Resize(100, 30); err != nil {
			return err
		}
		terminalState, err := environment.driver.State()
		if err != nil {
			return err
		}
		if terminalState.Columns != 100 || terminalState.Rows != 30 {
			return fmt.Errorf("expected resized terminal state 100x30, got %dx%d", terminalState.Columns, terminalState.Rows)
		}
		return nil
	})
}

func (state *scenarioState) actorShouldSeeI18nKey(actorLabel string, key string) error {
	if _, err := state.actors.GetActor(actorLabel); err != nil {
		return err
	}
	environment, err := state.requireEnvironment()
	if err != nil {
		return err
	}
	if err := environment.driver.WaitText(key, 10*time.Second); err != nil {
		return err
	}
	text, err := environment.driver.Text(false)
	if err != nil {
		return err
	}
	if !strings.Contains(text, key) {
		return fmt.Errorf("expected terminal text to contain i18n key %q, got %q", key, text)
	}
	return nil
}

func (state *scenarioState) actorCancelsInitialization(actorLabel string) error {
	return state.runActorAction(actorLabel, func() error {
		environment, err := state.requireEnvironment()
		if err != nil {
			return err
		}
		if err := environment.driver.Press("Ctrl+C"); err != nil {
			return err
		}
		return environment.driver.WaitExit(10 * time.Second)
	})
}

func (state *scenarioState) actorShouldObserveSuccessfulExit(actorLabel string) error {
	actor, err := state.actors.GetActor(actorLabel)
	if err != nil {
		return err
	}
	outcome, ok := actor.LastOutcome()
	if !ok {
		return errors.New("expected the cancellation action to record an outcome")
	}
	if outcome.Name != actionSucceededOutcome {
		return fmt.Errorf("cancellation failed: %s", outcome.Details)
	}
	environment, err := state.requireEnvironment()
	if err != nil {
		return err
	}
	terminalState, err := environment.driver.State()
	if err != nil {
		return err
	}
	if terminalState.Exited == nil {
		return errors.New("expected MMM to have exited")
	}
	if *terminalState.Exited != 0 {
		return fmt.Errorf("expected MMM exit status 0, got %d", *terminalState.Exited)
	}
	return nil
}

func (state *scenarioState) actorShouldFindNoConfiguration(actorLabel string) error {
	if _, err := state.actors.GetActor(actorLabel); err != nil {
		return err
	}
	environment, err := state.requireEnvironment()
	if err != nil {
		return err
	}
	for _, name := range []string{"modlist.json", "modlist-lock.json"} {
		path := filepath.Join(environment.workspace, name)
		if _, statErr := os.Stat(path); statErr == nil {
			return fmt.Errorf("unexpected configuration file created at %q", path)
		} else if !os.IsNotExist(statErr) {
			return fmt.Errorf("inspect configuration path %q: %w", path, statErr)
		}
	}
	return nil
}

func (state *scenarioState) runActorAction(actorLabel string, run func() error) error {
	actor, err := state.actors.GetActor(actorLabel)
	if err != nil {
		return err
	}
	outcome, err := actor.Run(actorAction{run: run})
	if err != nil {
		return err
	}
	if outcome.Name == actionFailedOutcome {
		return errors.New(outcome.Details)
	}
	return nil
}

func (state *scenarioState) requireEnvironment() (*scenarioEnvironment, error) {
	if state.environment == nil {
		return nil, errors.New("MMM E2E environment has not been prepared")
	}
	return state.environment, nil
}

func (state *scenarioState) cleanUp(currentContext context.Context, _ *godog.Scenario, scenarioErr error) (context.Context, error) {
	if state.environment == nil {
		return currentContext, nil
	}

	var cleanupErrors []error
	if scenarioErr != nil {
		if diagnostics := state.environment.driver.Diagnostics(); diagnostics != "" {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("terminal diagnostics:\n%s", diagnostics))
		}
	}
	if err := state.environment.driver.Close(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("close tui-test session: %w", err))
	}
	if err := os.RemoveAll(state.environment.workspace); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("remove E2E workspace %q: %w", state.environment.workspace, err))
	}
	state.environment = nil
	return currentContext, errors.Join(cleanupErrors...)
}

//go:build e2e

package e2e

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/meza/minecraft-mod-manager/testutil/bdd"
)

const placeholderOutcomeName = "placeholder-outcome"

type placeholderAction struct{}

func (placeholderAction) Execute(actor *bdd.Actor) bdd.Outcome {
	return bdd.NewOutcome(placeholderOutcomeName, actor.Label())
}

func (placeholderAction) Copy() bdd.Action {
	return placeholderAction{}
}

type scenarioState struct {
	actors      *bdd.ActorRegistry
	environment *scenarioEnvironment
}

func newScenarioState() *scenarioState {
	return &scenarioState{
		actors: bdd.NewActorRegistry(),
	}
}

func (state *scenarioState) reset() {
	state.actors = bdd.NewActorRegistry()
	state.environment = nil
}

func (state *scenarioState) actorIsAnActor(actorLabel string) error {
	actor := bdd.NewActor(actorLabel)
	return state.actors.AddActor(actorLabel, actor)
}

func (state *scenarioState) actorPerformsPlaceholderAction(actorLabel string) error {
	actor, err := state.actors.GetActor(actorLabel)
	if err != nil {
		return err
	}

	_, runErr := actor.Run(placeholderAction{})
	return runErr
}

func (state *scenarioState) actorShouldObservePlaceholderOutcome(actorLabel string, objectLabel string) error {
	actor, err := state.actors.GetActor(actorLabel)
	if err != nil {
		return err
	}

	outcome, ok, resolveErr := state.actors.ResolveOutcome(actor, objectLabel)
	if resolveErr != nil {
		return resolveErr
	}
	if !ok {
		return errors.New("expected outcome to be recorded")
	}
	if outcome.Name != placeholderOutcomeName {
		return fmt.Errorf("expected outcome %q, got %q", placeholderOutcomeName, outcome.Name)
	}
	return nil
}

// InitializeScenario registers BDD steps for godog.
func InitializeScenario(scenario *godog.ScenarioContext) {
	state := newScenarioState()

	scenario.Before(func(currentContext context.Context, _ *godog.Scenario) (context.Context, error) {
		state.reset()
		return currentContext, nil
	})
	scenario.After(state.cleanUp)

	scenario.Step(`^(\w+) (?:am|is) an actor$`, state.actorIsAnActor)
	scenario.Step(`^(\w+) (?:perform|performs) a placeholder action$`, state.actorPerformsPlaceholderAction)
	scenario.Step(`^(\w+) should observe (it|the placeholder outcome)$`, state.actorShouldObservePlaceholderOutcome)
	state.initializeTerminalSteps(scenario)
}

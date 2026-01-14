package bdd

// Actor represents a named persona that can execute actions.
// Actors are bound to a shared conversation context by the registry.
type Actor struct {
	label       string
	context     *ConversationContext
	lastOutcome *Outcome
}

// NewActor creates a new actor with the provided label.
func NewActor(label string) *Actor {
	return &Actor{
		label: label,
	}
}

// Label returns the actor's label as registered in the actor registry.
func (actor *Actor) Label() string {
	return actor.label
}

// Run executes an action and records its outcome in both the actor and conversation context.
//
// Contract:
// - Requires a non-nil action and a bound conversation context (set via ActorRegistry.AddActor).
// - Records the outcome as the actor's last outcome and the conversation's last outcome.
// - Returns ErrNilAction or ErrMissingContext when preconditions are not met.
func (actor *Actor) Run(action Action) (Outcome, error) {
	if action == nil {
		return Outcome{}, ErrNilAction
	}
	if actor.context == nil {
		return Outcome{}, ErrMissingContext
	}

	outcome := action.Execute(actor)
	actor.lastOutcome = &outcome
	actor.context.Record(action, actor, outcome)
	return outcome, nil
}

// LastOutcome returns the last outcome produced by the actor.
func (actor *Actor) LastOutcome() (Outcome, bool) {
	if actor.lastOutcome == nil {
		return Outcome{}, false
	}
	return *actor.lastOutcome, true
}

func (actor *Actor) setContext(context *ConversationContext) {
	actor.context = context
}

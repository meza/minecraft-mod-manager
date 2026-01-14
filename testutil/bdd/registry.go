package bdd

import "strings"

// ActorRegistry manages actors and resolves relative references like pronouns.
type ActorRegistry struct {
	actors         map[string]*Actor
	context        *ConversationContext
	relativeLabels map[string]struct{}
}

// NewActorRegistry returns a registry with a shared conversation context.
func NewActorRegistry() *ActorRegistry {
	return &ActorRegistry{
		actors:  map[string]*Actor{},
		context: NewConversationContext(),
		relativeLabels: map[string]struct{}{
			"he":     {},
			"her":    {},
			"hers":   {},
			"him":    {},
			"his":    {},
			"me":     {},
			"mine":   {},
			"my":     {},
			"myself": {},
			"she":    {},
			"them":   {},
			"their":  {},
			"theirs": {},
			"they":   {},
		},
	}
}

// Context returns the shared conversation context for all actors in the registry.
func (registry *ActorRegistry) Context() *ConversationContext {
	return registry.context
}

// AddActor registers an actor with a label and binds it to the shared context.
//
// Contract:
//   - Labels must be non-empty and not reserved relative labels (for example: "me", "he").
//   - Actor must be non-nil and either unlabeled or labeled with the same value.
//   - Returns ErrEmptyActorLabel, ErrReservedActorLabel, ErrNilActor, ErrActorAlreadyExists,
//     or ErrActorLabelMismatch when invariants are violated.
func (registry *ActorRegistry) AddActor(label string, actor *Actor) error {
	trimmedLabel := strings.TrimSpace(label)
	if trimmedLabel == "" {
		return ErrEmptyActorLabel
	}
	if actor == nil {
		return ErrNilActor
	}
	normalizedLabel := strings.ToLower(trimmedLabel)
	if _, isRelative := registry.relativeLabels[normalizedLabel]; isRelative {
		return ErrReservedActorLabel
	}
	if _, exists := registry.actors[trimmedLabel]; exists {
		return ErrActorAlreadyExists
	}
	if actor.label != "" && actor.label != trimmedLabel {
		return ErrActorLabelMismatch
	}

	actor.label = trimmedLabel
	actor.setContext(registry.context)
	registry.actors[trimmedLabel] = actor
	registry.context.lastActor = actor
	return nil
}

// GetActor resolves a label or relative reference and returns the corresponding actor.
//
// Contract:
// - Relative labels resolve to the most recently recorded actor in the conversation.
// - Updates the conversation's last actor when a non-relative label resolves.
// - Returns ErrEmptyActorLabel, ErrNoLastActor, or ErrActorNotFound when resolution fails.
func (registry *ActorRegistry) GetActor(label string) (*Actor, error) {
	trimmedLabel := strings.TrimSpace(label)
	if trimmedLabel == "" {
		return nil, ErrEmptyActorLabel
	}
	normalizedLabel := strings.ToLower(trimmedLabel)
	if _, isRelative := registry.relativeLabels[normalizedLabel]; isRelative {
		if registry.context.lastActor == nil {
			return nil, ErrNoLastActor
		}
		return registry.context.lastActor, nil
	}
	actor, exists := registry.actors[trimmedLabel]
	if !exists {
		return nil, ErrActorNotFound
	}
	registry.context.lastActor = actor
	return actor, nil
}

// ResolveOutcome returns the outcome referenced by the step for the provided actor.
//
// Contract:
// - "it" resolves to the most recent conversation outcome.
// - Any other reference resolves to the actor's last outcome.
// - Returns ErrNilActor or ErrEmptyOutcomeReference when preconditions fail.
func (registry *ActorRegistry) ResolveOutcome(actor *Actor, reference string) (Outcome, bool, error) {
	if actor == nil {
		return Outcome{}, false, ErrNilActor
	}
	trimmedReference := strings.TrimSpace(reference)
	if trimmedReference == "" {
		return Outcome{}, false, ErrEmptyOutcomeReference
	}
	normalizedReference := strings.ToLower(trimmedReference)
	if normalizedReference == "it" {
		outcome, ok := registry.context.LastOutcome()
		return outcome, ok, nil
	}
	outcome, ok := actor.LastOutcome()
	return outcome, ok, nil
}

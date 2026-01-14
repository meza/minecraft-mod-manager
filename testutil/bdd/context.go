package bdd

// ConversationContext stores the shared conversational state between steps.
// It keeps only the current subject and the last actor, action, and outcome.
type ConversationContext struct {
	subject     any
	lastActor   *Actor
	lastAction  Action
	lastOutcome *Outcome
}

// NewConversationContext returns a fresh, empty conversation context.
func NewConversationContext() *ConversationContext {
	return &ConversationContext{}
}

// SetSubject records the current subject of the conversation.
func (context *ConversationContext) SetSubject(subject any) {
	context.subject = subject
}

// Subject returns the current conversation subject.
func (context *ConversationContext) Subject() any {
	return context.subject
}

// Record stores the last action, actor, and outcome in the conversation context.
//
// Contract:
// - Records a copy of the action to avoid shared mutable state.
// - Updates last actor and last outcome for pronoun resolution.
func (context *ConversationContext) Record(action Action, actor *Actor, outcome Outcome) {
	actionCopy := action.Copy()
	context.lastActor = actor
	context.lastAction = actionCopy
	context.lastOutcome = &outcome
}

// LastActor returns the most recent actor recorded in the conversation.
func (context *ConversationContext) LastActor() *Actor {
	return context.lastActor
}

// LastAction returns the most recent action recorded in the conversation.
func (context *ConversationContext) LastAction() Action {
	return context.lastAction
}

// LastOutcome returns the most recent outcome recorded in the conversation.
func (context *ConversationContext) LastOutcome() (Outcome, bool) {
	if context.lastOutcome == nil {
		return Outcome{}, false
	}
	return *context.lastOutcome, true
}

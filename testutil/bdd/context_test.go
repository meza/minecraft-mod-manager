package bdd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testAction struct {
	outcome Outcome
}

func (action testAction) Execute(actor *Actor) Outcome {
	return action.outcome
}

func (action testAction) Copy() Action {
	return testAction{outcome: action.outcome}
}

func TestConversationContextRecordsValues(t *testing.T) {
	context := NewConversationContext()
	actor := NewActor("I")
	outcome := NewOutcome("observed", "details")
	action := testAction{outcome: outcome}

	context.SetSubject("subject")
	context.Record(action, actor, outcome)

	require.Equal(t, "subject", context.Subject())
	require.Same(t, actor, context.LastActor())
	require.NotNil(t, context.LastAction())
	recordedOutcome, ok := context.LastOutcome()
	require.True(t, ok)
	require.Equal(t, outcome, recordedOutcome)
}

func TestConversationContextLastOutcomeUnset(t *testing.T) {
	context := NewConversationContext()

	_, ok := context.LastOutcome()
	require.False(t, ok)
}

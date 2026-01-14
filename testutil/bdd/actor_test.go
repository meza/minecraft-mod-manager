package bdd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActorRunRecordsOutcome(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("I")
	require.NoError(t, registry.AddActor("I", actor))

	expectedOutcome := NewOutcome("ran", "ok")
	action := testAction{outcome: expectedOutcome}

	outcome, err := actor.Run(action)
	require.NoError(t, err)
	require.Equal(t, expectedOutcome, outcome)

	lastOutcome, ok := actor.LastOutcome()
	require.True(t, ok)
	require.Equal(t, expectedOutcome, lastOutcome)

	contextOutcome, ok := registry.Context().LastOutcome()
	require.True(t, ok)
	require.Equal(t, expectedOutcome, contextOutcome)
}

func TestActorLabel(t *testing.T) {
	actor := NewActor("I")

	require.Equal(t, "I", actor.Label())
}

func TestActorRunRequiresAction(t *testing.T) {
	actor := NewActor("I")
	actor.setContext(NewConversationContext())

	_, err := actor.Run(nil)
	require.ErrorIs(t, err, ErrNilAction)
}

func TestActorRunRequiresContext(t *testing.T) {
	actor := NewActor("I")

	_, err := actor.Run(testAction{outcome: NewOutcome("missing", "")})
	require.ErrorIs(t, err, ErrMissingContext)
}

func TestActorLastOutcomeFalseWhenUnset(t *testing.T) {
	actor := NewActor("I")

	_, ok := actor.LastOutcome()
	require.False(t, ok)
}

package bdd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActorRegistryAddAndGetActor(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("I")

	require.NoError(t, registry.AddActor("I", actor))

	resolvedActor, err := registry.GetActor("I")
	require.NoError(t, err)
	require.Same(t, actor, resolvedActor)
	require.Same(t, actor, registry.Context().LastActor())
}

func TestActorRegistryResolvesRelativeLabel(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("I")

	require.NoError(t, registry.AddActor("I", actor))

	resolvedActor, err := registry.GetActor("me")
	require.NoError(t, err)
	require.Same(t, actor, resolvedActor)
}

func TestActorRegistryReturnsErrorForMissingRelativeActor(t *testing.T) {
	registry := NewActorRegistry()

	_, err := registry.GetActor("me")
	require.ErrorIs(t, err, ErrNoLastActor)
}

func TestActorRegistryRejectsInvalidLabels(t *testing.T) {
	registry := NewActorRegistry()

	err := registry.AddActor("", NewActor("I"))
	require.ErrorIs(t, err, ErrEmptyActorLabel)

	err = registry.AddActor("me", NewActor("me"))
	require.ErrorIs(t, err, ErrReservedActorLabel)
}

func TestActorRegistryRejectsDuplicateAndNilActor(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("I")

	err := registry.AddActor("I", nil)
	require.ErrorIs(t, err, ErrNilActor)

	require.NoError(t, registry.AddActor("I", actor))

	err = registry.AddActor("I", NewActor("I"))
	require.ErrorIs(t, err, ErrActorAlreadyExists)
}

func TestActorRegistryRejectsLabelMismatch(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("Alice")

	err := registry.AddActor("Bob", actor)
	require.ErrorIs(t, err, ErrActorLabelMismatch)
}

func TestActorRegistryReturnsErrorWhenActorMissing(t *testing.T) {
	registry := NewActorRegistry()

	_, err := registry.GetActor("I")
	require.ErrorIs(t, err, ErrActorNotFound)
}

func TestActorRegistryGetActorRequiresLabel(t *testing.T) {
	registry := NewActorRegistry()

	_, err := registry.GetActor(" ")
	require.ErrorIs(t, err, ErrEmptyActorLabel)
}

func TestActorRegistryResolveOutcomeUsesItReference(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("I")
	require.NoError(t, registry.AddActor("I", actor))

	expectedOutcome := NewOutcome("observed", "details")
	_, err := actor.Run(testAction{outcome: expectedOutcome})
	require.NoError(t, err)

	outcome, ok, resolveErr := registry.ResolveOutcome(actor, "it")
	require.NoError(t, resolveErr)
	require.True(t, ok)
	require.Equal(t, expectedOutcome, outcome)
}

func TestActorRegistryResolveOutcomeUsesActorOutcomeForExplicitReference(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("I")
	require.NoError(t, registry.AddActor("I", actor))

	expectedOutcome := NewOutcome("observed", "details")
	_, err := actor.Run(testAction{outcome: expectedOutcome})
	require.NoError(t, err)

	outcome, ok, resolveErr := registry.ResolveOutcome(actor, "the placeholder outcome")
	require.NoError(t, resolveErr)
	require.True(t, ok)
	require.Equal(t, expectedOutcome, outcome)
}

func TestActorRegistryResolveOutcomeRejectsEmptyReference(t *testing.T) {
	registry := NewActorRegistry()
	actor := NewActor("I")

	_, _, resolveErr := registry.ResolveOutcome(actor, " ")
	require.ErrorIs(t, resolveErr, ErrEmptyOutcomeReference)
}

func TestActorRegistryResolveOutcomeRejectsNilActor(t *testing.T) {
	registry := NewActorRegistry()

	_, _, resolveErr := registry.ResolveOutcome(nil, "it")
	require.ErrorIs(t, resolveErr, ErrNilActor)
}

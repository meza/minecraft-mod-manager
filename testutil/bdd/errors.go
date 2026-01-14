package bdd

import "errors"

var (
	// ErrActorAlreadyExists indicates an actor label is already registered.
	ErrActorAlreadyExists = errors.New("actor already exists")
	// ErrActorLabelMismatch indicates the actor label does not match the registry label.
	ErrActorLabelMismatch = errors.New("actor label does not match the registry label")
	// ErrActorNotFound indicates an actor label could not be resolved.
	ErrActorNotFound = errors.New("actor not found")
	// ErrEmptyActorLabel indicates an actor label was empty or whitespace.
	ErrEmptyActorLabel = errors.New("actor label is required")
	// ErrMissingContext indicates an actor was not attached to a conversation context.
	ErrMissingContext = errors.New("actor is missing conversation context")
	// ErrNilAction indicates a nil action was provided.
	ErrNilAction = errors.New("action is nil")
	// ErrNilActor indicates a nil actor was provided.
	ErrNilActor = errors.New("actor is nil")
	// ErrEmptyOutcomeReference indicates an outcome reference was empty.
	ErrEmptyOutcomeReference = errors.New("outcome reference is required")
	// ErrNoLastActor indicates a relative reference was used before any actor acted.
	ErrNoLastActor = errors.New("no last actor is available for relative references")
	// ErrReservedActorLabel indicates the label is reserved for relative references.
	ErrReservedActorLabel = errors.New("actor label is reserved for relative references")
)

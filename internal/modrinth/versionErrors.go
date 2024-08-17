package modrinth

import (
	"errors"
	"fmt"
)

type VersionNotFoundError struct {
	Lookup VersionHashLookup
}

func (e *VersionNotFoundError) Error() string {
	return fmt.Sprintf("Version not found for: %s@%s", e.Lookup.algorithm, e.Lookup.hash)
}

func (e *VersionNotFoundError) Is(target error) bool {
	t, ok := target.(*VersionNotFoundError)
	if !ok {
		return false
	}
	return e.Lookup.algorithm == t.Lookup.algorithm && e.Lookup.hash == t.Lookup.hash
}

type VersionApiError struct {
	Lookup VersionHashLookup
	Err    error
}

func (e *VersionApiError) Error() string {
	return fmt.Sprintf("Version cannot be fetched due to an api error: %s@%s", e.Lookup.algorithm, e.Lookup.hash)
}

func (e *VersionApiError) Is(target error) bool {
	var t *VersionApiError
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	return e.Lookup.algorithm == t.Lookup.algorithm && e.Lookup.hash == t.Lookup.hash
}

func (e *VersionApiError) Unwrap() error {
	return e.Err
}

func VersionApiErrorWrap(err error, lookup VersionHashLookup) error {
	return &VersionApiError{
		Lookup: lookup,
		Err:    err,
	}
}

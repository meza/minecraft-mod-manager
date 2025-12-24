package modrinth

import (
	"errors"
	"fmt"
)

type VersionNotFoundError struct {
	Lookup VersionHashLookup
}

func (versionError *VersionNotFoundError) Error() string {
	return fmt.Sprintf("Version not found for: %s@%s", versionError.Lookup.algorithm, versionError.Lookup.hash)
}

func (versionError *VersionNotFoundError) Is(target error) bool {
	t, ok := target.(*VersionNotFoundError)
	if !ok {
		return false
	}
	return versionError.Lookup.algorithm == t.Lookup.algorithm && versionError.Lookup.hash == t.Lookup.hash
}

type VersionAPIError struct {
	Lookup VersionHashLookup
	Err    error
}

func (versionError *VersionAPIError) Error() string {
	return fmt.Sprintf("Version cannot be fetched due to an api error: %s@%s", versionError.Lookup.algorithm, versionError.Lookup.hash)
}

func (versionError *VersionAPIError) Is(target error) bool {
	var t *VersionAPIError
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	return versionError.Lookup.algorithm == t.Lookup.algorithm && versionError.Lookup.hash == t.Lookup.hash
}

func (versionError *VersionAPIError) Unwrap() error {
	return versionError.Err
}

func VersionAPIErrorWrap(err error, lookup VersionHashLookup) error {
	return &VersionAPIError{
		Lookup: lookup,
		Err:    err,
	}
}

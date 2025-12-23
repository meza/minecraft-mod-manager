// Package curseforge implements CurseForge API helpers and models.
package curseforge

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

type FingerprintAPIError struct {
	Lookup []int
	Err    error
}

func (e *FingerprintAPIError) Error() string {
	return fmt.Sprintf("Fingerprints for %d cannot be fetched due to an api error: %v", e.Lookup, e.Err)
}

func (e *FingerprintAPIError) Is(target error) bool {
	var t *FingerprintAPIError
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	return reflect.DeepEqual(t.Lookup, e.Lookup) && errors.Is(t.Err, e.Err)
}

func (e *FingerprintAPIError) Unwrap() error {
	return e.Err
}

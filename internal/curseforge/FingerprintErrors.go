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

func (fingerprintError *FingerprintAPIError) Error() string {
	return fmt.Sprintf("Fingerprints for %d cannot be fetched due to an api error: %v", fingerprintError.Lookup, fingerprintError.Err)
}

func (fingerprintError *FingerprintAPIError) Is(target error) bool {
	var t *FingerprintAPIError
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	return reflect.DeepEqual(t.Lookup, fingerprintError.Lookup) && errors.Is(t.Err, fingerprintError.Err)
}

func (fingerprintError *FingerprintAPIError) Unwrap() error {
	return fingerprintError.Err
}

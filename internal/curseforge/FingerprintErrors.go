package curseforge

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

type FingerprintApiError struct {
	Lookup []int
	Err    error
}

func (e *FingerprintApiError) Error() string {
	return fmt.Sprintf("Fingerprints for %d cannot be fetched due to an api error: %v", e.Lookup, e.Err)
}

func (e *FingerprintApiError) Is(target error) bool {
	var t *FingerprintApiError
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	return reflect.DeepEqual(t.Lookup, e.Lookup) && errors.Is(t.Err, e.Err)
}

func (e *FingerprintApiError) Unwrap() error {
	return e.Err
}

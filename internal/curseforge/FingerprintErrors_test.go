package curseforge

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFingerprintApiError_Error(t *testing.T) {
	err := &FingerprintApiError{
		Lookup: []int{1, 2, 3},
		Err:    errors.New("underlying error"),
	}
	expected := "Fingerprints for [1 2 3] cannot be fetched due to an api error: underlying error"
	assert.Equal(t, expected, err.Error())
}

func TestFingerprintApiError_Is(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err1 := &FingerprintApiError{
		Lookup: []int{1, 2, 3},
		Err:    underlyingErr,
	}
	err2 := &FingerprintApiError{
		Lookup: []int{1, 2, 3},
		Err:    underlyingErr,
	}
	err3 := &FingerprintApiError{
		Lookup: []int{4, 5, 6},
		Err:    underlyingErr,
	}
	err4 := &FingerprintApiError{
		Lookup: []int{1, 2, 3},
		Err:    errors.New("different error"),
	}
	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(err4))
	assert.False(t, err1.Is(errors.New("some other error")))
}

func TestFingerprintApiError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &FingerprintApiError{
		Lookup: []int{1, 2, 3},
		Err:    underlyingErr,
	}
	assert.Equal(t, underlyingErr, err.Unwrap())
}

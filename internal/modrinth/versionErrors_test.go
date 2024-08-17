package modrinth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionNotFoundError_Error(t *testing.T) {
	err := &VersionNotFoundError{
		Lookup: VersionHashLookup{
			algorithm: Sha1,
			hash:      "AABBCCDD",
		},
	}
	expected := "Version not found for: sha1@AABBCCDD"
	assert.Equal(t, expected, err.Error())
}

func TestVersionApiError_Error(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &VersionApiError{Lookup: VersionHashLookup{
		algorithm: Sha512,
		hash:      "AABBCCDD1",
	}, Err: underlyingErr}
	expected := "Version cannot be fetched due to an api error: sha512@AABBCCDD1"
	assert.Equal(t, expected, err.Error())
}

func TestVersionApiError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &VersionApiError{
		Lookup: VersionHashLookup{
			algorithm: "sha1",
			hash:      "AABBCCDD",
		}, Err: underlyingErr}
	assert.Equal(t, underlyingErr, err.Unwrap())
}

func TestVersionApiErrorWrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	lookup := VersionHashLookup{
		algorithm: "sha1",
		hash:      "AABBCCDDEE",
	}
	err := VersionApiErrorWrap(underlyingErr, lookup)
	expected := &VersionApiError{
		Lookup: lookup,
		Err:    underlyingErr,
	}
	assert.Equal(t, expected, err)
}

func TestVersionNotFoundError_Is(t *testing.T) {
	err1 := &VersionNotFoundError{
		Lookup: VersionHashLookup{
			algorithm: Sha1,
			hash:      "AABBCCDD",
		},
	}
	err2 := &VersionNotFoundError{
		Lookup: VersionHashLookup{
			algorithm: Sha1,
			hash:      "AABBCCDD",
		},
	}
	err3 := &VersionNotFoundError{
		Lookup: VersionHashLookup{
			algorithm: Sha1,
			hash:      "EEFFGGHH",
		},
	}
	err4 := &VersionNotFoundError{
		Lookup: VersionHashLookup{
			algorithm: Sha512,
			hash:      "AABBCCDD",
		},
	}
	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(err4))
	assert.False(t, err1.Is(errors.New("some other error")))
}

func TestVersionApiError_Is(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err1 := &VersionApiError{Lookup: VersionHashLookup{
		algorithm: Sha1,
		hash:      "AABBCCDD",
	}, Err: underlyingErr}
	err2 := &VersionApiError{Lookup: VersionHashLookup{
		algorithm: Sha1,
		hash:      "AABBCCDD",
	}, Err: underlyingErr}
	err3 := &VersionApiError{Lookup: VersionHashLookup{
		algorithm: Sha1,
		hash:      "EEFFGGHH",
	}, Err: underlyingErr}
	err4 := &VersionApiError{Lookup: VersionHashLookup{
		algorithm: Sha512,
		hash:      "AABBCCDD",
	}, Err: underlyingErr}
	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(err4))
	assert.False(t, err1.Is(errors.New("some other error")))
}

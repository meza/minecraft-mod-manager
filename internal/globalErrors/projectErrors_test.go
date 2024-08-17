package globalErrors

import (
	"errors"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectNotFoundError_Error(t *testing.T) {
	err := &ProjectNotFoundError{ProjectID: "AABBCCDD", Platform: models.CURSEFORGE}
	expected := "Project not found on curseforge: AABBCCDD"
	assert.Equal(t, expected, err.Error())
}

func TestProjectApiError_Error(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &ProjectApiError{ProjectID: "AABBCCDD", Err: underlyingErr, Platform: models.MODRINTH}
	expected := "Project cannot be fetched due to an api error on modrinth: AABBCCDD"
	assert.Equal(t, expected, err.Error())
}

func TestProjectApiError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &ProjectApiError{ProjectID: "AABBCCDD", Err: underlyingErr, Platform: models.CURSEFORGE}
	assert.Equal(t, underlyingErr, err.Unwrap())
}

func TestProjectApiErrorWrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := ProjectApiErrorWrap(underlyingErr, "AABBCCDD", models.MODRINTH)
	expected := &ProjectApiError{ProjectID: "AABBCCDD", Err: underlyingErr, Platform: models.MODRINTH}
	assert.Equal(t, expected, err)
}

func TestProjectNotFoundError_Is(t *testing.T) {
	err1 := &ProjectNotFoundError{ProjectID: "AABBCCDD", Platform: models.MODRINTH}
	err2 := &ProjectNotFoundError{ProjectID: "AABBCCDD", Platform: models.MODRINTH}
	err3 := &ProjectNotFoundError{ProjectID: "EEFFGGHH", Platform: models.MODRINTH}
	err4 := &ProjectNotFoundError{ProjectID: "EEFFGGHH", Platform: models.CURSEFORGE}
	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err3.Is(err4))
	assert.False(t, err1.Is(errors.New("some other error")))
}

func TestProjectApiError_Is(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err1 := &ProjectApiError{ProjectID: "AABBCCDD", Err: underlyingErr, Platform: models.CURSEFORGE}
	err2 := &ProjectApiError{ProjectID: "AABBCCDD", Err: underlyingErr, Platform: models.CURSEFORGE}
	err3 := &ProjectApiError{ProjectID: "EEFFGGHH", Err: underlyingErr, Platform: models.CURSEFORGE}
	err4 := &ProjectApiError{ProjectID: "EEFFGGHH", Err: underlyingErr, Platform: models.MODRINTH}
	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err3.Is(err4))
	assert.False(t, err1.Is(errors.New("some other error")))
}

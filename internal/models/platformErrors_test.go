package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModNotFoundErrorMessage(t *testing.T) {
	err := (&ModNotFoundError{Platform: MODRINTH, ProjectID: "abc"}).Error()
	assert.Contains(t, err, "mod not found")
	assert.Contains(t, err, "modrinth")
}

func TestNoCompatibleFileErrorMessage(t *testing.T) {
	err := (&NoCompatibleFileError{Platform: CURSEFORGE, ProjectID: "abc"}).Error()
	assert.Contains(t, err, "no compatible file")
	assert.Contains(t, err, "curseforge")
}

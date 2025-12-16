package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformString(t *testing.T) {
	assert.Equal(t, "modrinth", MODRINTH.String())
	assert.Equal(t, "curseforge", CURSEFORGE.String())
	assert.Equal(t, "custom", Platform("custom").String())
}

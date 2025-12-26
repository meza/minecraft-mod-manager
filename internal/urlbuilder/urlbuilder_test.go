package urlbuilder

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinEscapedPath_EscapesSegments(t *testing.T) {
	base, err := url.Parse("https://api.curseforge.com/v1")
	assert.NoError(t, err)

	joined := JoinEscapedPath(base, "mods", "123/45")
	assert.Equal(t, "/v1/mods/123%2F45", joined.EscapedPath())
	assert.Equal(t, "/v1/mods/123/45", joined.Path)
}

func TestJoinEscapedPath_AddsLeadingSlashForRootBase(t *testing.T) {
	base, err := url.Parse("https://api.modrinth.com")
	assert.NoError(t, err)

	joined := JoinEscapedPath(base, "v2", "project", "AABBCCDD")
	assert.Equal(t, "/v2/project/AABBCCDD", joined.EscapedPath())
	assert.Equal(t, "/v2/project/AABBCCDD", joined.Path)
}

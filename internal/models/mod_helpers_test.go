package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEffectiveAllowedReleaseTypesPrefersModOverrides(t *testing.T) {
	cfg := ModsJSON{DefaultAllowedReleaseTypes: []ReleaseType{Release}}
	mod := Mod{AllowedReleaseTypes: []ReleaseType{Beta}}

	chosen := EffectiveAllowedReleaseTypes(mod, cfg)

	assert.Equal(t, []ReleaseType{Beta}, chosen)
}

func TestEffectiveAllowedReleaseTypesFallsBackToConfig(t *testing.T) {
	cfg := ModsJSON{DefaultAllowedReleaseTypes: []ReleaseType{Release, Alpha}}
	mod := Mod{}

	chosen := EffectiveAllowedReleaseTypes(mod, cfg)

	assert.Equal(t, []ReleaseType{Release, Alpha}, chosen)
}

func TestLockIndexForModReturnsIndexWhenFound(t *testing.T) {
	mod := Mod{Type: CURSEFORGE, ID: "123"}
	lock := []ModInstall{
		{Type: MODRINTH, ID: "xyz"},
		{Type: CURSEFORGE, ID: "123"},
	}

	index := LockIndexForMod(mod, lock)

	assert.Equal(t, 1, index)
}

func TestLockIndexForModReturnsMinusOneWhenMissing(t *testing.T) {
	mod := Mod{Type: MODRINTH, ID: "missing"}
	lock := []ModInstall{{Type: MODRINTH, ID: "found"}}

	index := LockIndexForMod(mod, lock)

	assert.Equal(t, -1, index)
}

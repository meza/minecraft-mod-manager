package modfilename

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize_TrimsAndAcceptsJar(t *testing.T) {
	name, err := Normalize("  mod.JAR  ")
	require.NoError(t, err)
	assert.Equal(t, "mod.JAR", name)
}

func TestNormalize_RejectsEmpty(t *testing.T) {
	_, err := Normalize("   ")
	var validation Error
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonEmpty, validation.Reason)
}

func TestNormalize_RejectsPathSeparators(t *testing.T) {
	_, err := Normalize("mods/mod.jar")
	var validation Error
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonSeparator, validation.Reason)

	_, err = Normalize(`mods\mod.jar`)
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonSeparator, validation.Reason)
}

func TestNormalize_RejectsDriveLetters(t *testing.T) {
	_, err := Normalize("C:mod.jar")
	var validation Error
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonDriveLetter, validation.Reason)
}

func TestNormalize_RejectsUNCPaths(t *testing.T) {
	_, err := Normalize(`\\server\share\mod.jar`)
	var validation Error
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonUNCPath, validation.Reason)

	_, err = Normalize("//server/share/mod.jar")
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonUNCPath, validation.Reason)
}

func TestNormalize_RejectsNonJarExtension(t *testing.T) {
	_, err := Normalize("mod.zip")
	var validation Error
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonExtension, validation.Reason)

	_, err = Normalize("mod")
	assert.Error(t, err)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, ReasonExtension, validation.Reason)
}

func TestDisplay_UsesTrimmedValue(t *testing.T) {
	assert.Equal(t, "mod.jar", Display("  mod.jar  "))
	assert.Equal(t, "(empty)", Display("   "))
}

func TestErrorStringIncludesValueWhenPresent(t *testing.T) {
	err := Error{Value: "mod.jar", Reason: ReasonExtension}
	assert.Equal(t, "invalid mod filename mod.jar: extension", err.Error())
}

func TestErrorStringHandlesEmptyValue(t *testing.T) {
	err := Error{Value: "", Reason: ReasonEmpty}
	assert.Equal(t, "invalid mod filename: empty", err.Error())
}

func TestHelpers(t *testing.T) {
	assert.True(t, hasUNCPath(`\\server\share\mod.jar`))
	assert.True(t, hasUNCPath("//server/share/mod.jar"))
	assert.False(t, hasUNCPath("mod.jar"))

	assert.True(t, hasDriveLetter("C:mod.jar"))
	assert.True(t, hasDriveLetter("z:mod.jar"))
	assert.False(t, hasDriveLetter("C"))
	assert.False(t, hasDriveLetter("1:mod.jar"))
	assert.False(t, hasDriveLetter("mod.jar"))

	assert.True(t, isASCIIAlpha('a'))
	assert.True(t, isASCIIAlpha('Z'))
	assert.False(t, isASCIIAlpha('1'))
}

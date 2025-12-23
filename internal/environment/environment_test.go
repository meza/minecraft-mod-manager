package environment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModrinthAPIKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		original := modrinthAPIKeyDefault
		t.Cleanup(func() { modrinthAPIKeyDefault = original })
		modrinthAPIKeyDefault = "embedded_modrinth_placeholder"

		expected := "test_modrinth_placeholder"
		assert.NoError(t, os.Setenv("MODRINTH_API_KEY", expected))
		t.Cleanup(func() { assert.NoError(t, os.Unsetenv("MODRINTH_API_KEY")) })

		actual := ModrinthAPIKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		assert.NoError(t, os.Unsetenv("MODRINTH_API_KEY"))

		original := modrinthAPIKeyDefault
		t.Cleanup(func() { modrinthAPIKeyDefault = original })
		modrinthAPIKeyDefault = "embedded_modrinth_placeholder"

		expected := "embedded_modrinth_placeholder"
		actual := ModrinthAPIKey()
		assert.Equal(t, expected, actual)
	})
}

func TestCurseforgeAPIKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		original := curseforgeAPIKeyDefault
		t.Cleanup(func() { curseforgeAPIKeyDefault = original })
		curseforgeAPIKeyDefault = "embedded_curseforge_placeholder"

		expected := "test_curseforge_placeholder"
		assert.NoError(t, os.Setenv("CURSEFORGE_API_KEY", expected))
		t.Cleanup(func() { assert.NoError(t, os.Unsetenv("CURSEFORGE_API_KEY")) })

		actual := CurseforgeAPIKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		assert.NoError(t, os.Unsetenv("CURSEFORGE_API_KEY"))

		original := curseforgeAPIKeyDefault
		t.Cleanup(func() { curseforgeAPIKeyDefault = original })
		curseforgeAPIKeyDefault = "embedded_curseforge_placeholder"

		expected := "embedded_curseforge_placeholder"
		actual := CurseforgeAPIKey()
		assert.Equal(t, expected, actual)
	})
}

func TestPosthogAPIKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		original := posthogAPIKeyDefault
		t.Cleanup(func() { posthogAPIKeyDefault = original })
		posthogAPIKeyDefault = "embedded_posthog_placeholder"

		expected := "test_posthog_placeholder"
		assert.NoError(t, os.Setenv("POSTHOG_API_KEY", expected))
		t.Cleanup(func() { assert.NoError(t, os.Unsetenv("POSTHOG_API_KEY")) })

		actual := PosthogAPIKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		assert.NoError(t, os.Unsetenv("POSTHOG_API_KEY"))

		original := posthogAPIKeyDefault
		t.Cleanup(func() { posthogAPIKeyDefault = original })
		posthogAPIKeyDefault = "embedded_posthog_placeholder"

		expected := "embedded_posthog_placeholder"
		actual := PosthogAPIKey()
		assert.Equal(t, expected, actual)
	})
}

func TestAppVersion(t *testing.T) {
	expected := "REPL_VERSION"
	actual := AppVersion()
	assert.Equal(t, expected, actual)
}

func TestHelpURL(t *testing.T) {
	assert.Equal(t, "REPL_HELP_URL", HelpURL())
}

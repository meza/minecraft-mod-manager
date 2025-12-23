package environment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModrinthApiKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		original := modrinthApiKeyDefault
		t.Cleanup(func() { modrinthApiKeyDefault = original })
		modrinthApiKeyDefault = "embedded_modrinth_placeholder"

		expected := "test_modrinth_placeholder"
		assert.NoError(t, os.Setenv("MODRINTH_API_KEY", expected))
		t.Cleanup(func() { assert.NoError(t, os.Unsetenv("MODRINTH_API_KEY")) })

		actual := ModrinthApiKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		assert.NoError(t, os.Unsetenv("MODRINTH_API_KEY"))

		original := modrinthApiKeyDefault
		t.Cleanup(func() { modrinthApiKeyDefault = original })
		modrinthApiKeyDefault = "embedded_modrinth_placeholder"

		expected := "embedded_modrinth_placeholder"
		actual := ModrinthApiKey()
		assert.Equal(t, expected, actual)
	})
}

func TestCurseforgeApiKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		original := curseforgeApiKeyDefault
		t.Cleanup(func() { curseforgeApiKeyDefault = original })
		curseforgeApiKeyDefault = "embedded_curseforge_placeholder"

		expected := "test_curseforge_placeholder"
		assert.NoError(t, os.Setenv("CURSEFORGE_API_KEY", expected))
		t.Cleanup(func() { assert.NoError(t, os.Unsetenv("CURSEFORGE_API_KEY")) })

		actual := CurseforgeApiKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		assert.NoError(t, os.Unsetenv("CURSEFORGE_API_KEY"))

		original := curseforgeApiKeyDefault
		t.Cleanup(func() { curseforgeApiKeyDefault = original })
		curseforgeApiKeyDefault = "embedded_curseforge_placeholder"

		expected := "embedded_curseforge_placeholder"
		actual := CurseforgeApiKey()
		assert.Equal(t, expected, actual)
	})
}

func TestPosthogApiKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		original := posthogApiKeyDefault
		t.Cleanup(func() { posthogApiKeyDefault = original })
		posthogApiKeyDefault = "embedded_posthog_placeholder"

		expected := "test_posthog_placeholder"
		assert.NoError(t, os.Setenv("POSTHOG_API_KEY", expected))
		t.Cleanup(func() { assert.NoError(t, os.Unsetenv("POSTHOG_API_KEY")) })

		actual := PosthogApiKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		assert.NoError(t, os.Unsetenv("POSTHOG_API_KEY"))

		original := posthogApiKeyDefault
		t.Cleanup(func() { posthogApiKeyDefault = original })
		posthogApiKeyDefault = "embedded_posthog_placeholder"

		expected := "embedded_posthog_placeholder"
		actual := PosthogApiKey()
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

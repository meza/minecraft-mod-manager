package environment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModrinthApiKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		expected := "test_modrinth_api_key"
		os.Setenv("MODRINTH_API_KEY", expected)
		defer os.Unsetenv("MODRINTH_API_KEY")

		actual := ModrinthApiKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		os.Unsetenv("MODRINTH_API_KEY")

		expected := "REPL_MODRINTH_API_KEY"
		actual := ModrinthApiKey()
		assert.Equal(t, expected, actual)
	})
}

func TestCurseforgeApiKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		expected := "test_curseforge_api_key"
		os.Setenv("CURSEFORGE_API_KEY", expected)
		defer os.Unsetenv("CURSEFORGE_API_KEY")

		actual := CurseforgeApiKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		os.Unsetenv("CURSEFORGE_API_KEY")

		expected := "REPL_CURSEFORGE_API_KEY"
		actual := CurseforgeApiKey()
		assert.Equal(t, expected, actual)
	})
}

func TestPosthogApiKey(t *testing.T) {
	t.Run("environment variable set", func(t *testing.T) {
		expected := "test_posthog_api_key"
		os.Setenv("POSTHOG_API_KEY", expected)
		defer os.Unsetenv("POSTHOG_API_KEY")

		actual := PosthogApiKey()
		assert.Equal(t, expected, actual)
	})

	t.Run("environment variable not set", func(t *testing.T) {
		os.Unsetenv("POSTHOG_API_KEY")

		expected := "REPL_POSTHOG_API_KEY"
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

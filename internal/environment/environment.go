package environment

import (
	"os"
)

var (
	modrinthApiKeyDefault   = "REPL_MODRINTH_API_KEY"   // #nosec G101 -- build-time placeholder replaced in release builds.
	curseforgeApiKeyDefault = "REPL_CURSEFORGE_API_KEY" // #nosec G101 -- build-time placeholder replaced in release builds.
	posthogApiKeyDefault    = "REPL_POSTHOG_API_KEY"    // #nosec G101 -- build-time placeholder replaced in release builds.
)

func ModrinthApiKey() string {
	key, present := os.LookupEnv("MODRINTH_API_KEY")
	if present {
		return key
	}

	return modrinthApiKeyDefault
}

func CurseforgeApiKey() string {
	key, present := os.LookupEnv("CURSEFORGE_API_KEY")
	if present {
		return key
	}

	return curseforgeApiKeyDefault
}

func PosthogApiKey() string {
	key, present := os.LookupEnv("POSTHOG_API_KEY")
	if present {
		return key
	}

	return posthogApiKeyDefault
}

func AppVersion() string {
	return "REPL_VERSION"
}

func HelpURL() string {
	return "REPL_HELP_URL"
}

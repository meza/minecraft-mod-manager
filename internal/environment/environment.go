// Package environment reads runtime environment configuration.
package environment

import (
	"os"
)

var (
	modrinthAPIKeyDefault   = "REPL_MODRINTH_API_KEY"   // #nosec G101 -- build-time placeholder replaced in release builds.
	curseforgeAPIKeyDefault = "REPL_CURSEFORGE_API_KEY" // #nosec G101 -- build-time placeholder replaced in release builds.
	posthogAPIKeyDefault    = "REPL_POSTHOG_API_KEY"    // #nosec G101 -- build-time placeholder replaced in release builds.
)

func ModrinthAPIKey() string {
	key, present := os.LookupEnv("MODRINTH_API_KEY")
	if present {
		return key
	}

	return modrinthAPIKeyDefault
}

func CurseforgeAPIKey() string {
	key, present := os.LookupEnv("CURSEFORGE_API_KEY")
	if present {
		return key
	}

	return curseforgeAPIKeyDefault
}

func PosthogAPIKey() string {
	key, present := os.LookupEnv("POSTHOG_API_KEY")
	if present {
		return key
	}

	return posthogAPIKeyDefault
}

func AppVersion() string {
	return "REPL_VERSION"
}

func HelpURL() string {
	return "REPL_HELP_URL"
}

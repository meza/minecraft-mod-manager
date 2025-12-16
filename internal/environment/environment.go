package environment

import (
	"os"
)

func ModrinthApiKey() string {
	key, present := os.LookupEnv("MODRINTH_API_KEY")
	if present {
		return key
	}

	return "REPL_MODRINTH_API_KEY"
}

func CurseforgeApiKey() string {
	key, present := os.LookupEnv("CURSEFORGE_API_KEY")
	if present {
		return key
	}

	return "REPL_CURSEFORGE_API_KEY"
}

func PosthogApiKey() string {
	key, present := os.LookupEnv("POSTHOG_API_KEY")
	if present {
		return key
	}

	return "REPL_POSTHOG_API_KEY"
}

func AppVersion() string {
	return "REPL_VERSION"
}

func HelpURL() string {
	return "REPL_HELP_URL"
}

package terminal

import (
	"math/rand"
	"testing"

	"github.com/muesli/termenv"

	"github.com/meza/minecraft-mod-manager/internal/view"
)

// FixtureOptions controls shared terminal test setup.
type FixtureOptions struct {
	ColorProfile   termenv.Profile
	UnicodeEnabled bool
	SetMMMTestEnv  bool
	RandSeed       *int64
}

// FixtureOption mutates fixture options.
type FixtureOption func(options *FixtureOptions)

// WithColorProfile overrides the terminal color profile used in tests.
func WithColorProfile(profile termenv.Profile) FixtureOption {
	return func(options *FixtureOptions) {
		options.ColorProfile = profile
	}
}

// WithUnicodeEnabled toggles unicode support for tests.
func WithUnicodeEnabled(enabled bool) FixtureOption {
	return func(options *FixtureOptions) {
		options.UnicodeEnabled = enabled
	}
}

// WithoutMMMTestEnv disables setting MMM_TEST for the test.
func WithoutMMMTestEnv() FixtureOption {
	return func(options *FixtureOptions) {
		options.SetMMMTestEnv = false
	}
}

// WithRandSeed provides a deterministic rand generator via ApplyFixtures.
func WithRandSeed(seed int64) FixtureOption {
	return func(options *FixtureOptions) {
		options.RandSeed = &seed
	}
}

// ApplyFixtures configures shared terminal test settings and registers cleanup.
// It returns a deterministic rand generator when WithRandSeed is provided.
func ApplyFixtures(test testing.TB, options ...FixtureOption) *rand.Rand {
	if test == nil {
		return nil
	}
	fixtureOptions := FixtureOptions{
		ColorProfile:   termenv.Ascii,
		UnicodeEnabled: true,
		SetMMMTestEnv:  true,
	}
	for _, option := range options {
		option(&fixtureOptions)
	}

	if fixtureOptions.SetMMMTestEnv {
		test.Setenv("MMM_TEST", "true")
	}

	restoreColor := view.SetColorProfileFuncForTesting(func() termenv.Profile {
		return fixtureOptions.ColorProfile
	})
	test.Cleanup(restoreColor)

	restoreUnicode := view.SetUnicodeSupportFuncForTesting(func() bool {
		return fixtureOptions.UnicodeEnabled
	})
	test.Cleanup(restoreUnicode)

	if fixtureOptions.RandSeed != nil {
		//nolint:gosec // Deterministic test RNG for fixtures.
		return rand.New(rand.NewSource(*fixtureOptions.RandSeed))
	}
	return nil
}

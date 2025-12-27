package i18n

import (
	"embed"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

type mockLocaleProvider struct {
	LocaleProvider
}

func (provider mockLocaleProvider) GetLocales() ([]string, error) {
	return nil, errors.New("mock error")
}

type fakeLocaleProvider struct{}

func (provider fakeLocaleProvider) GetLocales() ([]string, error) {
	return []string{"fr_FR", "de_DE"}, nil
}

type emptyLocaleProvider struct{}

func (provider emptyLocaleProvider) GetLocales() ([]string, error) {
	return []string{"", "es_ES"}, nil
}

type customString string

func (value customString) String() string { return string(value) }

//go:embed __fixtures__/*.json
var testData embed.FS

func TestFormatKeyAndArgsReturnsEmptyOnWriteError(t *testing.T) {
	originalWriteString := i18nWriteString
	t.Cleanup(func() {
		i18nWriteString = originalWriteString
	})
	i18nWriteString = func(*strings.Builder, string) error {
		return errors.New("write failed")
	}

	assert.Equal(t, "", formatKeyAndArgs("key", Tvars{Count: 1}))
}

func TestFormatKeyAndArgsReturnsEmptyOnArgWriteError(t *testing.T) {
	originalWriteString := i18nWriteString
	t.Cleanup(func() {
		i18nWriteString = originalWriteString
	})
	callCount := 0
	i18nWriteString = func(*strings.Builder, string) error {
		callCount++
		if callCount == 2 {
			return errors.New("write failed")
		}
		return nil
	}

	assert.Equal(t, "", formatKeyAndArgs("key", Tvars{Count: 1}))
}

func TestNormalizeLocaleName(t *testing.T) {
	assert.Equal(t, "", normalizeLocaleName(" "))
	assert.Equal(t, "", normalizeLocaleName(".UTF-8"))
	assert.Equal(t, "fr-FR", normalizeLocaleName("fr_FR.UTF-8@euro"))
	assert.Equal(t, "en", normalizeLocaleName("C.UTF-8"))
	assert.Equal(t, "en", normalizeLocaleName("POSIX"))
}

//go:embed __fixtures_invalid__/*.json
var invalidLocales embed.FS

func TestSimpleTranslations(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"

	t.Run("simple translation", func(t *testing.T) {
		ResetForTesting()
		//Assuming that all systems running the tests have
		//English as their default language

		actual := T("test.simple")
		assert.Equal(t, "Hello World", actual)
	})

	t.Run("simple translation for tests", func(t *testing.T) {
		ResetForTesting()
		//Assuming that all systems running the tests have
		//English as their default language
		t.Setenv("MMM_TEST", "true")
		actual := T("test.simple")
		assert.Equal(t, "test.simple", actual)
	})

	t.Run("simple translation to german", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language
		ResetForTesting()

		t.Setenv("LANG", "de_DE")

		actual := T("test.simple")
		assert.Equal(t, "Hello World but in German", actual)
	})

	t.Run("custom type values are interpolated", func(t *testing.T) {
		ResetForTesting()
		actual := T("test.customType", Tvars{
			Data: &TData{"val": customString("XYZ")},
		})
		assert.Equal(t, "Value is XYZ", actual)
	})
}
func TestPluralsTranslations(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	t.Run("plurals in English", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language
		ResetForTesting()
		noPlural := T("test.multiple", Tvars{
			Data: &TData{"injectedData": "in English"},
		})
		assert.Equal(t, "Other message in English", noPlural)
		one := T("test.multiple", Tvars{
			Count: 1,
			Data:  &TData{"injectedData": "in English"},
		})
		assert.Equal(t, "One message: in English", one)
	})
	t.Run("plurals in German", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language
		ResetForTesting()
		t.Setenv("LANG", "de_DE")
		noPlural := T("test.multiple", Tvars{
			Data: &TData{"injectedData": "in English"},
		})
		assert.Equal(t, "Other message in English but in German", noPlural)
		one := T("test.multiple", Tvars{
			Count: 1,
			Data:  &TData{"injectedData": "in English"},
		})
		assert.Equal(t, "One message: in English but in German", one)
	})
	t.Run("plurals in test", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language
		t.Setenv("MMM_TEST", "true")
		noPlural := T("test.multiple", Tvars{
			Data: &TData{"injectedData": "in English"},
		})
		assert.Equal(t, "test.multiple, Arg 1: {Count: 0, Data: &map[injectedData:in English]}", noPlural)
		one := T("test.multiple", Tvars{
			Count: 1,
			Data:  &TData{"injectedData": "in English1"},
		})
		assert.Equal(t, "test.multiple, Arg 1: {Count: 1, Data: &map[injectedData:in English1]}", one)
	})
}

func TestMissingTranslation(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	ResetForTesting()

	t.Run("missing translation", func(t *testing.T) {
		actual := T("test.missing")
		assert.Equal(t, "test.missing", actual)
	})
}

func TestBadLangDir(t *testing.T) {
	enFS = testData
	langDir = "bogus"

	t.Run("bad lang dir", func(t *testing.T) {
		ResetForTesting()
		langDir = "badDir"
		assert.Equal(t, "test.simple", T("test.simple"))
	})
}

func TestInvalidLocaleFiles(t *testing.T) {
	enFS = invalidLocales
	langDir = "__fixtures_invalid__"
	ResetForTesting()

	assert.Equal(t, "test.simple", T("test.simple"))
}

func TestSetupKeepsDefaultFirst(t *testing.T) {
	originalFS := enFS
	originalLangDir := langDir

	enFS = testData
	langDir = "__fixtures__"
	ResetForTesting()

	t.Cleanup(func() {
		enFS = originalFS
		langDir = originalLangDir
		ResetForTesting()
	})

	assert.NoError(t, setup())

	supported := bundle.SupportedLanguages()
	assert.Equal(t, defaultLocale, supported[0].String())
}

func TestWrongNumberOfArguments(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"

	t.Run("wrong number of arguments", func(t *testing.T) {
		ResetForTesting()
		result := T("test.multiple",
			Tvars{Count: 1, Data: &TData{"injectedData": "first"}},
			Tvars{Count: 2, Data: &TData{"injectedData": "second"}},
		)
		assert.Equal(t, "test.multiple, Arg 1: {Count: 1, Data: &map[injectedData:first]}, Arg 2: {Count: 2, Data: &map[injectedData:second]}", result)
	})
}

func TestTestModeRequiresTestBinary(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	ResetForTesting()

	originalTestBinaryCheck := testBinaryCheck
	t.Cleanup(func() { testBinaryCheck = originalTestBinaryCheck })
	testBinaryCheck = func() bool { return false }

	t.Setenv("MMM_TEST", "true")

	actual := T("test.simple")
	assert.Equal(t, "Hello World", actual)
}

func TestFallbackToEnglish(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	localeProvider = mockLocaleProvider{}

	t.Run("fallback to English", func(t *testing.T) {
		ResetForTesting()

		actual := T("test.simple")
		assert.Equal(t, "Hello World", actual)
	})
}

func TestGetUserLocalesNoLang(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	localeProvider = mockLocaleProvider{}

	oldLang := os.Getenv("LANG")
	assert.NoError(t, os.Unsetenv("LANG"))
	t.Cleanup(func() { assert.NoError(t, os.Setenv("LANG", oldLang)) })

	locales := getUserLocales()
	assert.Equal(t, []string{language.English.String()}, locales)
}

func TestGetUserLocalesProviderSuccess(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	localeProvider = fakeLocaleProvider{}

	oldLang := os.Getenv("LANG")
	assert.NoError(t, os.Unsetenv("LANG"))
	t.Cleanup(func() { assert.NoError(t, os.Setenv("LANG", oldLang)) })

	locales := getUserLocales()
	assert.Equal(t, []string{"fr_FR", "de_DE"}, locales)
	assert.NotContains(t, locales, "")
}

func TestDefaultLocaleProvider(t *testing.T) {
	provider := DefaultLocaleProvider{}
	locales, err := provider.GetLocales()
	assert.NoError(t, err)
	assert.NotNil(t, locales)
}

func TestBuildLocalizerLocales(t *testing.T) {
	locales := buildLocalizerLocales([]string{"fr_FR", "de_DE", "fr_FR", ""})
	assert.Equal(t, []string{"fr-FR", "fr", "de-DE", "de"}, locales)

	withInvalid := buildLocalizerLocales([]string{"fr_FR", "???"})
	assert.Equal(t, []string{"fr-FR", "fr"}, withInvalid)
}

func TestBuildLocalizerLocalesNormalizesLangVariants(t *testing.T) {
	locales := buildLocalizerLocales([]string{"fr_FR.UTF-8", "C.UTF-8", "en_US.UTF-8"})
	assert.Equal(t, []string{"fr-FR", "fr", "en", "en-US"}, locales)
}

func TestGetUserLocalesSkipsEmptyEntries(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	localeProvider = emptyLocaleProvider{}

	oldLang := os.Getenv("LANG")
	assert.NoError(t, os.Unsetenv("LANG"))
	t.Cleanup(func() { assert.NoError(t, os.Setenv("LANG", oldLang)) })

	locales := getUserLocales()
	assert.Equal(t, []string{"es_ES"}, locales)
}

func TestTHandlesMissingLocalizerAfterInit(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	ResetForTesting()

	assert.Equal(t, "Hello World", T("test.simple"))

	translationMutex.Lock()
	localizer = nil
	translationMutex.Unlock()

	assert.Equal(t, "test.simple", T("test.simple"))
}

func TestConcurrentAccess(t *testing.T) {
	// This test verifies that concurrent calls to T() do not cause race conditions.
	// Run with `go test -race ./...` to detect data races.
	enFS = testData
	langDir = "__fixtures__"
	ResetForTesting()

	const goroutineCount = 100
	const iterationsPerGoroutine = 50

	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutineCount)

	// Start signal ensures all goroutines begin concurrently
	startSignal := make(chan struct{})

	for index := 0; index < goroutineCount; index++ {
		go func(routineIndex int) {
			defer waitGroup.Done()

			// Wait for start signal to maximize concurrent access
			<-startSignal

			for iteration := 0; iteration < iterationsPerGoroutine; iteration++ {
				// Mix of different translation calls to exercise various code paths
				switch iteration % 4 {
				case 0:
					result := T("test.simple")
					assert.NotEmpty(t, result)
				case 1:
					result := T("test.multiple", Tvars{
						Count: iteration,
						Data:  &TData{"injectedData": "concurrent"},
					})
					assert.NotEmpty(t, result)
				case 2:
					result := T("test.missing")
					assert.Equal(t, "test.missing", result)
				case 3:
					result := T("test.customType", Tvars{
						Data: &TData{"val": "test"},
					})
					assert.NotEmpty(t, result)
				}
			}
		}(index)
	}

	// Release all goroutines simultaneously
	close(startSignal)

	// Wait for all goroutines to complete
	waitGroup.Wait()
}

func TestConcurrentInitialization(t *testing.T) {
	// This test specifically verifies that concurrent initialization via sync.Once is thread-safe.
	// Multiple goroutines attempt to initialize the i18n system simultaneously.
	enFS = testData
	langDir = "__fixtures__"
	ResetForTesting()

	const goroutineCount = 50

	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutineCount)

	startSignal := make(chan struct{})

	results := make([]string, goroutineCount)

	for index := 0; index < goroutineCount; index++ {
		go func(routineIndex int) {
			defer waitGroup.Done()

			<-startSignal

			// All goroutines try to get a translation simultaneously
			// This forces concurrent initialization if not already initialized
			results[routineIndex] = T("test.simple")
		}(index)
	}

	// Release all goroutines simultaneously
	close(startSignal)

	// Wait for all goroutines to complete
	waitGroup.Wait()

	// All results should be identical since they use the same translation
	expectedResult := "Hello World"
	for index, result := range results {
		assert.Equal(t, expectedResult, result, "goroutine %d returned unexpected result", index)
	}
}

package i18n

import (
	"embed"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

type MockLocaleProvider struct {
	LocaleProvider
}

func (m MockLocaleProvider) GetLocales() ([]string, error) {
	return nil, errors.New("mock error")
}

type FakeLocaleProvider struct{}

func (f FakeLocaleProvider) GetLocales() ([]string, error) {
	return []string{"fr_FR", "de_DE"}, nil
}

//go:embed __fixtures__/*.json
var testData embed.FS

func TestSimpleTranslations(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"

	t.Run("simple translation", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language

		actual := T("test.simple")
		assert.Equal(t, "Hello World", actual)
	})

	t.Run("simple translation for tests", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language
		t.Setenv("MMM_TEST", "true")
		actual := T("test.simple")
		assert.Equal(t, "test.simple", actual)
	})

	t.Run("simple translation to german", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language
		localizer = nil

		t.Setenv("LANG", "de_DE")

		actual := T("test.simple")
		assert.Equal(t, "Hello World but in German", actual)
	})

}

func TestPluralsTranslations(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"

	t.Run("plurals in English", func(t *testing.T) {
		//Assuming that all systems running the tests have
		//English as their default language
		localizer = nil
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
		localizer = nil
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
	localizer = nil

	t.Run("missing translation", func(t *testing.T) {
		actual := T("test.missing")
		assert.Equal(t, "test.missing", actual)
	})
}

func TestBadLangDir(t *testing.T) {
	enFS = testData
	langDir = "bogus"

	t.Run("bad lang dir", func(t *testing.T) {
		langDir = "badDir"
		assert.Panics(t, func() {
			setup()
		})
	})
}

func TestWrongNumberOfArguments(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"

	t.Run("wrong number of arguments", func(t *testing.T) {
		localizer = nil
		assert.Panicsf(t, func() {
			T("test.simple", Tvars{}, Tvars{})
		}, "Too many arguments")
	})
}

func TestFallbackToEnglish(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	localeProvider = MockLocaleProvider{}

	t.Run("fallback to English", func(t *testing.T) {
		localizer = nil

		actual := T("test.simple")
		assert.Equal(t, "Hello World", actual)
	})
}

func TestGetUserLocalesNoLang(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	localeProvider = MockLocaleProvider{}

	oldLang := os.Getenv("LANG")
	os.Unsetenv("LANG")
	t.Cleanup(func() { os.Setenv("LANG", oldLang) })

	locales := getUserLocales()
	assert.Equal(t, []string{language.English.String()}, locales)
}

func TestGetUserLocalesProviderSuccess(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	localeProvider = FakeLocaleProvider{}

	oldLang := os.Getenv("LANG")
	os.Unsetenv("LANG")
	t.Cleanup(func() { os.Setenv("LANG", oldLang) })

	locales := getUserLocales()
	assert.Contains(t, locales, "fr_FR")
	assert.Contains(t, locales, "de_DE")
}

func TestDefaultLocaleProvider(t *testing.T) {
	provider := DefaultLocaleProvider{}
	locales, err := provider.GetLocales()
	assert.NoError(t, err)
	assert.NotNil(t, locales)
}

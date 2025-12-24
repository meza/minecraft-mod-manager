// Package i18n handles localized user-facing strings.
package i18n

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	goLocale "github.com/jeandeaual/go-locale"
	i18nLib "github.com/kaptinlin/go-i18n"
	"golang.org/x/text/language"
)

type LocaleProvider interface {
	GetLocales() ([]string, error)
}

type DefaultLocaleProvider struct{}

func (provider DefaultLocaleProvider) GetLocales() ([]string, error) {
	return goLocale.GetLocales()
}

//go:embed lang/*.json
var enFS embed.FS

const defaultLocale = "en-GB"

var localizer *i18nLib.Localizer
var bundle *i18nLib.I18n
var langDir = "lang"
var localeProvider LocaleProvider
var setupOnce sync.Once

// translationMutex guards localizer.Get() due to race conditions in go-i18n's internal cache.
var translationMutex sync.Mutex

func ResetForTesting() {
	translationMutex.Lock()
	localizer = nil
	bundle = nil
	translationMutex.Unlock()
	setupOnce = sync.Once{}
}

type TData map[string]interface{}

type Tvars struct {
	Count int
	Data  *TData
}

func ensureInitialized() {
	setupOnce.Do(setup)
}

func setup() {
	if localeProvider == nil {
		localeProvider = DefaultLocaleProvider{}
	}

	files, err := enFS.ReadDir(langDir)
	if err != nil {
		panic(err)
	}

	availableLocales := []string{defaultLocale}

	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			locale := strings.TrimSuffix(name, filepath.Ext(name))
			if strings.EqualFold(locale, defaultLocale) {
				continue
			}
			availableLocales = append(availableLocales, locale)
		}
	}

	newBundle := i18nLib.NewBundle(
		i18nLib.WithDefaultLocale(defaultLocale),
		i18nLib.WithLocales(availableLocales...),
	)

	if err := newBundle.LoadFS(enFS, fmt.Sprintf("%s/*.json", langDir)); err != nil {
		panic(err)
	}

	userLocales := buildLocalizerLocales(getUserLocales())
	newLocalizer := newBundle.NewLocalizer(userLocales...)

	translationMutex.Lock()
	bundle = newBundle
	localizer = newLocalizer
	translationMutex.Unlock()
}

func T(key string, args ...Tvars) string {
	_, present := os.LookupEnv("MMM_TEST")

	if present {
		return formatKeyAndArgs(key, args...)
	}

	ensureInitialized()

	if len(args) > 1 {
		panic("Too many arguments")
	}

	// Prepare vars before acquiring lock to minimize lock hold time
	var vars map[string]interface{}
	if len(args) > 0 {
		vars = make(map[string]interface{})
		if args[0].Data != nil {
			for varKey, value := range *args[0].Data {
				vars[varKey] = value
			}
		}
		vars["count"] = args[0].Count
	}

	translationMutex.Lock()
	defer translationMutex.Unlock()

	if len(args) == 0 {
		return localizer.Get(key)
	}

	return localizer.Get(key, i18nLib.Vars(vars))
}

func getUserLocales() []string {
	envLocale, present := os.LookupEnv("LANG")

	if present {
		return []string{envLocale}
	}

	detectedLocales, err := localeProvider.GetLocales()

	if err != nil {
		return []string{
			language.English.String(),
		}
	}

	locales := make([]string, 0, len(detectedLocales))
	for _, localeName := range detectedLocales {
		if localeName == "" {
			continue
		}
		locales = append(locales, localeName)
	}
	return locales
}

func formatKeyAndArgs(key string, args ...Tvars) string {
	var sb strings.Builder
	sb.WriteString(key)

	for i, arg := range args {
		sb.WriteString(fmt.Sprintf(", Arg %d: {Count: %d, Data: %v}", i+1, arg.Count, arg.Data))
	}

	return sb.String()
}

func buildLocalizerLocales(rawLocales []string) []string {
	locales := make([]string, 0, len(rawLocales)*2)
	seen := make(map[string]struct{}, len(rawLocales)*2)

	for _, localeName := range rawLocales {
		if localeName == "" {
			continue
		}

		tag, err := language.Parse(localeName)
		if err != nil {
			continue
		}

		canonical := tag.String()
		if _, ok := seen[canonical]; !ok {
			locales = append(locales, canonical)
			seen[canonical] = struct{}{}
		}

		if base, _ := tag.Base(); base.String() != "" {
			baseStr := base.String()
			if _, ok := seen[baseStr]; !ok {
				locales = append(locales, baseStr)
				seen[baseStr] = struct{}{}
			}
		}
	}

	return locales
}

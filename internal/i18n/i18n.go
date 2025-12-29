// Package i18n handles localized user-facing strings.
package i18n

import (
	"embed"
	"flag"
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
var setupError error
var testBinaryCheck = func() bool {
	return flag.Lookup("test.v") != nil
}

// translationMutex guards localizer.Get() due to race conditions in go-i18n's internal cache.
var translationMutex sync.Mutex

func ResetForTesting() {
	translationMutex.Lock()
	localizer = nil
	bundle = nil
	translationMutex.Unlock()
	setupOnce = sync.Once{}
	setupError = nil
}

type TData map[string]interface{}

type Tvars struct {
	Count int
	Data  *TData
}

var i18nWriteString = func(builder *strings.Builder, value string) error {
	_, err := builder.WriteString(value)
	return err
}

func ensureInitialized() {
	setupOnce.Do(func() {
		setupError = setup()
	})
}

func setup() error {
	if localeProvider == nil {
		localeProvider = DefaultLocaleProvider{}
	}

	files, err := enFS.ReadDir(langDir)
	if err != nil {
		return err
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
		return err
	}

	userLocales := buildLocalizerLocales(getUserLocales())
	newLocalizer := newBundle.NewLocalizer(userLocales...)

	translationMutex.Lock()
	bundle = newBundle
	localizer = newLocalizer
	translationMutex.Unlock()

	return nil
}

// T looks up a localized string by key and formats it with optional variables.
// Use it for all user-facing text so keys stay consistent across languages and
// tests, and so fallback behavior remains deterministic.
//
// Translation strings live in lang/*.json and use Go template variables.
// Example translation entry:
//
//	"mods.added": "Added {{.count}} mods"
//	"mods.remove.confirm": "Remove {{.name}}?"
//
// vars is optional. When provided:
// - Count becomes the template variable "count" (used for pluralization).
// - Data is a map of template variables (keys map to {{.key}} in the template).
//
// MMM_TEST turns T into test mode: it returns the key plus provided vars without
// attempting localization. Outside of test mode, T never returns raw key+vars.
//
// Examples:
//
//	title := i18n.T("tui.title", nil)
//	added := i18n.T("mods.added", &i18n.Tvars{Count: 2})
//	prompt := i18n.T("mods.remove.confirm", &i18n.Tvars{Data: &i18n.TData{
//	  "name": modName,
//	}})
func T(key string, vars *Tvars) string {
	if useTestMode() {
		return formatKeyAndArgs(key, vars)
	}

	ensureInitialized()

	if setupError != nil {
		return key
	}

	// Prepare vars before acquiring lock to minimize lock hold time
	var translatedVars map[string]interface{}
	if vars != nil {
		translatedVars = make(map[string]interface{})
		if vars.Data != nil {
			for varKey, value := range *vars.Data {
				translatedVars[varKey] = value
			}
		}
		translatedVars["count"] = vars.Count
	}

	translationMutex.Lock()
	defer translationMutex.Unlock()

	if localizer == nil {
		return key
	}

	if vars == nil {
		return localizer.Get(key)
	}

	return localizer.Get(key, i18nLib.Vars(translatedVars))
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

func formatKeyAndArgs(key string, vars *Tvars) string {
	var sb strings.Builder
	if err := i18nWriteString(&sb, key); err != nil {
		return ""
	}

	if vars == nil {
		return sb.String()
	}

	if err := i18nWriteString(&sb, fmt.Sprintf(", Arg 1: {Count: %d, Data: %v}", vars.Count, vars.Data)); err != nil {
		return ""
	}

	return sb.String()
}

func buildLocalizerLocales(rawLocales []string) []string {
	locales := make([]string, 0, len(rawLocales)*2)
	seen := make(map[string]struct{}, len(rawLocales)*2)

	for _, localeName := range rawLocales {
		normalizedLocale := normalizeLocaleName(localeName)
		if normalizedLocale == "" {
			continue
		}

		tag, err := language.Parse(normalizedLocale)
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

func normalizeLocaleName(localeName string) string {
	trimmed := strings.TrimSpace(localeName)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.SplitN(trimmed, ".", 2)[0]
	trimmed = strings.SplitN(trimmed, "@", 2)[0]
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.ReplaceAll(trimmed, "_", "-")
	upper := strings.ToUpper(trimmed)
	if upper == "C" || upper == "POSIX" {
		return language.English.String()
	}

	return trimmed
}

func useTestMode() bool {
	if !testBinaryCheck() {
		return false
	}

	_, present := os.LookupEnv("MMM_TEST")
	return present
}

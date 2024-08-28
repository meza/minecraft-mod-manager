package i18n

import (
	"embed"
	"encoding/json"
	"github.com/jeandeaual/go-locale"
	i18nLib "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"os"
)

//go:embed lang/*.json
var enFS embed.FS

var localizer *i18nLib.Localizer
var bundle *i18nLib.Bundle

type Tvars map[string]interface{}

func setup() {
	bundle = i18nLib.NewBundle(language.BritishEnglish)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	files, err := enFS.ReadDir("lang")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if !file.IsDir() {
			_, _ = bundle.LoadMessageFileFS(enFS, "lang/"+file.Name())
		}
	}

	localizer = i18nLib.NewLocalizer(bundle, getUserLocales()...)
}

func T(key string, args ...Tvars) string {

	if localizer == nil {
		setup()
	}

	localizeConfigWelcome := i18nLib.LocalizeConfig{
		MessageID: key,
	}

	if len(args) > 0 {
		localizeConfigWelcome.TemplateData = args[0]
	}

	localizationUsingJson, err := localizer.Localize(&localizeConfigWelcome)
	if err != nil {
		return key
	}
	return localizationUsingJson
}

func getUserLocales() []string {
	key, present := os.LookupEnv("LANG")

	if present {
		return []string{key}
	}

	detectedLocales, err := locale.GetLocales()

	var locales []string
	if err != nil {
		locales = []string{
			language.English.String(),
		}
	} else {
		locales = make([]string, len(detectedLocales))
		for _, localeName := range detectedLocales {
			locales = append(locales, localeName)
		}
	}
	return locales
}

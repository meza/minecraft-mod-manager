package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	goLocale "github.com/jeandeaual/go-locale"
	i18nLib "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type LocaleProvider interface {
	GetLocales() ([]string, error)
}

type DefaultLocaleProvider struct{}

func (d DefaultLocaleProvider) GetLocales() ([]string, error) {
	return goLocale.GetLocales()
}

//go:embed lang/*.json
var enFS embed.FS

var localizer *i18nLib.Localizer
var bundle *i18nLib.Bundle
var langDir = "lang"
var localeProvider LocaleProvider

type TData map[string]interface{}

type Tvars struct {
	Count int
	Data  *TData
}

func setup() {

	if localeProvider == nil {
		localeProvider = DefaultLocaleProvider{}
	}

	bundle = i18nLib.NewBundle(language.BritishEnglish)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	files, err := enFS.ReadDir(langDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if !file.IsDir() {
			// The path is a URL Path, not an OS dependent file path
			langFilePath := fmt.Sprintf("%s/%s", langDir, file.Name())
			_, _ = bundle.LoadMessageFileFS(enFS, langFilePath)
		}
	}

	locales := getUserLocales()
	localizer = i18nLib.NewLocalizer(bundle, locales...)
}

func T(key string, args ...Tvars) string {

	_, present := os.LookupEnv("MMM_TEST")

	if present {
		return formatKeyAndArgs(key, args...)
	}

	if localizer == nil {
		setup()
	}

	messageConfig := i18nLib.LocalizeConfig{
		MessageID: key,
	}

	if len(args) == 1 {
		messageConfig.TemplateData = args[0].Data
		messageConfig.PluralCount = args[0].Count
	}

	if len(args) > 1 {
		panic("Too many arguments")
	}

	localizationUsingJson, err := localizer.Localize(&messageConfig)
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

	detectedLocales, err := localeProvider.GetLocales()

	if err != nil {
		return []string{
			language.English.String(),
		}
	}

	locales := make([]string, 0, len(detectedLocales))
	for _, localeName := range detectedLocales {
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

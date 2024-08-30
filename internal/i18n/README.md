# I18N for the project

Unfortunately at the time of writing, there is no proper i18next support for golang so we had to be a bit creative.

We're using `go-i18n` under the hood, but the translation file format for that library is completely arbitrary and not
i18next compatible. So we're using a custom script to convert the i18next JSON files to the go-i18n format.

We're also using Locize as a translation management system, which is a great tool for managing translations.

## Understanding the directory structure

### `localise`

Contains the translation files used with Locize. These are the files that can be edited for tranlsation purposes.
With the correct API Key to Locize, running `make lang-sync` will download and/or upload the latest translations from Locize.

### `lang`

This is the directory where the go-i18n translation files are stored.
These files are generated from the files in the `localise` directory.

Editing these files directly is not recommended, as they will be overwritten by the `make lang-sync` command.


## Converting

Conversion is done by the cmd/lang package. To convert the files, run the following command:

```bash
go run cmd/lang/main.go
```

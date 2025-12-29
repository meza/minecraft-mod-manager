package init

import (
	"errors"
	"fmt"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/cobra"
)

func readInitOptions(cmd *cobra.Command, loader models.Loader) (initOptions, error) {
	gameVersion, err := cmd.Flags().GetString("game-version")
	if err != nil {
		return initOptions{}, err
	}
	modsFolder, err := cmd.Flags().GetString("mods-folder")
	if err != nil {
		return initOptions{}, err
	}
	releaseTypesRaw, err := cmd.Flags().GetStringSlice("release-types")
	if err != nil {
		return initOptions{}, err
	}
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return initOptions{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return initOptions{}, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return initOptions{}, err
	}

	releaseTypes, err := parseReleaseTypes(releaseTypesRaw)
	if err != nil {
		return initOptions{}, err
	}

	return initOptions{
		ConfigPath:   configPath,
		Quiet:        quiet,
		Debug:        debug,
		Loader:       loader,
		GameVersion:  gameVersion,
		ReleaseTypes: releaseTypes,
		ModsFolder:   modsFolder,
		Provided: providedFlags{
			Loader:       cmd.Flags().Changed("loader"),
			GameVersion:  cmd.Flags().Changed("game-version"),
			ReleaseTypes: cmd.Flags().Changed("release-types"),
			ModsFolder:   cmd.Flags().Changed("mods-folder"),
		},
	}, nil
}

func (flag *loaderFlag) String() string {
	return flag.value.String()
}

func (flag *loaderFlag) Set(value string) error {
	candidate := models.Loader(strings.TrimSpace(strings.ToLower(value)))
	for _, loader := range models.AllLoaders() {
		if loader == candidate {
			flag.value = candidate
			return nil
		}
	}
	return fmt.Errorf("%s", i18n.T("cmd.init.error.loader.invalid", &i18n.Tvars{
		Data: &i18n.TData{"loader": value},
	}))
}

func (flag *loaderFlag) Type() string {
	return "loader"
}

func isValidReleaseType(releaseType models.ReleaseType) bool {
	for _, r := range models.AllReleaseTypes() {
		if r == releaseType {
			return true
		}
	}
	return false
}

func parseReleaseTypes(raw []string) ([]models.ReleaseType, error) {
	releaseTypes := make([]models.ReleaseType, 0, len(raw))

	for _, part := range raw {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}

		candidate := models.ReleaseType(part)
		if !isValidReleaseType(candidate) {
			return nil, fmt.Errorf("%s", i18n.T("cmd.init.error.release-types.invalid", &i18n.Tvars{
				Data: &i18n.TData{"releaseType": part},
			}))
		}
		releaseTypes = append(releaseTypes, candidate)
	}

	if len(releaseTypes) == 0 {
		return nil, errors.New(i18n.T("cmd.init.error.release-types.empty", nil))
	}

	return releaseTypes, nil
}

func completeLoaders(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	loaders := models.AllLoaders()
	out := make([]string, 0, len(loaders))
	for _, loader := range loaders {
		out = append(out, loader.String())
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func completeReleaseTypes(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	releaseTypes := models.AllReleaseTypes()
	out := make([]string, 0, len(releaseTypes))
	for _, releaseType := range releaseTypes {
		out = append(out, string(releaseType))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func getAllReleaseTypes() string {
	releaseTypes := models.AllReleaseTypes()
	var releaseTypeList string

	for i, releaseType := range releaseTypes {
		releaseTypeList += string(releaseType)
		if i < len(releaseTypes)-1 {
			releaseTypeList += ", "
		}
	}

	return releaseTypeList
}

func getAllLoaders() string {
	loaders := models.AllLoaders()
	var loaderList string

	for i, loader := range loaders {
		loaderList += string(loader)
		if i < len(loaders)-1 {
			loaderList += ", "
		}
	}

	return loaderList
}

package init

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/meza/minecraft-mod-manager/internal/i18n"
	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/spf13/cobra"
	"os"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: i18n.T("cmd.init.short"),
		Run:   doInit,
	}

	cmd.Flags().StringP("loader", "l", "", i18n.T("cmd.init.usage.loader", i18n.Tvars{
		Data: &i18n.TData{"loaders": getAllLoaders()},
	}))
	cmd.Flags().StringP("release-types", "r", "", i18n.T("cmd.init.usage.release-types", i18n.Tvars{
		Data: &i18n.TData{"releaseTypes": getAllReleaseTypes()},
	}))
	cmd.Flags().StringP("game-version", "g", "", i18n.T("cmd.init.usage.game-version"))
	cmd.Flags().StringP("mods-folder", "m", "", i18n.T("cmd.init.usage.mods-folder", i18n.Tvars{
		Data: &i18n.TData{"cwd": getCurrentWorkingDirectory()},
	}))
	return cmd
}

func doInit(cmd *cobra.Command, args []string) {
	loader := cmd.Flag("loader").Value.String()
	gameVersion := cmd.Flag("game-version").Value.String()
	releaseTypes := cmd.Flag("release-types").Value.String()
	modsFolder := cmd.Flag("mods-folder").Value.String()

	if _, err := tea.NewProgram(NewModel(loader, gameVersion, releaseTypes, modsFolder)).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func getCurrentWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func getAllReleaseTypes() string {
	releaseTypes := models.AllReleaseTypes()
	var releaseTypeList string

	for i, releaseType := range releaseTypes {
		releaseTypeList += fmt.Sprintf("%s", string(releaseType))
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
		loaderList += fmt.Sprintf("%s", string(loader))
		if i < len(loaders)-1 {
			loaderList += ", "
		}
	}

	return loaderList
}

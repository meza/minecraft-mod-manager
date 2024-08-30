package init

import (
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/meza/minecraft-mod-manager/internal/fileutils"
)

func (m Model) selectedModsFolder() string {
	if m.options.modsFolder != "" && isValidModsFolder(m.options.modsFolder) {
		return m.options.modsFolder
	}

	return m.form.GetString("modsFolder")
}

func getModsFolderInput() *huh.FilePicker {
	return huh.NewFilePicker().
		CurrentDirectory(".").
		FileAllowed(false).
		DirAllowed(true).
		Key("modsFolder").
		Title("Select the mods folder").
		Validate(func(value string) error {
			if value == "" {
				return nil
			}

			if !isValidModsFolder(value) {
				return fmt.Errorf("invalid mods folder")
			}

			return nil
		})
}

func isValidModsFolder(modsFolder string) bool {
	if modsFolder == "" && !fileutils.FileExists(modsFolder) {
		return false
	}

	return true
}

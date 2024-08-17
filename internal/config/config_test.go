package config_test

import (
	"fmt"
	"github.com/meza/minecraft-mod-manager/internal/config"
	"testing"
)

func TestGetModsFolder(t *testing.T) {
	t.Run("mods folder exists", func(t *testing.T) {
		configuration, err := config.EnsureConfiguration("F:\\dev\\go\\src\\github.com\\meza\\minecraft-mod-manager\\modlist.json", false)
		if err != nil {
			return
		}

		fmt.Println(configuration)
	})
}

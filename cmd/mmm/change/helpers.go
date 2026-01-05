package change

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func changeModKey(mod models.Mod) string {
	return fmt.Sprintf("%s:%s", mod.Type, mod.ID)
}

type changeItemOrder int

const (
	changeItemOrderAlphabetical changeItemOrder = iota
	changeItemOrderConfig
)

func buildChangeItems(cfg models.ModsJSON, order changeItemOrder) ([]changeItem, map[string]int) {
	mods := make([]models.Mod, len(cfg.Mods))
	copy(mods, cfg.Mods)

	if order == changeItemOrderAlphabetical {
		sort.Slice(mods, func(i int, j int) bool {
			left := strings.ToLower(modDisplayName(mods[i]))
			right := strings.ToLower(modDisplayName(mods[j]))
			if left != right {
				return left < right
			}
			if mods[i].Type != mods[j].Type {
				return mods[i].Type < mods[j].Type
			}
			return mods[i].ID < mods[j].ID
		})
	}

	items := make([]changeItem, 0, len(mods))
	indexByKey := make(map[string]int, len(mods))
	for _, mod := range mods {
		item := changeItem{
			Mod:            mod,
			DisplayName:    modDisplayName(mod),
			CompatStatus:   changeCompatPending,
			DownloadStatus: changeDownloadPending,
			SwitchStatus:   changeSwitchPending,
		}
		indexByKey[changeModKey(mod)] = len(items)
		items = append(items, item)
	}
	return items, indexByKey
}

func cloneChangeItems(items []changeItem) []changeItem {
	clone := make([]changeItem, len(items))
	copy(clone, items)
	return clone
}

func modDisplayName(mod models.Mod) string {
	name := strings.TrimSpace(mod.Name)
	if name == "" {
		return mod.ID
	}
	return name
}

func colorModeForOutput(out io.Writer) view.ColorMode {
	if view.SupportsColor(out) {
		return view.ColorEnabled
	}
	return view.ColorDisabled
}

func indexLockByMod(lock []models.ModInstall) map[string]models.ModInstall {
	index := make(map[string]models.ModInstall, len(lock))
	for _, entry := range lock {
		index[fmt.Sprintf("%s:%s", entry.Type, entry.ID)] = entry
	}
	return index
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

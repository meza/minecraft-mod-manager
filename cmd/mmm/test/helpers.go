package test

import (
	"fmt"
	"io"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/models"
	"github.com/meza/minecraft-mod-manager/internal/view"
)

func testModKey(mod models.Mod) string {
	return fmt.Sprintf("%s:%s", mod.Type, mod.ID)
}

func buildTestItems(cfg models.ModsJSON) ([]testItem, map[string]int) {
	items := make([]testItem, 0, len(cfg.Mods))
	indexByKey := make(map[string]int, len(cfg.Mods))

	for _, mod := range cfg.Mods {
		item := testItem{
			Mod:    mod,
			Status: testItemStatusChecking,
		}
		indexByKey[testModKey(mod)] = len(items)
		items = append(items, item)
	}
	return items, indexByKey
}

func cloneTestItems(items []testItem) []testItem {
	clone := make([]testItem, len(items))
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
	if view.SupportsColor(out) && view.SupportsControlSequences(out) {
		return view.ColorEnabled
	}
	return view.ColorDisabled
}

func messageWithIcon(icon string, message string) string {
	return fmt.Sprintf("%s %s", icon, message)
}

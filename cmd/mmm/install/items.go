package install

import (
	"sort"
	"strings"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

type installItemStatus int

const (
	installItemPending installItemStatus = iota
	installItemDownloading
	installItemSuccess
	installItemFailed
	installItemAborted
)

type installItem struct {
	Mod           models.Mod
	DisplayName   string
	Status        installItemStatus
	FailureReason string
	Progress      *installProgress
}

type installProgress struct {
	ratio      float64
	downloaded int64
	total      int64
}

func buildInstallItems(cfg models.ModsJSON) ([]installItem, map[string]int) {
	items := make([]installItem, 0, len(cfg.Mods))
	for _, mod := range cfg.Mods {
		displayName := strings.TrimSpace(mod.Name)
		if displayName == "" {
			displayName = mod.ID
		}
		items = append(items, installItem{
			Mod:         mod,
			DisplayName: displayName,
			Status:      installItemPending,
		})
	}

	sort.SliceStable(items, func(leftIndex int, rightIndex int) bool {
		left := strings.ToLower(items[leftIndex].DisplayName)
		right := strings.ToLower(items[rightIndex].DisplayName)
		if left == right {
			leftPlatform := strings.ToLower(string(items[leftIndex].Mod.Type))
			rightPlatform := strings.ToLower(string(items[rightIndex].Mod.Type))
			if leftPlatform == rightPlatform {
				return strings.ToLower(items[leftIndex].Mod.ID) < strings.ToLower(items[rightIndex].Mod.ID)
			}
			return leftPlatform < rightPlatform
		}
		return left < right
	})

	indexByKey := make(map[string]int, len(items))
	for index, item := range items {
		indexByKey[installModKey(item.Mod)] = index
	}
	return items, indexByKey
}

func installModKey(mod models.Mod) string {
	return string(mod.Type) + ":" + mod.ID
}

func isTerminalInstallStatus(status installItemStatus) bool {
	switch status {
	case installItemSuccess, installItemFailed, installItemAborted:
		return true
	default:
		return false
	}
}

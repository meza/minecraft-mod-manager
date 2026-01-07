package install

import (
	"runtime"

	"github.com/meza/minecraft-mod-manager/internal/models"
)

var gomaxprocsFunc = runtime.GOMAXPROCS

func installMaxConcurrency(modCount int) int {
	limit := gomaxprocsFunc(0)
	if limit < 1 {
		limit = 1
	}
	if modCount > 0 && modCount < limit {
		return modCount
	}
	return limit
}

func snapshotInstallItems(state *installExecutionState, cfg models.ModsJSON) []installItem {
	if state != nil {
		return state.snapshot()
	}
	items, _ := buildInstallItems(cfg)
	return items
}

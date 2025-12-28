package models

// EffectiveAllowedReleaseTypes returns the mod-specific release types when set,
// otherwise it falls back to the config default list.
func EffectiveAllowedReleaseTypes(mod Mod, cfg ModsJSON) []ReleaseType {
	if len(mod.AllowedReleaseTypes) > 0 {
		return mod.AllowedReleaseTypes
	}
	return cfg.DefaultAllowedReleaseTypes
}

// LockIndexForMod returns the index of the lock entry matching the mod type/id,
// or -1 when no matching entry exists.
func LockIndexForMod(mod Mod, lock []ModInstall) int {
	for index := range lock {
		if lock[index].Type == mod.Type && lock[index].ID == mod.ID {
			return index
		}
	}
	return -1
}

package models

type ModsJson struct {
	Loader                     Loader        `json:"loader"`
	GameVersion                string        `json:"gameVersion"`
	DefaultAllowedReleaseTypes []ReleaseType `json:"defaultAllowedReleaseTypes"`
	ModsFolder                 string        `json:"modsFolder"`
	Mods                       []Mod         `json:"mods"`
}

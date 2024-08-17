package models

type Mod struct {
	Type                 Platform      `json:"type"`
	ID                   string        `json:"id"`
	AllowedReleaseTypes  []ReleaseType `json:"allowedReleaseTypes,omitempty"`
	Name                 string        `json:"name"`
	AllowVersionFallback *bool         `json:"allowVersionFallback,omitempty"`
	Version              *string       `json:"version,omitempty"`
}

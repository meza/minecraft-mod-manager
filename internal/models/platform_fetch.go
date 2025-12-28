package models

// RemoteMod is the normalized download metadata returned by platform-specific packages.
type RemoteMod struct {
	Name        string
	FileName    string
	ReleaseDate string
	Hash        string
	DownloadURL string
}

// FetchOptions describes how to select a mod file from a platform.
type FetchOptions struct {
	AllowedReleaseTypes []ReleaseType
	GameVersion         string
	Loader              Loader
	AllowFallback       bool
	FixedVersion        string
}

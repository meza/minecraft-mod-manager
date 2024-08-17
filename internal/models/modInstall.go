package models

type ModInstall struct {
	Type        Platform `json:"type"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	FileName    string   `json:"fileName"`
	ReleasedOn  string   `json:"releasedOn"`
	Hash        string   `json:"hash"`
	DownloadUrl string   `json:"downloadUrl"`
}

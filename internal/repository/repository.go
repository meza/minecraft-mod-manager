package repository

import "github.com/meza/minecraft-mod-manager/internal/models"

type Mod struct{}

type ModQuery struct {
	id                  string
	allowedReleaseTypes []models.ReleaseType
}

type Repository interface {
	FetchMod(query ModQuery) (Mod, error)
	LookupMod(hashes []string) (Mod, error)
}

func x(x models.ReleaseType) {
	if x == models.Alpha {
		println("Alpha")
	}
}

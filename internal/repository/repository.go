package repository

type Mod struct{}

type ReleaseType struct {
	slug string
}

type ModQuery struct {
	id                  string
	allowedReleaseTypes []ReleaseType
}

type Repository interface {
	FetchMod(query ModQuery) (Mod, error)
	LookupMod(hashes []string) (Mod, error)
}

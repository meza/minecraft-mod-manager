package models

type ReleaseType string

const (
	Alpha   ReleaseType = "alpha"
	Beta    ReleaseType = "beta"
	Release ReleaseType = "release"
)

func AllReleaseTypes() []ReleaseType {
	return []ReleaseType{
		Alpha,
		Beta,
		Release,
	}
}

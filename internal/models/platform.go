package models

type Platform string

const (
	CURSEFORGE Platform = "curseforge"
	MODRINTH   Platform = "modrinth"
)

func (p Platform) String() string {
	return string(p)
}

package models

type Platform string

const (
	CURSEFORGE Platform = "curseforge"
	MODRINTH   Platform = "modrinth"
)

func (platform Platform) String() string {
	return string(platform)
}

package tui

type ColorMode int

const (
	ColorDisabled ColorMode = iota
	ColorEnabled
)

func (mode ColorMode) Enabled() bool {
	return mode == ColorEnabled
}

type QuietMode int

const (
	QuietDisabled QuietMode = iota
	QuietEnabled
)

func (mode QuietMode) Enabled() bool {
	return mode == QuietEnabled
}

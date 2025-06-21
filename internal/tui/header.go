package tui

import (
	"github.com/charmbracelet/lipgloss"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	App     string
	Version string
	Extras  []string // e.g. {"Press 'q' to quit", "Press 'esc' to exit"}
}

// Pastel ramp – tweak/extend to taste.
var gradient = []string{
	"#89DCEB", "#99C9F5", "#B0B0FF", "#C49FFF", "#DB8AFF",
}

// ---------------------------------------------------------------------
// Utility: build a per-column gradient string exactly `width` runes wide
// ---------------------------------------------------------------------
func makeGradient(width int) []string {
	out := make([]string, width)
	for i := 0; i < width; i++ {
		col := gradient[i*len(gradient)/max(1, width)]
		out[i] = lipgloss.NewStyle().
			Background(lipgloss.Color(col)).
			Render(" ")
	}
	return out
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Header(cfg Config, width int) string {
	return RenderFloating(cfg, width)
}

func RenderFloating(cfg Config, width int) string {
	// 1. Build the plain text (with middot separators)
	parts := append([]string{
		cfg.App,
		"v" + cfg.Version,
	}, cfg.Extras...)

	plain := " " + strings.Join(parts, " | ")
	runes := []rune(plain)

	// 2. Prepare column-by-column output
	var out strings.Builder
	for col := 0; col < width; col++ {
		// Pick the colour for this column
		bgCol := gradient[col*len(gradient)/max(1, width)]

		// Decide the glyph (rune or space)
		glyph := " "
		if col < len(runes) {
			glyph = string(runes[col])
		}

		// Auto-select a readable foreground (black for light bg, white for dark)
		fgCol := "#FFFFFF"
		if isLight(bgCol) {
			fgCol = "#000000"
		}

		out.WriteString(
			lipgloss.NewStyle().
				Background(lipgloss.Color(bgCol)).
				Foreground(lipgloss.Color(fgCol)).
				Bold(true).
				Render(glyph),
		)
	}
	return out.String()
}

// very cheap luminance check: good enough for pastel vs dark
func isLight(hex string) bool {
	r, g, b := hexToRGB(hex)
	// relative luminance (Rec. 709)
	l := 0.2126*r + 0.7152*g + 0.0722*b
	return l > 0.5
}
func hexToRGB(h string) (r, g, b float64) {
	x := func(s string) float64 {
		v, _ := strconv.ParseUint(s, 16, 8)
		return float64(v) / 255
	}
	if strings.HasPrefix(h, "#") {
		h = h[1:]
	}
	switch len(h) {
	case 6:
		r, g, b = x(h[0:2]), x(h[2:4]), x(h[4:6])
	case 3: // short form #abc
		r, g, b = x(strings.Repeat(string(h[0]), 2)),
			x(strings.Repeat(string(h[1]), 2)),
			x(strings.Repeat(string(h[2]), 2))
	}
	return
}

var pillBG = lipgloss.Color("#313244") // Catppuccin surface-0

func RenderPills(cfg Config, width int) string {
	// ---- build pill strings with Powerline separators -------------------
	leftSep, rightSep := "", "" // U+E0B6 / U+E0B4
	base := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(pillBG).
		Bold(true).
		PaddingLeft(1).PaddingRight(1)

	cwd, _ := os.Getwd()
	words := append([]string{
		cfg.App, "v" + cfg.Version, filepath.Base(cwd),
	}, cfg.Extras...)

	var chipSB strings.Builder
	for _, w := range words {
		chipSB.WriteString(
			lipgloss.NewStyle().
				Foreground(pillBG). // separator takes pill colour
				Render(leftSep),
		)
		chipSB.WriteString(base.Render(w))
		chipSB.WriteString(
			lipgloss.NewStyle().
				Foreground(pillBG).
				Render(rightSep),
		)
	}
	chipRunes := []rune(chipSB.String())

	// ---- paint gradient underneath, then overwrite with pills ----------
	cells := makeGradient(width)
	for i, r := range chipRunes {
		if i >= width {
			break
		}
		// every rune translated to a single-width cell; safe for ASCII + Nerd glyphs
		cells[i] = string(r)
	}
	return strings.Join(cells, "")
}

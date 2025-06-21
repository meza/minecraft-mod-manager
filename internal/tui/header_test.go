package tui

import "testing"

func TestHeaderHelpers(t *testing.T) {
	g := makeGradient(5)
	if len(g) != 5 {
		t.Fatalf("expected gradient length 5, got %d", len(g))
	}
	if max(1, 2) != 2 || max(3, 1) != 3 {
		t.Fatalf("max not correct")
	}
	r, g2, b := hexToRGB("#ffffff")
	if r == 0 && g2 == 0 && b == 0 {
		t.Fatalf("hexToRGB failed")
	}
	hexToRGB("#abc")
	_ = isLight("#ffffff")
}

func TestRenderFloating(t *testing.T) {
	h := RenderFloating(Config{App: "A", Version: "1"}, 10)
	if len(h) == 0 {
		t.Fatalf("expected header string")
	}
}

func TestHeader(t *testing.T) {
	h := Header(Config{App: "A", Version: "1"}, 10)
	if len(h) == 0 {
		t.Fatalf("expected header")
	}
}

func TestRenderPills(t *testing.T) {
	p := RenderPills(Config{App: "A", Version: "1"}, 10)
	if len(p) == 0 {
		t.Fatalf("expected pills string")
	}
}

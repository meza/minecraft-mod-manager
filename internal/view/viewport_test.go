package view

import "testing"

func TestClampViewportHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
		want   int
	}{
		{name: "negative", height: -1, want: 0},
		{name: "zero", height: 0, want: 0},
		{name: "positive", height: 5, want: 5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ClampViewportHeight(test.height); got != test.want {
				t.Fatalf("ClampViewportHeight(%d) = %d, want %d", test.height, got, test.want)
			}
		})
	}
}

func TestViewportHeightOrContent(t *testing.T) {
	tests := []struct {
		name          string
		windowHeight  int
		contentHeight int
		want          int
	}{
		{name: "window height", windowHeight: 8, contentHeight: 2, want: 8},
		{name: "content height when window missing", windowHeight: 0, contentHeight: 3, want: 3},
		{name: "content height when window negative", windowHeight: -2, contentHeight: 4, want: 4},
		{name: "content height clamp", windowHeight: 0, contentHeight: -1, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ViewportHeightOrContent(test.windowHeight, test.contentHeight); got != test.want {
				t.Fatalf("ViewportHeightOrContent(%d, %d) = %d, want %d", test.windowHeight, test.contentHeight, got, test.want)
			}
		})
	}
}

func TestMaxViewportOffset(t *testing.T) {
	tests := []struct {
		name           string
		contentHeight  int
		viewportHeight int
		want           int
	}{
		{name: "content taller than viewport", contentHeight: 6, viewportHeight: 4, want: 2},
		{name: "content shorter than viewport", contentHeight: 2, viewportHeight: 5, want: 0},
		{name: "content height negative", contentHeight: -1, viewportHeight: 3, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MaxViewportOffset(test.contentHeight, test.viewportHeight); got != test.want {
				t.Fatalf("MaxViewportOffset(%d, %d) = %d, want %d", test.contentHeight, test.viewportHeight, got, test.want)
			}
		})
	}
}

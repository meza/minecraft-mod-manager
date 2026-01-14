package view

// ClampViewportHeight normalizes the viewport height to be non-negative.
func ClampViewportHeight(height int) int {
	if height < 0 {
		return 0
	}
	return height
}

// ViewportHeightOrContent returns the window height when available, otherwise the content height.
func ViewportHeightOrContent(windowHeight int, contentHeight int) int {
	if windowHeight > 0 {
		return windowHeight
	}
	return ClampViewportHeight(contentHeight)
}

// MaxViewportOffset returns the maximum scroll offset for the given heights, clamped to non-negative values.
func MaxViewportOffset(contentHeight int, viewportHeight int) int {
	offset := contentHeight - viewportHeight
	if offset < 0 {
		return 0
	}
	return offset
}

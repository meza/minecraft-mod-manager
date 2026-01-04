package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderIfColorEnabledUsesStyleWhenEnabled(t *testing.T) {
	value := "example"

	assert.Equal(t, TitleStyle.Render(value), RenderIfColorEnabled(ColorEnabled, TitleStyle, value))
}

func TestRenderIfColorEnabledReturnsValueWhenDisabled(t *testing.T) {
	value := "example"

	assert.Equal(t, value, RenderIfColorEnabled(ColorDisabled, TitleStyle, value))
}

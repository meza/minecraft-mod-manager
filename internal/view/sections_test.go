package view

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderViewSectionsJoinsWithSeparator(t *testing.T) {
	result := RenderViewSections([]string{"one", "two"}, SectionSeparatorParagraph)
	assert.Equal(t, "one\n\ntwo", result)
}

func TestRenderViewSectionsSkipsEmptySections(t *testing.T) {
	result := RenderViewSections([]string{"one", "", "two"}, SectionSeparatorParagraph)
	assert.Equal(t, "one\n\ntwo", result)
}

func TestRenderViewSectionsReturnsEmptyOnWriteError(t *testing.T) {
	restore := WriteString
	t.Cleanup(func() { WriteString = restore })
	WriteString = func(io.Writer, string) error {
		return errors.New("write failed")
	}

	assert.Equal(t, "", RenderViewSections([]string{"one"}, SectionSeparatorParagraph))
}

func TestRenderViewSectionsBuilderReturnsFalseOnWriteError(t *testing.T) {
	restore := WriteString
	t.Cleanup(func() { WriteString = restore })
	WriteString = func(io.Writer, string) error {
		return errors.New("write failed")
	}

	builder, ok := RenderViewSectionsBuilder([]string{"one"}, SectionSeparatorParagraph)
	assert.False(t, ok)
	assert.NotNil(t, builder)
}

func TestRenderViewSectionsBuilderReturnsFalseOnSeparatorWriteError(t *testing.T) {
	restore := WriteString
	t.Cleanup(func() { WriteString = restore })
	callCount := 0
	WriteString = func(writer io.Writer, value string) error {
		callCount++
		if callCount == 2 {
			return errors.New("write failed")
		}
		builder, ok := writer.(*strings.Builder)
		if !ok {
			return errors.New("unexpected writer")
		}
		_, err := builder.WriteString(value)
		return err
	}

	builder, ok := RenderViewSectionsBuilder([]string{"one", "two"}, SectionSeparatorParagraph)
	assert.False(t, ok)
	assert.NotNil(t, builder)
}

func TestRenderViewSectionsWithTrailingNewlineAddsNewline(t *testing.T) {
	result := RenderViewSectionsWithTrailingNewline([]string{"one"}, SectionSeparatorLine)
	assert.Equal(t, "one\n", result)
}

func TestRenderViewSectionsWithTrailingNewlineEmptySections(t *testing.T) {
	result := RenderViewSectionsWithTrailingNewline([]string{}, SectionSeparatorLine)
	assert.Equal(t, "", result)
}

func TestRenderViewSectionsWithTrailingNewlineReturnsEmptyOnWriteError(t *testing.T) {
	restore := WriteString
	t.Cleanup(func() { WriteString = restore })
	WriteString = func(writer io.Writer, value string) error {
		if value == "\n" {
			return errors.New("write failed")
		}
		builder, ok := writer.(*strings.Builder)
		if !ok {
			return errors.New("unexpected writer")
		}
		_, err := builder.WriteString(value)
		return err
	}

	assert.Equal(t, "", RenderViewSectionsWithTrailingNewline([]string{"one"}, SectionSeparatorLine))
}

func TestRenderViewSectionsWithTrailingNewlineReturnsEmptyOnBuilderError(t *testing.T) {
	restore := WriteString
	t.Cleanup(func() { WriteString = restore })
	WriteString = func(io.Writer, string) error {
		return errors.New("write failed")
	}

	assert.Equal(t, "", RenderViewSectionsWithTrailingNewline([]string{"one"}, SectionSeparatorLine))
}

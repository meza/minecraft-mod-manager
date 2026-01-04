package view

import (
	"io"
	"strings"
)

const (
	SectionSeparatorLine      = "\n"
	SectionSeparatorParagraph = "\n\n"
)

// WriteString is overridable for tests that need to force builder errors.
var WriteString = func(writer io.Writer, value string) error {
	_, err := writer.Write([]byte(value))
	return err
}

// RenderViewSections joins non-empty sections with the provided separator.
func RenderViewSections(sections []string, separator string) string {
	stringBuilder, ok := RenderViewSectionsBuilder(sections, separator)
	if !ok {
		return ""
	}
	return stringBuilder.String()
}

// RenderViewSectionsWithTrailingNewline joins sections with the provided separator and appends a trailing newline.
func RenderViewSectionsWithTrailingNewline(sections []string, separator string) string {
	stringBuilder, ok := RenderViewSectionsBuilder(sections, separator)
	if !ok {
		return ""
	}
	if stringBuilder.Len() > 0 {
		if err := WriteString(stringBuilder, "\n"); err != nil {
			return ""
		}
	}
	return stringBuilder.String()
}

// RenderViewSectionsBuilder returns a builder containing the combined sections.
func RenderViewSectionsBuilder(sections []string, separator string) (*strings.Builder, bool) {
	stringBuilder := &strings.Builder{}

	for _, section := range sections {
		if section == "" {
			continue
		}
		if stringBuilder.Len() > 0 {
			if err := WriteString(stringBuilder, separator); err != nil {
				return stringBuilder, false
			}
		}
		if err := WriteString(stringBuilder, section); err != nil {
			return stringBuilder, false
		}
	}

	return stringBuilder, true
}

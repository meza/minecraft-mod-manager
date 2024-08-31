package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransformJson(t *testing.T) {
	t.Run("Basics", func(t *testing.T) {
		input := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}

		expected := map[string]interface{}{
			"key1": map[string]interface{}{
				"other": "value1",
			},
			"key2": map[string]interface{}{
				"other": "value2",
			},
		}

		output, err := transformJson(input)

		assert.NoError(t, err)
		assert.Equal(t, expected, output)
	})

	t.Run("Plurals", func(t *testing.T) {
		input := map[string]interface{}{
			"key1":      "defaultValue1",
			"key1.one":  "oneValue1",
			"key2":      "defaultValue2",
			"key2.zero": "zeroValue2",
			"key2.few":  "fewValue2",
			"key2.two":  "twoValue2",
			"key3.many": "manyValue3",
		}

		expected := map[string]interface{}{
			"key1": map[string]interface{}{
				"other": "defaultValue1",
				"one":   "oneValue1",
			},
			"key2": map[string]interface{}{
				"other": "defaultValue2",
				"zero":  "zeroValue2",
				"few":   "fewValue2",
				"two":   "twoValue2",
			},
			"key3": map[string]interface{}{
				"many": "manyValue3",
			},
		}

		output, err := transformJson(input)

		assert.NoError(t, err)
		assert.Equal(t, expected, output)
	})
}

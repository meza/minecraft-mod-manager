//go:build !e2e

package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseBinaryIgnoresTestMode(t *testing.T) {
	enFS = testData
	langDir = "__fixtures__"
	ResetForTesting()

	originalTestBinaryCheck := testBinaryCheck
	t.Cleanup(func() { testBinaryCheck = originalTestBinaryCheck })
	testBinaryCheck = func() bool { return false }
	t.Setenv("MMM_TEST", "true")

	actual := T("test.simple", nil)

	assert.Equal(t, "Hello World", actual)
}

//go:build e2e

package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestE2EBinaryHonorsTestMode(t *testing.T) {
	originalTestBinaryCheck := testBinaryCheck
	t.Cleanup(func() { testBinaryCheck = originalTestBinaryCheck })
	testBinaryCheck = func() bool { return false }
	t.Setenv("MMM_TEST", "true")

	actual := T("test.multiple", &Tvars{
		Count: 2,
		Data:  &TData{"injectedData": "from-e2e"},
	})

	assert.Equal(t, "test.multiple, Arg 1: {Count: 2, Data: &map[injectedData:from-e2e]}", actual)
}

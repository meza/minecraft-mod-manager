package change

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveForcePolicy(t *testing.T) {
	policy, err := resolveForcePolicy(changeForcePolicyFlags{force: true})
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyUnset, policy)

	policy, err = resolveForcePolicy(changeForcePolicyFlags{force: true, keepConfig: true})
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyKeepConfig, policy)

	policy, err = resolveForcePolicy(changeForcePolicyFlags{force: true, pruneConfig: true})
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyPruneConfig, policy)

	policy, err = resolveForcePolicy(changeForcePolicyFlags{force: true, disableSkipped: true})
	assert.NoError(t, err)
	assert.Equal(t, changeForcePolicyDisableSkipped, policy)

	_, err = resolveForcePolicy(changeForcePolicyFlags{force: true, keepConfig: true, pruneConfig: true})
	assert.Error(t, err)

	_, err = resolveForcePolicy(changeForcePolicyFlags{force: false, keepConfig: true})
	assert.Error(t, err)
}

func TestSkippedSectionHeader(t *testing.T) {
	assert.Equal(t, "cmd.change.skipped.header", skippedSectionHeader(changeForcePolicyUnset))
	assert.Equal(t, "cmd.change.pruned.header", skippedSectionHeader(changeForcePolicyPruneConfig))
	assert.Equal(t, "cmd.change.disabled.header", skippedSectionHeader(changeForcePolicyDisableSkipped))
}

func TestSkippedItemSuffixKey(t *testing.T) {
	assert.Equal(t, "cmd.change.item.unsupported_skipped", skippedItemSuffixKey(changeForcePolicyUnset))
	assert.Equal(t, "cmd.change.item.unsupported", skippedItemSuffixKey(changeForcePolicyPruneConfig))
	assert.Equal(t, "cmd.change.item.unsupported", skippedItemSuffixKey(changeForcePolicyDisableSkipped))
}

func TestHasSkippedItems(t *testing.T) {
	assert.False(t, hasSkippedItems([]changeItem{{DisplayName: "Alpha"}}))
	assert.True(t, hasSkippedItems([]changeItem{{DisplayName: "Alpha", Skipped: true}}))
}

func TestChangePolicyFlagErrorMessage(t *testing.T) {
	assert.Equal(t, "multiple policy flags provided", changePolicyFlagError{kind: changePolicyFlagErrorMultiple}.Error())
	assert.Equal(t, "policy flags require --force", changePolicyFlagError{kind: changePolicyFlagErrorRequiresForce}.Error())
}

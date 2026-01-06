package change

type changeForcePolicy int

const (
	changeForcePolicyUnset changeForcePolicy = iota
	changeForcePolicyKeepConfig
	changeForcePolicyPruneConfig
	changeForcePolicyDisableSkipped
)

type changePolicyFlagErrorKind int

const (
	changePolicyFlagErrorMultiple changePolicyFlagErrorKind = iota
	changePolicyFlagErrorRequiresForce
)

type changePolicyFlagError struct {
	kind changePolicyFlagErrorKind
}

type changeForcePolicyFlags struct {
	force          bool
	keepConfig     bool
	pruneConfig    bool
	disableSkipped bool
}

func (err changePolicyFlagError) Error() string {
	switch err.kind {
	case changePolicyFlagErrorRequiresForce:
		return "policy flags require --force"
	default:
		return "multiple policy flags provided"
	}
}

func resolveForcePolicy(flags changeForcePolicyFlags) (changeForcePolicy, error) {
	policies := 0
	if flags.keepConfig {
		policies++
	}
	if flags.pruneConfig {
		policies++
	}
	if flags.disableSkipped {
		policies++
	}
	if policies > 1 {
		return changeForcePolicyUnset, changePolicyFlagError{kind: changePolicyFlagErrorMultiple}
	}
	if policies == 1 && !flags.force {
		return changeForcePolicyUnset, changePolicyFlagError{kind: changePolicyFlagErrorRequiresForce}
	}
	if flags.keepConfig {
		return changeForcePolicyKeepConfig, nil
	}
	if flags.pruneConfig {
		return changeForcePolicyPruneConfig, nil
	}
	if flags.disableSkipped {
		return changeForcePolicyDisableSkipped, nil
	}
	return changeForcePolicyUnset, nil
}

func skippedSectionHeader(policy changeForcePolicy) string {
	switch policy {
	case changeForcePolicyPruneConfig:
		return "cmd.change.pruned.header"
	case changeForcePolicyDisableSkipped:
		return "cmd.change.disabled.header"
	default:
		return "cmd.change.skipped.header"
	}
}

func skippedItemSuffixKey(policy changeForcePolicy) string {
	switch policy {
	case changeForcePolicyPruneConfig, changeForcePolicyDisableSkipped:
		return "cmd.change.item.unsupported"
	default:
		return "cmd.change.item.unsupported_skipped"
	}
}

func hasSkippedItems(items []changeItem) bool {
	for _, item := range items {
		if item.Skipped {
			return true
		}
	}
	return false
}

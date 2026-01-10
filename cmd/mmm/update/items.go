package update

func cloneUpdateItems(items []updateItem) []updateItem {
	if len(items) == 0 {
		return nil
	}
	clone := make([]updateItem, len(items))
	copy(clone, items)
	return clone
}

func isTerminalUpdateStatus(status updateItemStatus) bool {
	switch status {
	case updateItemStatusUpToDate, updateItemStatusUpdated, updateItemStatusSkipped, updateItemStatusFailed:
		return true
	default:
		return false
	}
}

func hasTerminalItems(items []updateItem) bool {
	for _, item := range items {
		if isTerminalUpdateStatus(item.Status) {
			return true
		}
	}
	return false
}

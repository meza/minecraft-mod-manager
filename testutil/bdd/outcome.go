package bdd

// Outcome captures an observable result from running an action.
type Outcome struct {
	Name    string
	Details string
}

// NewOutcome returns an Outcome with the provided name and details.
func NewOutcome(name string, details string) Outcome {
	return Outcome{
		Name:    name,
		Details: details,
	}
}

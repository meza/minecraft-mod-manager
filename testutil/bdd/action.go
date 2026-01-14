package bdd

// Action encapsulates a domain operation that can be executed by any actor.
// Implementations must remain stateless so the same action can be reused across actors.
type Action interface {
	Execute(actor *Actor) Outcome
	Copy() Action
}

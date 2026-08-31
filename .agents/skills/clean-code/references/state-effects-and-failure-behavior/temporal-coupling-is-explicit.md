# Temporal coupling is explicit

If operations must happen in a particular order, that requirement is encoded or enforced rather than socially remembered.

Whenever order matters, the code should make that dependency obvious through the API, control structure, transaction, or invariant. Requirements that exist only in comments or team memory are fragile. In review, this is about whether the code preserves truth under bad input, surprising state, and real failure conditions rather than only under ideal conditions. Strong signals are explicit contracts, visible failure behavior, and boundaries that stop invalid or unsafe state from spreading. Weak signals are silent fallback, ambiguous error handling, implicit ordering requirements, and cleanup that only works on the happy path. The educational point is that correctness depends as much on what happens when things go wrong as on what happens when they go right. For this specific symptom, the reviewer should ask whether the change makes 'Temporal coupling is explicit' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
post = Post("Temporal coupling")
post.addText("Encode required call order.")
post.requestReview()
post.approve()
post.publish()
```

### Good

```text
draft = DraftPost("Temporal coupling")
draft.addText("Encode required call order.")
review = draft.requestReview()
approved = review.approve()
published = approved.publish()
```

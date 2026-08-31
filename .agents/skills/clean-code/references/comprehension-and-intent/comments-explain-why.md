# Comments explain why

Comments justify intent, tradeoffs, or non-obvious constraints rather than narrating what the code already says.

Useful comments preserve intent that the code itself cannot show, such as why a tradeoff exists, what external constraint shaped the solution, or which assumption must remain true. They should not duplicate the visible control flow. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Comments explain why' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
// Wait five seconds before retrying.
wait(5 seconds)
retry request
```

### Good

```text
// The provider rejects retries made within five seconds of a rate-limit response.
wait(5 seconds)
retry request
```

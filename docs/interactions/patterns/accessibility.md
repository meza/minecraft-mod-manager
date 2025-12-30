# Accessibility and inclusivity constraints

This document records interaction-level constraints we need to keep true as the CLI and TUI evolve.

## Keyboard only

All interactive flows must be usable with keyboard only.
Key bindings must be discoverable.

## Color independence

Do not rely on color alone to convey meaning.
Provide text or icon equivalents and ensure non-color output remains clear.

## Plain language

Error messages and prompts should:
- Say what happened
- Say what to do next
- Avoid jargon when a clearer term exists

## Safe defaults

Default answers in destructive actions should prefer safety.
Users should not lose data due to an accidental keypress or Enter.

## Screen reader friendly structure

Even in terminals, structure matters:
- Prefer stable headings in docs
- Prefer consistent prompt patterns
- Avoid dense decorative output


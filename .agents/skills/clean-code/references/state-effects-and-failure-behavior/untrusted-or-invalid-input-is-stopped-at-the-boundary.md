# Untrusted or invalid input is stopped at the boundary

External input is validated, normalized, sanitized, and rejected early enough that invalid or unsafe state does not spread through the system.

Systems should become stricter as data moves inward, not looser. The safest and most maintainable pattern is to validate and normalize external input at the boundary, apply any context-appropriate sanitization before it reaches sensitive destinations, and fail explicitly when input violates assumptions or cannot be trusted. This combines correctness and security into one review lens because both are about preventing bad state from spreading.

Strong signs include explicit boundary checks, parsing and normalization near entry points, destination-aware sanitization or escaping, and failures that make invalid assumptions visible instead of silently tolerating them. Weak signs include raw external values flowing deep into business logic, sanitization deferred until late or applied inconsistently, impossible states represented as ordinary values, and quiet continuation after invalid input in the hope that later code will cope.

This symptom matters because once malformed or unsafe input leaks into the trusted core, every downstream unit has to defend itself and reviewers lose confidence in system truth. Early boundary enforcement reduces bug surface, simplifies reasoning, and lowers security risk. The code becomes easier to review because the trusted and untrusted parts of the system are separated more clearly.

## Examples

### Bad

```text
function register(request):
  user = users.create(request.body.email, request.body.displayName)
  mailer.sendWelcome(request.body.email)
  audit.record("registered", request.body)
  return user
```

### Good

```text
function register(request):
  input = RegistrationInput.validateAndNormalize(request.body)
  if input.invalid: return BadRequest(input.errors)
  return registerTrusted(input.value)
function registerTrusted(input):
  user = users.create(input.email, input.displayName)
  mailer.sendWelcome(input.email)
  audit.record("registered", input)
  return user
```

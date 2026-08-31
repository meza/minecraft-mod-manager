# Hidden state is avoided

A unit does not rely on ambient mutable state, invisible caches, implicit globals, or surprising external mutation.

Hidden state creates non-local behavior and makes debugging much harder. If correctness depends on caches, globals, ambient context, or invisible mutation, the reader loses the ability to reason from the code in front of them. In review, this is about how much interpretation work the reader must do before they can trust what the code is trying to accomplish. Strong signals are names, structure, and local flow that let another engineer build a correct mental model quickly. Weak signals are vague labels, mixed levels of abstraction, hidden assumptions, or a need to chase many references before the unit makes sense. The educational point is that readability is not cosmetic; it is what makes future change, debugging, and review safe. For this specific symptom, the reviewer should ask whether the change makes 'Hidden state is avoided' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
global current_user
global audit_log
function approve(request):
    request.status = "approved" if current_user.can_approve
    audit_log.append(request.id)
```

### Good

```text
function approve(request, actor, audit_log):
    updated_log = audit_log.with_entry(request.id)
    if not actor.can_approve:
        return (request, updated_log)
    approved = request.with_status("approved")
    return (approved, updated_log)
```

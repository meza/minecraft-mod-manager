# Accessibility is considered in user-facing paths

User-visible outputs, flows, and interfaces do not casually exclude people through avoidable design choices.

Accessibility review checks whether user-facing behavior excludes people through avoidable assumptions about perception, input method, timing, wording, or semantics. It is part of product quality, not just visual polish. In review, this is about whether quality is defined broadly enough to include the people who must use, read, or depend on the system. Strong signals are wording and behavior that are precise, respectful, and accessible without unnecessary assumptions. Weak signals are exclusionary terminology, user-facing flows that presume one kind of user, or semantics that make correct use harder for part of the audience. The educational point is that maintainable systems do not externalize avoidable difficulty onto users. For this specific symptom, the reviewer should ask whether the change makes 'Accessibility is considered in user-facing paths' easier to see and rely on, or whether it makes the surrounding code more ambiguous. A good detail line here should help a future reviewer explain not only what this symptom means, but also why its absence raises maintenance cost, defect risk, or review uncertainty.

## Examples

### Bad

```text
render <div class="save-icon" onClick=saveDraft>
  💾
</div>
```

### Good

```text
render <button type="button" onClick=saveDraft>
  Save draft
</button>
```

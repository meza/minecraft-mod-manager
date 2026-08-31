# Security privacy and user harm

Trust boundaries, privileges, secrets, data exposure, auditability, defaults, and user-facing language protect the people the system touches.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Sensitive data is protected in telemetry](security-privacy-and-user-harm/sensitive-data-is-protected-in-telemetry.md) - Logs, traces, and messages do not leak secrets, private data, or unsafe internal detail.
- [Security boundaries are explicit](security-privacy-and-user-harm/security-boundaries-are-explicit.md) - Trust levels, privilege transitions, and sensitive operations are visible in the design and code paths.
- [Least privilege is followed](security-privacy-and-user-harm/least-privilege-is-followed.md) - Code, services, and components operate with the minimum permissions they need.
- [Secrets are not embedded](security-privacy-and-user-harm/secrets-are-not-embedded.md) - Credentials, tokens, keys, and sensitive configuration are not hard-coded into source or artifacts.
- [Data exposure is minimized](security-privacy-and-user-harm/data-exposure-is-minimized.md) - The system only collects, stores, returns, and retains what is needed for its legitimate purpose.
- [Sensitive actions are auditable](security-privacy-and-user-harm/sensitive-actions-are-auditable.md) - Important security or privacy-relevant events can be traced after the fact.
- [Unsafe defaults are avoided](security-privacy-and-user-harm/unsafe-defaults-are-avoided.md) - The easiest path through the code does not accidentally produce insecure, destructive, or privacy-harming behavior.
- [Accessibility is considered in user-facing paths](security-privacy-and-user-harm/accessibility-is-considered-in-user-facing-paths.md) - User-visible outputs, flows, and interfaces do not casually exclude people through avoidable design choices.
- [Inclusive language is used](security-privacy-and-user-harm/inclusive-language-is-used.md) - Naming, docs, messages, and user-facing outputs avoid unnecessary exclusionary or misleading language.


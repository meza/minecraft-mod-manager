# Optional ticket concern scan

Use this scan only when the work has cross-cutting, user-facing, security, operational, or rollout implications. Add only material answers to the ticket. Omit concerns that do not apply.

## Product and experience

- Is required content or design available, owned, and maintained through a known source?
- Are user flows, empty states, validation failures, and recovery behavior defined?
- Do accessibility, keyboard, screen-reader, responsive, localization, or design-system requirements change?
- Does the change introduce a URL, discoverability, or search-engine requirement?

## Data and security

- Does the work introduce secrets, permissions, personal data, dependencies, or new trust boundaries?
- Are authentication, authorization, storage, transmission, retention, and regulatory constraints explicit?
- Is a migration, compatibility strategy, or data recovery path required?

## Delivery and operation

- Is a feature flag, staged rollout, or explicit rollback needed to limit blast radius?
- Could the change increase external request volume or weaken shared infrastructure safeguards?
- Are new logs, metrics, traces, alerts, dashboards, runbooks, service objectives, or performance budgets needed?
- Who responds when the new behavior fails, and what evidence will they have?

## Verification and maintenance

- Do acceptance scenarios cover success, empty results, validation failures, recoverable errors, and fatal errors where applicable?
- Are snapshot, integration, security, performance, or regression checks required by repository policy?
- Which documentation, API descriptions, architecture diagrams, or operator guidance must stay synchronized?
- Who owns the capability and its supporting content after delivery?

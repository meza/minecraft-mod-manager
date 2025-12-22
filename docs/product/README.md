# Product Artifacts

This directory holds product artifacts that don't fit elsewhere. Most product work belongs in:

- **Issue tracker** — Requirements, success criteria, bugs, tasks
- **`docs/specs/`** — Detailed behavioral specifications
- **`docs/commands/`** — User-facing documentation
- **`doc/adr/`** — Decisions needing attribution and rationale

Add to this directory sparingly. If something can live in the issue tracker or existing documentation, it should.

## When to Use This Directory

Valid uses:
- Roadmaps spanning multiple releases
- Open questions awaiting stakeholder resolution
- PRDs for larger features that span multiple issues

Invalid uses:
- Anything that should be a ticket
- Specifications (use `docs/specs/`)
- Decision records (use `doc/adr/`)
- User documentation (use `docs/commands/`)

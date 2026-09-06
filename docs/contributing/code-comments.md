# Code comment policy

## Write comments only when they protect understanding

Prefer clear names, types, and structure over comments. Add a comment only when it records a reason
the code cannot express, warns about a non-obvious hazard, identifies a value fixed outside the
system, or links to an upstream defect.

Do not use comments for deferred work, ticket keys or other internal issue references, ADR numbers
or pointers to ADRs, review-feedback notes, plan echoes, commented-out code, authorship, change
history, banners, or narration of ordinary code. Before
keeping any comment, apply one test: would a reader who never saw the authoring conversation,
ticket, or review still need it? If not, delete it.

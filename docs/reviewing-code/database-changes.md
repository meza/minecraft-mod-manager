# Database changes

Review PostgreSQL changes against the [database contribution guide](../contributing/database.md)
and the guidance it selects for the affected concern. Existing migrations and neighbouring SQL
describe the current state but are not precedent for new work.

## Trace the database boundary

Follow the change from the production migration through the resulting schema, constraints, grants,
policies, application query shape, and relevant test evidence. Confirm that responsibilities remain
with their documented owner: the database owns structure, relational integrity, row-level access
control, and mechanical timestamp maintenance; application code owns product behaviour and
orchestration.

For migration changes, verify rollout and verification against the
[migration guidance](../contributing/database/migrations.md). For runtime queries, verify binding,
native result types, bounded reads, and changed access paths against the
[runtime SQL guidance](../contributing/database/runtime-sql.md).

Reject any new or changed SQL that constructs a JSON transport result from otherwise
typed values, rows or collections. A query-executor limitation and neighbouring JSON-returning SQL
do not justify preserving or copying the pattern. Require native PostgreSQL results and application-
owned transport mapping before approval.

## Review the authorization gate

Read the [RLS contribution guidance](../contributing/database/rls.md) and the
[RLS harness guide](../../apps/web/mig/rls/README.md) when a migration changes a table,
relationship, constraint, grant, policy, or probe row shape.

Every production migration is applied automatically before the RLS fixtures. There is no separate
migration list to update. The RLS gate then loads its own current-schema dataset; it does not load
the golden dataset used by API-contract and end-to-end tests. Golden seed changes therefore do not
provide RLS coverage and must not determine RLS fixture cardinalities.

Verify that the contribution reconciles every affected part of the gate:

- the minimum relational graph under `apps/web/mig/rls/seed/`;
- shared fixture identifiers and topology preflight assertions;
- affected table matrices, RPC probes, write probes, and catalog assertions; and
- positive access and populated cross-boundary denials for the applicable personas.

Treat case declarations as the authorization contract. An expectation change requires an explicit
product-authorization decision. Fixture cardinalities may change only when the smaller or larger
graph preserves the authorization and boundary claim. A denial must reach the intended RLS or grant
layer rather than pass because of an unrelated constraint, empty target, or missing privilege.

A new protected table is incomplete without positive access, cross-boundary denial, applicable
write behaviour, complete policy inventory, and assertions that both RLS and forced RLS remain
enabled. If current-schema data exposes an unexplained mismatch, require it to be reported as a
probable production-policy defect rather than skipped, weakened, or rebaselined.

## Use the right evidence

The required automated gate is `pnpm run test:rls`, which runs on PGlite through Turbo. During a
pull request review, use its CI result according to the
[pull request review guidance](./pull-requests.md) rather than rerunning CI-owned validation.

Docker parity is a manual diagnostic. Request it when PostgreSQL-version behaviour, extensions,
catalog behaviour, or driver parity is material to the change; do not make it routine evidence for
every database contribution.

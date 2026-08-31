# Operability and production behavior

The system can be configured, observed, scaled, migrated, and released by people who did not write it, and its signals reflect what is really happening.

Use the descriptions below to decide which individual quality references are relevant. Open only the references that can materially affect the current decision, code, or failure.

- [Configuration is explicit, externalized, and validated](operability-and-production-behavior/configuration-is-explicit-externalized-and-validated.md) - Environment-dependent behavior is configured outside core logic, read in controlled places, and checked early enough that the system does not run on broken assumptions.
- [Performance and scale behavior are intentional](operability-and-production-behavior/performance-and-scale-behavior-are-intentional.md) - Load handling, hot paths, optimization choices, and scalability are shaped deliberately so the system continues to behave predictably as demand grows without spreading complexity unnecessarily.
- [Logs are structured and useful](operability-and-production-behavior/logs-are-structured-and-useful.md) - Production logs carry useful context and machine-readable shape instead of noise or free-form guessing text.
- [Diagnostics preserve signal](operability-and-production-behavior/diagnostics-preserve-signal.md) - Errors, traces, and metrics contain enough information to diagnose failures without drowning operators in clutter.
- [Health signals reflect real health](operability-and-production-behavior/health-signals-reflect-real-health.md) - Health endpoints and readiness checks reflect whether the service can actually perform its role.
- [Observability matches critical paths](operability-and-production-behavior/observability-matches-critical-paths.md) - The system emits signals around the behaviors that matter most to users and operators.
- [Data migrations are survivable](operability-and-production-behavior/data-migrations-are-survivable.md) - Schema and data evolution are designed to happen safely across versions and deployments.
- [Versioning and compatibility are understood](operability-and-production-behavior/versioning-and-compatibility-are-understood.md) - The system handles evolving contracts, clients, data shapes, and deployment skew with intention.
- [Delivery mechanics and quality support the intended architecture](operability-and-production-behavior/delivery-mechanics-and-quality-support-the-intended-architecture.md) - Build and delivery behavior is reproducible, quality gates check meaningful risks, and tooling reinforces the architectural shape instead of undermining it.


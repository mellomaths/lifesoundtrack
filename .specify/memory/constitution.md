<!--
Sync Impact Report
Version: 1.2.1 → 1.3.0
Modified principles: I (Code quality) — added linting standards (config, CI, suppressions, rule changes)
Added sections: (none)
Removed sections: (none)
Templates: .specify/templates/plan-template.md — ✅ updated (Constitution Check)
Templates: .specify/templates/spec-template.md — ✅ updated (NFR lint callout)
Templates: .specify/templates/tasks-template.md — unchanged (T003 already covers lint)
Runtime: README.md — unchanged
Follow-up TODOs: none
-->

# lifesoundtrack Constitution

## Core Principles

### I. Code Quality

Code MUST be readable, reviewable, and consistent with the language and project conventions. Each change MUST follow the project’s **formatting and style** rules, **static analysis** where it is in the agreed toolchain, and idiom-appropriate structure for that stack. Changes MUST be scoped and avoid unrelated refactors in the same change set. Rationale: readable, conventional code and small changes make review and maintenance predictable.

**Linting** — When the project uses linters, their **tooling, default rule set, and configuration files** MUST be **kept in version control** and be the same ones used in **CI** when CI enforces lint. The commands to run linters locally SHOULD be discoverable (e.g. in `README`, `Makefile`, or `package` scripts) so contributors can fix issues before review. A change MUST **not** add **unjustified** new suppressions, global disables, or rule downgrades; where a one-off suppression is unavoidable, it MUST be **as narrow as the fix allows** and **include a brief comment** (or link) explaining why. Broad rule changes or bypasses MUST be recorded in the pull request and, if they weaken the project baseline, in **Complexity Tracking** (or the same plan’s exceptions) with rationale. Rationale: shared, versioned lint config and reviewable exceptions keep style and safety rules consistent; uncontrolled waivers erode the benefit of automation.

### II. REST API Standards

When the project exposes HTTP or REST-style APIs, those surfaces MUST use consistent resource-oriented design, the correct HTTP methods for the operation, and HTTP status codes that match the outcome. Request and response bodies MUST be validated; errors MUST be represented in a single, documented shape (e.g. JSON) so clients can handle failures predictably. Breaking changes to public API contracts MUST be versioned, feature-flagged, or otherwise migrated with a clear plan. Rationale: predictable HTTP semantics and contracts reduce client bugs and support burden. If a feature has no HTTP surface, this principle applies only when one is added.

### III. Testing Standards

Tests MUST protect critical behavior and regression-prone areas. New or changed behavior SHOULD be covered at the appropriate level: unit tests for pure logic, integration tests for boundaries (database, HTTP, external services) where the feature risk warrants it. The codebase SHOULD use **parametric, table-based, or otherwise repeatable** test patterns that fit the language and improve clarity. Tests MUST run in CI when CI exists, or the plan MUST document the manual gate. Rationale: explicit testing discipline keeps releases safe as the system grows.

### IV. User Experience Consistency

User-facing text and interaction patterns for each product channel (e.g. web, CLI, or messaging) MUST be consistent in tone, structure, and naming so users can form a reliable mental model. Terminology and entry points MUST not duplicate the same action under confusing aliases without documentation. Rationale: consistent UX reduces confusion and support load. This applies when a product surface exists; pure tooling in this repository without an end user follows **IX** for accuracy only.

### V. Monitoring

Production and staging deployments MUST expose enough observability to detect failure and misuse: at minimum, a health or liveness path where applicable, and key metrics (e.g. error rates, latency, business-relevant counts) that the team agrees to track for the workload. Rationale: without defined monitoring signals, incidents are discovered late and root cause takes longer. Where no deployed service exists, plan work against future surfaces without claiming production today.

### VI. Logging

Logging MUST be sufficient to diagnose production issues: clear messages, appropriate levels, and a structure or format that is parseable in your log pipeline when one exists. Log entries for a single request or operation SHOULD be correlatable (e.g. request ID, user or session ID where policy allows) when more than one log line is involved. Secrets, tokens, and passwords MUST NOT appear in logs. Rationale: good logs make incidents actionable without guessing; log hygiene protects users and the system.

### VII. Performance Requirements

Features with latency, throughput, or resource sensitivity MUST state explicit targets or limits in the plan (e.g. p95 latency, memory ceiling, or “N/A with justification” for non–performance-critical work). Work that risks hot paths MUST be profiled or measured when complexity or load grows. Rationale: unstated performance assumptions lead to late rework; stated budgets enable tradeoffs during design and review.

### VIII. Containerization and local development

When the repository **contains** first-party **runnable** artifacts (applications, workers, or long-running services), each such artifact MUST include a **Dockerfile** that builds and runs it in a documented way, and the repository MUST maintain **Docker Compose** (or an equivalent, versioned compose file) that assembles those services and their local dependencies for **local development and debugging** (start commands, ports, env files, logs, debugger attachment where applicable). Introducing or materially changing a runnable component MUST include updates to image definitions and the compose graph. Before any runnable exists, this principle is **latent**—it applies at the time such components are added. Rationale: containerized, compose-driven local workflows reduce drift; principles must not claim containers that the tree does not yet ship.

### IX. Accurate documentation and decommissioning

The **Technology alignment** section, root **README**, and other user-facing or contributor-facing documentation MUST match what is **actually present** on the default branch. Removal of code, product specs, or deployment assets MUST be followed by **updating** this constitution, templates, and README in the same change (or a follow-up plan item with owner) so nothing continues to point at deleted paths or retired behavior. Rationale: stale documentation voids reviews and misleads automation; cleanup is part of delivery.

## Technology alignment

The repository is built around **Spec Kit** (`.specify/`: templates, memory, workflows) and **Cursor** project rules and skills (`.cursor/`). Feature implementation and product documentation (for example a service module, `spec/` command reference, and operator `README` sections) are **added and maintained as the product is developed**; do not assume a fixed on-disk layout beyond what is committed.

**Languages, frameworks, and API implementation styles** MUST follow the repository’s **Cursor** rules in `.cursor/rules/` for whatever stack is in the tree, when those rules are present. For HTTP APIs, follow **II**. When runnables and local dependencies exist, follow **VIII** for Docker and Compose. Governance and version history: this file.

## Specification and delivery

Substantive features SHOULD be specified and planned with the project’s Spec Kit workflow: specification, implementation plan, and tasks, so that work is traceable and reviewable. `plan.md` MUST satisfy the **Constitution Check** gates before heavy implementation begins. If delivery must deviate from a principle, the plan MUST record the exception in **Complexity Tracking** (or equivalent) with rationale. Large removals or restructures MUST trigger **IX** in the same effort.

## Governance

This constitution is the source of truth for project-wide quality and delivery expectations. It supersedes ad-hoc practices when they conflict. Amendments require updating this document, bumping the version per semantic rules below, and keeping dependent templates in sync. Reviewers MUST treat constitution compliance as part of design and code review. Version **MAJOR** for incompatible changes or removed principles; **MINOR** for new principles or materially new guidance; **PATCH** for clarifications and non-semantic refinements.

**Version**: 1.3.0 | **Ratified**: 2026-04-23 | **Last Amended**: 2026-04-23

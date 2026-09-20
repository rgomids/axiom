# Tasks — Specification 003: E2E Codex POC

## Status

Approved and implementation-authorized on 2026-09-19. T31–T39 are delivered in
dedicated PRs to `poc/e2e-axiom-codex`; T40 is the final technical reconciliation.
Human acceptance and final merge remain separate gates.

## Dependency graph

```text
T31
 +-> T32
 +-> T33
 +-> T34 -> T35
          -> T36 -> T37 -> T38 -> T39 -> T40
```

## Task definitions

### T31 / #31 — E2E contract

Specify journeys, naming compatibility, boundaries, failures, Evidence and
delivery plan. No product implementation.

### T32 / #32 — Local installation

Build/install Lingo into an explicit user PATH root, preserve unowned content,
support idempotent rerun and expose source/version metadata.

### T33 / #33 — Codex Runtime

Install validated thin `axiom-*` skills at user scope. Verify conflict behavior,
delegation and discovery independently of repository CWD.

### T34 / #34 — Global Project resolution

Persist/resolve validated Project catalog and repository bindings by ID/slug.
Fail safely for ambiguity, missing/moved paths and inconsistent local state.

### T35 / #35 — Project configuration

Compose existing portable/local contracts into one atomic configure use case and
expose repeatable CLI plus Codex entrypoint.

### T36 / #36 — GitHub Work Item

Implement narrow provider-neutral application contracts and a GitHub adapter for
create/select/comment/close with explicit external mutation authority.

### T37 / #37 — Executable workflow

Persist ordered gates, bounded repository target, validation/review Evidence,
interruption/resume and completion rules.

### T38 / #38 — Equivalent entrypoints

Complete CLI/JSON/help surface and map every installed skill to one stable command.

### T39 / #39 — E2E Evidence

Add deterministic acceptance tests and execute/version the real unrelated-CWD
Codex plus direct CLI dogfood journey. Reconcile #19/#20 Evidence.

### T40 / #40 — Final reconciliation

Reconcile README, commands, Specification, roadmap, limitations and #21. Run
final review/validation, open final PR and mark #30/#40 ready for human acceptance.

## Shared completion rule

Each task needs acceptance criteria, deterministic tests where applicable,
documentation, reproducible Evidence, no known Blocker/Major, self-review,
dedicated PR, squash merge to POC branch and issue reconciliation. Final human
acceptance remains outside agent authority.

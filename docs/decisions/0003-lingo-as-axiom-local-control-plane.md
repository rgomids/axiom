# ADR-0003 — Lingo as Axiom Local Control Plane

## Status

Accepted on 2026-09-10 after human review.

Acceptance establishes the architectural direction only. Implementation still
requires a Specification defining observable behavior, boundaries, failure
handling, security constraints, migration impact, and acceptance evidence.

## Context

Axiom owns product intent, domain language, lifecycle, policies, and contracts.
Its current Codex-first skills manually exercise workflows before an executable
product exists.

If each runtime embeds a complete Axiom workflow independently, Codex, Claude,
Kiro, and future integrations would duplicate central behavior. That creates
workflow drift, inconsistent evidence, repeated updates, and coupling between
Axiom semantics and runtime-specific harness formats.

Axiom also needs a coherent future location for Project configuration, runtime
and model resolution, capability negotiation, agent planning, orchestration,
execution state, and Evidence collection. These are executable control-plane
concerns, but they must not redefine the Axiom domain or force runtime-specific
details into it.

Constraints:

- Axiom and Lingo must remain distinct;
- `Project != Repository`, `Role != Model`, and `Execution != Agent`;
- Provider, Capability, Integration, and Transport remain separate concepts;
- Projects must be portable across runtimes;
- secrets never enter portable Project configuration;
- Spec-Kit remains a strategic upstream reference under ADR-0002;
- this ADR does not authorize implementation without an approved Specification.

## Decision

Use **Lingo as Axiom's executable local control plane**.

Axiom remains the product, domain, policies, lifecycle, and contracts. Lingo
will execute or coordinate those contracts through Project configuration,
workflows, Agent Planning, orchestration, runtime/model/capability resolution,
integration bootstrap, execution state, Evidence collection, and a CLI/wizard.

Runtime adapters will bridge Lingo to Codex, Claude, Kiro, and future runtimes.
Runtime skills should tend toward thin entrypoints:

```text
runtime skill
-> Lingo command
-> Axiom workflow
```

Detailed contracts, persistence, commands, schemas, rollout, and migration from
the current harness remain subject to future Specifications and decisions.

## Alternatives considered

### Logic inside each runtime skill

Fastest path for one runtime. Duplicates workflows across targets, increases
drift, makes evidence inconsistent, and raises migration cost as runtimes grow.

### Library embedded directly in runtimes

Centralizes some logic, but runtime hosts still own lifecycle and integration.
Embedding constraints may vary, portability is weaker, and operational state
can fragment across hosts.

### Remote control plane

Centralizes coordination and collaboration. Adds deployment, availability,
identity, tenancy, network, privacy, and operational complexity before the
local product contracts are proven.

### Lingo local control plane

Keeps executable behavior close to local repositories and credentials while
providing one place for workflow and execution semantics. Requires careful
portable/local state separation and a stable adapter boundary.

## Consequences

### Positive

- one source of executable workflow behavior across runtimes;
- lower duplication and drift in runtime skills;
- Axiom domain remains independent from runtime and provider products;
- local-first operation and Project portability remain possible;
- capability negotiation, orchestration, and Evidence collection gain a
  coherent executable boundary;
- runtime and model choices can remain configurable.

### Negative / trade-offs

- Lingo becomes a significant architectural boundary and potential bottleneck;
- adapter and version-compatibility contracts must be designed and tested;
- local state, credentials, upgrades, failures, and recovery need explicit
  semantics;
- thin skills may expose only the behavior Lingo supports;
- migration from current self-contained skills requires staged compatibility;
- a local control plane may later need synchronization or remote coordination.

## Revisit when

- a Specification shows that Lingo cannot preserve required runtime behavior or
  Project portability;
- embedding a library demonstrates materially simpler operation without
  duplicating lifecycle or state;
- collaboration requirements justify a remote or hybrid control plane;
- capability negotiation or execution evidence cannot be expressed without
  leaking runtime-specific concepts into the Axiom domain;
- local state, security, or upgrade costs outweigh the drift reduction.

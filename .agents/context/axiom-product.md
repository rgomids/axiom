# Axiom Product Context

## Product thesis and canonical sources

Axiom owns the product, domain, policies and contracts for governing AI-assisted
software development. Lingo is the accepted local-first executable control-plane
direction under [ADR-0003](../../docs/decisions/0003-lingo-as-axiom-local-control-plane.md).
The names are not synonyms. Runtime skills are entrypoints, not the workflow
source of truth; durable Evidence is not raw chat history.

Use these canonical sources rather than expanding this routing summary:

- [Product Foundation](../../docs/product/foundation.md): facts, requirements,
  decisions, hypotheses and open questions.
- [Constitution](../../docs/product/constitution.md): normative governance.
- [Product index](../../docs/product/README.md#documentation-authority): discovery,
  versioned technical truth and operational tracking responsibilities.
- [Architecture](../../docs/architecture/README.md): target boundaries and implemented foundation.
- [Specification index](../../docs/specifications/README.md): current approval,
  implementation and acceptance state; merge never grants next-Task authority.
- [Roadmap](../../docs/product/roadmap.md): capabilities and dependencies, not delivery authorization.

## Product boundaries

A Project may relate multiple independent Repositories. It does not require
submodules, common providers or colocated checkouts. Preserve `Project != Repository`,
`Role != Model`, `Execution != Agent`, `Provider != Transport` and `Integration != MCP`.

[ADR-0004](../../docs/decisions/0004-portable-project-manifest.md) accepts the
versioned Portable Project Manifest, currently `axiom.yaml`, distinct from local
installation state. Project UUID is immutable; slug and name are mutable.
Portable working copies use `projects/<slug>`; local state uses
`state/projects/<id>`. Secrets and machine-local bindings never belong in portable intent.

[Specification 002](../../docs/specifications/002-lingo-project-initialization/spec.md)
and its approved Plan define minimal init, explicit update, reopening and local
installation. Update may start from partial intent but must materialize and
validate complete proposed state before persistence. Optional Git backing creates
no Repository association; Project mutation, local commit and remote push/sync
have separate authority and outcomes. External mutations require explicit authority.
The Plan and clarifications own detailed formats and revision semantics.

## Implemented foundation and active slice

Specification 002 is the first active vertical slice, delivered incrementally.
The Go module contains Project invariants (`internal/project`), application
contracts and authority (`internal/projectapp`), the portable manifest codec
(`internal/manifest`), local JSON codec and minimal filesystem stores
(`internal/local`), and a limited executable Lingo CLI (`internal/cli`,
`cmd/lingo`). The POC covers one-manifest Project operations; complete
Specification 002 use cases, runtime adapters and orchestration remain open.

Implementation requires approved artifacts and explicit authorization for the
bounded Task. Consult the Specification index before advancing; neither roadmap
direction, code availability nor technical merge substitutes for human acceptance.

[Specification 001](../../docs/specifications/001-codex-agent-harness-generation/spec.md)
remains Proposed. Its agent-factory flow and
[dogfood evidence](../../docs/product/dogfooding/001-go-pr-review-agent.md)
inform the existing Codex-first harness; they do not replace the active slice.

## Future capabilities and unresolved questions

Future work may include runtime/model discovery, capability negotiation,
Integration bootstrap, Agent Planning, Execution graphs and orchestration.
Configuration declarations already exist; their presence does not establish
connectivity or implemented execution. No provider or runtime is mandatory at
the domain level, and useful deterministic behavior must remain possible without an LLM.

Still unresolved beyond the approved slice contracts:

- Workspace persistence/ownership and cross-Project repository sharing;
- internal Work Item identity and future provider ownership/conflict policies;
- concrete filesystem publication/recovery mechanisms satisfying the Plan;
- remote/hybrid collaboration and synchronization protocols;
- runtime discovery, capability negotiation and Integration bootstrap behavior;
- credential-source resolution, execution/Evidence schemas, retention and privacy;
- Agent Planner/Orchestrator contracts and future action/risk approval policies;
- CLI framework, packaging and distribution strategy.

Do not reopen resolved identity, minimal-init, portable/local format or Project
update/authority questions. Consult approved Specifications and ADRs before
classifying any issue as undecided.

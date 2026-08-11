# ADR-0002 — Axiom Relationship with GitHub Spec-Kit

## Status

Proposed on 2026-08-11. Human acceptance required.

## Context

Axiom and GitHub Spec-Kit overlap in intent-first development, constitution,
specification, clarification, planning, tasks, implementation, and validation.
Spec-Kit `v0.16.2` also provides mature Codex integration, checklists,
cross-artifact analysis, convergence, extensions, presets, workflows, bundles,
manifests, and upgrades.

Axiom's target domain is broader than the evaluated Spec-Kit feature flow. It
needs a logical Project distinct from Repository, multi-repository delivery,
Work Item/provider boundaries, accountable Decisions, Executions, Evidence,
Artifacts, Releases, and living-document reconciliation.

A controlled same-scenario experiment found:

- both workflows completed the documentation-only scenario with one human
  clarification batch and no application code or dependency;
- Axiom was leaner and aligned with its domain/review/reconcile concepts;
- Spec-Kit used more time/tokens but provided stronger standardized artifacts,
  cross-artifact analysis, and convergence;
- Spec-Kit analysis caught an unsupported identifier-allocation inference that
  Axiom's self-review missed;
- neither run demonstrated Axiom's required multi-repository and Evidence model.

Durable evidence: [Axiom and GitHub Spec-Kit Evaluation](../research/axiom-speckit-evaluation.md).
Temporary reproducibility evidence:
[`experiments/speckit-evaluation/`](../../experiments/speckit-evaluation/).

## Proposed decision

Axiom should implement **conceptual compatibility with Spec-Kit without a
mandatory runtime dependency**.

Axiom should:

- keep its own domain and state authoritative;
- define explicit conceptual mappings where semantics genuinely align;
- adopt proven interaction and quality patterns such as bounded clarification,
  requirements checklists, cross-artifact analysis, and convergence;
- avoid copying upstream templates or promising file compatibility by default;
- add import/export or an executable Spec-Kit adapter only through a separately
  approved specification and evidence;
- keep encapsulation/orchestration as a reconsideration path.

This proposal does not authorize implementation, Spec-Kit installation,
dependency adoption, Lingo changes, or an interoperability contract.

## Alternatives considered

### A — Direct dependency

**Benefits:** Immediate reuse of mature SDD artifacts, Codex support, analysis,
convergence, extensibility, and upstream improvements.

**Limitations/trade-offs:** Highest coupling to upstream layout, commands,
templates, versions, and upgrade lifecycle. Axiom must still build its broader
domain around Spec-Kit. Lingo and multi-repository evolution become harder to
separate later.

### B — Encapsulation/orchestration

**Benefits:** Reuse selected Spec-Kit capabilities behind an Axiom experience
while keeping Project/Evidence orchestration above it.

**Limitations/trade-offs:** Wrapper may leak upstream state and errors. Requires
version pinning, compatibility matrices, migrations, provenance, rollback, and
two workflow models. Viable after a narrow adapter experiment.

### C — Conceptual compatibility

**Benefits:** Preserves Axiom domain, customization, multi-repository direction,
living documentation, and future Lingo while learning from proven Spec-Kit
patterns. Keeps optional interoperability reversible.

**Limitations/trade-offs:** Axiom owns implementation and can duplicate generic
SDD, drift, or claim compatibility too vaguely. Mappings and reuse boundaries
must be explicit and tested.

### D — Reference only

**Benefits:** Lowest external coupling and maximum freedom.

**Limitations/trade-offs:** Highest reinvention, no interoperability promise,
and full ownership of generic SDD/Codex/upgrade mechanics.

## Consequences

### Positive

- Axiom Project, Execution, Evidence, provider, Release, and documentation
  lifecycles remain unconstrained by one upstream project/feature layout.
- No Spec-Kit runtime/version/upgrade dependency enters Axiom now.
- Proven Spec-Kit quality patterns can influence Axiom's specifications and
  deterministic contracts.
- A later optional adapter remains possible without migrating authoritative
  Axiom state first.

### Negative / trade-offs

- Axiom must implement and maintain its own workflow contracts.
- Generic SDD functionality may be duplicated without strict capability-level
  reuse-or-own decisions.
- Conceptual mappings can drift as both projects evolve.
- Spec-Kit users receive no automatic artifact interoperability.
- Axiom needs its own equivalent of observed analyze/converge value.

## Risks

- **Compatibility theater:** names align while semantics differ.
- **Reimplementation waste:** Axiom rebuilds mature generic features without a
  domain-specific reason.
- **Drift:** upstream changes invalidate mappings.
- **Under-validation:** Axiom repeats the self-review gap observed in the
  experiment.
- **Delayed reuse:** avoiding a dependency now may postpone valuable adapter
  evidence.

Controls if accepted:

- version every conceptual mapping and state semantic differences;
- require deterministic cross-artifact coverage and convergence in Axiom;
- evaluate generic capabilities individually before rebuilding them;
- keep upstream monitoring as research, not implicit architecture;
- require a separate ADR/specification before any runtime adapter or import/export
  contract.

## Revisit when

- a bounded Axiom workflow demonstrates a pinned Spec-Kit adapter with lower
  validated cost than Axiom-native implementation;
- upgrades, failures, rollback, provenance, privacy, and multi-repository
  aggregation are tested;
- users require Spec-Kit artifact interoperability;
- Spec-Kit's domain model materially converges with Axiom Project, Execution,
  Evidence, Release, provider, or living-document requirements;
- Axiom's generic SDD implementation cost or quality becomes unacceptable.

# ADR-0030 — Canonical results, terminal output and host presentation

## Status

Proposed — 2026-10-18, AXM-32 specify only. Explicit human acceptance of the
exact revision remains required. Merge, checks and publication do not accept
this proposal or authorize implementation. Human D1–D3 choices recorded on
2026-10-18 refine this text; they are not formal acceptance of the revised ADR.

## Context

[#232](https://github.com/rgomids/axiom/issues/232) delivered and explicitly
accepted deterministic Markdown with literal Runtime skill relay in v0.13.0.
[PR #325](https://github.com/rgomids/axiom/pull/325) retained that relay and added
optional auxiliary explanation. Personal v0.19.0 testing reports verbosity and
a success heading obscuring blocked Execution readiness. The current application
can legitimately complete structural validation with success while returning an
Execution blocker. Presentation cannot redefine either fact.

The [AXM-32 Specification](../specifications/008-output-separation/spec.md)
records the baseline, exact conflicting contracts, proposed behavior, migration
requirements and resolved D1–D3 choices, with technical JSON coverage still
unverified before Plan. This is a durable public
CLI/host boundary change, so an ADR is justified even without changing domain
status or persistence.

## Decision

The proposal incorporates human D1–D3 choices recorded in the
[GitHub decision](https://github.com/rgomids/axiom/pull/328#issuecomment-6245936652)
and [Linear decision](https://linear.app/rgomids/issue/AXM-32/separar-saida-do-lingo-da-apresentacao-conversacional-da-ia#comment-43e5ffd1)
on 2026-10-18, following the
[technical review](https://github.com/rgomids/axiom/pull/328#pullrequestreview-5561505065).
Formal acceptance of this exact revised ADR remains pending.

Keep one application/domain-owned structured
result and complete compatible JSON transport; let Lingo produce deterministic
native terminal text and let the existing operating AI synthesize Markdown in
its own host from canonical JSON. No dedicated formatting LLM call is added.
Human surfaces preserve material truth, exact authority/digests and references;
they need not reproduce every nonmaterial internal field. JSON remains lossless.

Keep status, readiness, effect certainty, operation identity and acceptance
separate. Success of `project validate` must not conceal blocked Execution,
nor be recoded as a new domain failure by a renderer. Preserve domain/skill
ownership, local control-plane authority and explicit consent.

D1 keeps no-flag human output in TTY AND pipes/redirection, as native CLI text,
not Markdown. `--human` selects that mode; `--json` explicitly selects automation
and internal consumption by all four skills. Non-TTY never selects JSON
implicitly; pipes have no ANSI/color, `NO_COLOR` suppresses color and unsupported
terminal capabilities degrade to plain text. Status, exit codes, invocation
identity and authority remain unchanged.

D2 includes prospective replacement of #232 default Markdown/exhaustive relay
and AXM-31 literal relay/Auxiliary obligations in AXM-32 itself, coordinated
across the binary and all four installed skills for Codex/Claude. It cannot be
deferred to a cleanup Work Item or retain mandatory literal relay as the final
contract. Migration must inventory consumers and verify compatible binary/skill
sets, upgrade protection, explicit JSON/flags and rollback while preserving history.

D3 preserves the current typed JSON wire contract, names/types/enums/status and
invocation identity. No speculative envelope, field or version is introduced.
R1 fact coverage must be inventoried with concrete producer/consumer evidence
before Plan; it remains unverified. A proven material gap requires explicit
review and acceptance of a compatible/additive or versioned proposal identifying
consumers and tests before code changes. This technical prerequisite does not
make the D3 policy an open human decision.

## Alternatives considered

| Alternative | Complexity, coupling and portability | Operability/security, cost and migration |
|---|---|---|
| Retain literal Markdown relay and auxiliary explanation | Least change; hosts remain coupled to terminal Markdown | Deterministic but preserves duplicate/verbose presentation; no additional inference cost; smallest migration |
| One canonical JSON, deterministic terminal, host synthesis (selected direction; ADR Proposed) | Clear transport/presentation ownership; shared host behavior needs integration checks | No extra model call; host omission/injection risk requires fact-preservation checks; coordinated skill/binary migration |
| Deterministic host-owned Markdown templates over JSON | Strong repeatability; duplicates presentation rules across hosts and constrains contextual synthesis | No extra inference call, but template maintenance and host-specific migration cost; less adaptable to the user's objective |

The selected direction costs stronger host integration validation in exchange
for concise native conversation. Deterministic byte equivalence remains suitable
for JSON/terminal snapshots; conversation requires semantic assertions and native
observations. Automation must not depend on human prose.

## Consequences

### Positive

- Canonical facts and authority remain inspectable without AI interpretation.
- Terminal users receive native text; host users receive one concise answer.
- Automation retains explicit JSON without a new inference dependency.

### Negative / trade-offs

- Host synthesis can omit or overstate facts; typed equivalence checks and native
  Codex/Claude evidence are required, not just instruction string tests.
- Existing Markdown goldens, literal-relay instructions and consumers of human
  output require migration within the coordinated AXM-32 delivery. Exact revised
  artifacts still need acceptance; old accepted Evidence stays.
- After consumers rely on the new boundary, reverting output/skill contracts is
  harder: rollback needs a compatible binary/skill pair and preserved state.

## Relationship to accepted decisions

No accepted ADR fragment requiring supersession was identified.
[ADR-0003](0003-lingo-as-axiom-local-control-plane.md),
[ADR-0006](0006-machine-local-detail-artifacts.md),
[ADR-0022](0022-dedicated-workflow-conversational-surface.md) and
[ADR-0025](0025-public-stage-dispatch-and-capability-proof.md) retain their
control-plane, artifact, skill-ownership and authority decisions. This proposal
prospectively replaces bounded presentation obligations of #232, the command
reference and AXM-31, not their acceptance history or diagnostic semantics.

If human review finds an accepted ADR conflict, identify the exact partial/total
scope in a new proposal revision and apply
[ADR-0018](0018-adr-evolution-and-supersession-governance.md) only after explicit
acceptance. No old ADR status, text or supersession annotation changes here.

## Revisit when

Existing JSON cannot safely express required facts; a host cannot consume it
without a separate formatting hop; native tests cannot preserve material truth;
or verified consumers cannot migrate safely in the coordinated delivery. Such findings require
explicit decision rather than silent schema, authority or default changes.

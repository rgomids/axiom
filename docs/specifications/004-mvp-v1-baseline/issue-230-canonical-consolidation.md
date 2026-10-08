# Issue #230 — Canonical skills consolidation

## Approved direction and historical boundary

On 2026-10-08 the maintainer explicitly authorized local planning,
implementation and validation of the pre-MVP two-skill product surface.
This supersedes #229's decision to retain six operation-specific compatibility
entrypoints. It does not supersede its thin Runtime/application architecture,
argument discovery, completion, ownership or authority contracts.

Historical #229 and #230 Evidence describes what earlier revisions actually
shipped. v0.10.0 (`c7260797aad7865419f555eea94a2eef7ae78285`) is the historical
baseline, not a tag to rewrite. Current runtime distribution must contain only
`axiom-project` and `axiom-work-item`. Repository maintainer skills remain
separate from these product Runtime skills.

No new ADR is needed: this changes the pre-MVP product operation surface, while
ADR-0003 (thin Runtime/application boundary), ADR-0005/0007 (protected state and
publication), ADR-0016 (candidate-derived receipt), and ADR-0014 (maintainer
skill discovery) retain their decisions. Historical digest/receipt recognition
is internal migration metadata, never an active invocation surface.

## Invariants and acceptance

- Preserve FR-068–FR-084, AC-50–AC-61 and the frozen v1 lifecycle matrix;
  reconcile current routing in its delivery/consolidation sections.
- Keep every supported CLI operation. Unsupported Provider delete, credential
  revocation, sequential Execution cancel and history mutation stay unsupported.
- Natural-language intent selects a declared operation/mode; ambiguity stops
  for clarification. Intent does not supply authority, identities or Evidence.
- Lingo owns parser/argument discovery, validation, identity, state, preview,
  digest/revision verification, domain decisions and effects.
- Fresh Codex/Claude installation converges from zero to two skills. Recognized
  six/eight sets converge to two. Reinstall from two is idempotent.
- Retire only content whose private directory, file shape and digest match
  trusted Axiom history. Unknown/modified/linked/foreign content is preserved;
  malformed receipts fail closed. No entire Runtime root is deleted.
- Interruption reports confirmed/partial effects accurately. Rerun revalidates
  remaining content; it never fabricates atomic cross-root success.
- FR-026 release corpus must cover `operational.json` from a reproducible
  v0.10.0 writer and detect future persisted writer gaps before release. Release
  preparation validates the frozen local/compatibility contracts before building
  and clears the fixture generator variable.

## Bounded plan and tasks

| Task | Scope | Acceptance / evidence |
|---|---|---|
| C230-T01 | Capture trusted historical eight-skill bytes; reduce embedded distribution; safe owned retirement in install/upgrade/recovery; release manifests | 0/6/8→2, 2→2; conflicts preserved; interruption; Codex/Claude parity; archive journey |
| C230-T02 | Review both canonical skills; remove legacy inspection/discovery; executable routing/argument/matrix checks | Every supported route, authority/mode/payload preserved; unknown/ambiguity fail safely |
| C230-T03 | Historical state fixture and corpus enforcement; assess show/validate composition; workflow list output tests | Exact tag provenance; FR-026; empty/multiple/large/order/damaged output |
| C230-T04 | Reconcile specification, plan/tasks, docs and historical Evidence links; prepare native acceptance | Real Runtime/Provider acceptance explicitly pending until isolated authority exists |
| C230-T05 | Aggregate validation, archive proof and engineering/security review | Build/vet/mod/test/race/repository/runtime/release/upgrade/security results; AC ledger |

T01–T03 can progress independently in disjoint files. T04 uses their contracts;
T05 integrates them. Main session retains scope, authority and result claims.

## Test matrix and authority

Tests use temporary HOME, Projects, State, Codex and Claude roots. The migration
matrix covers absent/current/historical/partial skill sets, modified/foreign/
linked entries, malformed receipts, deterministic reruns, inspection and bundle
inventory. Lifecycle regressions run unchanged. Native Runtime selection is a
separate acceptance observation, not implied by parser/stub tests.

Authorized: local changes, isolated tests, public read-only issue/release data.
Pending authority: real Runtime installation, real Provider mutation, commit,
push, merge, release, Issue closure and human acceptance. No automated test may
mutate a production GitHub resource.

[Execution evidence and handoff](evidence-230-canonical-consolidation.md).

# Tasks 010 — Reconciled causal execution map

**PROPOSED — R1, 2026-10-20. No tasks resumed.**
Use only after exact [baseline/ADR acceptance](reconciliation.md#grouped-human-decision-h1)
and a separate complete Implement prompt. Same AXM-37/PR #335; no duplicate
Execution or dispatch. [Plan R1](plan-reconciled.md) and [Spec R1](spec-reconciled.md)
own behavior. E01–E16 remain stable Evidence identifiers, not new artifacts/states.

## Historical state preserved

At entry `5dd66917d601357126f3c53c85b01581ead3d664`: T01–T04 PARTIAL,
T02/D2 and G1 BLOCKED; T05–T16 BLOCKED/NOT_RUN, G2/G3 and full AC/V rows NOT_RUN.
C1 accepted, C2 macOS oracle FAIL, native 24-case positive suites and 21-transition
models retained with original limitations. R1 is an unexecuted replacement work
map; no row below changes those statuses or claims existing production completion.
Earlier work may be reused only after checking its applicability to the new oracle.

## Outputs, causal dependencies and done evidence

Dependencies below are direct inputs, not permission grants. Every unit also
requires the baseline/Implement authority above. Rows are internal units in one
cohesive implementation, not separate handoffs or approval gates.

| ID / retained owner | Observable output / minimum scope | Causal inputs | Evidence and ready criterion / AC |
| --- | --- | --- | --- |
| T01 / contracts | Bounded amendment of accepted D1/D3: compiled descriptor semantics and exact image-field delta from Plan R1, acyclic build graph, v1 pins, known receipt variants and migration matrix. Preserve numeric budgets | none | E01: exact contract/vector comparison, unknown/duplicate/version refusals and noncircular DAG; no image claim from disk hash. Final quota freeze joins T05/T06 before T07. AC-02/05/06 |
| T02 / association proof | Small native association/refusal tests using descriptor/current-install facts; reuse applicable experiments, no new full-image research | T01 | E02: all three native host rows, C2 B-content refusal, surviving A/B, same-label, aliases/spaces, copy/move, stale/unsafe target and snapshot read. Original C2 FAIL retained; no full-image qualification claimed. AC-02/05 |
| T03 / publication boundaries | Review/reconcile existing lock and state-transition fixture to descriptor; installation key covers same target/different receipts; no generic transaction engine | T01 | E03: two conflicting publishers cannot stale-commit; lock replacement, stage/commit/uncertain/postcommit failures classify effects and preserve prior/foreign data. Fixture not installer PASS. AC-03/05 |
| T04 / engineering review | Internal contract/security review of T01–T03; retire original G1 fan-in only upon accepted R1, not by declaring old G1 PASS | T02, T03 | E04: reviewed descriptor oracle, boundary assumptions and remaining risks, no unresolved material contradiction before write integration. AC-02/03/05 |
| T05 / corpus | Dedicated English allowlist and complete minimum operational content/local links/source mapping | none | E05: all content classes, exclusions/notices, no checkout-only journey. No T02/T04 predecessor. AC-01/06 |
| T06 / generator | Deterministic reference/man from real CLI tree plus coverage and provenance; include final public inspect tree before packaging | T01, T05 | E06: two identical documentary builds, exact help parity, all public leaves/flags mapped, complete corpus size/header/manifest measurements. Freeze existing budgets or stop before T07, never trim coverage. AC-06/07 |
| T07 / package | Coordinated v2 producer/verifiers/shell and Go readers, compiled descriptor build order; v1 dispatch unchanged | T01, T06 | E07: accepted exact limits, closed codec/archive refusals, pinned old/new matrix and no candidate execution on invalid package. AC-05/06 |
| T08 / generation publisher | Complete confined immutable generation/binding publication with exact preview effects | T04, T07 | E08: bounded safe writes, equivalent no-op, foreign conflicts preserved, inventory/descriptor verified before publication. AC-03/05 |
| T09 / installer lifecycle | Fresh/upgrade integration, installation-wide lock, generation-before-binary ordering, receipts, read-back and supported recovery | T08 | E09: actual installer concurrent/fault tests (not only model), confirmed effects truthful, ADR-0016 receipt proof unchanged, no downgrade/GC, data preserved. AC-03/04/05 |
| T10 / read-only consumer | One descriptor/installation/content resolver using validated package/binding fixtures; no dependency on completed installer | T04, T07 | E10: current content pinned/revalidated or refused, crossed/C2/alias/race/unknown cases; zero network/write/extraction/repair/Runtime startup; help independent. AC-01/02/05 |
| T11 / CLI | Accepted leading presentation grammar, bounded additive docs result and truthful association fields | T10 | E11: schema/status/reason/exit and human/JSON checks, verified-only paths, no global grammar changes; regenerate T06 outputs after final command integration. AC-02/06 |
| T12 / skills/man | Four active skills consume inspect/help, local revalidation and host-specific session man recipes | T11, T06 | E12: expected/installed/loaded honest, no parallel resolver or fallback; actual page identity vs binary and explicit A selection, no persistent profile/cache writes. AC-02/07/08 |
| T13 / artifact qualification | Integrated exact-artifact native lifecycle/man/migration matrix V1–V7 | T09, T12 | E13: all three supported native host rows, exact package/source hashes, A/B, actual compatible return, v1/new formats, v0.1.0 floor qualification or explicit UNVERIFIED. Full AC-02–07; no unsupported migration claim |
| T14 / offline journey | Clean local artifact installation and reading without checkout/network (V8) | T09, T12 | E14: all three native host rows, denied network/source access, index and all content classes read, paths/hashes and controls. Does not depend on every T13 migration row. AC-01 |
| T15 / native agents | Separate real Codex/Claude sessions using installed docs/help/readiness fixture (V9) | T09, T12 | E15: unchanged Q6 seven-step controls and historical host/Runtime matrix; independent OS file/network/descendant observation, controls and full A/B/mismatch/absent/tampered cases. NOT_RUN/UNVERIFIED for gaps, never synthetic PASS. AC-08 |
| T16 / integration readiness | Reconcile installed-user docs, Evidence and complete technical review; explicit remaining human/release boundary | T13, T14, T15 | E16: AC/OD→outputs/evidence coverage complete, local/CI/native claims separate, all required gaps dispositioned before full qualification. No release, merge, acceptance or unblock inferred. AC-01–08 |

T11 changes the real tree; T06's generator is then rerun as part of T11 and final
artifact construction, not a dependency backedge. T07 may use complete provisional
outputs containing the proposed command descriptor; T13–T15 use only regenerated
final-tree artifacts. This is normal rebuild after integration, not a cycle.

## Reduced execution shape

1. Content/contracts: T01, T05 → T06 → T07. Association/publication review:
   T01 → T02 and T03 → T04. Independent fronts may be executed sequentially by
   one engineer; no multi-agent execution is necessary or implied.
2. Integrate delivery: T04 + T07 → T08 → T09. Reader/CLI/skills:
   T04 + T07 → T10 → T11 → T12; T06 generator refresh joins final package build.
3. Qualify final artifacts: T09 + T12 → T13, T14, T15 → T16. Run separately only
   when disposable hosts/observers/real sessions have explicit authority.

Removed: blanket T04→T05/T06, T02→T03, T09→T10 and T13→T14/T15 sequencing.
These are not causal inputs for content, transition modeling, fixture reader work
or independent offline/agent trials. Preserved: reviewed safety before mutation
integration, measured coverage/limits before packaging, final exact artifacts
before acceptance tests. Quota/security failures block dependent effects.

Deferred/removed scope: exact original full-file image research and private native
ABI adoption from T02/T10 (requires H1/ADR-0034); no replacement generic identity
framework. Automatic GC/rollback and source-doc qualification remain deferred as
before. No historical unit is deleted or marked done; T04 becomes an internal
review checkpoint, T16 remains one readiness integration. Missing qualification,
new contract/ABI requirement, quota change or migration-floor narrowing is a
material decision, reported together rather than a new series of C3/C4/C5 gates.

## Bounded implementation continuation (2026-10-21)

[v2/gzip r2 Evidence](evidence-bounded-v2-gzip-r2.md) maps the authorized
T02/T03/T07/T09/T12 delta to AC-01/02/03/05/06/07. The task definitions above
remain the accepted R1 map, with their original proposal history preserved.
The continuation retains T13/T15 NOT_RUN, T14/T16 PARTIAL, AC-04/08 NOT_RUN and
v0.1.0 UNVERIFIED; no default activation or independent T16 acceptance follows.

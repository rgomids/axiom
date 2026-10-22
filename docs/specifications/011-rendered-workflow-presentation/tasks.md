# Specification 011 Tasks — Rendered Workflow presentation

## Status and exact authority

**Revision 1 — PROPOSED, 2026-10-22; attempt-01; next gate `review_tasks`.**
This artifact decomposes approved planning; it does not authorize implementation,
qualify a host, start a canonical Execution or accept product behavior. All four
Tasks below are **NOT_STARTED**; all future behavioral Evidence is **NOT_RUN**.
Owner: [AXM-38](https://linear.app/rgomids/issue/AXM-38).
Only [PR #336](https://github.com/rgomids/axiom/pull/336), branch
`axm-38-specify-rendered-workflow`, is used. Observed local and remote HEAD:
`5939c7aac0f05f53d94116c4ebd89fb774e36950` (OPEN/Draft).

[Human approval of both exact artifacts](https://github.com/rgomids/axiom/pull/336#issuecomment-6345522075)
supersedes their historical Proposed/pending approval wording, without editing it:

| Approved input | Git blob | SHA-256 |
| --- | --- | --- |
| [Specification 011 editorial e1](spec.md) | `ba83987e96bc31bd31778ef80ab9e4a2890b1706` | `921389e131fa203366a013102dfde429df8a160f228029f995cadebf48208be1` |
| [Plan r1](plan.md) | `469fd7a7a94778b8dc275640839238fe9902e90a` | `2418a6814b03bbc3e7472a19dd6c3052de2eb0e66bb0cd4100ec65fbaed0dd24` |

Both blobs exist at the observed HEAD. Linear approval receipts:
[Spec e1](https://linear.app/rgomids/issue/AXM-38#comment-bbe9f2ad) and
[Plan r1](https://linear.app/rgomids/issue/AXM-38#comment-e0d2b09c).
P-01–P-05 apply only within those approved conditions; P-05 neither adopts a host
nor proves compatibility. CR-001/CR-002 technical closure and all historical
`completion-receipt*.json` remain untouched. Preparation Evidence and actual
authority/publication state belong to [this attempt's receipt](completion-receipt-tasks.json).

Preserve [Spec 007](../007-configurable-workflows/spec.md),
[ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md),
[ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) and its
[effort amendment](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md).
Prospective local configuration retirement does not rewrite historical Workflow
schema-1 effort or immutable execution facts. Follow [SDD](../../../.agents/skills/axiom-sdd/SKILL.md),
[documentation governance](../../documentation.md) and [tracking ownership](../../development/linear-github-issue-tracking.md).
Linear stays In Progress; AXM-39/AXM-12 relationships stay unchanged.

## Shared completion rules and dependency graph

IDs AXM38-T01–AXM38-T04 are stable and map one-to-one to Plan §5 packages 1–4.
Candidate paths are impact bounds, not permission to edit every listed file.
No new domain/schema, Go/Lingo renderer, persisted graph, CLI command, dashboard,
service, plugin, runtime, credential, platform promise or formatting-only LLM call.
JSON and terminal contracts remain intact; host presents canonical facts only.

| Task | Causal prerequisites | Future execution class | Completion boundary |
| --- | --- | --- | --- |
| AXM38-T01 | Approved Tasks and separate implementation authority | Local, synthetic | Reviewed fixture/oracle and packaging inventory |
| AXM38-T02 | T01 fixture/oracle baseline | Local guidance and offline evaluation; real host output separately authorized | Semantic mapping, fallback and safe-label rules evaluated; host qualification remains T03 |
| AXM38-T03 | T01 and explicit bounded host-session authority to start; final T02 to complete | Real external session | Exact supported host visually qualified, failures retained; final T02 bytes rechecked |
| AXM38-T04 | Final T02; T03 for graphical acceptance | Local packaging checks; live installation separately authorized | Packaging/compatibility verified; AC1 requires T03 evidence |

Task DAG edges: T01→T02, T01→T03, T02→T03, T02→T04, T03→T04. No cycle.
T02→T03 is a completion prerequisite: bounded T03 capability probing may start
independently after T01, but T03's final qualification uses final T02 bytes.
Early probe feedback does not require T03 completion before T02. Minimum safe
order: pin T01 → evaluate T02 → authorized
T03 final candidate qualification → T04 final integration/review. Early T03 probing
can inform syntax in T02. Critical completion joins both paths at T04; duration
is unknown. Without T03 authority, finish authorized local T01/T02 and prepare T04,
but T04 remains PARTIAL and graphical AC unmet. No artificial `continue` gates.

Each Task requires exact input, source/skill revision, command/check and outcome,
sanitized evidence location and limitations. Synthetic envelopes are never called
captured CLI results. Offline oracle success is never host/model success. Retain
failures and NOT_RUN results; no retroactive proof or changed historical receipts.

## AXM38-T01 — Canonical fixtures, semantic oracle and packaging baseline

**Result:** reproducible canonical-shaped inputs with independently reviewed
expected facts and an inventory of the existing shared installer consumers.
**Scope:** Plan §§1–3/5–6; no production mapping or renderer, installed-state edits,
new canonical format, substitute JCS implementation or real host sessions.

**Candidate code/docs:** [producer](../../../cmd/lingo/workflow_authoring.go),
[report](../../../internal/projectapp/workflow_authoring.go),
[definition tests](../../../internal/workflowdefinition/definition_test.go),
[skill tests](../../../internal/codexruntime/workflow_skill_content_test.go),
[shared manifest](../../../internal/codexruntime/manifest.go),
[runtime installation](../../../internal/codexruntime/runtime.go),
[upgrade tests](../../../internal/codexruntime/domain_skills_upgrade_test.go),
Spec 007 examples and future test-only fixtures adjacent to existing tests.
Producer/report/schema files are inspection sources, not planned production edits.

**Dependencies:** no other Task; approved Tasks plus implementation grant first.
**Completion/tests:**

- Pin source commit and actual `workflowAuthoring` completion shape, including
  operation identity, outcome, diagnostics, prerequisites/effects and provenance.
  Wrap default r1/r2 and custom r1/r2 through existing validation; preserve original
  digests. Mark constructed wrappers SYNTHETIC; record fixture hash and reference.
- Expected inventory records complete four-field identity, selection relationship,
  Project revision separately, ordered Stage IDs, scoped Agent IDs, exact predecessor
  sets, gate kind/timing, I/O references and declared constraints/configuration.
  Manually checked expected cases must detect a missing/extra/reversed edge and
  wrong identity/order; computing expected and actual with the same mapper is insufficient.
- Cover multi-agent sequential chain including declared extra predecessors; DAG
  independent roots and integrator join; single-Agent DAG; DAG concurrency 1.
  Equal object-key/whitespace variants preserve semantic facts; changed arrays,
  edges, identity and declaration values change only corresponding expectations.
- Prepare selected/divergent/absent/stale identity, schema-1 historical effort and
  schema-2 absence, missing/failed/partial/unknown/truncated payload, invalid schema,
  cycle/dangling/self/duplicate dependencies and malicious Unicode/markup scenarios.
  Invalid inputs expect diagnostics/refusal, never corrected success graphs.
- Include valid 32-Stage, 32-Agent Stage and dense 496-edge DAG cases; 1 MiB
  definition/snapshot boundary and over-bound rejection, bounded prose 16384 UTF-8
  bytes and over-bound, 64-item lists, attempts 1/10 and timeout 1/86400.
  Cross-Stage aggregate limits remain payload-constrained; do not require all
  individual maxima simultaneously. Recheck bounds against current producer.
- Inventory active four-skill bytes/digests, shared history/receipts, Codex/Claude
  consumers, installation/conflict/rollback test seams. Checkout inventory is not
  installed inventory. Preserve history/fixtures and existing compatibility versions.

**Expected Evidence:** fixture manifest, manually checked inventories, negative
control results, packaging source map and focused `go test ./internal/workflowdefinition
./internal/codexruntime` results. Extend producer tests only if envelope fidelity
requires them. These commands are future checks, not checks run in this stage.
**Coverage:** AC2/3/5/6; E2/E3/E4/E6/E8/E9/E10; VP-03–10/12/14/15.
**Authority/rollback/handoff:** local synthetic preparation only after implementation
authorization; no installed CLI/state required. On source drift, pin revised source
and reconcile expected facts before T02/T03. Revert only this Task's new fixtures/tests
if rejected; preserve accepted examples/digests. Hand off manifest and oracle hashes.

## AXM38-T02 — Host presentation, semantic mapping and independent fallback

**Result:** shared Workflow skill guides three-level host presentation from one
canonical input, with evaluated fidelity and complete independent text fallback.
**Scope:** host-owned transient mapping per Plan §§2–3; no formatter executable,
Go graph engine, wire change, persistence, new domain or mandatory interaction.
**Candidates:** [installed Workflow skill](../../../internal/codexruntime/skills/axiom-workflow/SKILL.md),
[existing content tests](../../../internal/codexruntime/workflow_skill_content_test.go),
T01 test-only semantic evaluation fixtures. Inspect
[Project skill](../../../internal/codexruntime/skills/axiom-project/SKILL.md)
for conflicting handoff only; selection remains Project-owned.
**Dependencies:** T01 baseline. T03 probes may inform reversible syntax; no format
is frozen until actual compatibility evidence exists.

**Completion/tests:**

- Workflow overview shows Definition label, full identity/selection, complete ordered
  Stage index, purposes, mode/count and boundary gate markers. Stage detail preserves
  exact scoped Agent nodes and all declared edges; Agent detail exposes declared I/O,
  roles, constraints, Profile intent, capabilities/effect ceilings, criteria ownership,
  attempts/timeouts and instructions on request. No array-derived DAG chain or
  transitive reduction. Gate declarations never imply satisfaction.
- Keep Stage progression, Agent dependency and I/O provenance distinct. Show potential
  concurrency separately from observed activity. No invented Execution, progress,
  success, validation, capability, resolved Runtime/Profile or consent. Schema-1
  effort remains historical; do not synthesize effort into schema 2.
- Full-reference comparison prevents selection divergence and stale follow-up mixing;
  absent selection stays absent. Re-read exact source when needed, disclose drift,
  never select or mutate. Preserve failed/partial/unknown diagnostics and effect certainty.
- Both visual and text derive from the same input inventory. Text alone provides
  three levels, exact identity, Stage order, all Agent predecessor lists including
  independent roots, gates and material warnings. Missing renderer, literal fence,
  link-only output, clipped/empty/failed graphic and unknown capability yield honest
  fallback; none counts as graphical success.
- Segment ordered Stages and dense Stage detail with full index, segment position,
  identity, continuation endpoints and cross-segment predecessor lists. All 496 edges
  stay accounted for; no silently lost nodes/gates/configuration. Reachable full detail
  remains exact; disclose any unavailable scope. Do not lower domain ceilings.
- Use opaque ordinal syntax IDs, context-specific escaping and inert labels. Test
  quotes, brackets, backticks, newlines, HTML/XML, Unicode controls/bidi/combining
  text and format directives; preserve exact source detail without normalization.
  No script/link action, remote resource, URL/file load or instruction execution.
- Evaluate actual candidate presentation/fallback specimens against T01 inventories:
  exact ordered Stage list, node/edge/gate/identity/constraint sets and negative
  controls. A keyword assertion on SKILL.md checks guidance only. A handcrafted
  specimen or offline test mapper is synthetic, never proof that the operating AI
  follows guidance. T03 compares actual host responses and retains any discrepancy;
  without that evidence, end-to-end presentation remains UNVERIFIED.

**Expected Evidence:** skill diff/digest, semantic comparison and escape results,
full fallback specimens, field/source map and explicit evaluation provenance.
Run focused affected tests; real response fidelity is witnessed under T03 authority.
**Coverage:** AC1–AC6 (AC1 contribution only); E1–E6/E8–E10;
VP-01–12/14/15; T03 owns VP-13 empirical qualification.
**Authority/rollback/handoff:** local implementation grant permits guidance and
synthetic checks, not host sessions. Keep format-neutral fallback while capability
is unknown. If no safe supported visual route is demonstrated, retain local work
and hand off gap; no support expansion. Roll back changed skill/test bytes through
reviewed source change; live installation restoration belongs to T04's gated path.

## AXM38-T03 — Empirical qualification of an existing supported host

**Result:** one exact already-supported surface qualifies visibly, or an honest
failed/unverified report and fallback identify the unmet graphical requirement.
**Scope:** Plan §4; no new platform/plugin/service, web server, credential provisioning,
remote renderer/upload, live Execution, state creation or domain mutation.
**Candidates:** Spec 011 qualification Evidence/captures and minimal host-specific
guidance in the existing Workflow skill if evidence warrants it. No renderer code
is presumed. **Dependencies:** T01 and explicit bounded external-session authority;
final comparison uses final T02 bytes. Authorization absence does not block T01/T02.

**Completion/tests:**

- Before opening a session, record grant identifying existing host/surface, permitted
  session/actions/inputs and any cost/credential boundary. Record application/build,
  OS, settings/capability, renderer/format, binary/source and installed or candidate
  skill revision/digest. No installed revision may be inferred from checkout.
- Probe already available Mermaid/SVG/native routes only within granted effects.
  Choose reversible syntax after evidence; documentation, tools or vendor names
  alone do not qualify. Existing supported surface must be established independently
  of rendering capability. Chat/cloud comparator is not silently adopted support.
- Exercise default and custom canonical-shaped inputs, sequential and independent DAG,
  Workflow→Stage→Agent, gates, identity divergence and unselected cases. Synthetic
  source stays synthetic. An authorized real read-only result, if used, is a separately
  identified capture; never rename `reviewed-delivery` to observed `teste-sdd`.
- Retain visible screenshot/capture and dated named manual witness tying host output
  to input hash/full reference and skill revision. Graphic must actually appear as
  primary presentation, with complementary text. Literal fences, source, links/files
  without visible rendering and documentation-only claims FAIL AC1. Compare actual
  visible/output facts to T01 oracle with zero missing/extra/wrong semantic facts.
- Check normal zoom/viewport, legible IDs/identity/gates and dense/long-label segmented
  detail. Account for all 496 declared edges through visible scope and complete reachable
  text; record segmentation and cross-segment references. Text reading order and
  keyboard access to any adopted controls must preserve independent access to facts;
  color alone carries no meaning. Record accessibility method and unverified limits.
- Force unknown/unavailable rendering, literal fence, clipped/empty graphic and error;
  capture complete fallback and honest limitation. Record qualified/failed/UNVERIFIED
  per exact host/build/settings/surface/format; no vendor-wide extrapolation.
- Recheck final T02 skill bytes on the qualified route; invalidate qualification if
  relevant content/input/settings drift. Failure remains Evidence, not a waived AC.

**Expected Evidence:** grant reference, input/skill manifest, actual response and
sanitized visible capture hashes, manual witness/date, semantic diff, accessibility
observations, per-host outcome and failed-render/fallback captures (E1/E7/E9).
**Coverage:** AC1/2/3/4/5/6 empirical portions; E1–E10 comparison where exercised;
VP-01/02/06/07/10/11/13/14/15 plus T02 graph/declaration checks.
**Authority/rollback/handoff:** AUTHORIZATION_REQUIRED; currently NOT_RUN and all
hosts UNVERIFIED. No session is launched in Tasks. Unsupported/failed route falls
back without installation/state changes. If no existing supported route can satisfy
AC1, report concrete evidence and only then raise material PRODUCT_BUSINESS or
ARCHITECTURE choice. Do not invent acceptance or broaden scope. Hand off exact
qualified tuple and failures; no T04 graphical PASS without valid T03 Evidence.

## AXM38-T04 — Shared packaging, compatibility and acceptance evidence

**Result:** final shared skill bytes integrate through existing Codex/Claude
mechanism with reproducible compatibility/rollback checks and complete AC/Evidence
matrix. **Scope:** Plan §§1/6–7, no new installer/version generation by implication,
release/publication, manual receipt edits or unapproved live installation.
**Candidates:** [manifest/history](../../../internal/codexruntime/manifest.go),
[shared history tests](../../../internal/codexruntime/shared_history_test.go),
[upgrade tests](../../../internal/codexruntime/domain_skills_upgrade_test.go),
[output separation tests](../../../internal/codexruntime/output_separation_test.go),
[runtime bootstrap](../../../internal/runtimebootstrap),
[skill check script](../../../scripts/test-codex-skills.sh),
Spec 011 Evidence/compatibility notes and changed installed operational guidance only.
**Dependencies:** final T02 bytes; T03 qualification for graphical completion.

**Completion/tests:**

- Integrate shared bytes identically for Codex and Claude using existing embedding,
  digest/history and receipt protocol. Append replaced revision through existing
  shared history; preserve historical bytes and other skills' ownership. No binary
  compatibility bump without demonstrated incompatibility and appropriate review.
- In isolated temporary test roots cover clean install, supported owned upgrade,
  repeated no-op, user-modified/unowned skill conflict refusal, interrupted upgrade,
  mixed inventory detection and compatible binary/skill pair rollback/refusal.
  Prove no partial success claim, silent overwrite, automatic downgrade or deletion
  of operator data; unsupported rollback must refuse safely and retain recovery facts.
  Source rollback does not prove installed rollback. Live roots require separate grant.
- Preserve JSON shape/fields, flags, exit/status, native human terminal output,
  Definition/Execution separation, selection and immutable state. Compare canonical
  state before/after isolated inspection and record no new persisted presentation.
- Run `go test ./internal/workflowdefinition ./internal/codexruntime`, affected
  bootstrap/CLI checks through `bash scripts/test-codex-skills.sh`, then `go test ./...`
  for shared packaging integration. Record script dependency warnings as limitations;
  no silent native/visual claim from unit tests. Execute repository checks appropriate
  to final changes and report CI only for actually observed exact remote head.
- Reconcile every E1–E10/AC1–AC6 row with precise source kind, hash/revision, actual
  result and missing checks. Document exact-host compatibility/fallback and supported
  rollout/rollback. Review semantic/security/escaping/provenance/install risks together.
  Independent engineering review, Tasks acceptance, later implementation authorization
  and human product acceptance remain separate gates.

**Expected Evidence:** source/embedded/temporary-installed digests for both consumers,
installation/upgrade/conflict/interruption/rollback results, canonical before/after
comparison, test logs and complete matrix linked to T03 visible captures.
**Coverage:** AC1–AC6 consolidation; E1–E10; VP-01–VP-15.
**Authority/rollback/handoff:** local temp-root tests follow implementation grant;
real install/upgrade, Git publication, merge, release and acceptance each retain their
own authority. T03 unauthorized/failed means **PARTIAL**, even with all local checks
passing; text-only mitigation does not satisfy graphical AC1. Restore only through
supported compatible pair mechanism after authorization; preserve conflicts/state.
Hand off exact candidate hashes, qualification validity and open gates for review.

## Coverage and measurable technical acceptance

This matrix refines approved Spec E1–E10 and Plan §6 without changing their meaning.
Every behavioral row below is **NOT_RUN** in this Tasks attempt. PASS requires linked
actual Evidence; proposed fixtures, code presence and documentary checks are insufficient.

| Evidence | AXM AC | VP coverage | Task ownership | Required future oracle / measurable acceptance |
| --- | --- | --- | --- | --- |
| E1 | AC1 | VP-01/02/13 | T02 input/guidance; T03 proof; T04 consolidation | Default/custom overview actually visible on ≥1 already-supported exact host; full identity and complementary text; fences/link-only FAIL |
| E2 | AC2 | VP-03/04/05/12 | T01 oracle; T02 mapping; T03 comparison; T04 | Exact Stage sequence and complete sequential predecessor sets, concurrency 1, criteria/gates equal source; zero added/omitted edges |
| E3 | AC2 | VP-04/05/12 | T01/T02/T03; T04 | Independent implementer/reviewer; both edges into integrator, no edge between roots; DAG concurrency 1 stays independent; I/O distinct from scheduling |
| E4 | AC3 | VP-06/07/08/09 | T01/T02/T03; T04 | All four reference fields/selection exact; absent/divergent/stale visible; schema-1 effort historical and schema-2 absent; zero inferred Execution/resolution/authority |
| E5 | AC4 | VP-02/11/13/14 | T02 fallback; T03 host failures; T04 | Unsupported/unknown/literal/empty/clipped/error cases retain ordered index, all predecessors, identity/gates/warnings and reachable three-level text |
| E6 | AC4/AC5 | VP-10/11 | T01 oracle; T02 offline; T03 actual response; T04 | Repeated/equivalent JSON has equal semantic facts; changed array/edge/identity changes corresponding facts; negative controls fail comparison |
| E7 | AC1/AC5 | VP-01/02/13/14 | T03 witness; T04 validation | Authorized real capture + witness/date + host/build/settings/viewport/zoom + source/input/skill identity; visible diagram and independent accessible fallback |
| E8 | AC5/AC6 | VP-07/10/15 | T01/T02 checks; T03 read-only witness; T04 state | Zero presentation writes/persisted graph/selection changes; material diagnostics/effects/provenance match source; before/after state equal |
| E9 | AC2 | VP-12/14 | T01 bounds; T02 segments; T03 readability; T04 | 32 Stages, 32 Agents, 496 dense edges all accounted for; 1 MiB global definition/snapshot limits, Unicode/16384-byte prose, 64-item lists, attempts/timeouts retained; full cross-segment predecessors |
| E10 | AC6 | VP-07/10/12/15 | T01 negative cases; T02 handling; T03 host safety; T04 | Missing/failed/partial/unknown/truncated/inconsistent/invalid inputs retain diagnostics/limits; no repaired success, satisfied gate or executable label/resource action |

T04 checks complete union AC1–AC6, VP-01–VP-15 and E1–E10. AC3 includes full
`source` despite the card's shorter identity wording. No pixel/latency budget or
new accepted product requirement is introduced. Synthetic and real source kinds
must remain separate even when they exercise the same E row.

## Decisions, preparation Evidence and handoff

`technical_decisions[]` (executor-owned, reversible; implementation remains gated):

```json
[
  {"id":"TASKS-TD-01","category":"TECHNICAL_LOCAL","chosen_option":"Four stable Tasks matching Plan packages; T03 final qualification rechecks final T02 bytes","alternatives":["Task per field/fixture/host","Serial host gate before all local preparation"],"rationale":"Actual causal dependencies permit local preparation while preserving graphical acceptance join","evidence":["plan.md#5-incremental-implementation-order--future-not-tasks"],"executor":"executor-ai","scope_impact":"none"},
  {"id":"TASKS-TD-02","category":"TECHNICAL_LOCAL","chosen_option":"tasks.md plus compact preparation receipt; update owning Specifications index only","alternatives":["Additional Evidence file and mirrored task statuses","Rewrite approved Spec/Plan headers"],"rationale":"Receipt holds observed preparation checks; index owns lifecycle; approved bytes and historical receipts remain immutable","evidence":["../../documentation.md","../../../.agents/skills/axiom-sdd/SKILL.md"],"executor":"executor-ai","scope_impact":"none"},
  {"id":"TASKS-TD-03","category":"TECHNICAL_LOCAL","chosen_option":"Manual expected fixtures plus semantic output comparisons; actual host response checked separately","alternatives":["Keyword-only skill assertions","Test-only mapper treated as runtime proof"],"rationale":"Avoid circular oracle and false visual acceptance; defer reversible syntax until supported-host evidence","evidence":["plan.md#2-transient-model-and-deterministic-mapping","spec.md"],"executor":"executor-ai","scope_impact":"none"}
]
```

No delegation: strict lower capability hierarchy is not proven; tightly coupled
four-package reconciliation is performed directly. No material architecture/product/
design decision is needed to finish Tasks. Missing host evidence is an EVIDENCE_GAP;
opening a real session is AUTHORIZATION_REQUIRED, not a vote on local test structure.

Current delivery is local PROPOSED Tasks and preparation receipt; no effective grant
for this attempt's Git commit/push was established. Earlier Specify/Plan publication
authority is checkpoint-bound. The single requested Linear callback is separately
authorized. It reports PARTIAL publication and exact local hashes, not a GitHub file
URL that does not yet exist. No canonical Execution is created or implied.

**Next action:** review these exact local Tasks; authorize commit/push of the Tasks,
receipt and bounded index delta to this existing branch/PR when ready. Re-read remote
head/diff and run applicable pre-commit checks before publication; reconcile drift,
never reset/force-push. Next gate remains `review_tasks` and explicit human Tasks
acceptance; implementation and host sessions require later independent authority.

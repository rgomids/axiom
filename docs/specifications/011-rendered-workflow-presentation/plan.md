# Specification 011 Plan — Rendered Workflow presentation (AXM-38)

## Status, authority and sources

**PROPOSED — Plan r1, 2026-10-22.** Documentation only; no Tasks, implementation,
renderer, CLI/skill edits, native tests, merge or release are authorized here.
This Plan implements the semantic baseline of [Specification 011 r1](https://github.com/rgomids/axiom/blob/c2298a9ae60d32ecbfea96c19a051a7514fe78b6/docs/specifications/011-rendered-workflow-presentation/spec.md),
Git blob `21705f315bbaa2365ac1078ad1bc868f74725555`, explicitly
[accepted by Rafael](https://github.com/rgomids/axiom/pull/336#issuecomment-6333507483).
VP-01–VP-15 and P-01–P-05 are ACCEPTED only within that revision's conditions;
P-05 does not accept a platform or prove compatibility. Editorial e1 of
[local Spec](spec.md) remains PROPOSED pending exact reconfirmation.

Resume: clean worktree, local/PR HEAD `c2298a9ae60d32ecbfea96c19a051a7514fe78b6`,
[PR #336](https://github.com/rgomids/axiom/pull/336) OPEN/Draft, branch
`axm-38-specify-rendered-workflow`, base `main` at
`2e53228bfd27cff206c5c955f363bfff18fe745a`. No HEAD delta to reconcile.
Linear AXM-38 remains In Progress, blockedBy empty; AXM-39/AXM-12 remain downstream.
No canonical Execution identity/state was obtained or inferred from those facts.
No current explicit commit/push grant was found: previous Specify publication
was bounded to that checkpoint. This attempt supplies local reviewable artifacts;
GitHub publication and CR-001 closure there remain pending authority.

Accepted dependencies: [Spec 007](../007-configurable-workflows/spec.md#stage-and-agent-semantics-wf-005006),
[ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md),
[ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) and its
[valid amendment](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md).
Use current amendments, including prospective proof-retirement annotations, without
implementing adjacent AXM work. Governance: [tracking policy](../../development/linear-github-issue-tracking.md),
[versioned SDD flow](../../../.agents/skills/axiom-sdd/SKILL.md) and
[constitution](../../product/constitution.md). No separate versioned SDD playbook
was found by bounded path/content search; the routed canonical SDD skill is used.
[Plan Evidence](evidence-plan.md) owns actual checks and hashes;
[plan receipt](completion-receipt-plan.json) owns the callback checkpoint.

## 1. Boundary and affected components

Target flow: exact read-only `axiom --json project workflow show` result →
checked canonical facts → transient presentation inventory → supported host
visual plus complementary text, or complete text fallback. One invocation,
one canonical JSON result; no formatting LLM, runtime hop or mutation replay.

| Owner / inspected path | Planned impact; preserved contract |
| --- | --- |
| `internal/cli/workflow_authoring.go`, `cmd/lingo/workflow_authoring.go`, `internal/projectapp/workflow_authoring.go` | Read/fixture contract sources only. Keep existing flags, show result, diagnostics, wire schema, exit/status and terminal text unchanged |
| `internal/workflowdefinition/{definition.go,validation.go,workflow.schema.json,workflow-v2.schema.json}` | Read/fixture bounds and semantics only; no new graph type, decoder, digest or persisted schema in Lingo |
| `internal/codexruntime/skills/axiom-workflow/SKILL.md` | Primary future edit: replace optional visualization guidance with accepted three-level presentation and explicit fallback/qualification rules, retaining JSON, discovery and authority rules |
| `internal/codexruntime/skills/axiom-project/SKILL.md` | Inspect selection handoff; change only if actual duplicate presentation instructions conflict. Selection stays Project-owned; no new operation |
| `internal/codexruntime/{runtime.go,manifest.go,*upgrade*_test.go,output_separation_test.go,workflow_skill_content_test.go}` | Existing shared skill packaging/history and synthetic upgrade checks for Codex/Claude; exact touched bookkeeping depends on final skill bytes, not a new compatibility generation by default |
| Host's existing presentation surface | Own visual syntax/layout and limitations; never domain validation, scheduling, persistence, Runtime resolution or consent |
| Spec 011 folder / installed operational documentation sources | Bounded Evidence, qualification results and any changed user guidance; preserve published historical fixtures/receipts |

No new dashboard, graph engine, service, upload, plugin, dependency, supported
runtime, state store or CLI command is adopted. Maintainer `.agents/skills` are
not installed product skills and are not a visualization implementation target.

## 2. Transient model and deterministic mapping

**PROPOSED technical design (VP-03–10/12/15; accepted P-02).** A logical in-memory
view, not a new serialized API, cache or application/domain object. Keep immutable
source JSON available for detail. Minimal inventory:

- Identity: `workflowAuthoring.reference.{workflowId,revision,digest,source}`,
  `selection` as full reference or absent, `projectId`, `projectRevision` separately.
  Compare all four reference fields; never infer selected from workflowId alone.
- Ordered Stage entries: source index/path, ID, purpose, phase/checkpoint, mode,
  concurrency, gates with kind/timing, pointers to inputs/outputs/criteria/validators,
  failure policy and full instructions. Initial summaries do not replace source.
- Per-Stage Agent entries: scoped ID/path, role, declared dependencies, responsibility,
  I/O, complexity, runtimeConstraints, profileRef, historical schema-1 effort,
  capabilities/effect ceilings, timeout/attempts and integration/validation ownership.
- Separate relation kinds: Stage progression; same-Stage Agent dependency;
  I/O provenance. Gates annotate Stage boundaries; no synthetic executable node.
- Material canonical status/diagnostics/prerequisites/effects/provenance retained
  alongside view; presentation limitations kept distinct from canonical results.

Transformation procedure, shared by visual/text semantic oracles:

1. Check complete event/target/status and supported schema before using definition.
   Preserve failure/partial/unknown, missing payload, diagnostics and provenance.
   No successful graph for malformed/contradictory identity; no guessed repair.
   Structural presentation checks do not constitute canonical domain validation.
2. Copy exact reference and declaration facts. Definition has no self-digest;
   take digest/source from `reference`. Never compute a replacement identity from
   prose or a projected subset. Absent selection remains not reported.
3. Traverse `stages` in array order, retaining ordinals. Overview arrows mean
   ordered progression only. Do not draw a general inter-Stage scheduling DAG.
4. Within each Stage, retain Agent array order for stable indexing; enumerate every
   declared `dependsOn` as predecessor → Agent, scoped to that Stage. No transitive
   reduction, inferred chain or cross-Stage Agent edge. In sequential mode preserve
   all declared edges, including required immediate predecessors, concurrency 1.
   In DAG mode preserve independence even when concurrency 1 serializes execution.
5. For layout only, use stable topological layers (ties by source ordinal); enumerate
   predecessor text in canonical source order. A cycle/dangling/self/duplicate edge
   triggers an honest presentation limitation, not rewritten successful topology.
   Complexity O(V+E) per Stage; layout ordering never changes source order/identity.
6. Assign internal syntax tokens by Stage/Agent ordinals, never raw labels. Escape
   label text for the selected host grammar and text context. Retain exact source
   IDs in text. Derive both views from the same inventory, then compare node/edge,
   identity, gate and constraint sets; on incomplete visual output retain full text.

Determinism means identical semantic inventory, identity, ordering, relations and
material truth for equivalent JSON; prose/layout may vary. Pure source mapping
and fixture oracle can be deterministic without pretending host synthesis is.
No production formatter subprocess/helper is introduced: current skill says
invoke only `axiom` for executable domain behavior. Mapping lives with host guidance;
synthetic test projection is an oracle, never a second runtime source of truth.
Native output must later be compared to that oracle; instruction-string tests
alone cannot prove transformation/fallback behavior.

## 3. Disclosure, bounds and security

**ACCEPTED behavior, PROPOSED layout:** Workflow overview shows full identity and
selection relationship, Definition label, all Stage IDs in order, purpose summaries,
mode/agent count and gate markers. Stage request by stable ID shows its complete
Agent DAG, concurrency, gate timing, criteria/validators and I/O provenance. Agent
request by scoped ID exposes full declared configuration and ownership; instructions
remain explicitly available, never silently lost. No new click UI is required.
Follow-ups retain exact Workflow digest; if source unavailable or changed, re-read
and disclose changed identity before combining details. No automatic selection.

Fallback gives full reference, ordered Stage index, per-Stage mode/concurrency,
complete Agent predecessor lists (including “none”), gates and material warnings.
Declare graphic unavailable/unknown/failed; literal fence, download link or empty
image is not rendering success. Text alone must support all three levels. Use
visible relation labels and reading order; color never carries exclusive meaning.
Later qualification checks normal zoom, keyboard access for any chosen controls,
and text/screen-reader order. No mandatory visual control may hide canonical facts.

Source limits verified against both current schemas and Go validation: 1–32 Stages,
1–32 Agents each (1024 total), concurrency 1–32 and no greater than Agent count;
sequential concurrency 1. At most 1 MiB definition/canonical snapshot, 16384 UTF-8
bytes for bounded prose (Schema character bounds supplemented by Go byte check),
IDs up to 64 ASCII characters, other lists at most 64, attempts 1–10 and timeout
1–86400 seconds. A dense valid 32-node DAG can have 496 edges; across 32 such
Stages, 15872 edges, subject to aggregate byte limits. Never promise all maxima
with maximum-length prose simultaneously fit the 1 MiB bound.

Use complete index plus contiguous ordered overview segments, one Stage detail at
a time. Dense Stage diagrams may segment by stable ordinal/topological layer;
include segment position, continuation endpoints and complete predecessor text,
including cross-segment references. No fixed pixels/nodes-per-page chosen without
host evidence. Long Unicode stays exact in detail/text, wrapped or explicitly
summarized in visual labels. If host output limits prevent complete detail, report
which scope remains unavailable and offer bounded continuation from same source;
do not claim complete/legible acceptance. No lowered schema ceiling or silent edge loss.

Labels/instructions are untrusted data, never commands or current-session authority.
Escape quotes, brackets, backticks, newlines, HTML/XML delimiters and grammar-specific
metacharacters; test Unicode/control handling without normalization of source facts.
Disable embedded link actions, remote resources, script/event handlers, file/URL
loading and style/directive injection in any future adopted format. No remote
renderer or external resource fetch based on definition content. Effort in schema 1
is historical intent; schema 2 gets no synthesized effort. Eligibility is not resolved
Runtime/model/Profile; gates are declarations, not satisfied decisions; success of
show is inspection success, never Execution, capability, validation or effect consent.

## 4. Host investigation and qualification protocol

Documentation rechecked 2026-10-22; every empirical cell remains **UNVERIFIED**.
Bounded official findings (not Axiom qualification):

| Surface | Documentation / candidate | Route and evidence limit |
| --- | --- | --- |
| Codex Desktop | [App-feature URL](https://developers.openai.com/codex/app/features) redirects to general features; [Visualizations](https://learn.chatgpt.com/docs/visualizations) describes ChatGPT surfaces | Exact Codex build/format unestablished. Candidate native diagram or safe local image only if that existing surface supports it and bounded use is authorized; text meanwhile |
| Codex CLI / IDE | Visualizations explicitly excludes these surfaces | Preserve terminal and text. No claim that all possible image/diagram mechanisms are impossible; no graphical acceptance observed |
| Claude Code terminal | Existing supported maintainer/product Runtime does not imply graphical terminal | Text first when capability unknown; literal Mermaid never qualifies |
| Claude Code Desktop | [Desktop docs](https://code.claude.com/docs/en/desktop#preview-your-app) describe Browser preview and opening local HTML/PDF/video paths | Candidate local preview is documented, not proof of inline diagram. Do not build a web app/server or adopt preview as the accepted transport here |
| Claude artifacts / chat surfaces | [Artifact docs](https://support.claude.com/en/articles/17153992-what-are-artifacts-and-how-do-i-use-them) distinguish artifact panel/session publication and prerequisite capabilities | Research comparator only; cloud execution/private URL is an external effect, not an accepted substitute host or support expansion |

Format comparison: Mermaid is concise and inspectable but host may show only source;
SVG permits escaping/stable local layout but inline support and safe delivery are
unknown; native host diagrams may fit conversation but are surface-specific and
need semantic checks. **PROPOSED:** prefer an already available inline host surface
with no new transport/effects after qualification; otherwise fall back. No format
is declared ACCEPTED or selected by documentation. This is an evidence prerequisite,
not a human decision about a reversible syntax choice.

Later protocol (all **NOT_RUN**, real sessions **AUTHORIZATION_REQUIRED**):

1. Obtain explicit bounded authority naming existing supported Codex/Claude surface,
   session/effects, synthetic/public inputs and any cost/credentials. Record actual
   host identity/build/settings/OS, binary and installed skill revision. Do not use
   `teste-sdd` as an observed fixture or create a Project/Workflow to manufacture it.
2. First use public sequential/DAG canonical-shaped synthetic events, identified as
   synthetic, then authorized real read-only show if required. Record complete input
   identity/hash and source; no live Execution or mutation. No credentials in Evidence.
3. Try smallest available safe host presentation; record format/tool and exact
   limits. A screenshot must show diagram actually visible in conversation/supported
   surface, with identity and matching nodes/edges. A preview/file route that only
   offers a link fails VP-01; do not redefine primary display to make it pass.
4. Exercise Workflow → Stage → Agent, independent nodes, gates, dense/long labels,
   normal zoom and text accessibility; force missing renderer, literal fence, failed
   image and unknown capability. Compare semantic output against the offline oracle.
5. Retain sanitized screenshot/capture and manual witness/date, viewport/zoom,
   renderer/skill/source revisions, input digest, failures and fallback text. Mark
   each exact host/format qualified, failed or UNVERIFIED; never generalize by vendor.
6. At least one supported host must meet VP-01/E7 for product acceptance. Others
   need honest documented fallback. If existing hosts demonstrably cannot meet it
   within scope, retain fallback and bring a concrete PRODUCT_BUSINESS/ARCHITECTURE
   choice: adopt bounded support/transport vs explicitly revise product requirement.
   Evidence precedes that decision. No new ADR unless material conflict is proven.

## 5. Incremental implementation order — future, not Tasks

All increments below are **PROPOSED / NOT_RUN**, outside this execution's authority.
They are bounded work packages for a later Tasks gate, not newly created Tasks.

1. Pin canonical event fixtures and expected semantic inventories; inventory active
   shared skill packaging/consumers. Establish regressions for sequential, DAG,
   identity/absence/failure and source bounds before altering presentation guidance.
2. Implement minimal shared host mapping and complete text/disclosure rules in the
   existing Workflow skill, plus synthetic semantic projection/escape/coverage checks.
   Depends on 1; no application/CLI graph renderer. Extend current testing location,
   avoiding a standalone validator service or durable script unless actually needed.
3. Qualify an existing graphical surface under separate bounded authorization;
   select its reversible safe syntax from actual evidence, add minimal host guidance
   and fallback cases. Depends on 1; can inform 2 before final packaging. If no host
   qualifies, retain completed safe mapping work; graphical AC remains unmet.
4. Package shared skill bytes through existing Codex/Claude upgrade mechanism;
   verify conflict/modified-install refusal, interrupted upgrade and compatible pair
   rollback. Depends on 2/3 for final content. Reconcile compatibility documentation,
   actual native results and all E1–E10 Evidence before independent review/acceptance.

Do not split every field/host/fixture into a separate work item. No dependency on
AXM-39, AXM-12, Runtime capabilities research or unrelated packaging work is added.
If a proven coupling appears, record exact source and impact before expansion.

## 6. Verification design and acceptance traceability

Future deterministic checks use canonical event envelopes, not definitions alone.
Existing [default r1](../007-configurable-workflows/examples/default-sdd-r1.json),
[default r2](../007-configurable-workflows/examples/default-sdd-r2.json),
[custom r1](../007-configurable-workflows/examples/custom-r1.json) and custom r2 seed
fixtures. Wrap using actual `workflowAuthoring`/completion shapes; explicitly label
synthetic wrappers. Validate future fixtures through existing definition validation,
not a competing Python JCS implementation. Preserve original reference digests.

| CARD AC / accepted Spec evidence | Future deterministic oracle and retained evidence |
| --- | --- |
| AC1 → E1/E7; VP-01/02/13 | Authorized real visible supported-host witness and screenshot, input/skill/renderer identity; literal fence/link-only is FAIL. Documentation is not oracle |
| AC2 → E2/E3/E9; VP-03/04/05/12/14 | Exact Stage order/Agent node and edge sets; multi-agent sequential, independent roots and integrator join from custom fixture; DAG concurrency 1; all 496 dense edges, 32 Stages/32 Agents and 1 MiB boundary. No transitive reduction |
| AC3 → E4; VP-06/07/08/09 | Full reference equality vs differing source/revision/digest/selection, absent selection, project vs Workflow revision; schema-1 effort vs schema-2 absence; no resolved Runtime or active Execution inferred |
| AC4 → E5/E6; VP-02/11/13/14 | Text retains facts through all levels and segmentation; graphics disabled, literal fence, clipped/empty graphic, renderer error; independent text reading/keyboard path |
| AC5 → E6/E7/E8; VP-10/11/15 | Repeated identical JSON, object-key/whitespace variants yield same semantic inventory; array/edge/identity changes alter corresponding facts. Native capture compared to oracle; pure oracle PASS alone cannot qualify host |
| AC6 → E8/E10; VP-07/10/12/15 | Input/state unchanged, no persistence/mutation; missing/failed/truncated/unknown schema, dangling/cycle, unsafe labels/markup; exact diagnostics/limitations and authority preserved |

Add long Unicode/escaping, 64-item list coverage, 1/10 attempts and 1/86400 timeout,
full-detail retrieval and cross-segment predecessors. Invalid fixtures assert refusal
without repaired graphs; no domain limits changed. Fixture boundary tests establish
transformation safety, not a new product performance/latency budget.

Likely later commands: `go test ./internal/workflowdefinition ./internal/codexruntime`
for affected definition fixtures/skill installation; `bash scripts/test-codex-skills.sh`
after reviewing its actual current assertions. Add focused semantic output evaluation
alongside existing tests; literal content assertions only check packaging/guidance.
Broaden to `go test ./...` and repository CI when implementation touches shared
packaging. Native sessions have their own authority and evidence, never inferred
from Go tests or a stub host. No such implementation tests run in this Plan stage.

## 7. Compatibility, rollout, rollback and risks

Existing typed JSON and native human terminal modes remain identical. No Workflow,
Project or Runtime state migration. Inventory both Runtime installations and current
skill receipts before a later authorized upgrade; do not edit installed skills or
canonical receipts by hand. Source checkout is not installed inventory. Preserve
historical skill fixtures/history; register final content hashes with the existing
shared history mechanism. A wire/binary compatibility bump needs a demonstrated
incompatibility, not visual prose alone. Other three domain skills retain ownership.

Stage rollout: synthetic checks → authorized exact-host qualification → shared
skill package tests → review/exact human approval → separately authorized install.
Rollback restores a known compatible binary/shared-skill pair by supported install
mechanism; preserve user-modified skill conflicts and all state. Disabling visuals
retains complete text and discloses loss of graphical AC; it is mitigation, not
full product success. No automatic downgrade, state rewrite or remote publication.

Concrete risks: host features/builds differ (qualify exact surface); model omits edges
(compare complete semantic sets); dense graphs clip (segments plus complete text);
labels inject syntax/actions (opaque syntax IDs, escaping/no resources); stale
follow-up mixes revisions (full identity re-read); installer leaves mixed skills
(existing interruption/compatibility checks). No new logging subsystem; sanitized
Evidence records canonical identities, checks, limitations and empirical failures.
No new mandatory requirement beyond CARD/ACCEPTED is introduced by test technique.

## 8. technical_decisions[] and gates

Local choices below are executor-owned, reversible, **PROPOSED implementation design**,
not accepted architecture/product changes. Verification is source/documentary only.

```json
[
  {"id":"PLAN-TD-01","category":"TECHNICAL_LOCAL","owner":"executor-ai","chosen_option":"Host-owned transient semantic projection; preserve typed JSON and terminal","alternatives":["Go application graph/renderer","Persisted intermediate schema"],"rationale":"ADR-0030 assigns host presentation; existing WorkflowReport contains required declarations","verification":"PASS: source inventory and accepted boundary comparison; behavioral NOT_RUN","links":["../../../internal/projectapp/workflow_authoring.go","../../decisions/0030-canonical-results-terminal-and-host-presentation.md"],"scope_impact":"none"},
  {"id":"PLAN-TD-02","category":"TECHNICAL_LOCAL","owner":"executor-ai","chosen_option":"Stable ordinal indexing, complete declared edges and predecessor fallback; O(V+E) layering","alternatives":["Agent array interpreted as chain","Transitive reduction","One image for entire definition"],"rationale":"Preserves independence/all edges at verified 32/32 bounds with bounded detail","verification":"PASS: schemas/validation and fixture inspection; algorithm/host tests NOT_RUN","links":["../../../internal/workflowdefinition/validation.go","../007-configurable-workflows/workflow-v2.schema.json"],"scope_impact":"none"},
  {"id":"PLAN-TD-03","category":"TECHNICAL_LOCAL","owner":"executor-ai","chosen_option":"Qualify existing surface before selecting diagram syntax; unknown uses text","alternatives":["Assume fenced Mermaid renders","Adopt cloud artifact/platform from documentation"],"rationale":"Official documentation does not establish exact supported-host rendering; P-05 is conditional","verification":"PASS: official pages opened; empirical UNVERIFIED / NOT_RUN","links":["https://learn.chatgpt.com/docs/visualizations","https://code.claude.com/docs/en/desktop#preview-your-app"],"scope_impact":"none"}
]
```

- **COMPLETED:** documentary Plan and local CR-001 editorial patch, source/schema
  investigation and checks recorded in Evidence. No behavioral AC declared complete.
- **AUTHORIZED_PENDING:** review this local Plan/editorial checkpoint; no further
  implementation work authorized in this attempt.
- **AUTHORITY_REQUIRED:** commit/push/update only PR #336; exact editorial Spec
  reconfirmation; exact Plan human approval; later Tasks/Implement and bounded real
  host sessions. `AUTHORIZATION_REQUIRED` for external effects differs from a
  material design decision; none is currently proven necessary to finish the Plan.
- **TECHNICALLY_BLOCKED:** none for documentary planning. Rendering capability is
  UNVERIFIED, not proven impossible; publication pending authority is not a defect.
- **NOT_RUN:** implementation, deterministic transformation/renderer tests, native
  host sessions/screenshots, new CI, merge/release/product acceptance.

Next gate: `review_plan` plus re-review/reconfirmation of exact editorial e1 Spec.
CR-001 is locally patched, **PARTIAL / pending publication**, not closed on GitHub.
No new ADR, Tasks, Execution or automatic acceptance. Handoff must preserve exact
hashes and obtain publication authority before pushing these files to the same PR.

# Specification 011 — Rendered Workflow presentation

## Status, provenance and authority

**Revision 1, editorial checkpoint e1 — PROPOSED reconciliation, 2026-10-22.**
Behavioral baseline: accepted r1 at `c2298a9ae60d32ecbfea96c19a051a7514fe78b6`;
this edited revision requires exact re-review and human reconfirmation. Product owner:
[AXM-38](https://linear.app/rgomids/issue/AXM-38/apresentar-workflow-como-diagrama-renderizado-com-texto-complementar).
The user explicitly authorizes translating this card's product requirements into
repository documentation; no unrelated provider content is imported.
The original Specify operational gate was `READY_FOR_specify`, comment UUID
`1c7a61cd-01b9-4386-a57f-e6ba0711280f`. Prompt publication and dispatch acceptance
are not canonical Execution start/completion. This document records no Execution
identity or transition. Linear status and downstream AXM-39/AXM-12 remain untouched.

Source baseline: `2e53228bfd27cff206c5c955f363bfff18fe745a`, clean detached HEAD
before authoring. [Evidence](evidence-specify.md) records source checks and limits.
The original pre-publication checkpoint is recorded in the immutable historical
[completion receipt](completion-receipt.json) and [Specify Evidence](evidence-specify.md).
Its `pr_url: null`, uncommitted state and then-next gate describe that original
observation, not present publication. Revision 1 was subsequently published in
[PR #336](https://github.com/rgomids/axiom/pull/336) at
`c2298a9ae60d32ecbfea96c19a051a7514fe78b6`, blob
`21705f315bbaa2365ac1078ad1bc868f74725555`, and received
[explicit human approval](https://github.com/rgomids/axiom/pull/336#issuecomment-6333507483)
for VP-01–VP-15 and P-01–P-05 within that exact text's conditions.
That approval authorizes preparing the Plan and editorial CR-001 correction;
it does not transfer acceptance to these edited bytes or establish native rendering,
implementation, merge, release or product acceptance. See [Plan Evidence](evidence-plan.md)
for exact hashes, editorial diff and publication/authority limits.

Requirement origins are explicit:

- `CARD`: behavior requested by AXM-38 and this Specify instruction; proposed
  specification of existing product intent, not evidence of delivery.
- `ACCEPTED`: invariant already owned by an accepted contract; preserved here.
- `PROPOSED`: new refinement or compatibility/design option; recommendation only,
  **not a MUST or accepted AC until explicit material approval**.

Origin and PROPOSED labels below preserve the r1 drafting vocabulary. Subsequent
human approval applies only to the exact r1 baseline linked above; editorial e1
remains pending reconfirmation. No requirement or refinement text is changed.

## Problem, objectives and non-goals

The card reports that examining `default-sdd` and then `teste-sdd` produced only
stage lists. Operators need a rendered diagram as the primary presentation on a
capable, tested host, backed by useful text. A fenced Mermaid source block alone
does not satisfy the visual requirement. The report is product context, not a
reproduced observation of either installed definition in this session.

Actors: operator inspecting a Project's Workflow, AI consuming canonical JSON,
host presenting the response, and reviewer checking fidelity and visual evidence.
Objectives: understand the ordered Workflow, inspect one Stage's actual agent
dependencies/gates, then examine Agent configuration without a JSON dump.
Definition inspection remains read-only and conveys no execution authority.

Non-goals: renderer/CLI/skill implementation in Specify; a graphical engine in
Lingo; dashboard, new interface or infrastructure; another canonical format or
persisted graph; new Workflow semantics; arbitrary inter-Stage DAG scheduling;
extra formatting-only LLM call; credential provisioning, host publication or
new supported platforms. AXM-32 and accepted historical Evidence remain intact.

## Accepted sources and observed contracts

1. [Spec 007](../007-configurable-workflows/spec.md), accepted 2026-10-09:
   ordered Stages, one active Stage, bounded same-Stage agent graph, gates,
   immutable revision identity and explicit selection. Historical baseline
   narrative in that document is not proof of present implementation.
2. [ADR-0030](../../decisions/0030-canonical-results-terminal-and-host-presentation.md),
   accepted 2026-10-18: domain/application own structured facts; JSON remains
   compatible; Lingo owns native terminal text; existing AI owns host presentation.
   Operation success, readiness, effects and human acceptance remain distinct.
3. [ADR-0024](../../decisions/0024-model-profile-effort-ownership.md) and
   [accepted #306 amendment](../007-configurable-workflows/amendment-306-profile-effort-onboarding.md):
   schema 2 forbids Workflow effort overrides; Model Profile owns effort.
   Historical schema-1 effort is not silently reinterpreted or copied to schema 2.
4. [Spec 008 acceptance index](../README.md) establishes acceptance despite its
   historical Proposed header. AXM-32 is operationally Done in Linear; that does
   not prove AXM-38 rendering support or real-host behavior.
5. Current [CLI parser](../../../internal/cli/workflow_authoring.go),
   [application report](../../../internal/projectapp/workflow_authoring.go),
   [definition types](../../../internal/workflowdefinition/definition.go),
   [schema 1](../../../internal/workflowdefinition/workflow.schema.json),
   [schema 2](../../../internal/workflowdefinition/workflow-v2.schema.json) and
   [commands](../../commands.md) establish available flags and field names.

### Read-only command syntax

Confirmed by source inspection, not an installed-CLI smoke test:

```text
axiom --json project workflow list --project <uuid-or-slug>
axiom --json project workflow show --project <uuid-or-slug> --workflow <workflowId> --revision <n> --digest <sha256> --source <builtin|project>
```

Take the complete exact reference from `list` or a canonical selection; never
invent a digest or choose the largest revision. `show` defaults source to `project`
in the parser; examples make it explicit to avoid confusing built-ins and custom
definitions. Both `default-sdd` revisions 1 and 2 are exposed in the current
producer. The requested `teste-sdd` name is not a repository fixture and its
installed revision/content has **not** been observed. Do not create it to answer
an inspection request. `axiom workflow list` lists Executions, not definitions.

The completion event contains `workflowAuthoring`: `definition`, `reference`,
`selection`, `projectId`, `projectRevision`, available/retired revision references,
and applicable diagnostics/prerequisites/effects. `definition` has no self-digest;
the exact digest/source come from `reference`. Optional omissions remain omissions.
Do not add fields to the wire contract to support presentation.

## Journeys

- **J1 — default:** operator asks to inspect `default-sdd`. AI obtains the exact
  canonical reference and definition, shows its revision/selection and ordered
  Stage overview as a rendered diagram on a verified capable surface, with concise
  text. It offers Stage-level follow-up without starting an Execution.
- **J2 — custom:** operator asks to examine `teste-sdd`. Use its observed reference,
  not assumptions based on its name or the default. Explain sequential versus DAG
  inside each Stage. Missing definition yields the canonical failure and next safe
  read, not a fabricated ten-stage diagram.
- **J3 — detail:** operator names Stage ID, then Agent ID. Keep Workflow identity
  visible, expand relevant gates, dependencies, inputs/outputs and configuration.
  Independent agents remain independent; follow-up does not execute their tasks.
- **J4 — fallback:** visual capability absent, unknown or rendering fails. Explain
  the limitation briefly and provide accessible ordered stages and same-Stage
  adjacency text. The inspection stays useful without images, color or a plugin.

## Requirements and invariants

| ID | Origin | Observable behavior / preserved invariant |
|---|---|---|
| VP-01 | CARD | On a capable and tested supported host, primary overview is an actually rendered diagram with concise supporting text. Source code, fences, a download link alone or a claim of rendering do not qualify. At least one supported host needs real visual acceptance evidence. |
| VP-02 | CARD | Offer three conceptual levels: Workflow overview; Stage detail with gates/dependencies; Agent roles/configuration/bindings. Avoid dumping JSON or every instruction in the initial answer. |
| VP-03 | ACCEPTED | Stage sequence equals definition array order; only one Stage can be active in an Execution. Overview edges denote ordered progression, not arbitrary scheduling or inferred inter-Stage concurrency. |
| VP-04 | ACCEPTED | Agent dependency edges derive only from `dependsOn` within their Stage. `sequential` has concurrency 1 and immediate predecessor dependencies; preserve all declared dependencies. `dag` permits independent nodes. Array order does not create an edge. |
| VP-05 | CARD | Show Stage mode and declared concurrency; distinguish potential parallelism from observed simultaneous execution. Independent nodes are visibly independent, even when concurrency is 1. Explain informative inputs, outputs, criteria and gates. |
| VP-06 | ACCEPTED | Identify `workflowId`, `revision`, full `digest`, `source` and exact selection relationship. Compare complete references; inspected definition may differ from selected definition. Missing selection is not an inferred default. Project revision is distinct from Workflow revision. |
| VP-07 | ACCEPTED | Label inspection as Definition. `show` success means successful inspection, not Stage success. No active Stage, progress coloring, completed gate, resolved Runtime or Execution status without facts from the appropriate canonical Execution source. |
| VP-08 | ACCEPTED | `runtimeConstraints` express declared Runtime eligibility, not resolved Runtime/model/Profile or proof of capability. `profileRef`, including `policy-default`, is declared intent. Resolved details require matching canonical Execution/Stage Plan facts and their provenance; inspection alone cannot supply them. |
| VP-09 | ACCEPTED | Role, complexity, effort, Runtime and Model Profile remain distinct. Schema-1 effort is historical declared intent; schema-2 absence is not missing configuration to repair. Capabilities/effect ceilings are requirements/limits, never observed capability, consent or performed effects. |
| VP-10 | CARD | Derive the minimum presentation information in memory from canonical JSON. No parallel persistence/source of truth; no invented effect, status, dependency, validation outcome or decision. Repeated equivalent inputs yield equivalent semantic graph/fallback. |
| VP-11 | CARD | Text remains usable when rendering is unavailable. It preserves identity, sequence, independent agents, prerequisites/gates and material warnings, rather than repeating an unreadable code fence. |
| VP-12 | ACCEPTED | Human gates retain `kind` and `timing` and required evidence/authority semantics. A declaration is not a satisfied gate. Inputs referencing prior Stage outputs denote data provenance, not permission for arbitrary inter-Stage graph execution. |
| VP-13 | CARD | Record host/surface/version, format, empirical outcome and fallback separately; document unknown rendering as `UNVERIFIED`. Documentation or exposed tools alone never establish rendered acceptance. |
| VP-14 | CARD | Keep bounded definitions readable without silently dropping nodes, dependencies, gates or identity. Full configuration remains retrievable through proportional detail. |
| VP-15 | ACCEPTED | Presentation cannot mutate selection, schema, CLI flags/exit codes, domain authority, validation results or lifecycle. Human/JSON/host presentation keep material truth under ADR-0030. |

### Proposed disclosure and derived information (not a new wire schema)

**P-01 — PROPOSED, DESIGN refinement:** initial answer shows all Stage IDs in
order, purpose summaries, mode/agent count and gate markers, plus concise text
describing notable constraints. Requesting a Stage expands its full same-Stage
graph and criteria; requesting an Agent expands responsibilities, I/O references,
dependencies, declared Runtime/Profile constraints, capability/effect ceilings,
timeout/attempt limits and integration/validation ownership. Instructions remain
available on request; omission is explicit. Conversation by stable ID is sufficient;
no click interaction, panel, new command or dashboard is required.

**P-02 — PROPOSED, presentation design:** derive only identity/selection context,
ordered Stage labels, per-Stage mode/concurrency, Agent IDs/labels, declared edges,
gate timing and pointers to source detail. Distinguish three relationship kinds:
Stage progression, agent dependency and input/output reference. Gate markers
attach to their Stage boundary and are not invented executable Stages. This is a
conceptual information inventory, not serialized JSON, API, new domain objects,
graph digest, cache or persistence contract. Preserve source paths for traceability.
The same canonical facts feed visual and text views. Layout and prose may vary;
identity, relationships, ordering and material facts may not.

**P-03 — PROPOSED, DESIGN/legibility:** wrap or split an overview into explicitly
ordered contiguous Stage segments when it cannot fit legibly. Retain a complete
Stage index, segment position, continuation endpoints and Workflow identity. Show
one Stage's agents at a time; for dense graphs use labelled detail segments and
complete predecessor lists, including cross-segment references. Never reduce an
independent set to a chain or hide an edge without a reachable textual equivalent.
No fixed pixel, latency or node-per-page budget is adopted without host evidence.

At current schema revisions: 1–32 Stages, 1–32 Agents per Stage, concurrency
1–32 (sequential: 1), definition/canonical snapshot at most 1 MiB; bounded strings
at most 16384 UTF-8 bytes. These are domain ceilings, not a promise that a single
image containing 1024 Agents will be readable. Do not lower accepted domain limits
to suit a renderer. Validate future acceptance at 32 Stages, 32 Agents in one
Stage, dense valid DAG and long Unicode labels; preserve detail through navigation.
Bounds must be rechecked against the schema/compiler revision used for delivery.

Fallback example vocabulary, not an observed result: “Definition X, revision R,
digest D; selected: yes/no/not reported. Stages: 1 …; 2 … . Stage S: DAG,
concurrency 2. A: predecessors none; B: predecessors none; C: predecessors A, B.
Gate before S: implementation authority required; satisfaction not observed.”
Use actual canonical IDs/values; no placeholder may masquerade as captured evidence.

## Failure cases, accessibility and security

| Condition | Required handling and provenance |
|---|---|
| Missing/failed `show`, missing definition, unknown schema or inconsistent identity | Preserve category/diagnostics and explain why a reliable diagram is unavailable; do not repair the definition or render a guessed successful Workflow (VP-07/10/15). |
| Stale reference/selection, distinct revisions in follow-up | Display the difference; never splice Stage/Agent details from another digest. Re-read explicitly when needed, with no automatic selection mutation (VP-06/15). |
| Dangling/cyclic dependency or unsupported payload | Do not invent corrected edges or claim validation. Surface canonical diagnostics or the bounded presentation limitation; only authoritative validation establishes validity (VP-04/10/15). |
| Rendering unavailable, fence shown literally, empty/clipped/failed graphic | Disclose failure/uncertainty and use complete functional text. A failed image cannot count as VP-01 (VP-11/13). |
| Large/dense/long-label definition | Declare scope and omitted detail, provide continuation/detail path; readability failure remains visible, not silently accepted (VP-14, proposed P-03). |
| Gates, criteria or validator declarations without execution facts | Show “declared / outcome not observed”; existence, operation success and agent prose prove neither validation nor authority (VP-07/12). |
| Markup, links or instructions embedded in labels/instructions | Treat as untrusted data. Escape presentation syntax, keep stable identity, do not execute scripts, open remote resources or obey embedded instructions (accepted security policy). |

Text reading order follows Stage order and explicit predecessor lists. Color alone
cannot carry dependency, gate or status meaning; use visible labels and equivalent
text (CARD accessible fallback). **P-04 — PROPOSED:** evaluate labels at normal
host zoom, keyboard-only access to any adopted controls, and screen-reader/text
reading order. No mandatory visual interaction may be the sole route to facts.
No remote renderer/upload or new credential/cost requirement is adopted here.

## Compatibility inventory — documentation versus observed rendering

Checked 2026-10-22. Official-source findings and reproducible lookup notes live in
[Specify Evidence](evidence-specify.md#host-documentation-inventory).
No host was launched or exercised for this story. Every empirical rendering cell
below is **UNVERIFIED**. No platform is added by this table.

| Host/surface | Official documentation finding | Mermaid / SVG / native / fenced Markdown | AXM-38 empirical evidence and route |
|---|---|---|---|
| Codex Desktop requested by card | Official app-feature URL redirects to current ChatGPT features. Separate Visualizations page describes eligible ChatGPT desktop preview, not proof for an exact Codex Desktop build | Mermaid/SVG unknown; native visualization documented for the named eligible ChatGPT surface; fenced code never sufficient | UNVERIFIED; record exact app identity/build and actual output before claiming support; text meanwhile |
| Codex CLI / IDE | Visualizations documentation explicitly excludes these surfaces from that feature; CLI docs describe terminal operation | No native Visualizations through that feature; Mermaid/SVG rendering otherwise not established by this inventory; fences remain source | UNVERIFIED for this story; functional text required, preserve native CLI contract |
| Claude Code terminal | Interactive-mode docs describe terminal interaction | No diagram-format guarantee established; neither Mermaid nor SVG nor fences assumed to render | UNVERIFIED; text route; terminal support is not graphical proof |
| Claude Code Desktop | Desktop docs establish a separate surface; artifact docs describe conditional artifact availability | General artifact capability is not proof of inline Mermaid/SVG or Workflow rendering | UNVERIFIED; verify exact build/settings/format, otherwise text |
| Claude chat/Desktop/Cowork | Custom-visual docs describe inline diagrams on web/desktop, distinct from Code terminal; artifact docs describe separate artifact surface and capability prerequisites | Native custom visuals documented; SVG export is not itself inline SVG or Mermaid rendering proof; fenced source remains insufficient | UNVERIFIED; research candidate only, not newly promised Axiom runtime support |
| Other hosts | None additionally established as supported by the inspected Axiom contracts | Unknown | No promise. Inventory a concrete adopted surface before extending compatibility |

**P-05 — PROPOSED, compatibility policy:** qualify support per exact surface and
format with a real rendered witness. If the supported surface cannot render, retain
text and document the gap. Do not satisfy the at-least-one-host AC by substituting
an unadopted platform, silently adding a plugin/dependency or publishing to a site.
If no existing supported host can meet VP-01 within current boundaries, the material
decision is PRODUCT_BUSINESS/ARCHITECTURE: adopt bounded support/transport or revise
the product requirement explicitly. Evidence collection precedes that decision.

## Acceptance and future Evidence matrix

These criteria describe later behavioral verification, **not tests executed in
Specify**. Documentary review checks completeness/consistency only. CARD AC mapping:
AC1 → E1/E7; AC2 → E2/E3/E9; AC3 → E4; AC4 → E5/E6;
AC5 → E6/E7/E8; AC6 → E8/E10.

| Evidence ID | Requirement/origin | Input or scenario | Observable oracle and retained Evidence | This stage |
|---|---|---|---|---|
| E1 | VP-01/02/13 CARD | Canonical `show` for default and a custom definition on one supported qualified host | Visible rendered overview, complementary text, host/build/settings/format and manual witness tied to input digest; a literal fence fails | NOT_RUN |
| E2 | VP-03/04 ACCEPTED, VP-05 CARD | Sequential fixture; include later multi-agent sequential case | Exact Stage order; agent predecessor edges/concurrency 1; gates/criteria match canonical declarations. Semantic expected graph and readable text | NOT_RUN |
| E3 | VP-04 ACCEPTED, VP-05 CARD | Existing `custom-r1.json` implementation Stage: implementer/reviewer independent, integrator depends on both, concurrency 2 | No edge between independent agents; both edges into integrator. Show possible parallelism, not observed execution; preserve gate timing and I/O | NOT_RUN |
| E4 | VP-06/07/08/09 ACCEPTED | Distinct inspected/selected revisions; unselected Project; schema 1/2; unresolved `policy-default`; operation success | Full exact reference and selection; no inferred active Execution/Stage/status/resolution/capability/authority; compare text and visual semantics | NOT_RUN |
| E5 | VP-11/13 CARD | Unsupported/unknown renderer; literal Mermaid, missing image, rendering error; plain terminal | Same identity/order/dependencies/gates available in readable text; limitation explicit; no rendering success claim | NOT_RUN |
| E6 | VP-10/11 CARD | Same JSON repeatedly; object key/whitespace variants; changed array/dependency/revision/digest | Deterministic semantic transformation and fallback; preserve arrays, identity and changed facts. Fixed renderer bytes only where an implementation later defines them | NOT_RUN |
| E7 | VP-01/13 CARD, P-04 PROPOSED | Authorized real session with sequential and DAG inputs, then forced fallback | Screenshot/capture plus human visual/manual testimony: graphic visible, no clipped IDs/edges, identity legible, three levels usable, text usable independently. Record source/checkpoint, host/build, format, viewport/zoom, failures and witness/date | NOT_RUN |
| E8 | VP-10/15 CARD/ACCEPTED | Read-only presentation, before/after canonical/state comparison | No persisted derived graph, no mutation/new domain facts; material diagnostics/authority/digests retained; deterministic transformation/fallback assertions reserved for implementation | NOT_RUN |
| E9 | VP-14 CARD, P-01/03/04 PROPOSED | 32 Stages; 32 Agents in one Stage; dense DAG; long Unicode; single-Agent DAG | Domain bounds unchanged; complete navigation/text predecessors; legible tested segments, no invented dependencies. Proposed UX acceptance depends on approval | NOT_RUN |
| E10 | VP-07/12/15 ACCEPTED | Failed/unknown/recovery result, absent gates' satisfaction, invalid payload, unsafe label markup | Honest failure/limitation, no guessed validation/decision or executable label content; semantic fact comparison | NOT_RUN |

Existing [default r1](../007-configurable-workflows/examples/default-sdd-r1.json),
[default r2](../007-configurable-workflows/examples/default-sdd-r2.json) and
[custom r1](../007-configurable-workflows/examples/custom-r1.json) are source
fixtures, not captured `workflowAuthoring` completion events or host tests.
Future tests wrap actual canonical event shapes and add bounded cases as needed.
Do not rename `reviewed-delivery` to `teste-sdd` in evidence; that would invent
an observed installed definition. No behavioral tests/fixtures are implemented here.

## Decisions, ambiguities and next gate

No accepted-contract conflict was found: host-owned visuals extend presentation,
while ordered Stages, intra-Stage DAG, terminal text and JSON authority stay intact.
No supersession or new ADR is adopted. A later incompatible proposal must identify
the exact clause and follow [ADR-0018](../../decisions/0018-adr-evolution-and-supersession-governance.md),
without editing acceptance history by implication.

At the original Specify checkpoint, P-01–P-05 awaited review. The subsequent
approval above accepts them within the exact baseline conditions, without proving
compatibility or adopting a renderer. Original recommendation: conversational detail/segmentation with textual
equivalence and evidence-based surface qualification. Alternative: demand one
complete static diagram; simpler navigation, but dense maximum graphs can become
unreadable. Renderer technology remains open; no Mermaid/SVG/native choice is
made. No choice blocks finishing this documentary proposal.

`EVIDENCE_GAP`: actual `teste-sdd` payload, exact host builds/availability and rendered
behavior. Investigate later; these are not votes or invented human-decision gates.
`AUTHORIZATION_REQUIRED`: real external sessions, credentials/costs if needed,
and commit/push/Draft PR without explicit publication authority. Existing authenticated
read access alone does not grant write authority. The explicitly requested single
Linear completion callback is authorized separately.

`technical_decisions[]` for this Specify attempt (executor AI, TECHNICAL_LOCAL,
scope impact none, reversible):

- **TD-01:** use next local numbered Specification directory and a bounded Evidence
  companion/index entry. Alternative: amend accepted Spec 007. Separate proposal
  preserves accepted bytes and scope; rollback removes only new proposal/index text.
- **TD-02:** verify command/field/limit claims against current source and schemas;
  inspect existing fixtures instead of launching CLI/hosts or creating sample state.
  Alternative: native smoke now. Source evidence is sufficient for Specify syntax,
  with runtime behavior explicitly unverified. No behavioral confidence is claimed.
- **TD-03:** retain format-neutral information requirements; no renderer selection
  from documentation alone. Alternative: prematurely choose Mermaid or SVG.
  Documentation does not prove delivery-surface rendering; later evidence can guide
  a reversible local implementation choice within approved boundaries.

References for all three: this Specification, [Evidence](evidence-specify.md) and
the source links above. No subagent was used: strict lower-model hierarchy was not
proven, and bounded research did not justify delegation overhead.

**Current next gate: `review_plan` plus exact editorial reconfirmation.** The
original `review_specification` gate was followed by publication and the linked
human approval of r1. [Plan](plan.md) is drafted against that accepted semantic
baseline. Review the exact Plan and editorial e1 checkpoint before human approval;
publication remains separately authorized. Do not start Tasks or implementation
in this Plan attempt. Product ACs and native host acceptance remain unfulfilled.

# Specification 007 amendment — dedicated `axiom-workflow` conversational surface (#303)

## Status and authority

**Proposed, 2026-10-10**, for human decision under
[Issue #303](https://github.com/rgomids/axiom/issues/303) (Linear AXM-7),
together with [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md).
It records the maintainer direction of 2026-10-10 to introduce a dedicated
`axiom-workflow` **configuration** skill and keep the complete #275
conversational testing/acceptance journey available through the Work Item
surface in both Codex and Claude. The maintainer refined HD-005 during
[PR #304 review](https://github.com/rgomids/axiom/pull/304):
Project selects the active workflow; Workflow defines/configures it; Work Item
initiates and controls work; Execution is its revision-bound instance. This
scope decision does not constitute acceptance of the exact revised artifacts. Revised the same day after the maintainer's
[contract review](https://github.com/rgomids/axiom/pull/304#issuecomment-6098704209),
which requested changes and accepted no revision. Until the maintainer accepts this exact
revision and decisions HD-005–HD-008, accepted
[Specification 007](spec.md), HD-002 and ADR-0020 remain authoritative
unchanged, and no `axiom-workflow` implementation is authorized.

Inspected baseline: `main` at `3766273` (release v0.15.0). The historical #275
delivery record ([issue-275-implementation.md](issue-275-implementation.md)),
its PR #301 and the v0.15.0 release are not reopened or reinterpreted.

## Superseded fragments (applied on acceptance)

On acceptance, before merge, the following accepted Specification text is
preserved struck through with an inline note linking this amendment and
ADR-0022. Nothing else in Specification 007 changes. ADR-0020 is not
annotated: its sentence "Authoring stays a Project operation; execution stays
under the Work Item domain surface" states domain ownership (Work Item target
and immutable binding), which remains in force; ADR-0022 only clarifies that it
does not fix the conversational skill name.

| Location | Accepted text | Replacement scope |
|---|---|---|
| [spec.md, Public operations](spec.md#public-operations-and-boundary-dtos-wf-009011013) | "and thin `axiom-work-item`; do not create an `axiom-workflow` skill" and the sentences "A dedicated top-level workflow management surface would separate authoring conceptually but add selector/skill overlap; revisit only if independently managed cross-Project definitions become approved." | Conversational skill surface only. "Keep authoring under `axiom project workflow <operation>`" and "Execution uses existing `axiom workflow` command family" remain in force. |
| [spec.md, Architectural decisions](spec.md#architectural-decisions--explicit-maintainer-acceptance), HD-002 row | Accepted option "Extend Project authoring + Work Item execution" **as the skill surface** | CLI command families and domain ownership remain as accepted; the skill-surface part is replaced by HD-005. The 2026-10-09 approval ledger is history and stays unchanged. |

## Intent and invariants

A developer working in Codex or Claude configures a Project-owned workflow in
natural language through `axiom-workflow`: stages, agent topology, validators,
gates, dependencies, Runtime/Profile references and revision authoring. The
Runtime gathers missing configuration facts, prepares bounded drafts, validates
them with Axiom, presents the canonical preview/effects, requests explicit
approval and reads back the result. The developer never manually authors JSON
or computes digests for a supported configuration journey.

**Domain and conversational responsibility are separate:**

- **Project chooses:** the Project owns workflow definitions and selects the
  active workflow revision through `axiom-project`.
- **Workflow defines:** `axiom-workflow` configures and validates the
  Project-owned definitions, but does not initiate or control Executions.
- **Work Item requests and controls:** `axiom-work-item` owns the conversational
  `run`, `status`, stage planning and Execution lifecycle for a **specific**
  Work Item, including the conversational #275 acceptance journey.
- **Execution performs:** starting a Work Item resolves the Project's active
  workflow revision via canonical application behavior and immutably binds it
  to the new Execution. Later Project workflow changes do not rewrite existing
  Execution bindings; a missing or invalid selection fails closed.

Work Item `run` should not require the operator to select a workflow revision
manually when a valid active Project selection exists. This amendment defines
the intended conversational behavior without inventing an unverified new CLI
capability: if the existing canonical start operation cannot resolve that
selection, this gap must be surfaced and specified before implementation,
not worked around by the skill.

Unchanged invariants: Project ≠ Repository; Runtime ≠ Model; Role ≠ Model;
Execution ≠ Agent; Skill ≠ workflow truth; Evidence ≠ raw chat. Workflow
definitions are Project-owned; Executions keep their Work Item target and
immutable binding; Lingo/application services own validation, digests,
preview/apply, authority, state and the canonical result. No new workflow
engine, scheduler, store, approval mechanism, result protocol, HTTP server or
mandatory MCP integration.

## Requirement

| ID | Required behavior | Primary owner | Acceptance |
|---|---|---|---|
| WF-015 | Dedicated `axiom-workflow` **configuration** skill in Codex and Claude; Project-owned active selection; Work Item-owned run/status/plan/acceptance journey; revision-bound Execution; compatible canonical operations and Runtime-prepared drafts | #303 | AC-015 |

WF-001–WF-014 owners and acceptance are unchanged. WF-015 adds no behavior to
WF-009 (#276), WF-011 (#277) or WF-012 (#278).

## Skill ownership

| Skill | Primary conversational ownership after this amendment | Compatibility |
|---|---|---|
| `axiom-project` | Project lifecycle, identity, integrations, Runtime/Model Profile policy at creation, and **selection of the active Project workflow revision** | Existing `workflow.*` routes remain operational. |
| `axiom-workflow` | **Definition configuration only**: list/show/validate/create/edit/remove/recover workflow definitions, stages and agents, plus read-only configuration readiness | New skill, no Execution lifecycle or Work Item mutation routes. |
| `axiom-work-item` | Work Item lifecycle and the Work Item-bound **run/status/plan/Execution interactions**, including the #275 conversational acceptance procedure | Existing `run`, `status`, `plan` remain supported. |

Rules:

1. `axiom-workflow` routes configuration operations to the **existing** Project
   workflow commands; domain data and validation continue to belong to Project
   and Lingo. A route's flags, authority and results are taken from the
   canonical command registry. Metadata parity is tested per action and mode.
2. The **active revision is selected via `axiom-project`**, not by treating the
   `axiom-workflow` configuration skill as the Project selector. The existing
   `axiom project workflow select` command is unchanged. A configuration
   conversation can explain the next Project-selection step, without silently
   selecting or activating a revision.
3. `axiom-work-item run` (the conversational intent, not a newly mandated CLI
   spelling) identifies the specific Work Item and its Project, resolves the
   Project-selected active workflow through canonical operations, and creates
   an immutable `WorkflowBinding`. It must not execute a free-standing
   workflow or fall back silently to another revision. `axiom workflow start`
   and other existing CLI operations remain canonical and unchanged.
4. `axiom-work-item` continues to own stage Plan proposals, Execution
   operations, readiness in execution context, and the #275 acceptance
   procedure. When that procedure needs a configured workflow, it invokes
   the configuration surface for that **configuration subtask only**; it does
   not transfer Work Item or Execution ownership to `axiom-workflow`.
5. Historical routes in `axiom-project` and `axiom-work-item` remain
   available with identical metadata and authority until a separate versioned
   retirement decision (HD-007). A cross-skill journey never combines
   independent canonical previews into a single approval.

## `axiom-workflow` configuration operation catalog

Operation names are proposed conversational contracts. All flags, requirements
and authority derive from the existing canonical operations via
`axiom skill inspect axiom-workflow` (no duplicate registry).

| Operation | Canonical command(s) | Effect | Authority |
|---|---|---|---|
| `definition.list` / `definition.show` | `axiom project workflow list` / `show` | read-only | none |
| `definition.validate` | `axiom project workflow validate` | read-only | none |
| `definition.create` | `axiom project workflow create` (`--from-default` or Runtime-prepared `--file`) | local mutation | exact preview/revision and `--authorize-local` |
| `definition.edit` / `definition.remove` / `definition.recover` | `axiom project workflow edit` / `remove` / `recover` | local mutation | as above |
| `configuration.readiness` | `axiom runtime profile validate` / `preview`; Runtime status/auth observations | read-only diagnostic | none |

**Not `axiom-workflow` operations:** `definition.select` (Project),
`execution.list/status/run/reconcile`, `stage.plan` and
`acceptance(stage-plan)` (Work Item), and any future dispatch/retry/cancel,
coordination, delivery, decision or rework (#276/#277). The `axiom-work-item`
skill can expose its existing `run/status/plan` routes and a bounded #275
acceptance procedure with their original canonical commands and authority.
No new workflow execution engine or top-level CLI command family is implied.

### Conversational configuration coverage

All stage and agent fields of [workflow.schema.json](workflow.schema.json) are
configured through `definition.create`/`edit` drafts: stage purpose,
instructions, inputs/outputs, completion criteria, validators, human gates,
failure policy, `sequential`/`dag` mode and concurrency; agent role,
responsibilities, inputs/outputs, dependencies, integration and validation
ownership, capabilities, complexity, Runtime constraints, Profile reference or
`policy-default`, effort, timeout, attempts and effect ceilings. There is no
persistent Agent Profile entity; agents exist only inside a stage definition.

## Runtime-prepared drafts

1. The Runtime may write bounded draft inputs (workflow definitions when using
   `axiom-workflow`; stage Plan documents/results only when using the
   Work Item-owned planning/acceptance journey) to a Runtime-owned scratch
   directory outside
   the Project working copy, every Repository working copy and Axiom state
   roots. It never writes Project, workflow index, Execution, Runtime Profile or
   receipt files.
2. A draft is built only from the user's statements, the current canonical
   read results and explicit defaults that the skill names. Unknown or
   ambiguous material values (stage, scope paths, effects, authority ceiling,
   Runtime constraint, Profile, effort) are asked, never invented.
3. Every draft passes through its owner's canonical validation
   (`definition.validate` / configuration preview or Work Item-owned
   `stage.plan`) before it is presented. Lingo's
   diagnostics are presented as returned; the Runtime does not pre-judge them.
4. Presentation shows the user a readable summary **and** the exact canonical
   preview/effects/digest returned by Lingo; approval binds to that exact
   preview, never to the summary.
5. **Work Item-owned Plan confirmation (not an `axiom-workflow` action).**
   When `axiom-work-item` prepares the #275 stage-plan acceptance, the Runtime
   writes the complete candidate Plan
   document to scratch in the existing input format, including `approved:
   true`, `planRevision` and `planDigest`, runs the read-only `stage.plan` on
   it and presents the scope, effects, authority ceiling and Lingo's proposal
   with its `plan.planDocumentDigest`.
   - `approved: true` in the candidate is a **provisional input assertion**
     that the read-only planner requires (it refuses unapproved Plans). It is
     not a committed approval, gate fact, authority grant or Evidence of user
     approval. The skill never labels a candidate as human-approved because it
     wrote that field.
   - Human confirmation is a separate, explicit event: the user confirms the
     exact reviewed proposal. Confirmation binds to Lingo's canonical
     `planDocumentDigest` (computed over the canonical encoding, so formatting
     alone does not change it) **plus** the bindings `stage.plan` reports: the
     Execution ID and expected revision, `stageId`, `workflowRef` and the
     proposal `plan.digest`. A change to the canonical Plan content or to any
     of those bindings requires a fresh `stage.plan` and a fresh
     confirmation. The Runtime defines no digest of its own and claims no
     byte-for-byte identity that the implementation does not enforce.
   - `planRevision` and `planDigest` are opaque references with no authority:
     `planDigest` is the SHA-256 of the exact bytes of a reviewed Plan/Tasks
     artifact the user names or, when none exists, of a bounded scope artifact
     the Runtime writes to scratch and shows.
   - The confirmation is recorded only in the conversation and the Evidence
     report as a statement of who confirmed which digest; no Axiom fact is
     committed (G-5). Dispatch remains #276 scope.
6. Drafts are working files, not Evidence. Evidence cites canonical results and
   digests.

## Runtime and Model Profile capability matrix

| Concern | Owner / layer | Current operation | `axiom-workflow` coverage |
|---|---|---|---|
| Project allowlist, Model Profiles, preferences at Project creation | Project (portable policy v2) | `project configure` CREATE `--runtime`, `--model-profile`, `--runtime-preference` | Explains and hands off to `axiom-project configure` with its own preview/authority |
| Same policy on an **existing** Project | Project | none (CREATE-only) | Unavailable: reported as G-1, never patched |
| Stage/agent Runtime constraint, Profile reference, effort | Workflow definition | `definition.create`/`edit` | Supported |
| Machine-local Runtime Profile store (adapters, models, credential references) | Machine-local | `runtime profile validate` (read-only) | Read-only; authoring unavailable (G-2) |
| Runtime installation / skill integration / auth observation | Machine-local | `runtime codex\|claude status`, `auth`; `runtime profile preview` | Read-only configuration diagnostics; Work Item owns execution readiness |
| Capability and explicit-effort proof beyond `axiom-skills` | Machine-local observation | production observer proves only `axiom-skills` | Reported truthfully as `runtime_unresolvable` / `unsupported_effort` (G-3, #272) |

The skill never infers a capability from an executable on `PATH`, a model name,
a login state or the Runtime it is running in.

## Gaps and dependencies

| Gap | Missing contract | Impact | Proposed disposition |
|---|---|---|---|
| G-1 | Preview/apply edit of Project Runtime/Model Profile policy on an existing Project | Policy changes need a new Project or a historical authored-manifest path | Bounded follow-up Issue (Spec 002 amendment); not in #303 (HD-006) |
| G-2 | Preview/apply authoring of the machine-local Runtime Profile store | A fresh sandbox cannot create the Profile configuration that `workflow start` and `stage.plan` need; live Execution-dependent acceptance is blocked | Bounded follow-up Issue with its own credential-reference security review; not in #303. Required for R-3 of C–G (HD-006) |
| G-3 | Authoritative capability / model-specific effort observation | Positive explicit-effort or non-`axiom-skills` planning is only provable synthetically | Existing #272 |
| G-4 | Provider-free Work Item linkage for sandboxes | A live Execution needs a linked existing Provider Issue (`work-item select`, Provider read, sandbox-local link) | No new contract; use an existing Issue the user names, never create one without external authority |
| G-5 | Recorded Plan approval fact | Approval is asserted by the Plan document | Existing #276 binds the reviewed `plan.digest` before dispatch |

**Follow-up traceability gate.** G-1 and G-2 are not tracked by any Issue
at this revision (checked 2026-10-10; closed #140 delivered the existing
CREATE-time policy, not G-1/G-2). Before this amendment is accepted as
reconciled, and before #303 delivery is considered complete, one bounded
GitHub Issue each for G-1 and G-2 must exist, created under the maintainer's
authority, and be cross-referenced here by number. The G-2 Issue must carry the
credential-reference security review as its own acceptance criterion. Both
stay outside #303's skill implementation. Until the references are recorded,
this gate reads `open` and HD-006 cannot be accepted as reconciled.

| Gap | Follow-up Issue | Status |
|---|---|---|
| G-1 | to be created | open |
| G-2 | to be created | open |

## Work Item-owned conversational #275 acceptance (`axiom-work-item`, stage-plan mode)

This remains an explicit #303 outcome, but **not an `axiom-workflow` operation**.
A single Work Item-bound natural-language request to `axiom-work-item` in either Runtime starts it. The Runtime assembles
and runs the procedure, asks only for genuinely missing consent/inputs, and
reports results in natural language with a bounded Evidence report.

**Safe environment.** Every Lane L step runs the installed `axiom` with fresh
temporary `LINGO_STATE_ROOT`, `AXIOM_CODEX_SKILLS_ROOT`, `CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, `HOME`/`USERPROFILE` and a temporary Git working copy. The
user's real Axiom state, skills and Repositories are never read for mutation
or written. No Provider resource is created or mutated and nothing is
published.

**Inference boundary.** Two kinds of activity are distinct:

- **Lane L and Lane S checks** (CLI, planner and synthetic tests) run no
  vendor inference and dispatch no stage agent. The only vendor processes
  they start are the existing read-only status/auth observations of
  `runtime.readiness` (for example `codex login status`), which the report
  names.
- **R-2 native sessions** (below) are interactive Codex or Claude sessions,
  and the session itself performs vendor inference. Each one is permitted only
  after the user's prior explicit consent for that Runtime, and is a separate
  step from the Lane L/Lane S checks. A native session still dispatches no
  stage agent through Axiom: `workflow stage run`, coordination and any other
  #276 behavior stay outside #303.

**Credentials and isolation.** A native session authenticates through that
Runtime's own existing login on the user's machine; the Axiom sandbox isolates
Axiom state, installed skills and Repositories, not the vendor login. The
Runtime and the skill never copy, export, print or move credentials, tokens,
login files or vendor configuration into scratch, Evidence, the sandbox state
roots or the isolated Runtime homes used for Lane L. Lane L `runtime.readiness`
observations of an isolated home therefore report it as unauthenticated, which
is the expected sandbox result, not a failure. Evidence records only Runtime
name, version and the canonical Axiom results; never credential values,
environment values or host paths.

**Evidence lanes** (how a result was produced).

- **Lane L — live local:** the installed binary and its production observer in
  the sandbox. Classified `operational-local`; never vendor inference proof.
- **Lane S — synthetic:** the release source at the installed version's exact
  provenance revision, run through a durable acceptance runner registered in
  `scripts/automation-registry.json`, which maps scenario IDs to the existing
  controlled-observation tests and emits a bounded JSON report. Classified
  `synthetic`. Fetching that source is a network read and needs the user's
  consent; a missing Go toolchain or revision mismatch is `blocked`, never
  substituted.

| Scenario | Lane L (installed binary, sandbox) | Lane S (synthetic) |
|---|---|---|
| A Setup | version/provenance, `skill inspect` for both Runtimes after sandbox install, Runtime status, sandbox Project and Repository, `runtime profile validate` | fixture Project/Work Item/Execution identities |
| B Workflow setup | `axiom-workflow` configures/validates/previews/applies draft; `axiom-project` selects the active revision with its own review/approval; read-back | covered by existing #273 tests |
| C Single agent | blocked by G-2 (no Profile store) and G-4 (linked Provider Issue needs a consented Provider read with credentials the sandbox does not hold) | `executionKind: single`, one resolution, no graph identity |
| D Multi-agent | blocked by G-2 and G-4 | independent Codex/Claude agents, integrator depending on both, concurrency, child inputs/outputs, integration ownership |
| E Determinism | blocked by G-2 and G-4 | repeated identical digests; config/observation change invalidates; Execution revision binding; no fallback |
| F Security | definition-level refusals (cycle, dangling dependency, invalid definition); plan-level cases blocked by G-2 and G-4 | invalid Plan, stale revision, missing inputs, unavailable Runtime, unsupported effort, cycle, dangling, unsafe overlap, authority escalation, `..`, `C:/outside`, `C:outside` |
| G Read-only | before/after canonical status and state-tree digests around every read-only step | no Execution/attempt/stage advance/gate/authority/Provider/artifact change |
| H Evidence | report below | same report |

When G-2 is delivered **and** the user consents to link an existing Provider
Issue they name (G-4), C–G gain Lane L coverage for stages requiring only
`axiom-skills` with runtime-default effort; explicit effort stays Lane S until G-3.

### Result levels

Every report states three results separately; none is derived from another,
and none is inferred from a passing lower level.

| Level | Meaning | Satisfied by | Never satisfied by |
|---|---|---|---|
| R-1 Technical verification | Contract, catalog, routing, install/upgrade and planner behavior are correct | Repository tests, `skill inspect`, catalog/routing tests, Lane S, Lane L binary checks | — |
| R-2 Native conversational operability | A real Codex session and a real Claude session complete the journey from natural language | The native scenarios below, run with explicit consent, one per Runtime | Binary commands, `skill inspect`, catalog tests, synthetic routing, Lane S |
| R-3 AXM-7 functional acceptance | The maintainer explicitly accepts the essential real scenarios | An explicit human decision over R-2 Evidence plus Lane L results for C–G | Any Lane S result, any `blocked` Lane L scenario, merge, green CI, or this report alone |

A report in which Lane L is `blocked` and Lane S is `passed` reads
`R-1 passed; R-3 blocked (G-n)` for that scenario. It never counts as positive
live acceptance. Blocked scenarios stay `blocked`, with their exact G-n
dependencies, until an operational demonstration satisfies them; synthetic
Evidence is intermediate technical Evidence only. Live C–G (R-3) depend on G-2
and G-4; until G-2 ships, AXM-7 functional acceptance for those scenarios is
blocked by it. Recorded by the maintainer on the
[PR #304 review](https://github.com/rgomids/axiom/pull/304#issuecomment-6098704209):
AXM-7 stays `In Progress` and keeps `Blocks AXM-8` until its essential real
scenarios are demonstrated and explicitly accepted.

### Native conversational scenarios (R-2)

At least one bounded, authorized native scenario runs in a real **Codex**
session and one in a real **Claude** session, each in the safe environment
above, from a natural-language request (not a typed command). Each covers:

1. skill discovery and ownership routing (`axiom-workflow` for definition configuration, `axiom-project` for active revision selection, `axiom-work-item` for Work Item-bound acceptance);
2. workflow configuration from natural language into a Runtime-prepared draft;
3. canonical validation and preview returned by Lingo;
4. explicit human confirmation of each exact preview, followed by bounded
   sandbox mutations (for example configuration `definition.create`, then
   Project-owned `definition.select`);
5. read-back of the persisted result;
6. interpretation and a bounded Evidence entry.

The Runtime session itself performs vendor inference, so each scenario needs
the user's prior explicit consent before it starts, and it follows the
inference boundary and credential rules above. The scenarios do not dispatch
stage agents (#276) or deliver end to end (#278). Their Evidence records the
Runtime and version, skill-set digest, the canonical results and digests of
each step, who confirmed which preview, and a bounded description of the
interaction, not raw chat. A scenario without that consent is `not run`, never
`passed`.

**Report (H).** Axiom version and source revision; Runtime name/version as
observed (or unknown); skill set digest; sandbox identification (no host
paths); scenarios with `passed` / `failed` / `blocked` and the classified
category for each refusal; digests and canonical references; lane and Evidence
class per result; R-1, R-2 and R-3 results stated separately; known limitations (G-1–G-5); unresolved acceptance
requirements. Raw logs are not the default presentation and credentials,
environment values and host paths never appear.

## Codex/Claude parity

Both Runtimes install the same embedded `axiom-workflow` configuration bytes
and resolve the same catalog. Validation covers, per Runtime: discovery after
install, upgrade and reinstall; natural-language configuration intents and
canonical operations; Project-owned selection; Work Item-owned execution,
stage planning and acceptance routing; missing-input acquisition; preview and
confirmation; failure classification; human-readable presentation; Evidence. Runtime-specific differences are limited to
installation location and invocation syntax (`$axiom-workflow` /
`/axiom-workflow`) and must not change semantics, authority or state.
Synthetic routing tests are distinguished from native Runtime behavioral runs,
which need explicit authorization when they involve vendor inference.

## Acceptance criteria (AC-015)

1. Accepted amendment and ADR-0022, with the superseded Specification fragments annotated.
2. `axiom-workflow` is embedded, installed and discoverable for Codex and
   Claude, including upgrade from v0.15.0 and reinstall; skill-set history pins
   the replaced v0.15.0 set; `skill inspect`, help and routing catalogs agree.
3. Every `axiom-workflow` **configuration** action routes to its canonical
   Project workflow command with matching metadata; active selection stays
   Project-owned, execution and planning stay Work Item-owned, and existing
   routes and CLI commands remain unchanged.
4. Definition, stage and agent configuration work from natural language
   through Runtime-prepared drafts. Project selects the active revision with
   its own approval. Starting a specific Work Item resolves that Project
   selection without a free-standing workflow run or silent fallback; each
   Execution binds the selected immutable revision. If canonical start lacks
   that behavior, its gap is surfaced instead of invented in the skill.
5. The **Work Item-owned** conversational #275 acceptance produces the report
   above in both Runtimes, routing workflow configuration to `axiom-workflow`
   and active revision selection to `axiom-project` without transferring
   Execution ownership. Every scenario is classified, every blocked item tied
   to G-n, and R-1, R-2 and R-3 reported separately.
6. R-2: one native scenario per Runtime (Codex and Claude) passes with
   explicit consent, covering every step listed above.
7. #303 technical delivery may complete on R-1 and R-2. R-3 (AXM-7
   functional acceptance) remains a separate maintainer decision; scenarios
   blocked by G-2/G-4 stay `blocked` and are never reported as accepted.
8. Canonical CLI, result, digest, preview/apply, security and authority
   semantics are unchanged (existing tests pass unmodified).
9. An independent review reports no unresolved Blocker/Major.

## Implementation plan (authorized only after acceptance)

1. Configuration skill and catalog: `internal/codexruntime/skills/axiom-workflow/SKILL.md`;
   `skillOperationSpecs("axiom-workflow")` reusing **only definition/configuration**
   specs; embedded inventory (`codexruntime`, `install`), receipts, skill-set
   history; route selection to Project and run/plan/acceptance to Work Item.
2. Install/upgrade/retirement and parity tests for Codex and Claude.
3. **Work Item-owned** #275 acceptance runner (Lane S) and report schema with
   R-1/R-2/R-3 levels, registered as durable automation; Lane L procedure in
   `axiom-work-item` (using `axiom-workflow` only for configuration).
4. Native R-2 scenarios in Codex and Claude, run only with explicit consent;
   their bounded Evidence recorded in this amendment's Evidence section.
5. Documentation: `docs/commands.md`, `docs/agent-harness.md`, Spec index, this
   amendment's Evidence section.

## Human decisions requested

| ID | Decision | Options | Recommendation | Harder to change later |
|---|---|---|---|---|
| HD-005 | Dedicated configuration surface and explicit ownership | A keep HD-002; **B `axiom-workflow` configuration only, `axiom-project` selects, `axiom-work-item` runs/plans/accepts, Execution binds an immutable selected revision (ADR-0022)**; C plus new CLI tree; D new workflow domain | **B — maintainer direction recorded 2026-10-10; exact revised contract still Proposed** | Skill and operation names become public conversational contracts |
| HD-006 | Runtime/Profile gaps G-1/G-2 | **a. track G-1 and G-2 as bounded follow-up Issues, not in #303; live C–G stay `blocked` on G-2/G-4**; b. deliver G-2 (and G-1) inside #303 | a — G-2 adds credential-reference authoring that needs its own Spec 002 amendment and security review; acceptance requires the follow-up traceability gate to be closed | With (a), AXM-7 functional acceptance (R-3) of C–G is blocked by G-2 until that follow-up ships |
| HD-007 | Compatibility window | **keep routes until a separate removal decision**; remove at the next minor | keep | Removal later requires retirement/upgrade protocol |
| HD-008 | Synthetic Evidence for #275 conversational acceptance | **accept as intermediate technical Evidence (R-1) only, classified `synthetic`, never R-2, R-3 or vendor proof** (direction recorded on the PR #304 review); require Lane L only | accept as R-1 only | R-3 still needs real scenarios; #278 still owns live two-Runtime proof |

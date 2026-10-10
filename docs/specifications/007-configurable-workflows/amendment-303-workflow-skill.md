# Specification 007 amendment — dedicated `axiom-workflow` conversational surface (#303)

## Status and authority

**Accepted on 2026-10-10** by the Axiom maintainer's explicit formal decision
recorded in [Issue #303](https://github.com/rgomids/axiom/issues/303#issuecomment-6099596104),
together with [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md).
The accepted exact review is [PR #304](https://github.com/rgomids/axiom/pull/304)
head `7db80ac95b5e83cd8cc9aaaadc785633c3ee3fa9` with amendment blob
`9d1895b6ab2dfea7e6c8ab4cd026bb111f31ce97` and ADR-0022 blob
`13a52a62e1273ec8c928f064563d297c3798d152`.

**Approved HD-005–HD-008 contract:**
Project permits/selects, `axiom-workflow` configures Workflow definitions,
`axiom-work-item` initiates/controls a specific Work Item, and Execution
immutably binds the Project-selected revision (HD-005). Project Runtime/Profile
policy remains portable, separately from machine-local Runtime/Profile
configuration; #305 and #306 own the bounded follow-ups and #231 retains
Project readiness (HD-006 A). Duplicate definition-authoring *skill routes*
are removed in the same #303 implementation, without compatibility aliases;
canonical CLI contracts stay unchanged (HD-007 B). #303 acceptance is
**technical R-1**, supported by deterministic/synthetic Evidence. Real native
Codex and Claude R-2 and final human E2E MVP R-3 remain in [#278](https://github.com/rgomids/axiom/issues/278)
/ AXM-12 (HD-008 A).

Accepted Specification 007 HD-002's historical *skill-surface* text is
annotated in [spec.md](spec.md), with the originally accepted decision kept
readable. ADR-0020 and all underlying Project-owned canonical operations,
Work Item Execution authority, historical #275/v0.15.0 Evidence and other
Specification 007 requirements remain in force.

**Authority boundary:** this decision accepts the *versioned architecture
and Specification amendment*, not any implementation, CLI/skill modification,
vendor inference, Provider effect, merge, release, acceptance of AXM-7's
technical implementation, closure or AXM-8 unblock. Those retain separate
explicit gates.
## Superseded fragments (reconciled after acceptance)

After formal acceptance, the following historical skill-surface fragments
in [spec.md](spec.md) are preserved struck through and annotated with a link
to this amendment and ADR-0022. Their canonical CLI/domain ownership remains
unchanged, and the rest of Specification 007 is unmodified. ADR-0020 is not
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

Work Item `run` does not require the operator to select a workflow revision
manually when a valid active Project selection exists. **Verified against
`main` at `3766273` (v0.15.0):** canonical `axiom workflow start` already
resolves the Project's `workflowSelection` during admission
(`cmd/lingo/workflow_binding.go`, the Project's resolved portable snapshot):

- an absent selection returns `workflow_selection_required` with the next
  action "select a Project workflow revision with `project workflow select`",
  and no default is substituted;
- a selected Project revision must be indexed `published`, and its content must
  match the selected reference; otherwise `recovery_required`;
- a Project source revision that differs from the resolved installation
  returns `project_configuration_drift`;
- the selected reference, Project revision and context digests are bound into
  the reviewed start preview, so a selection change after review requires a
  fresh preview, and the confirmed Execution retains its snapshot
  ([#274 record](issue-274-implementation.md#binding-and-execution)).

No new CLI capability or gap is therefore needed for this behavior, and the
skill implements no selection logic of its own.

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

**Migration policy (HD-007):** no legacy skill compatibility window. The
new skill is introduced and overlapping old skill routes are **removed within
the same #303 implementation**; the current PR remains a contract-only
proposal and does not itself change installed Runtime packages.

| Skill | Exclusive primary conversational ownership after implementation |
|---|---|
| `axiom-project` | Project lifecycle, identity, integrations, Runtime/Model Profile policy and **selection of the active Project workflow revision** (`workflow.select` and minimal Project-owned active-selection inspection). Definition authoring/validation routes are **removed** from this skill. |
| `axiom-workflow` | **Definition configuration only**: list/show/validate/create/edit/remove/recover definitions, stages and agents, plus read-only configuration readiness. The sole canonical *skill* entrypoint for these intents. No Project selection or Work Item/Execution operation. |
| `axiom-work-item` | Work Item lifecycle and Work Item-bound **run/status/stage planning/Execution interactions**, including the #275 conversational acceptance procedure. These routes remain here; no workflow-definition routes are introduced. |

Rules:

1. `axiom-workflow` routes configuration operations to the **existing**
   Project-owned canonical CLI/application commands. Domain data and validation
   remain Project/Lingo-owned; operation flags, authorization and canonical
   results come from that command registry. **Domain ownership is not skill
   routing ownership.**
2. **HD-007: Option B — remove overlapping skill routes in the same #303
   implementation release**, with no deprecation window or separate removal
   PR. Remove `axiom-project workflow.list/show/create/edit/validate/remove/recover`
   from its embedded skill instructions, operation catalog, help, inspection and
   natural-language routing. Add equivalent **configuration-only** operations
   to `axiom-workflow`. Do not retain aliases, forwarding shims, duplicate
   registered skill actions or a legacy instruction path. Old instructions
   are replaced on both Codex and Claude install/upgrade/reinstall.
3. **Active selection stays in `axiom-project`**: retain its existing
   `workflow.select` route and only the Project-owned read-only active
   selection inspection needed for that task. Selection is not an
   `axiom-workflow` action; its CLI `axiom project workflow select` is
   unchanged. Editing/creating a definition never activates it implicitly.
4. **Execution remains in `axiom-work-item`**: retain its existing
   Work Item-bound `run`, `status`, `plan` and Execution operations with
   unchanged semantic owner and authority. `axiom-work-item run` is
   conversational intent, not a newly mandated CLI spelling. It resolves the
   Work Item's Project-selected revision and binds an immutable
   `WorkflowBinding`, without free-standing workflow execution or fallback.
5. **Strict conversational routing**: a request targeting the wrong skill is
   directed to the owning skill, not served by duplicating a foreign operation
   in the invoked skill. A multi-domain journey may invoke each owner's
   explicit canonical operation as a separate step, with its own preview,
   human authorization and read-back; never combine approval boundaries.
6. **No CLI break**: `axiom project workflow *`, `axiom workflow *` and
   their flags/results continue to work. Removing redundant *skill surfaces*
   does not remove those commands or move domain ownership. Removing/renaming
   any canonical CLI operation would require an independent versioned decision.
7. **Upgrade contract**: the installed v0.15.0 skill-set remains immutable in
   versioned skill-set history/receipts for provenance, but a current upgrade
   replaces old skill bytes, regenerated catalogs and routing guidance
   atomically. After upgrade, old configuration routes must be absent from
   `axiom-project` `skill inspect`, help and both Runtime installations,
   while Project selection and Work Item operations still work. Reinstall and
   upgrade are idempotent; no stale route in active inventory. Historical
   receipts remain evidence, not active executable aliases.

## `axiom-workflow` configuration operation catalog

Operation names are accepted future conversational contracts, not implemented commands. All flags, requirements
and authority derive from the existing canonical operations via
`axiom skill inspect axiom-workflow` (no duplicate registry).

| Operation | Canonical command(s) | Effect | Authority |
|---|---|---|---|
| `definition.list` / `definition.show` | `axiom project workflow list` / `show` | read-only | none |
| `definition.validate` | `axiom project workflow validate` | read-only | none |
| `definition.create` | `axiom project workflow create` (`--from-default` or Runtime-prepared `--file`) | local mutation | exact preview/revision and `--authorize-local` |
| `definition.edit` / `definition.remove` / `definition.recover` | `axiom project workflow edit` / `remove` / `recover` | local mutation | as above |
| `configuration.readiness` | `axiom runtime profile validate` / `preview`; Runtime status/auth observations | read-only diagnostic | none |

**Not `axiom-workflow` operations:** active revision selection (Project: `axiom-project` `workflow.select`),
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

**HD-006 accepted (2026-10-10):** Project Runtime policy describes permission,
not necessarily host operational availability. The exact amendment/ADR revisions
were accepted by the maintainer; implementation authorization remains separate.

| Layer | Owner | Existing contract / missing behavior |
|---|---|---|
| Machine-local Runtime/Profile configuration | Runtime configuration, not a Project or a Workflow skill | Local installed/observed Runtimes, adapters, models, credential references. Read-only `runtime profile validate/preview` exists; supported local create/edit/remove/preview/apply is **G-2**, [#306](https://github.com/rgomids/axiom/issues/306). |
| Portable allowed Runtime/Profile policy | Project / `axiom-project` | Spec 002 Runtime policy v2 (#140): `runtimes[]`, `modelProfiles[]`, `runtimePreferences[]` established at CREATE. Safe edit of an existing Project is **G-1**, [#305](https://github.com/rgomids/axiom/issues/305). |
| Stage/Agent selection intent | Workflow definition / `axiom-workflow` configuration | Existing `definition.create/edit`: `runtimeConstraints`, `profileRef` or `policy-default`, `complexity`, `effort`. Neither Project policy nor local credentials are mutated by the skill. |
| Concrete Work Item execution readiness | Work Item/Execution and canonical resolver | Resolve Project allowlist ∩ machine-local enabled/matching Profiles ∩ observed capabilities ∩ stage constraints; fail closed. |
| Project bootstrap/readiness | [#231](https://github.com/rgomids/axiom/issues/231) | Existing proposed Project validation and Provider/Repository/binding diagnostics; no duplicate global Doctor implementation inside #303, #305 or #306. |

### HD-006 configuration invariants

1. A Project's Runtime list is its **allowed set**, not a single selected
   Runtime and not proof of local installation. Removing Claude from Project A
   affects only A. Disabling or removing Claude from the **local machine**
   affects readiness for every dependent Project, but never rewrites portable
   Project policies, workflow definitions or existing immutable Executions.
2. Multiple **user-named Profiles per Runtime** are supported as a product
   outcome. `economy`, `balanced`, `advanced` are optional examples only,
   not a hardcoded three-profile limit or comparable vendor quality ranking.
   Respect accepted Project policy v2 limits (**8 Runtimes, 32 profiles,
   32 preferences**); relaxing those requires a separate versioned contract.
3. Runtime != Model; Role != Model; Profile != agent complexity !=
   reasoning effort. Each local Profile belongs to a Runtime and resolves
   a concrete model/configuration as supported. The Workflow Stage/Agent
   declares constraints and logical Profile references rather than credentials
   or executable paths. Effort and capabilities must be observed and supported,
   with no fabricated cross-vendor equivalence.
4. **Preserve the current Spec 002 v2 matching rule:** a portable Project
   `modelProfiles` entry currently includes an explicit logical key,
   `runtimeRef` **and `model` field**; local binding must match required
   IDs, associations and concrete model. Changing the portable model field
   into a purely logical reference would require an independently reviewed
   migration, not an implicit change in HD-006.
5. No silent fallback to the initiating Runtime, another Profile or another
   model when the intersection is empty, ambiguous, disabled, unauthenticated
   or capability/effort-incompatible. Return a precise blocker.
6. G-1 changes a **Project-owned portable** policy through canonical
   read/validate/preview/apply, expected-revision and explicit authorization.
   G-2 changes **machine-local** Runtime/Profile bindings through separately
   gated operations; it requires credential-reference and filesystem security
   review. Neither operation grants dispatch or Provider effects and neither
   copies secrets, login files, tokens or host paths to portable data/Evidence.
7. **Doctor/readiness** should aggregate existing canonical, read-only
   Project, Provider, Repository and Runtime/Profile validators. #231 owns the
   Project-scoped readiness proposal. A potential future `axiom doctor` is
   only an aggregate UX idea, not a new requirement, Issue or implementation
   in #303. Structural validity ≠ operational readiness ≠ active vendor
   inference; live probes require their own explicit operator authorization.

`axiom-workflow` may explain available Profiles and validate Stage/Agent
references. It does not create/edit Project policy (G-1) or local Profiles
(G-2). These remain distinct stores and authority boundaries even if one
conversation guides the user across them.

## Gaps and dependencies

| Gap | Missing contract | Impact / disposition |
|---|---|---|
| G-1 | Edit existing Project allowed Runtimes/Model Profile policy | [#305](https://github.com/rgomids/axiom/issues/305), bounded Project config follow-up; requires its own Spec 002 reconciliation if needed; not part of #303 skill implementation. |
| G-2 | Production authoring of local Runtime/Profile store | [#306](https://github.com/rgomids/axiom/issues/306), bounded Runtime config follow-up and mandatory credential-reference security review; prerequisite for AXM-7 R-3 live scenarios C–G; not part of #303 skill implementation. |
| G-3 | Authoritative model-specific capability/effort observation | Existing observation boundary (#272); positive unproven capabilities remain blocked. |
| G-4 | Live linked Provider Work Item | Requires consented use of an existing user-named Provider Issue; no new Provider resource without authority. |
| G-5 | Recorded Plan approval fact | #276 owns durable approval binding before real dispatch. |

**Follow-up traceability gate: satisfied.** G-1 and G-2 are linked
technical follow-ups [#305](https://github.com/rgomids/axiom/issues/305)
and [#306](https://github.com/rgomids/axiom/issues/306) under
[#303](https://github.com/rgomids/axiom/issues/303), without adding
scope to the frozen [#15](https://github.com/rgomids/axiom/issues/15)
MVP epic. Their creation does **not** mean delivery or authorize
implementation. The #303 R-1 technical milestone can be reviewed
separately (R-2 is deferred to #278 under HD-008), but blocked Lane L scenarios cannot be accepted at R-3 until
G-2 is genuinely resolved (and G-4 consented linkage is available).
G-1 is tracked as a separate Project-policy evolution; its unavailable
operations must be reported truthfully.

| Gap | Follow-up | Status |
|---|---|---|
| G-1 | [#305](https://github.com/rgomids/axiom/issues/305) | Open technical dependency |
| G-2 | [#306](https://github.com/rgomids/axiom/issues/306) | Open technical dependency; blocks R-3 C–G |
| Doctor/Project readiness | [#231](https://github.com/rgomids/axiom/issues/231) | Existing Project diagnostic scope; no new Doctor in #303 |

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

**Inference boundary and stage of delivery (HD-008).** The **#303 technical
phase** runs canonical CLI/contract checks, installation/parity tests and
controlled/synthetic planning tests. It **does not require or dispatch**
real vendor model inference, interactive native Codex/Claude sessions, live
stage agents or external Provider effects. Safe local deterministic binary
checks may run without calling a vendor; they are not evidence of real
vendor behavior. Read-only status/auth observation (e.g. `codex login status`)
proves only the specific observation and is never promoted to live inference
proof.

**[Issue #278](https://github.com/rgomids/axiom/issues/278) / AXM-12**
owns the **final MVP operational/human phase**, after #276/#277 and required
readiness dependencies: interactive natural-language journeys in real Codex
and Claude sessions, consented login/auth probes and real Runtime dispatch,
workflow and Work Item end-to-end execution, authorized artifacts/rework,
and the maintainer's explicit accept/reject decision. These operations
require separate actual user authorization at that time; no present decision
authorizes inference, paid effects, Provider mutation or acceptance.

**Credentials and isolation (future #278 real tests).** A native session authenticates through that
Runtime's own existing login on the user's machine; the Axiom sandbox isolates
Axiom state, installed skills and Repositories, not the vendor login. The
Runtime and the skill never copy, export, print or move credentials, tokens,
login files or vendor configuration into scratch, Evidence, the sandbox state
roots or the isolated Runtime homes used for Lane L. Lane L `runtime.readiness`
observations of an isolated home therefore report it as unauthenticated, which
is the expected sandbox result, not a failure. Evidence records only Runtime
name, version and the canonical Axiom results; never credential values,
environment values or host paths.

**Evidence lanes** (how a result was produced; L/S are technical validation only and do not claim real vendor execution).

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
| R-1 Technical verification (**#303**) | Contract, skill catalog/routing, fresh install/upgrade, planner and fail-closed behavior are proven technically | Deterministic repository and CLI tests, read-only local binary checks where safe, Lane S controlled observations and sanitized Evidence | Never vendor operation, native conversation or end-to-end acceptance |
| R-2 Native conversational operability (**#278**) | Real Codex and Claude sessions can drive the accepted workflow/Work Item journeys through language | Explicitly authorized native sessions and actual Runtime observations at final MVP validation | CLI/skill metadata, mocks, synthetic routing, Lane S, binary-only checks |
| R-3 MVP end-to-end functional/human acceptance (**#278 / AXM-12**) | Essential real Project → Workflow → Work Item → Execution → delivery/rework scenarios work and receive the maintainer's explicit acceptance | Real authenticated/dispatched Codex and Claude, the end-to-end Evidence required by #278, review and a separate human accept/reject decision | Any synthetic PASS, untested/blocked scenario, technical issue closure, merge, green CI or release alone |

A `Lane S = passed`, `Lane L = blocked / not run` scenario may contribute
to **R-1 PASS only**. Its R-2/R-3 fields must read **`deferred_to_278`**
(or `blocked: G-n` when an actual prerequisite is unsatisfied), never
`passed`, `accepted` or implicit vendor proof. The #303 technical outcome
is **reviewed R-1 Evidence**, not a waiver of the real MVP journey.

**Lifecycle boundary:** once #303 is implemented, its technical contract/tests
pass and any required *technical* reviewer/maintainer authorization is given,
the product may reconcile the AXM-7 *technical dependency* for the following
#276/AXM-8 work; **do not keep AXM-8 blocked merely because E2E testing has
been deliberately assigned to final #278**. This is not an automatic Linear
status transition, product-wide acceptance or unblocking now. #278 / AXM-12
keeps the real validation and explicit final human acceptance gate. G-2
[#306](https://github.com/rgomids/axiom/issues/306), consented G-4 linkage
and other prerequisites must be satisfied before those real scenarios can
actually pass. Findings from #278 are then addressed as bounded follow-ups
to their owning capability; historical synthetic PASS is never reinterpreted.

### Deferred native conversational scenarios (R-2, owned by #278 / AXM-12)

These scenarios are a **handoff checklist**, not execution or acceptance
criteria for #303. A final real scenario must run in an authenticated
**Codex** session and an authenticated **Claude** session, with the
maintainer's authorization at the time, starting from natural language:

1. Discover the configured skills and verify HD-005/HD-007 ownership
   (`axiom-workflow` configures, `axiom-project` selects active revision,
   `axiom-work-item` runs a specific Work Item).
2. Create/edit a workflow definition through a Runtime-prepared draft.
3. Validate and present the canonical Lingo preview and effects.
4. Capture explicit approval of each exact preview, mutate only authorized
   sandbox/local targets and read back the result.
5. Resolve Project Runtime/Profile allowlist vs local available Profiles
   through the canonical resolver, without credential leakage or fallback.
6. Exercise real Work Item/Execution and Stage/Agent flow, including Codex +
   Claude, as supported after #276/#277 delivery; verify provenance, safety,
   refusal/recovery, authorized PR/rework and immutable workflow binding.
7. Obtain the **separate human MVP accept/reject decision** over sanitized
   operational Evidence; document any defects as bounded owned corrections.

Vendor inference, local authentication probes and external effects remain
unexecuted until their precise future authorization. Real vendor session
Evidence must cite observed versions, effective identity/mode, skill-set and
canonical command digests, without tokens, host paths or raw chat. A
`not_run` scenario is never `passed`.

**Report (H).** Axiom version and source revision; observed Runtime information
(or `unknown`); skill-set digest; sandbox identifier (no host paths); R-1
scenarios with `passed` / `failed` / `blocked`, classified refusals,
provenance, canonical digests and Evidence class. R-2/R-3 are reported
separately as `deferred_to_278` or with a concrete blocker (G-1–G-5), not
`passed`. Include the #278/AXM-12 handoff and known limitations. Raw logs,
credentials, environment values and host paths are never published.

## Codex/Claude parity

Both Runtimes install the same embedded `axiom-workflow` configuration bytes
and resolve the same catalog. Validation covers, per Runtime: discovery after
install, upgrade and reinstall; natural-language configuration intents and
canonical operations; Project-owned selection; Work Item-owned execution,
stage planning and acceptance routing; missing-input acquisition; preview and
confirmation; failure classification; human-readable presentation; Evidence. Runtime-specific differences are limited to
installation location and invocation syntax (`$axiom-workflow` /
`/axiom-workflow`) and must not change semantics, authority or state.
Synthetic skill-routing tests prove **technical parity only** in #303.
Native Codex/Claude behavioral sessions and all real vendor inference are
explicitly deferred to #278/AXM-12, where separate authorization is required.

## Acceptance criteria (AC-015)

1. Accepted amendment and ADR-0022, with the superseded Specification fragments annotated.
2. `axiom-workflow` is embedded, installed and discoverable for Codex and
   Claude, including upgrade from v0.15.0 and reinstall. Skill-set history
   retains the historical v0.15.0 set for provenance, while active installed
   skills, `skill inspect`, help and routing catalogs reflect **no duplicate
   authoring/validation routes** after replacement.
3. Every `axiom-workflow` **configuration** action routes to its existing
   canonical Project workflow CLI command with matching flags/authority/results.
   The old definition list/show/create/edit/validate/remove/recover **skill
   routes are absent from `axiom-project`**, with no aliases or forwarding;
   `workflow.select` stays Project-owned and `run/status/plan` Work Item-owned.
   Existing canonical CLI operations and domain semantics remain unchanged.
4. Definition, stage and agent configuration work from natural language
   through Runtime-prepared drafts. Project selects the active revision with
   its own approval. Starting a specific Work Item resolves that Project
   selection without a free-standing workflow run or silent fallback; each
   Execution binds the selected immutable revision. Canonical start already
   provides this (verified above); the skill adds no selection logic.
5. The **Work Item-owned** #275 acceptance test procedure is operable as a
   **deterministic/synthetic technical R-1** runner, with correct skill
   ownership and canonical preview/Plan behavior. In Codex/Claude, installed
   skill inventories, parity and routing are tested without requiring an
   actual vendor conversation. Every synthetic scenario is explicitly
   classified, including G-n blockers, and the report separates R-1, R-2
   and R-3 statuses.
6. **HD-008: no real Codex/Claude sessions, paid inference, live stage dispatch,
   Provider effects or human end-to-end tests are a #303 technical delivery
   condition.** R-2 and R-3 are reported as deferred to #278 / AXM-12, never
   marked passed based on synthetic Evidence. A bounded handoff/checklist is
   published for #278.
7. #303 technical delivery requires R-1, all other applicable technical
   acceptance criteria, review and its own normal authorization. The final
   native interaction and end-to-end human MVP test belongs to #278 / AXM-12;
   G-2/G-4 may block live scenarios there, but do not silently waive them.
8. Canonical CLI, result, digest, preview/apply, security and authority
   semantics are unchanged (existing tests pass unmodified).
9. HD-006 policy-vs-local registry boundaries are verified: #305 and #306
   are linked; no fixed three-profile constraint, no Project/local conflation,
   no unproven Runtime capability and no hidden credential/authority mutation.
   Project readiness reuses #231 without implementing a new Doctor here.
10. An independent review reports no unresolved Blocker/Major.

## Implementation plan (authorized only after acceptance)

1. Configuration skill and catalog: `internal/codexruntime/skills/axiom-workflow/SKILL.md`;
   `skillOperationSpecs("axiom-workflow")` reusing **only definition/configuration**
   specs; embedded inventory (`codexruntime`, `install`), receipts, skill-set
   history; route selection to Project and run/plan/acceptance to Work Item.
2. In the **same #303 implementation**, remove obsolete `axiom-project`
   configuration routes and all duplicate help/catalog/routing text, retaining
   Project selection and Work Item execution. Validate fresh install, upgrade
   from v0.15.0, reinstall, idempotence, receipt/history preservation and
   negative discovery in Codex and Claude; do not schedule a later cleanup.
3. **Work Item-owned** #275 acceptance runner (Lane S) and report schema with
   R-1 technical Evidence and explicit R-2/R-3 `deferred_to_278` states,
   registered as durable automation. Deterministic Lane L CLI checks are
   optional within #303 and are never vendor-inference proof.
4. Prepare and cross-link the **future** real Codex/Claude R-2 and end-to-end
   human R-3 checklist in #278 / AXM-12. Do not execute those sessions in
   #303 or require their Evidence for #303 delivery.
5. Documentation: `docs/commands.md`, `docs/agent-harness.md`, Spec index, this
   amendment's Evidence section.

## Human decisions requested

| ID | Decision | Options | Recommendation | Harder to change later |
|---|---|---|---|---|
| HD-005 | Dedicated configuration surface and explicit ownership | A keep HD-002; **B `axiom-workflow` configuration only, `axiom-project` selects, `axiom-work-item` runs/plans/accepts, Execution binds an immutable selected revision (ADR-0022)**; C plus new CLI tree; D new workflow domain | **B — formally accepted 2026-10-10; exact reviewed revision bound in the status section** | Skill and operation names become public conversational contracts |
| HD-006 | Project allowlist vs local Runtime/Profile registry, Workflow selection, readiness | **A (maintainer direction):** keep Project policy portable/editable [#305], manage multiple named local Profiles per Runtime [#306], Workflow binds Stage/Agent references, Project/workflow/host readiness is verified by canonical validators (#231). Separate the technical follow-ups from #303 implementation; **B:** conflate portable Project policy and machine-local Profiles inside #303 | **A — formally accepted 2026-10-10**; preserve Spec 002 v2 intersection/bounds and security/authority. Both Issues tracked; implementation separately gated. | G-2 still blocks real Lane L C–G and AXM-7 R-3 until shipped; adding a global Doctor or changing Project schema needs its own reviewed scope |
| HD-007 | Immediate removal of overlapping skill routes | A keep legacy `axiom-project` definition-authoring routes for a compatibility window; **B remove those routes in the same #303 implementation and make `axiom-workflow` their only skill owner, with no aliases or deprecation window** | **B — formally accepted 2026-10-10**, pre-MVP with no external users. No compatibility period. Preserve Project `workflow.select`, Work Item `run/status/plan` and existing canonical CLI. | Active skill inventory/metadata changes immediately on upgrade; historical v0.15.0 skill-set receipt/history remains intact for provenance. Renaming CLI commands is out of scope. |
| HD-008 | Synthetic verification now; native and human E2E at final MVP acceptance | **A (selected):** accept deterministic and synthetic Evidence for #303 technical R-1, without requiring native sessions or real stage dispatch; defer R-2 and R-3 to #278 / AXM-12. **B:** require real R-2/R-3 before #303 technical delivery. | **A — formally accepted 2026-10-10**; R-1 synthetic PASS is never real Runtime proof. R-2/R-3 remain future gates. | #278 is the explicit owner of real Codex/Claude E2E and final human accept/reject; defects found there become scoped corrections; AXM-8 technical dependency is not gated on final-MVP tests once #303 is otherwise accepted. |

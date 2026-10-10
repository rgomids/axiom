# Specification 007 amendment — dedicated `axiom-workflow` conversational surface (#303)

## Status and authority

**Proposed, 2026-10-10**, for human decision under
[Issue #303](https://github.com/rgomids/axiom/issues/303) (Linear AXM-7),
together with [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md).
It records the maintainer direction of 2026-10-10 ("adopt a dedicated
`axiom-workflow` conversational skill and make the complete #275 testing/acceptance
preparation operable in natural language in Codex and Claude"). That direction
is not acceptance of this text. Until the maintainer accepts this exact
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

A developer working in Codex or Claude describes workflow outcomes in natural
language — author or change a workflow, configure stages and agents, select a
revision, plan a stage, run the #275 acceptance — and the Runtime discovers
state, collects only genuinely missing facts, prepares drafts, validates them
through Axiom, presents the canonical preview and effects, obtains explicit
approval for the exact preview, applies only that, and reads back the result.
The developer never authors JSON, computes a digest, looks up an internal
revision, assembles CLI arguments or knows a test name for a supported journey.

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
| WF-015 | Dedicated installed `axiom-workflow` conversational surface in Codex and Claude over existing canonical operations, with compatibility routes, Runtime-prepared drafts and a conversational #275 acceptance procedure | #303 | AC-015 |

WF-001–WF-014 owners and acceptance are unchanged. WF-015 adds no behavior to
WF-009 (#276), WF-011 (#277) or WF-012 (#278).

## Skill ownership

| Skill | Primary conversational ownership after this amendment | Workflow routes kept for compatibility |
|---|---|---|
| `axiom-project` | Project lifecycle, identity, readiness, Repository associations, Work Item Provider and Integrations, Project Runtime/Model Profile policy authoring at create | `workflow.list/show/create/edit/validate/select/remove/recover` unchanged |
| `axiom-work-item` | Work Item create/select/list/show/update/comment/close/reopen | `run`, `status`, `plan` unchanged |
| `axiom-workflow` | Workflow definitions and revisions, stage/agent configuration, Runtime/Profile references and read-only readiness, Execution workflow interaction, stage planning, workflow acceptance procedures | — |

Rules:

1. One canonical action, many routes: every `axiom-workflow` operation maps
   to already registered command actions; the same action keeps the same
   flags, authority metadata and result in every skill that routes it. Routing
   tests assert identical `skill inspect` metadata per action and mode;
   operation groupings may differ (for example compatibility `run` also
   contains reconcile, which `axiom-workflow` exposes as `execution.reconcile`).
2. Compatibility routes stay installed and behave identically for at least the
   compatibility window chosen in HD-007. Their skill text points to
   `axiom-workflow` as primary; their `skill inspect` payload does not change
   shape. Removal needs a separate versioned decision and the skill retirement
   protocol.
3. Natural-language workflow intent arriving at `axiom-project` or
   `axiom-work-item` is still served (it may be routed to the same commands);
   it is never refused only because another skill is primary.
4. A workflow intent that also needs a Project or Work Item lifecycle effect
   (for example a new Project or linking a Provider Issue) uses that owner's
   canonical command with its own preview and authority; `axiom-workflow` may
   orchestrate the sequence but never merges two previews into one approval.

## `axiom-workflow` operation catalog

Operation names are proposed conversational contracts; flags, requirements and
authority come from `axiom skill inspect axiom-workflow`, generated from the
existing command flag sets (no second registry).

| Operation | Canonical command(s) | Effect | Authority |
|---|---|---|---|
| `definition.list` / `definition.show` | `axiom project workflow list` / `show` | read-only | none |
| `definition.validate` | `axiom project workflow validate` | read-only | none |
| `definition.create` | `axiom project workflow create` (`--from-default` or a Runtime-prepared `--file` draft) | local mutation | preview first; exact `--expected-revision`, `--preview-digest`, `--authorize-local` |
| `definition.edit` / `definition.select` / `definition.remove` / `definition.recover` | `axiom project workflow edit` / `select` / `remove` / `recover` | local mutation | as above |
| `execution.list` / `execution.status` | `axiom workflow list` / `status`, `evidence` | read-only | none |
| `execution.run` | `axiom workflow start`, `advance`, `fact`, `resume` | local mutation | unchanged reviewed start (`--runtime-preview`), exact-revision gate rules, `fact` only with `--authorize-local` (exact metadata from `skill inspect`) |
| `execution.reconcile` | `axiom workflow reconcile` | external mutation | preview first; exact `--preview-digest`, `--authorize-external` |
| `stage.plan` | `axiom workflow stage plan` | read-only | none; proposal never dispatches, grants effects or satisfies gates |
| `runtime.readiness` | `axiom runtime profile validate` / `preview`, `axiom runtime codex\|claude status` / `auth` | read-only | none |
| `acceptance` (mode `stage-plan`) | Guided composition of the rows above plus the registered acceptance runner (see below) | sandbox-local only | each step keeps its own preview/authority; no external effect |

Stage dispatch/retry/cancel, coordination, delivery, decision and rework (#276,
#277) are **not** registered. The skill reports them as not yet available and
names the owning Issue. A route is added only by the change that delivers the
operation.

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

1. The Runtime may write bounded draft inputs (workflow definitions, stage Plan
   documents, stage results) only to a Runtime-owned scratch directory outside
   the Project working copy, every Repository working copy and Axiom state
   roots. It never writes Project, workflow index, Execution, Runtime Profile or
   receipt files.
2. A draft is built only from the user's statements, the current canonical
   read results and explicit defaults that the skill names. Unknown or
   ambiguous material values (stage, scope paths, effects, authority ceiling,
   Runtime constraint, Profile, effort) are asked, never invented.
3. Every draft passes through the canonical validation (`definition.validate`,
   a mutating preview or `stage.plan`) before it is presented. Lingo's
   diagnostics are presented as returned; the Runtime does not pre-judge them.
4. Presentation shows the user a readable summary **and** the exact canonical
   preview/effects/digest returned by Lingo; approval binds to that exact
   preview, never to the summary.
5. **Plan approval.** The Runtime writes the complete candidate Plan document
   to scratch exactly as it would be used, including `approved: true`,
   `planRevision` and `planDigest`, runs the read-only `stage.plan` on it and
   presents the scope, effects, authority ceiling and Lingo's proposal with
   its `plan.planDocumentDigest`. The user's approval binds only to that exact
   `planDocumentDigest`; any byte change requires a fresh `stage.plan` and a
   fresh approval. `planRevision` and `planDigest` are opaque references with
   no authority: `planDigest` is the SHA-256 of the exact bytes of a reviewed
   Plan/Tasks artifact the user names, or, when none exists, of a bounded
   scope artifact the Runtime writes to scratch and shows. The Runtime never
   defines another canonical encoding. `approved: true` is the operator
   assertion described in G-5, decided in the conversation; it creates no
   authority fact, gate fact or dispatch authority.
6. Drafts are working files, not Evidence. Evidence cites canonical results and
   digests.

## Runtime and Model Profile capability matrix

| Concern | Owner / layer | Current operation | `axiom-workflow` coverage |
|---|---|---|---|
| Project allowlist, Model Profiles, preferences at Project creation | Project (portable policy v2) | `project configure` CREATE `--runtime`, `--model-profile`, `--runtime-preference` | Explains and orchestrates through `axiom-project configure` with its own preview/authority |
| Same policy on an **existing** Project | Project | none (CREATE-only) | Unavailable: reported as G-1, never patched |
| Stage/agent Runtime constraint, Profile reference, effort | Workflow definition | `definition.create`/`edit` | Supported |
| Machine-local Runtime Profile store (adapters, models, credential references) | Machine-local | `runtime profile validate` (read-only) | Read-only; authoring unavailable (G-2) |
| Runtime installation / skill integration / auth observation | Machine-local | `runtime codex\|claude status`, `auth`; `runtime profile preview` | Read-only `runtime.readiness` |
| Capability and explicit-effort proof beyond `axiom-skills` | Machine-local observation | production observer proves only `axiom-skills` | Reported truthfully as `runtime_unresolvable` / `unsupported_effort` (G-3, #272) |

The skill never infers a capability from an executable on `PATH`, a model name,
a login state or the Runtime it is running in.

## Gaps and dependencies

| Gap | Missing contract | Impact | Proposed disposition |
|---|---|---|---|
| G-1 | Preview/apply edit of Project Runtime/Model Profile policy on an existing Project | Policy changes need a new Project or a historical authored-manifest path | Follow-up Spec 002 amendment; not in #303 (HD-006) |
| G-2 | Preview/apply authoring of the machine-local Runtime Profile store | A fresh sandbox cannot create the Profile configuration that `workflow start` and `stage.plan` need; live Execution-dependent acceptance is blocked | Follow-up with its own credential-reference security review (HD-006) |
| G-3 | Authoritative capability / model-specific effort observation | Positive explicit-effort or non-`axiom-skills` planning is only provable synthetically | Existing #272 |
| G-4 | Provider-free Work Item linkage for sandboxes | A live Execution needs a linked existing Provider Issue (`work-item select`, Provider read, sandbox-local link) | No new contract; use an existing Issue the user names, never create one without external authority |
| G-5 | Recorded Plan approval fact | Approval is asserted by the Plan document | Existing #276 binds the reviewed `plan.digest` before dispatch |

## Conversational #275 acceptance (`acceptance`, mode `stage-plan`)

One natural-language request in either Runtime starts it. The Runtime assembles
and runs the procedure, asks only for genuinely missing consent/inputs, and
reports results in natural language with a bounded Evidence report.

**Safe environment.** Every live step runs the installed `axiom` with fresh
temporary `LINGO_STATE_ROOT`, `AXIOM_CODEX_SKILLS_ROOT`, `CLAUDE_CONFIG_DIR`,
`CODEX_HOME`, `HOME`/`USERPROFILE` and a temporary Git working copy. The
user's real Axiom state, skills and Repositories are never read for mutation
or written. No agent is dispatched and no vendor inference runs; the only
vendor processes are the existing read-only status/auth observations of
`runtime.readiness` (for example `codex login status`), which the report names.
No Provider resource is created or mutated and nothing is published.

**Evidence lanes.**

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
| B Workflow authoring | `definition.list` default, custom draft, validate, create/edit preview → approval → apply, select, read-back | covered by existing #273 tests |
| C Single agent | blocked by G-2 (no Profile store) and G-4 (linked Provider Issue needs a consented Provider read with credentials the sandbox does not hold) | `executionKind: single`, one resolution, no graph identity |
| D Multi-agent | blocked by G-2 and G-4 | independent Codex/Claude agents, integrator depending on both, concurrency, child inputs/outputs, integration ownership |
| E Determinism | blocked by G-2 and G-4 | repeated identical digests; config/observation change invalidates; Execution revision binding; no fallback |
| F Security | definition-level refusals (cycle, dangling dependency, invalid definition); plan-level cases blocked by G-2 and G-4 | invalid Plan, stale revision, missing inputs, unavailable Runtime, unsupported effort, cycle, dangling, unsafe overlap, authority escalation, `..`, `C:/outside`, `C:outside` |
| G Read-only | before/after canonical status and state-tree digests around every read-only step | no Execution/attempt/stage advance/gate/authority/Provider/artifact change |
| H Evidence | report below | same report |

When G-2 is delivered **and** the user consents to link an existing Provider
Issue they name (G-4), C–G gain Lane L coverage for stages requiring only
`axiom-skills` with runtime-default effort; explicit effort stays Lane S until G-3.

**Report (H).** Axiom version and source revision; Runtime name/version as
observed (or unknown); skill set digest; sandbox identification (no host
paths); scenarios with `passed` / `failed` / `blocked` and the classified
category for each refusal; digests and canonical references; lane and Evidence
class per result; known limitations (G-1–G-5); unresolved acceptance
requirements. Raw logs are not the default presentation and credentials,
environment values and host paths never appear.

## Codex/Claude parity

Both Runtimes install the same embedded `axiom-workflow` bytes and resolve the
same catalog. Validation covers, per Runtime: discovery after install, upgrade
and reinstall; natural-language intent resolution; explicit operation
invocation; missing-input acquisition; preview and confirmation; workflow
configuration; stage planning; failure classification; human-readable
presentation; Evidence generation. Runtime-specific differences are limited to
installation location and invocation syntax (`$axiom-workflow` /
`/axiom-workflow`) and must not change semantics, authority or state.
Synthetic routing tests are distinguished from native Runtime behavioral runs,
which need explicit authorization when they involve vendor inference.

## Acceptance criteria (AC-015)

1. Accepted amendment and ADR-0022, with the superseded Specification fragments annotated.
2. `axiom-workflow` is embedded, installed and discoverable for Codex and
   Claude, including upgrade from v0.15.0 and reinstall; skill-set history pins
   the replaced v0.15.0 set; `skill inspect`, help and routing catalogs agree.
3. Every catalog action and mode routes to the canonical command with
   metadata identical to its compatibility route; compatibility routes are
   unchanged.
4. Definition, stage and agent configuration work from natural language
   through Runtime-prepared drafts; required approvals stay explicit.
5. The conversational #275 acceptance produces the report above in both
   Runtimes, with every scenario classified and every blocked item tied to G-n.
6. Canonical CLI, result, digest, preview/apply, security and authority
   semantics are unchanged (existing tests pass unmodified).
7. An independent review reports no unresolved Blocker/Major.

## Implementation plan (authorized only after acceptance)

1. Skill and catalog: `internal/codexruntime/skills/axiom-workflow/SKILL.md`;
   `skillOperationSpecs("axiom-workflow")` reusing existing specs; embedded
   inventory (`codexruntime`, `install`), receipts, skill-set history;
   compatibility text in the two existing skills.
2. Install/upgrade/retirement and parity tests for Codex and Claude.
3. Acceptance runner (Lane S) and report schema, registered as durable
   automation; Lane L procedure in the skill.
4. Documentation: `docs/commands.md`, `docs/agent-harness.md`, Spec index, this
   amendment's Evidence section.

## Human decisions requested

| ID | Decision | Options | Recommendation | Harder to change later |
|---|---|---|---|---|
| HD-005 | Dedicated surface | A keep HD-002; **B thin `axiom-workflow` (ADR-0022)**; C plus new CLI tree; D new workflow domain | B | Skill and operation names become public conversational contracts |
| HD-006 | Runtime/Profile gaps G-1/G-2 | **a. record as follow-ups; accept Lane S for C–G**; b. deliver G-2 (and G-1) inside #303 | a — G-2 adds credential-reference authoring that needs its own Spec 002 amendment and security review | With (a), live C–G stays blocked until the follow-up ships |
| HD-007 | Compatibility window | **keep routes until a separate removal decision**; remove at the next minor | keep | Removal later requires retirement/upgrade protocol |
| HD-008 | Synthetic Evidence for #275 conversational acceptance | **accept, classified `synthetic`, never operational vendor proof**; require Lane L only | accept | #278 still owns live two-Runtime proof |

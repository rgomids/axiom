# Plan and Tasks — #303 dedicated `axiom-workflow` configuration skill

## Status

**Proposed, 2026-10-10**, for independent review (Code Review Guardian) and
then explicit maintainer acceptance. This document is **not accepted** and
authorizes no implementation, merge, release, Issue or Linear change.

| Item | Value |
|---|---|
| Work Item | [#303](https://github.com/rgomids/axiom/issues/303) / Linear AXM-7 |
| Contract (immutable for this plan) | [Spec 007](spec.md), [amendment](amendment-303-workflow-skill.md), [ADR-0022](../../decisions/0022-dedicated-workflow-conversational-surface.md); **Accepted 2026-10-10** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6099596104)) on PR #304 `7db80ac95b5e83cd8cc9aaaadc785633c3ee3fa9` (amendment blob `9d1895b6…`, ADR blob `13a52a62…`) |
| Planned on | PR #304 branch at `804123915d1c2eccaa81a3fbf01253aa3910ae48`; differs from `7db80ac` only in status/annotation text |
| Planning decisions | **PD-1 to PD-9 individually accepted** by the maintainer on #303 (links in §8). Individual PD acceptance is not acceptance of this Plan/Tasks as a whole, nor of implementation or merge. The PD-1 Project read path needs one further explicit authorization of its additive output delta before T08a (§3.4, R10) |
| Code baseline | `main` `9378c41` (v0.15.0 + #298 CI routing); every file cited below is unchanged from v0.15.0 `3766273` |
| Starting point | Amendment §"Implementation plan" steps 1–5, refined here; HD-005–HD-008 are not reopened |

Observed statements cite `file:line` on the baseline. Statements marked
*(inferred)* need confirmation in the named task.

---

# Part I — Plan

## 1. Technical objective

#303 introduces `axiom-workflow`, an embedded Runtime skill for Codex and
Claude. It is the **only** conversational owner of Project workflow
**definition configuration**: list, show, validate, create (built-in copy or
Runtime-prepared draft), edit, remove and recover definitions, stages and
agents, plus read-only Runtime/Profile configuration readiness. Every
operation routes to an existing canonical command.

In the same delivery:
- `axiom-project` loses its duplicate definition-authoring skill routes and
  keeps Project lifecycle, policy and `workflow.select`. Its own `show`/`list`
  report each Project's active workflow (ID, name, revision) read-only (PD-1);
- `axiom-work-item` keeps `run`/`status`/`plan` and gains the documented #275
  stage-plan acceptance procedure, backed by a deterministic synthetic **R-1**
  runner.

**User value:**
- one unambiguous skill per workflow intent ("Project selects, Workflow
  configures, Work Item runs, Execution performs");
- no hand-written definition JSON;
- no digest lookup or CLI assembly for supported configuration journeys.

**Outside scope:**
- G-1 [#305](https://github.com/rgomids/axiom/issues/305),
  G-2 [#306](https://github.com/rgomids/axiom/issues/306),
  readiness [#231](https://github.com/rgomids/axiom/issues/231), and any new
  global Doctor;
- #276 dispatch, #277 delivery/rework, #278 R-2/R-3;
- any new or renamed CLI command, flag, payload, schema, engine, Runtime
  registry, approval mechanism, scheduler or fixed Profile tier. **The single
  named exception** is the PD-1 additive, read-only `activeWorkflow` result
  field on `project show`/`project list` (§3.4), which is **not authorized by
  this document** and needs the explicit authorization in R10 before T08a;
- `--file` regular-file/symlink hardening, owned by
  [#307](https://github.com/rgomids/axiom/issues/307) (PD-9 A);
- real inference, native sessions, Provider effects, release, and the
  Lane L sandbox runner (PD-5 A: not built).

## 2. Current architecture (observed)

| Concern | Module / function | Current behavior | #303 impact |
|---|---|---|---|
| Canonical skill text | `internal/codexruntime/skills/{axiom-project,axiom-work-item}/SKILL.md`; embedded by `//go:embed skills/*/SKILL.md` (`internal/codexruntime/runtime.go:15`) | Two domain skills | Add `axiom-workflow/SKILL.md`; edit both |
| Operation catalog | `internal/cli/skill_inspect.go:200-208` `skillOperationSpecs`; `inspectSkill` (`:267`) | Switch on two names; Project case appends `workflowAuthoringSkillSpecs()` | New case; split specs |
| Authoring specs | `internal/cli/workflow_authoring.go:107-122` | 8 `workflow.*` specs over `project_workflow_*` actions | 7 move to `definition.*`; `select` stays |
| Inspect flag registry | `internal/cli/skill_inspect.go` `skillFlagSet`; default `projectFlagSet` (`internal/cli/cli.go:808-812`) | Runtime actions not mapped (default would add `--slug`) | Map `runtime_profile_validate/preview` (`runtimePreviewFlagSet`, `internal/cli/runtime_preview.go:94`) and `runtime_{codex,claude}_{status,auth}` |
| Project workflow commands | `internal/cli/commands.go:38`, `workflow_authoring.go:22-48,124-130`; `cmd/lingo/workflow_authoring.go:21-40`; `internal/projectapp/workflow_authoring.go:58` `ApplyWorkflow` | Preview/apply with `--expected-revision`, `--preview-digest`, `--authorize-local`; `--file` size-bounded, no regular-file/symlink check | Unchanged; `--file` hardening owned by #307 (PD-9 A) |
| Active selection | `internal/projectapp/workflow_authoring.go:69-76,205-248`; stored in the portable Project `workflowSelection` (`internal/project/project.go:84-91`, validated by `internal/project/workflow.go:9-23`) | Every authoring report includes `selection` as a ref (ID, revision, digest, source) **without a name**; `select` without authority returns a read-only preview of a change | `select` unchanged; **not** used as the Project inspection (PD-1) |
| Project show / list | `cmd/lingo/project_lifecycle.go:70-95` `listProjects`, `:101-133` `showProject`; `internal/projectapp/list.go:37-42` `ProjectSummary{ID, Slug, Name, Status}`; views `internal/cli/cli.go:263-277` `ProjectView`/`ProjectListView` | Neither result exposes the active workflow; the list catalog already reads portable definitions for display names (`list.go:54-60`) | Additive read-only `activeWorkflow` field (PD-1, §3.4, T08a), subject to R10 authorization |
| Workflow validation | `internal/workflowdefinition/definition.go:144,244` `Encode`/`Decode`, `StrictJSONBounded` | Strict schema, limits, DAG rules | Unchanged; drafts validated through it |
| Work Item operations | `internal/cli/skill_inspect.go` `workItemRun/Status/Plan`; `internal/cli/workflow_stage_plan.go`; `cmd/lingo/workflow_stage_plan.go` | `run`/`status`/`plan` | Catalog unchanged; SKILL text extended |
| Execution binding | `cmd/lingo/workflow_binding.go:40-110` | Resolves `workflowSelection`, fails closed | Unchanged |
| Runtime/Profile resolution | `internal/runtimeprofile/runtimeprofile.go:127` `Resolver.Resolve`, `:193` `Validate`; `internal/runtimeapplication/policy.go:59` `Preview`, `:148` `Check` | Intersection, fail-closed | Unchanged; cited for HD-006 tests |
| Stage planning | `internal/workflowcompiler/stage_plan.go:117` `Compiler.PlanStage` | Read-only proposal | Unchanged; R-1 runner source |
| Skill installation | `internal/codexruntime/runtime.go:21` `skillNames`; `Service.Install` (`:~140-250`); `integration.go:~49` `NewClaude` | Same code for both Runtimes; absent skill created; foreign bytes conflict; rerun no-op | Name added |
| Codex root / Claude root | `cmd/lingo/main.go:172` (`AXIOM_CODEX_SKILLS_ROOT` or `~/.agents/skills`); `internal/runtimebootstrap/bootstrap.go:~150` (`$CLAUDE_CONFIG_DIR/skills` or `~/.claude/skills`) | — | Unchanged |
| Skill-set history | `internal/codexruntime/manifest.go:13-14` (`SkillSetVersion`/`BinaryCompatibility` "2"), `sharedSkillHistory` (newest entry v0.14.0, `:~207-211`), `currentRevision`; `shared_history_test.go:22` `publishedSharedRevisions` | Current v0.15.0 set (`axiom-project d80cd1d1…`, `axiom-work-item e47c9256…`) is not yet a history entry | Append it; pin new digest |
| Retirement | `internal/codexruntime/retirement.go` (`retiredSkillNames`, `runtime.go:24-27`) | Removes retired directory names only | Not used (no directory retired) |
| Archive / installer | `internal/install/archive.go:35` `skillNames`, `:225` exactly `3+len(skillNames)` lines; `internal/install/upgrade.go:871` `planSkills`, `:~966` `installedSkillManifest`; `scripts/install-release.sh:199-206` (5 lines, 2 skills, 2 names) | Upgrade publishes Codex root; Claude converges via `first-run` (ADR-0016) | Names and counts change; v0.15.0 compatibility risk R1 |
| Routing contract tests | `internal/cli/skill_catalog_contract_test.go:13,35` `TestCanonicalRoutingContract`; `domain_skill_routing_test.go:24,125-139`; catalog block in [issue-230 matrix](../004-mvp-v1-baseline/issue-230-resource-lifecycle-matrix.md) | Three-way equality: SKILL.md table, inspect, matrix doc | Update all three atomically |
| Automation / CI | `scripts/automation-registry.json`, `scripts/check-automation-registry.py`; `scripts/classify-ci-changes.py:20-47`; `.github/workflows/ci.yml` | Every governed script registered; `internal/**` triggers go/verify/upgrade; `scripts/**` triggers a full run | Register the runner |

## 3. Proposed architecture

1. **Single catalog, three owners.**
   - Split `workflowAuthoringSkillSpecs()`: `definition.list|show|validate|create|edit|remove|recover`
     reuse the same `project_workflow_*` actions, modes and authority
     strings under a new `axiom-workflow` case.
   - The `axiom-project` case keeps only `workflow.select` among workflow
     operations; its existing `show`/`list` operation metadata (actions,
     modes, flags, authority `none`) is unchanged and carries the PD-1 read.
   - No `definition.list` or other definition read is added to
     `axiom-project`.
   - Add `configuration.readiness` with read-only modes over
     `runtime_profile_validate`, `runtime_profile_preview`,
     `runtime_codex_status`, `runtime_claude_status`, `runtime_codex_auth` and
     `runtime_claude_auth` (PD-2 A,
     [decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101705557)),
     and add matching `skillFlagSet` cases.
   - The skill reports three distinct readiness facts, never merged into one
     verdict: **Configuration Valid** (`runtime profile validate/preview`),
     **Runtime Ready** (`runtime codex|claude status`) and **Authentication
     Ready** (`runtime codex|claude auth`). Auth is a read-only status
     observation: no login, no credential write, and it never blocks
     authoring. Every result is reported as point-in-time.
   - Invariant: no action is registered in two skills.
2. **Draft preparation (skill text only).** The Runtime writes a draft only
   to its own scratch outside the Project, Repository and Axiom state roots.
   It builds the draft from user statements, canonical reads and named
   defaults, and asks for any material unknown. The sequence is:
   - `definition.validate`;
   - mutating preview;
   - the user approves the exact preview digest;
   - apply with `--expected-revision`, `--preview-digest`, `--authorize-local`;
   - `definition.show` read-back.

   Creating or editing a definition never selects it. Activation is a
   separate `axiom-project` `workflow.select` preview and approval.
3. **Removal without forwarding.**
   - `axiom-project` SKILL.md drops its authoring rows and the "Workflow
     definitions" prose, and gains one ownership rule pointing definition
     intents to `axiom-workflow`. The rule is guidance, not an operation,
     alias or shim.
   - An explicit removed operation is reported as unsupported.
4. **Preserved boundaries.**
   - `workflow.select` metadata and behavior stay unchanged. It is used only
     to **change** the selection (preview, then separate approval), never as
     an inspection.
   - **Project active-workflow read path (PD-1 A,
     [decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101636404)).**
     A user asking the Project which workflow is active gets the answer from
     the Project's own canonical reads, with no target reference and no
     change preview:
     - `axiom project show --selector <slug-or-id>` (its existing selector, `internal/cli/cli.go:821-823`, `internal/cli/skill_inspect.go:91-92`; no new flag or alias) gains an additive, read-only
       `project.activeWorkflow` object; `axiom project list` gains the same
       object per Project entry, when applicable;
     - fields: `status` (`selected` | `none` | `unresolvable`) and, when a
       selection is recorded, `workflowId`, `revision`, `digest`, `source`;
       `name` only when resolved; `category` when `unresolvable`.

     **Name resolution (safe, no fallback).** The ref comes from the portable
     Project `workflowSelection`, through the same observation the authoring
     port and start binding already use (`cmd/lingo/workflow_authoring.go:72`
     `ObserveWorkflows`; `cmd/lingo/workflow_binding.go:40-110`):
     - `source: builtin`: the name comes from `workflowdefinition.Builtin()`
       only if its ID, revision and canonical digest equal the ref;
     - `source: project`: the name comes from the catalog document for that
       ID/revision only if the index entry matches the ref's ID, revision and
       digest and the document decodes strictly;
     - no selection: `status: none`, with no defaulting to the built-in;
     - any mismatch, missing document, undecodable document or unreadable
       source: `status: unresolvable`, with the ref fields still reported, no
       `name`, and the canonical category start binding already returns for
       that condition (for example `project_configuration_drift`,
       `recovery_required`). No new category is introduced without
       authorization.

     The read never selects, repairs, lists other definitions or falls back
     to another revision. For `list`, an unresolvable entry never hides the
     Project or fails the whole listing.

     **Delta and authorization.** This is the only output delta in #303:
     additive JSON/human output on two existing read-only commands, with no
     new command, flag, operation, authority or state write, and with every
     existing field and category unchanged. Because AC-015.3/AC-015.8 state
     that canonical CLI result semantics stay unchanged, this plan does not
     treat the field as covered by the accepted contract. **T08a is blocked
     until the maintainer explicitly authorizes the delta (R10)** as either
     compatible with AC-015.3/AC-015.8 (additive, existing semantics
     unchanged) or as a recorded contract amendment. Without that
     authorization, nothing in T08a is implemented.

     Rejected: reusing `project workflow list` (the `definition.list` action)
     under Project, which duplicates a Workflow-owned action (HD-007);
     treating the `select` preview as an inspection (rejected by the PD-1
     decision); having `axiom-project` call `axiom-workflow` reads, which
     breaks single ownership.
   - `axiom-work-item` `run`/`status`/`plan` metadata stays unchanged. It gains
     handoff rules and an "acceptance (stage-plan, R-1)" procedure section
     (PD-8). That section keeps the Plan-confirmation rules: provisional
     `approved: true`, and confirmation bound to `planDocumentDigest` plus
     Execution revision, stage, `workflowRef` and `plan.digest`.
5. **HD-006 in skill text.**
   - `axiom-workflow` sets Stage/Agent `runtimeConstraints`, `profileRef` or
     `policy-default`, `complexity` and `effort` only.
   - It never writes Project policy (G-1) or local Profiles (G-2), reports
     both as unavailable with #305/#306, and assumes no fixed tiers.
6. **Codex/Claude parity.** Both Runtimes embed the same bytes and resolve the
   same catalog. They differ only in install root and invocation syntax
   (`$axiom-workflow` / `/axiom-workflow`).
7. **Installation and retirement.**
   - Add the name to every enumerated list.
   - Append the replaced v0.15.0 set to `sharedSkillHistory`.
   - Retirement is not used: `axiom-project` is replaced as an owned revision.
8. **R-1 runner.** Add `scripts/acceptance/stage-plan-r1.py` with a schema and
   an offline unittest (PD-4 A: durable, owner `maintainer-acceptance`, manual trigger, no new required CI check). It maps scenarios A–H to existing Go tests,
   runs `go test -json -run` with bounded output and a timeout, and writes
   one sanitized Evidence JSON. Its R-2/R-3 fields are fixed to
   `deferred_to_278`.

   **Provenance binding** (amendment §Lane S: source at the exact provenance
   revision; a mismatch is `blocked`, never substituted). The runner takes
   an explicit target revision and version: the 40-hex commit and release
   version of the installed or release build, from `axiom --json version`
   provenance with `sourceState` clean, or from the release tag.

   Before any test runs, it observes the tested source:
   - `git rev-parse HEAD` must equal the target;
   - `git status --porcelain` must be empty;
   - the Go toolchain must be present.

   A missing or unverifiable target, a mismatch, a dirty tree or a missing
   toolchain makes every scenario and the overall R-1 result `blocked`, with
   a classified reason. No test result is reported as `passed` for an
   unverified revision.

   **Version binding.** A declared target version must be provable, by one
   of two routes:
   - the release tag `v<version>` resolves (`git rev-parse v<version>^{commit}`)
     to the target revision; or
   - a supplied release binary's `axiom --json version` reports that version,
     that revision and `sourceState: clean`.

   A declared version that cannot be proven, or that disagrees, is `blocked`
   (`version_unverifiable` / `version_mismatch`). An untagged implementation
   head is run with no version claim and is recorded as
   `axiomVersion: "unreleased"`, never with an invented version.

   **Canonical result digests.** Positive scenarios carry the digests of the
   canonical results they produced:
   - `workflowRef.digest`;
   - `plan.digest` and `plan.planDocumentDigest`;
   - `plan.stageInputRef.digest`;
   - `plan.graphProposalRef.digest` (graph stages only);
   - the authoring preview digests for scenario B.

   Their source is the canonical JSON the tests already decode. The two
   stage-plan tests and the authoring test emit them through bounded
   `t.Logf("r1-evidence {…}")` markers (a test-only change), which the runner
   extracts from `go test -json` output. The runner validates each value as
   64-character lowercase hex and never computes or substitutes one. A
   required digest that is missing or malformed makes the scenario `blocked`
   (`canonical_digest_missing`). Scenario E additionally requires two equal
   `plan.digest` values for identical inputs and a different value after
   drift.

   The report carries every Report H field:
   - Axiom version (proven, or `unreleased`) and target/observed revisions;
   - canonical result digests per scenario;
   - observed Runtime information as `controlled` (Lane S fake observations),
     never a real Runtime;
   - skill-set version and per-skill SHA-256 taken from
     `axiom --json runtime codex status` against an empty temporary skills
     root built from the tested source;
   - a sanitized sandbox identifier (random, no host paths);
   - per-scenario results with test names and classified refusal categories;
   - provenance and Evidence class `synthetic`;
   - known limitations (G-1–G-5);
   - the #278/AXM-12 handoff reference.

## 4. Migration

| Path | Expected behavior | Verification |
|---|---|---|
| Fresh install, Codex and Claude | Three skills + current receipt; rerun `already_configured` | T06 |
| Upgrade from v0.15.0 via the **new** release installer (`install.sh` → candidate `install-release`) | The v0.15.0 set is recognized from history; owned skills are replaced; `axiom-workflow` is created; Codex receipt published; Claude converged by `first-run`; rerun `unchanged` | T06, CI `upgrade-journeys` (N = newest tag) |
| Upgrade via an installed **v0.15.0** `axiom upgrade --archive <new>` | **Refused before effects**: the v0.15.0 parser requires two skills (`archive.go:225`) | T06 negative check (PD-3) |
| Older roots (v0.6.0 six skills, v0.1.1) | Converge to three skills; existing retirement of old names unchanged | T06 |
| Reinstall | Idempotent; a foreign or modified `axiom-workflow` gives `*_skill_conflict` with zero effects | T06 |
| Active inventory | No authoring route in installed `axiom-project`, `skill inspect` or help after any path | T07 |
| Historical Evidence | v0.15.0 set stays an owned revision; published receipts and skill testdata stay frozen | T05 |
| Failure / recovery | Existing anchored lock, pre-check-then-apply, `.axiom-upgrade-skill.*` resume, `recovery_required`; no new mechanism | existing tests |
| Rollback | Revert the implementation PR before release. After release, downgrade is not supported (the installer refuses downgrade); a v0.15.0 binary would treat `axiom-workflow` as unowned *(inferred; record in T06)* | T06 Evidence |

## 5. Security

- **Authority.** Every mutation keeps its canonical preview, expected revision,
  preview digest and `--authorize-local`. Skill text describes authority and
  never supplies it. A draft or a Runtime-written `approved: true` is never
  approval.
- **Binding.** Approval binds to Lingo's canonical digests:
  - definitions: the preview digest and the expected Project revision;
  - Plans: `planDocumentDigest` plus the reported bindings.

  Any change requires a fresh preview.
- **Filesystem.** Drafts go only to Runtime scratch. The skill never writes
  Project, index, Execution, Profile or receipt files. Skill installation
  keeps its anchored-root, ownership and conflict checks.
- **Known `--file` limitation (PD-9 A,
  [decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6102203860)).**
  `project workflow --file` uses `os.Open` with no regular-file/symlink check
  (`cmd/lingo/workflow_authoring.go:26-37`). The limitation stays known and
  **is not fixed by #303**; [#307](https://github.com/rgomids/axiom/issues/307)
  owns the hardening and is not a gate for #303. #303 does not claim it fixed
  and does not include the hardening silently. The skill mitigates exposure
  only by writing drafts as regular files in Runtime scratch.
- **Project read path (PD-1).** `activeWorkflow` is read-only, needs no
  authority input, writes no state, never selects or repairs and never falls
  back; it discloses only the ref and definition name, no host path.
- **Credentials and disclosure.** No credential, environment value or host
  path appears in skill text, Evidence or runner output. Readiness `auth`
  runs only the existing read-only vendor status observation (PD-2 A).
- **Fail-closed.** Wrong-skill intents are redirected. #276/#277 operations
  and G-1/G-2 are reported as unavailable. There is no Runtime, Profile or
  model fallback, and a synthetic result never fills R-2/R-3.
- **External effects.** None. The runner runs local `go test` only, with fakes
  and controlled observations.
- **Negative tests.** T07 (stale routes, unsupported operations), T08a
  (absent/unresolvable selection, no fallback, no write), T09
  (stale/unauthorized apply), T10 scenario F, T11 (HD-006/credential).

## 6. Testing (R-1 deterministic/synthetic)

| Area | Tests (task) |
|---|---|
| Skill registration | manifest-driven `skill_inspect_test.go:41`; `cmd/lingo/skill_inspect_test.go`; runtime tests (T02, T04) |
| Operation metadata | `TestSkillDiscoveryExactCommands`, `TestSkillDiscoveryMatchesParserAndAcceptedForms` (T02) |
| Routing | `TestCanonicalRoutingContract`, `TestCanonicalSkillRoutingTableConvergesWithInspection`, `TestExplicitOperationRoutingIsDeterministic`, `TestSemanticResolutionIsBoundedAndNeverMutatesOnAmbiguity` (T02, T07) |
| Draft handling, validation, preview/apply | new draft journey test (T09); existing #273 authoring tests |
| Project active workflow | new `project show`/`list` `activeWorkflow` tests: selected built-in, selected project revision, none, unresolvable, list isolation, no write (T08a); `select` behavior unchanged (T08) |
| Readiness | three-fact readiness mapping, auth read-only and non-blocking (T02, T03) |
| Work Item stage planning | `TestWorkflowStagePlanPublicJourneySingleAndMixedRuntimeGraph`, `TestWorkflowStagePlanExecutableFailsClosedAndNeverMutates` (T08, T10) |
| Negatives | `TestUnsupportedDomainOperationsFailSafely`, `TestReadPlanDocumentRefusesNonRegularFiles`, `TestCompileRejectsInvalidTopologyControlsAndReferences`, `TestCompileSequentialConcurrencyAndUnsafeOverlap`, `TestPlanStageFailsClosedWithSpecificationCategories` (T07, T10, T11) |
| Skill migration and parity | `domain_skills_upgrade_test.go` (codexruntime and install), `upgrade_test.go`, `skill_receipt_test.go`, `first_run_blackbox_test.go`, `scripts/test-upgrade-journeys.sh` (T06) |
| Historical Evidence | `TestEveryPublishedSharedRevisionStaysOwned`, `TestRuntimeOnlyHistoryIsFrozen`, `TestEveryPublishedReceiptIsRecognizedAsAxiomOwned`, `TestEveryPublishedSkillDigestIsKnown`, `TestEveryPublishedClaudeInstallationConverges` (T05) |
| Binding preserved | `TestConfiguredExecutableRevisionIsolationAndAuthority` (T08) |

**Deferred, never reported as passed:**
- **R-2:** native Codex and Claude natural-language sessions;
- **R-3:** human end-to-end MVP acceptance.

Both belong to [#278](https://github.com/rgomids/axiom/issues/278) / AXM-12 and
need G-2 (#306), consented G-4 linkage, G-3 observation for explicit effort and
#276/#277. Evidence required there: the observed Runtime version and mode,
skill-set digest, canonical command digests, a sanitized interaction summary
and the maintainer's accept/reject decision.

## 7. Delivery

- **Sequence:** T01 → T02 → T03 → T04 → T05 → T06 → T07 → T13. Off that path:
  - T08 follows T02, and T09 follows T03 and T08;
  - T08a follows T02 and the R10 authorization;
  - T10 runs from T01;
  - T11 starts from T01 but closes after T03;
  - T12 starts after T03 but closes after T10.
- **Technical dependencies:**
  - T02 must be atomic, because the routing tests enforce a three-way
    equality;
  - T05 pins digests only after the final SKILL bytes;
  - any later text change requires re-pinning.
- **Risks:**

  | ID | Risk | Mitigation | Blocks |
  |---|---|---|---|
  | R1 | v0.15.0 `axiom upgrade` refuses a three-skill archive | **PD-3 Accepted — Option A:** new release installer is the supported path; T06 must verify the legacy binary refuses without effects and document this in release notes | No — decision accepted; T06 verification remains required |
  | R2 | v0.15.0 set missing from history makes v0.15.0 installs look foreign | T05 before release | No |
  | R3 | Drift between table, inspect and matrix | Atomic T02; existing tests | No |
  | R4 | Wrong inspect flags for runtime actions | `skillFlagSet` cases plus parser-parity test | No |
  | R5 | `--file` accepts non-regular files/symlinks | **PD-9 A:** known limitation owned by [#307](https://github.com/rgomids/axiom/issues/307); not fixed or claimed fixed by #303; drafts are regular files in Runtime scratch | No — #307 is not a #303 gate |
  | R6 | Runner map goes stale | Missing test gives `blocked`; unittest pins the map | No |
  | R9 | Runner Evidence attributed to an untested revision or version, or missing canonical digests | Provenance gate (target = `HEAD`, clean tree, toolchain, version proven by tag or release binary, else `unreleased`); digests taken only from canonical results via test markers; otherwise `blocked`; unittest covers divergence | No |
  | R7 | PR #304 not on `main` | T01 gate | Yes |
  | R8 | Claude converges only via `first-run` after `axiom upgrade` | Assert in T06; document in T12 | No |
  | R10 | The PD-1 `activeWorkflow` field changes `project show`/`list` output, while AC-015.3/AC-015.8 keep canonical CLI results unchanged | Additive, read-only, existing fields/categories unchanged; explicit maintainer authorization of the delta (compatible with AC-015.3/8, or a recorded amendment) before T08a | **Yes:** T08a, and therefore T13 closure, R-1 completion and `Completes-Issues: #303`; other tasks proceed |

- **Documentation:**
  - `docs/commands.md` skill sections (`:125-128`, `:521-529`, `:1256`), the Project workflow authoring section and the `project show`/`list` output (T08a field);
  - `docs/installation.md:281`, `README.md`, `docs/README.pt-BR.md`, `site/index.html`;
  - the issue-230 catalog block;
  - `internal/codexruntime/testdata/published-skills/PROVENANCE.md` wording;
  - `docs/specifications/README.md`;
  - new `issue-303-implementation.md`.

  CHANGELOG is left to release automation.
- **Review requirements:** bounded independent review of contract
  compatibility, domain duplication, authority, migration and Evidence, with
  no unresolved Blocker/Major; then human review.
- **Required Evidence:** `issue-303-implementation.md` with the base SHA, test
  output summaries, the runner Evidence JSON, CI on head, R-1 results and
  R-2/R-3 `deferred_to_278`.
- **Rollback:** see §4. Revert the PR before release. No persisted-state
  migration is introduced.

## 8. Planning decisions (individually accepted)

All nine were decided by the maintainer on #303. They fall within the accepted
contract and none reopens HD-005–HD-008. **Individual PD acceptance does not
accept this Plan/Tasks as a whole**, which stays Proposed, and authorizes no
implementation or merge.

| ID | Decision | Accepted outcome | Applied in |
|---|---|---|---|
| PD-1 | Project active-workflow inspection — **Accepted, refined** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101636404)) | The Project reports its active workflow (ID, name, revision) through its own `show`/`list`, with no target reference and no change preview; `select` only changes the selection; no `definition.list` in Project; absent/unresolvable selection reported without fallback. Needs the R10 delta authorization before T08a | §2, §3.1, §3.4, §5, R10, T02, T03, T08, T08a, T12 |
| PD-2 | `configuration.readiness` scope — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101705557)) | `runtime profile validate/preview` plus `runtime codex` / `runtime claude` `status` and `auth`, reported as Configuration Valid / Runtime Ready / Authentication Ready; auth read-only, no login, never blocks authoring; point-in-time | §3.1, §5, T02, T03 |
| PD-3 | Upgrade from v0.15.0 — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6100657230)) | New release installer is the supported path; v0.15.0 `axiom upgrade` must refuse without effects (verified and documented in T06). Versioning the archive skill manifest was **not selected** | §4, R1, T06 |
| PD-4 | R-1 runner form — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101761143)) | python3 runner + JSON schema + offline unittest; durable; owner `maintainer-acceptance`; manual trigger; no new required CI check | §3.8, T10 |
| PD-5 | Lane L in #303 — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101798981)) | Lane L is not built; T02–T13 all remain required | §1, T13 |
| PD-6 | Skill-set constants — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101819334)) | Keep `SkillSetVersion`/`BinaryCompatibility` "2"; append-only history. A real incompatibility found in T05/T06 stops the work for a new decision | T05, T06 |
| PD-7 | v0.15.0 receipt fixture — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6101853398)) | Mandatory fixture derived from official v0.15.0 artifacts, or reproduced through that version's official mechanism, with documented provenance (tag, asset, digest); never fabricated; unprovable → blocked and escalated | Common rules, T05 |
| PD-8 | Work Item acceptance surface — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6102010231)) | SKILL.md procedure section; no new inspect operation | §3.4, T03 |
| PD-9 | `--file` hardening — **Accepted, Option A** ([decision](https://github.com/rgomids/axiom/issues/303#issuecomment-6102203860)) | Separate Issue [#307](https://github.com/rgomids/axiom/issues/307); known limitation, not fixed or claimed fixed by #303; #307 is not a #303 gate | §1, §2, §5, R5, T09, T11 |

---

# Part II — Tasks

Common rules for every task:
- No CLI, application or schema change unless the task names it; the only
  named one is T08a, gated by R10.
- No vendor inference, Provider effect or network access beyond the Go module
  cache, except T05's read-only retrieval of official v0.15.0 release
  artifacts (PD-7 A).
- Existing tests change only where they enumerate skill names or catalog rows, plus **one bounded exception (R-1 instrumentation, T10):**
  - purely additive `t.Logf("r1-evidence %s", …)` lines in `cmd/lingo/workflow_stage_plan_test.go` and `cmd/lingo/workflow_authoring_test.go`;
  - they emit values already present in the canonical JSON those tests decode;
  - no existing line, assertion, fixture, control flow or production code may change.
- Each task ends with `go build ./...`, `go vet ./...` and its focused tests.

### T01 — Contract baseline gate (human)
- **Objective:** implementation starts from a base containing the accepted contract.
- **Scope:** PR #304 merged, or explicit authorization to stack on its head; then the implementation branch is created.
- **Files:** none.
- **Dependencies:** maintainer acceptance of this document as a whole (PD-1 to PD-9 are already individually accepted, §8). T08a additionally needs the R10 authorization.
- **Requirements:** no agent merge.
- **Acceptance criteria:** the base contains the accepted amendment and ADR bytes.
- **Verification:** `git rev-parse <base>:<path>` matches the accepted content.
- **Evidence:** base SHA in `issue-303-implementation.md`.
- **Definition of Done:** branch exists on an approved base.
- **Parallel:** no (root).

### T02 — Catalog split and atomic routing contract
- **Objective:** `axiom-workflow` owns the definition operations; `axiom-project` keeps only `workflow.select`.
- **Scope:** Go catalog, minimal SKILL.md routing tables, matrix catalog block, catalog test lists.
- **Files:**
  - `internal/cli/workflow_authoring.go`, `internal/cli/skill_inspect.go`;
  - `internal/codexruntime/skills/*/SKILL.md` (tables and "Supported domain operations" sentence only);
  - `docs/specifications/004-mvp-v1-baseline/issue-230-resource-lifecycle-matrix.md` (catalog block);
  - `internal/cli/{skill_inspect,skill_catalog_contract,domain_skill_routing}_test.go`, `cmd/lingo/skill_inspect_test.go`.
- **Dependencies:** T01.
- **Requirements:**
  - `definition.*` reuse the `project_workflow_*` actions and authority strings exactly;
  - add `configuration.readiness` (PD-2 A) and its `skillFlagSet` cases;
  - no action in two skills; no `definition.*` read in `axiom-project`;
  - `axiom-project` `show`/`list`/`workflow.select` and all `axiom-work-item` inspect metadata byte-identical to before.
- **Acceptance criteria:**
  - `axiom-workflow` inspect lists exactly the planned operations, with arguments equal to the parsers;
  - `axiom-project` inspect lacks the seven removed operations;
  - the three-way equality holds.
- **Verification:** `go test ./internal/cli ./cmd/lingo -run 'Skill|Routing|Catalog'`.
- **Evidence:** test output; inspect JSON diff.
- **Definition of Done:** green, in one commit.
- **Parallel:** no (critical path).

### T03 — Skill instructional content
- **Objective:** natural-language behavior per HD-005 to HD-008.
- **Scope:** prose of all three SKILL.md files, plus content assertions.
- **Files:** `internal/codexruntime/skills/{axiom-workflow,axiom-project,axiom-work-item}/SKILL.md`; content tests in `internal/codexruntime`/`internal/cli`.
- **Dependencies:** T02.
- **Requirements:**
  - §3.2–3.6 rules;
  - full stage/agent field coverage per `workflow.schema.json`;
  - G-1/G-2 reported as unavailable with #305/#306;
  - #276/#277 reported as unavailable;
  - redirection rules;
  - Work Item acceptance procedure (PD-8 A) with Plan-confirmation rules;
  - `axiom-project`: "which workflow is active" answered from `project show`/`list` `activeWorkflow` (PD-1); a change only via `workflow.select` preview plus separate approval; `none`/`unresolvable` reported as such, with no guessed name or fallback;
  - `axiom-workflow` readiness reported as the three PD-2 facts, point-in-time, with auth never blocking authoring;
  - no credentials, paths or tiers.
- **Acceptance criteria:**
  - each file has the inspect pointer, the exact operations sentence and the canonical result contract;
  - no skill claims a foreign operation.
- **Verification:** content tests; reviewer checklist.
- **Evidence:** test output; checklist in the PR.
- **Definition of Done:** green, reviewed text.
- **Parallel:** with T10 and T11.

### T04 — Inventory names and installers
- **Objective:** every enumerated skill list includes `axiom-workflow`.
- **Scope:** name lists, help text, installer counts.
- **Files:**
  - `internal/codexruntime/runtime.go:21`, `internal/install/archive.go:35`;
  - `internal/cli/help.go:285,297,314`;
  - `scripts/install-release.sh:199-206`, `scripts/test-install-posix-facade.sh:107`, `scripts/dogfood-poc.sh:136`;
  - name lists in `internal/runtimebootstrap/bootstrap_test.go`, `internal/codexruntime/{runtime,retirement,execution_target_skills}_test.go`, `internal/install/*_test.go`, `cmd/lingo/{first_run_blackbox,s9_dogfood_blackbox,skill_protocol}_test.go`, `internal/cli/help_test.go`.
- **Dependencies:** T03.
- **Requirements:**
  - add the name in sorted order;
  - `install-release.sh` expects 6 manifest lines and 3 `skill.*` entries;
  - `retiredSkillNames` stays unchanged.
- **Acceptance criteria:**
  - an archive from `scripts/build-release-archives.sh` installs through `install-release.sh`;
  - help lists three stable skills.
- **Verification:** `scripts/test-install-posix-facade.sh`, `scripts/test-release-archives.sh`, focused Go tests.
- **Evidence:** script and test output.
- **Definition of Done:** green.
- **Parallel:** no.

### T05 — History pinning and receipt provenance
- **Objective:** v0.15.0 installs stay owned and the new set is pinned.
- **Scope:** shared history, published-revision list, mandatory v0.15.0 receipt fixture (PD-7 A).
- **Files:**
  - `internal/codexruntime/manifest.go` (append v0.15.0: `axiom-project d80cd1d1…`, `axiom-work-item e47c9256…`);
  - `internal/codexruntime/shared_history_test.go:22`;
  - `testdata/published-receipts/v0.15.0/` (mandatory, PD-7 A);
  - `testdata/published-skills/PROVENANCE.md` and the receipt provenance record.
- **Dependencies:** T04 (final bytes).
- **Requirements:**
  - append only; Codex-only history and existing fixtures stay frozen;
  - PD-6 A: constants stay "2"; a real incompatibility stops the task for a new decision;
  - PD-7 A: the receipt fixture comes from the official v0.15.0 release artifacts (read-only), or is reproduced through v0.15.0's official install mechanism from those artifacts. Provenance records the tag `v0.15.0`, its commit, the asset name and the asset SHA-256. Bytes are never hand-written or fabricated; if provenance cannot be proven, the task is `blocked` and escalated.
- **Acceptance criteria:**
  - the five history/receipt tests in §6 pass, including the v0.15.0 receipt;
  - the recorded asset digest matches the official release asset.
- **Verification:** `go test ./internal/codexruntime`.
- **Evidence:** test output; recorded digests; fixture provenance (tag, commit, asset, digest).
- **Definition of Done:** green, with proven fixture provenance.
- **Parallel:** no. Re-pin if T03 or T04 bytes change.

### T06 — Install, upgrade and reinstall convergence
- **Objective:** prove every §4 path for Codex and Claude.
- **Scope:** tests only, plus a comment fix in the journey script.
- **Files:** `internal/codexruntime/{domain_skills_upgrade,runtime}_test.go`, `internal/install/{domain_skills_upgrade,upgrade,skill_receipt}_test.go`, `cmd/lingo/first_run_blackbox_test.go`, `scripts/test-upgrade-journeys.sh:162-163` (comment).
- **Dependencies:** T05.
- **Requirements:** cover:
  - fresh install;
  - a v0.15.0 root (seeded from history bytes) upgraded to three skills;
  - a v0.6.0 six-skill root;
  - rerun no-op;
  - a foreign or modified `axiom-workflow` giving a conflict with zero effects;
  - Claude converging via `first-run` after `axiom upgrade`;
  - PD-3 old-parser refusal with zero effects (a synthetic v0.15.0-format parser fixture, or a journey assertion with the downloaded N binary);
  - rollback behavior recorded.
- **Acceptance criteria:** installed bytes are byte-identical to the embedded bytes in both Runtimes; receipts are current; refusal is fail-closed.
- **Verification:** focused Go tests; CI `upgrade-journeys` on linux and macOS.
- **Evidence:** test output; CI run.
- **Definition of Done:** green locally and in CI.
- **Parallel:** no.

### T07 — Negative discovery and stale-route removal
- **Objective:** no obsolete Project authoring route remains active anywhere.
- **Scope:** test additions.
- **Files:** `internal/cli/{skill_inspect,help,domain_skill_routing}_test.go`, `cmd/lingo/skill_inspect_test.go`.
- **Dependencies:** T06.
- **Requirements:** after fresh install, upgrade and reinstall:
  - `axiom-project` inspect, help and installed bytes contain no `workflow.{list,show,create,edit,validate,remove,recover}`;
  - an explicit removed operation is reported as unsupported, with no alias;
  - `axiom-workflow` exposes no select, execution or plan operation.
- **Acceptance criteria:** the assertions pass for both Runtime roots.
- **Verification:** focused tests.
- **Evidence:** test output.
- **Definition of Done:** green.
- **Parallel:** with T08.

### T08 — Preservation of selection, execution and CLI
- **Objective:** HD-005 boundaries hold and the CLI is unchanged apart from the T08a field.
- **Scope:** one new Project-skill test, plus a check that existing tests are untouched.
- **Files:** a new test in `internal/cli` or `cmd/lingo`; existing `internal/cli/workflow_authoring_test.go`, `cmd/lingo/{workflow_authoring,workflow_binding,workflow_stage_plan}_test.go`.
- **Dependencies:** T02.
- **Requirements:**
  - `workflow.select` metadata and behavior unchanged; it is not used as an inspection;
  - Work Item `run`/`status`/`plan` metadata unchanged;
  - CLI help tree and flags unmodified; results unmodified except the T08a `activeWorkflow` field.
- **Acceptance criteria:**
  - the #273/#274/#275 tests pass with their assertions and fixtures unmodified;
  - the only permitted diff in them is the T10 R-1 instrumentation exception.
- **Verification:**
  - `git diff <base> -- internal/cli/workflow_authoring_test.go cmd/lingo/workflow_binding_test.go` is empty;
  - for `cmd/lingo/workflow_authoring_test.go` and `cmd/lingo/workflow_stage_plan_test.go`, `git diff --unified=0 <base>` contains no removed lines, and every added line matches `^\+\s*t\.Logf\("r1-evidence `;
  - test output.
- **Evidence:** diff and output.
- **Definition of Done:** green.
- **Parallel:** with T07.

### T08a — Project active-workflow read path (PD-1)
- **Objective:** a Project-only query identifies the active workflow by ID, name and revision; changing it still needs a separate preview and authorization.
- **Gate:** **blocked until the R10 authorization** of the additive output delta. Without it, nothing here is implemented, T08a stays `blocked`, and T13 cannot close (see T13).
- **Scope:** application read model and CLI view for `project show`/`list`; tests; no new command, flag, operation, authority, category or state write.
- **Files:** `cmd/lingo/project_lifecycle.go` (`listProjects`, `showProject`); `internal/projectapp/list.go` (`ProjectSummary` gains the resolved selection) plus a small resolver beside `internal/projectapp/workflow_authoring.go`; `internal/cli/cli.go` (`ProjectView`/`ProjectListView` gain `activeWorkflow,omitempty`); human renderer if it lists Project fields; new tests in `cmd/lingo` and `internal/projectapp`.
- **Dependencies:** T02; R10 authorization.
- **Requirements:**
  - the ref comes from the portable Project `workflowSelection`, via the existing observation used by authoring and start binding;
  - name resolution exactly as §3.4: built-in only on an exact ID/revision/digest match; project source only on a matching index entry and strictly decoded document;
  - `status: none` without a selection; `status: unresolvable` with the ref, no name and the existing binding category on any mismatch or unreadable source;
  - no fallback, defaulting, repair, selection or listing of other definitions;
  - `list`: per-Project resolution; one unresolvable entry never hides a Project or fails the listing;
  - existing fields, categories, messages and `next` text unchanged.
- **Acceptance criteria:**
  - tests cover: built-in selected; project revision selected; no selection; digest mismatch; missing document; unreadable portable source; list with mixed entries;
  - a tree snapshot proves both commands write nothing;
  - existing project show/list tests pass unmodified. If one compares the full payload exactly, it is named in the PR and changed only by adding the new field, under the R10 authorization;
  - after `workflow.select` apply, `project show` reports the new selection; before apply, it reports the old one.
- **Verification:** `go test ./cmd/lingo ./internal/projectapp ./internal/cli -run 'Project|ActiveWorkflow'`; diff review of existing tests.
- **Evidence:** test output; JSON sample of each `status` (sanitized).
- **Definition of Done:** green, with R10 authorization recorded.
- **Parallel:** with T07 and T08.

### T09 — Draft and authority integration
- **Objective:** the draft journey maps onto the canonical preview/apply contract with no new behavior.
- **Scope:** a new `cmd/lingo` test.
- **Files:** a new `cmd/lingo/*_test.go`.
- **Dependencies:** T03, T08.
- **Requirements:**
  - draft in a temp scratch directory outside the Project;
  - `validate` → `create` preview → apply with the exact authority → `show` read-back;
  - a changed draft after preview is refused with the canonical category;
  - apply without authority has no effect;
  - creation does not change the selection;
  - `select` preview then apply activates it;
  - tree snapshot around each refused step;
  - drafts written as regular files only; `--file` hardening is **not** added (#307, PD-9 A), and no test claims non-regular-file refusal.
- **Acceptance criteria:** the categories returned today; no state change on refusals.
- **Verification:** focused test.
- **Evidence:** test output.
- **Definition of Done:** green.
- **Parallel:** with T07, T10 and T11.

### T10 — Work Item-owned #275 R-1 runner
- **Objective:** deterministic synthetic Evidence for scenarios A–H (Lane S), bound to the exact tested revision.
- **Scope:** runner, schema, offline unittest, registry entry, test-only digest markers.
- **Files:** `scripts/acceptance/stage-plan-r1.py`, `scripts/acceptance/stage-plan-r1.schema.json`, `scripts/acceptance/test_stage_plan_r1.py`, `scripts/automation-registry.json`; test-only marker lines in `cmd/lingo/workflow_stage_plan_test.go` and the #273 authoring test (`cmd/lingo/workflow_authoring_test.go`); wiring into `scripts/validate-repository.sh` only if it matches how existing acceptance unittests are run.
- **Dependencies:** T01.
- **Requirements:**
  1. **Inputs:** `--target-revision <40-hex>`, optional `--target-version <version>`, optional `--release-binary <path>`, `--source <dir>`, `--output <new-path>`.
  2. **Provenance gate, before any test runs:**
     - `git -C <source> rev-parse HEAD` must equal the target revision;
     - `git status --porcelain` must be empty;
     - `go version` must succeed.

     - when `--target-version` is given, it must be proven: either `git rev-parse v<version>^{commit}` equals the target revision, or `--release-binary` reports that version, that revision and `sourceState: clean` in `axiom --json version`. Without `--target-version`, the version is recorded as `unreleased`.

     Otherwise every scenario and `r1` are `blocked`, with reason `revision_mismatch`, `dirty_source`, `target_unverifiable`, `toolchain_missing`, `version_unverifiable` or `version_mismatch`. No test results are recorded as `passed`.
  3. **Scenario map:**
     - A, C, D, E: `TestWorkflowStagePlanPublicJourneySingleAndMixedRuntimeGraph`;
     - B: #273 authoring tests;
     - F: the journey negatives, `TestWorkflowStagePlanExecutableFailsClosedAndNeverMutates`, `TestReadPlanDocumentRefusesNonRegularFiles`, `TestCompileRejectsInvalidTopologyControlsAndReferences`, `TestCompileSequentialConcurrencyAndUnsafeOverlap`, `TestPlanStageFailsClosedWithSpecificationCategories`;
     - G: the `snapshotTrees` assertions.

     The runner uses `go test -json -run`, with bounded output and a timeout.
  3a. **Canonical digest source (test-only change):** add bounded `t.Logf("r1-evidence %s", …)` markers to `cmd/lingo/workflow_stage_plan_test.go` (both tests) and the #273 authoring test. They emit, from the canonical JSON already decoded there, the scenario ID together with the digests below:
     - `workflowRef.digest`, `plan.digest`, `plan.planDocumentDigest`, `plan.stageInputRef.digest`;
     - `plan.graphProposalRef.digest` (graph only);
     - authoring `previewDigest`;
     - refusal categories.

     Assertions and production code are unchanged. The runner parses only these markers, checks that each digest is 64-character lowercase hex and never computes a digest itself. This is the general-rules instrumentation exception, enforced by T08's diff verification: additive marker lines only, with no other change to those files.
  4. **Report H fields** (closed schema, `additionalProperties: false`):
     - `axiomVersion` (proven, or `unreleased`), `versionProof` (`tag` | `release-binary` | `none`), `targetRevision`, `observedRevision`, `sourceClean`, `goVersion`;
     - `runtimeObservation: "controlled"`;
     - `skillSet` (version and per-skill SHA-256 from `axiom --json runtime codex status` against an empty temporary root);
     - `sandboxId` (random, no host path);
     - `lane: "S"`, `class: "synthetic"`;
     - per-scenario `{result: passed|failed|blocked, tests[], refusalCategories[], canonicalDigests{}}`. Positive scenarios A–E require their digests; scenario E requires repeated equal `plan.digest` values and a changed value after drift. A missing or malformed digest gives `blocked` (`canonical_digest_missing`);
     - `r1`;
     - `r2`/`r3` fixed to `deferred_to_278`;
     - `limitations` (G-1–G-5) and `handoff` (#278/AXM-12).
  4a. **Operation (PD-4 A):** registered as durable with owner `maintainer-acceptance`; run by manual trigger only; no new required CI check or workflow job is added.
  5. **Classification:** a missing or renamed test gives `blocked`; a timeout or exceeded bound gives `failed`. The runner writes only to a new output path; no network, vendor process, credential, environment value or host path appears in the output.
- **Acceptance criteria:**
  - the schema rejects `passed`/`accepted` for R-2/R-3, missing H fields and unknown fields;
  - the unittest proves that each of the following yields `blocked` with no `passed` scenario: revision mismatch, dirty source, unverifiable target, missing toolchain, unproven or divergent target version (tag absent or pointing elsewhere; release binary reporting another version, revision or a non-clean state), and a missing or malformed canonical digest;
  - the unittest also covers mapping, missing test, timeout and output bound, and confirms no host path or environment value in the output;
  - `check-automation-registry.py` passes.
- **Verification:** `python3 scripts/acceptance/test_stage_plan_r1.py`; `scripts/check-automation-registry.py`; one runner run on a clean checkout at the implementation head, with that head as the target revision.
- **Evidence:** the Evidence JSON, referenced from `issue-303-implementation.md`.
- **Definition of Done:** green, and Evidence produced for the exact head.
- **Parallel:** yes, from T01.

### T11 — Security, HD-006 and negative coverage
- **Objective:** close AC-015.9 and confirm negative coverage.
- **Scope:** analysis, plus tests only where a gap exists.
- **Files:** existing `internal/runtimeprofile`, `internal/runtimeapplication`, `internal/workflowcompiler` tests; new tests only for gaps.
- **Dependencies:** start after T01 (test coverage); **completion requires T03** (SKILL text review).
- **Requirements:** cite or add tests for:
  - multiple operator-named Profiles per Runtime resolving deterministically, with no fixed tier set;
  - Project ∩ local intersection with no fallback when empty, ambiguous or disabled;
  - no credential reference in serialized output;
  - unsupported effort failing closed.

  Also confirm the T03 text has no credential, path or tier assumption, and record the `--file` limitation as known, owned by #307 and not fixed by #303 (PD-9 A).
- **Acceptance criteria:** every item has a cited passing test.
- **Verification:** focused tests.
- **Evidence:** test list in `issue-303-implementation.md`.
- **Definition of Done:** checklist complete, including the T03 text review.
- **Parallel:** test-coverage part from T01; closes after T03.

### T12 — Documentation reconciliation
- **Objective:** user and contract docs match the delivered surfaces.
- **Scope:** the §7 documentation list, plus a #278 handoff checklist kept in the repository.
- **Files:** the files in §7; new `docs/specifications/007-configurable-workflows/issue-303-implementation.md`.
- **Dependencies:** start after T03; **completion requires T10** (Evidence links).
- **Requirements:**
  - describe the three-skill ownership and the removed Project routes;
  - document the `activeWorkflow` field only if T08a was authorized and delivered;
  - record the `--file` limitation as known and owned by #307;
  - make no R-2/R-3 claim;
  - leave the accepted amendment/ADR text unchanged;
  - make no Issue edits without authority.
- **Acceptance criteria:** `validate-repository.sh`, `check-adr-governance.py`, `scripts/test-site-language.py` and `scripts/validate-landing-page.sh` pass.
- **Verification:** validator output.
- **Evidence:** output.
- **Definition of Done:** green, with T10 Evidence linked.
- **Parallel:** with T06–T11 after T03; closes after T10.

### T13 — Final validation and review readiness
- **Objective:** an R-1-complete implementation PR ready for human review.
- **Scope:** full validation, independent review, PR.
- **Files:** `issue-303-implementation.md` final section.
- **Dependencies:** T02–T12, **including T08a delivered** under the R10 authorization. A blocked T08a is not a satisfied dependency.
- **Requirements:**
  - `go test ./...`, `go vet ./...`, `go build ./...`, `go mod verify`;
  - `scripts/check-go-quality.sh all`;
  - `scripts/validate-repository.sh .`;
  - the T10 runner;
  - a bounded independent review;
  - no Lane L artifact (PD-5 A); every task T02–T13 is still required;
  - Draft PR with `Related-Issues: #303` and `Relates to AXM-7`; `Completes-Issues: #303` only when T13 closes (below), otherwise `Completes-Issues: none`;
  - no merge, release, closure, AXM-7 acceptance or AXM-8 unblock.
- **Acceptance criteria (closure):**
  - R10 authorized and T08a delivered with its tests green;
  - every check mapped to an AC-015 row green; NOT_RUN is allowed only for a check no AC-015 row depends on, with a reason;
  - no unresolved Blocker/Major;
  - R-2/R-3 `deferred_to_278`.
- **Blocked state:** if any closure criterion is unmet (for example R10 still pending), T13 is `blocked`, not done. A Draft PR may still record the work, but it reports `blocked` with the unmet items, uses `Completes-Issues: none`, and claims neither R-1 completion nor #303 technical completion, even when every other check is green.
- **Verification:** CI on head; review record; R10 authorization record.
- **Evidence:** final implementation record, stating `complete` or `blocked` with the unmet items.
- **Definition of Done:** every closure criterion met and the Draft PR pushed under separate authorization. A Draft PR in the blocked state does not satisfy this DoD.
- **Parallel:** no.

## Task dependency graph

```text
T01 -> T02 -> T03 -> T04 -> T05 -> T06 -> T07 -> T13   (critical path)
T02 -> T08 -> T13
T02 + R10 authorization -> T08a -> T13   (T13 cannot close while T08a is blocked)
T03 + T08 -> T09 -> T13
T01 -> T10 -> T13
T01 -> T11(start); T03 -> T11(close) -> T13
T03 -> T12(start); T10 -> T12(close) -> T13
```

**Critical path:** T01 → T02 → T03 → T04 → T05 → T06 → T07 → T13.

**Completion dependencies (start ≠ close):**
- T11 starts after T01 but closes only after T03;
- T12 starts after T03 but closes only after T10.

**Parallel-safe:**
- T10 from T01;
- T11's test-coverage part from T01;
- T08 after T02;
- T08a after T02 and R10;
- T09 after T03 and T08;
- T12 drafting after T03.

T02–T05 run sequentially: they change the same embedded bytes and digests.

## AC-015 traceability

| AC-015 | Plan § | Task(s) | Test(s) | Expected Evidence |
|---|---|---|---|---|
| 1 Accepted contract, annotated | Status | T01, T12 | ADR governance; Spec index | Contract bytes on base; annotations present |
| 2 Embedded/installed/discoverable; v0.15.0 upgrade; reinstall; history; no duplicate active routes | 3.7, 4 | T04, T05, T06, T07 | runtime/install/upgrade tests; upgrade journeys; inspect tests | Converged three-skill inventories; owned v0.15.0 set |
| 3 Configuration routes = canonical commands; Project routes absent; select/run/status/plan kept; CLI unchanged | 3.1, 3.3, 3.4 | T02, T03, T07, T08, T08a | routing/catalog/inspect/help tests; `activeWorkflow` tests | Parity; negative discovery; Project-only active-workflow query; R10 authorization of the additive field |
| 4 NL configuration via drafts; Project selects; start resolves and binds | 3.2, 3.4 | T03, T08, T08a, T09 | content tests; draft journey; `TestConfiguredExecutableRevisionIsolationAndAuthority` | Journey output |
| 5 Work Item-owned #275 R-1 runner; classified scenarios; R-1/R-2/R-3 separated | 3.4, 3.8 | T10 | runner + unittest (revision/version mismatch, dirty source, missing digest → `blocked`) | Evidence JSON A–H with the full Report H fields, canonical digests per scenario, bound to the tested revision and a proven or `unreleased` version |
| 6 No real sessions/inference/dispatch/Provider/human E2E; #278 handoff | 1, 6 | T10, T12 | schema rule; review | `deferred_to_278`; handoff checklist |
| 7 R-1 + review + authorization; G-2/G-4 not waived | 7 | T08a, T13 | full validation; review; R10 record | Review record; T13 `complete` only with R10 authorized and T08a delivered, otherwise `blocked` with blockers listed |
| 8 CLI/result/digest/preview/security unchanged | 3, 5 | T08, T08a, T13 | unmodified existing tests | Diff review; the only output delta is the additive `activeWorkflow` field, delivered only under R10 |
| 9 HD-006 boundaries; #305/#306 linked; no tiers/conflation/unproven capability/credential mutation | 3.5, 5 | T03, T11 | profile/resolver tests; content tests | Cited tests |
| 10 Independent review, no unresolved Blocker/Major | 7 | T13 | independent review | Review record |

**Unresolved technical dependencies:**
- *blocking implementation:* R7/T01 (accepted contract on approved implementation base; acceptance of this Plan/Tasks as a whole still needed);
- *blocking T08a, and therefore T13 closure and R-1 completion:* R10 (explicit authorization of the PD-1 additive `activeWorkflow` output delta);
- *accepted decisions:* PD-1 to PD-9, individually accepted (§8); their Evidence (for example T05 provenance, T06 refusal) is still required;
- *known, outside #303:* `--file` hardening #307 (PD-9 A, not a gate);
- *outside #303:* G-1 #305, G-2 #306, G-3 #272, G-4 consent, G-5 #276.

R-2 and R-3 rows always read `deferred_to_278`. A synthetic PASS is never
real Runtime proof.

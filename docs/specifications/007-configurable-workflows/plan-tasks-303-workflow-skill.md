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
  keeps Project lifecycle, policy and `workflow.select`;
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
  registry, approval mechanism, scheduler or fixed Profile tier;
- real inference, native sessions, Provider effects, release, and the
  optional Lane L sandbox runner (PD-5).

## 2. Current architecture (observed)

| Concern | Module / function | Current behavior | #303 impact |
|---|---|---|---|
| Canonical skill text | `internal/codexruntime/skills/{axiom-project,axiom-work-item}/SKILL.md`; embedded by `//go:embed skills/*/SKILL.md` (`internal/codexruntime/runtime.go:15`) | Two domain skills | Add `axiom-workflow/SKILL.md`; edit both |
| Operation catalog | `internal/cli/skill_inspect.go:200-208` `skillOperationSpecs`; `inspectSkill` (`:267`) | Switch on two names; Project case appends `workflowAuthoringSkillSpecs()` | New case; split specs |
| Authoring specs | `internal/cli/workflow_authoring.go:107-122` | 8 `workflow.*` specs over `project_workflow_*` actions | 7 move to `definition.*`; `select` stays |
| Inspect flag registry | `internal/cli/skill_inspect.go` `skillFlagSet`; default `projectFlagSet` (`internal/cli/cli.go:808-812`) | Runtime actions not mapped (default would add `--slug`) | Map `runtime_profile_validate/preview` (`runtimePreviewFlagSet`, `internal/cli/runtime_preview.go:94`) and `runtime_{codex,claude}_{status,auth}` |
| Project workflow commands | `internal/cli/commands.go:38`, `workflow_authoring.go:22-48,124-130`; `cmd/lingo/workflow_authoring.go:21-40`; `internal/projectapp/workflow_authoring.go:58` `ApplyWorkflow` | Preview/apply with `--expected-revision`, `--preview-digest`, `--authorize-local`; `--file` size-bounded | Unchanged (PD-9 for `--file`) |
| Active selection | `internal/projectapp/workflow_authoring.go:69-76,205-248` | Every report includes `selection`; `select` without authority returns a read-only preview | Used by `axiom-project` as its inspection (PD-1) |
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
   - The `axiom-project` case keeps only `workflow.select`.
   - Add `configuration.readiness` with read-only modes over
     `runtime_profile_validate`, `runtime_profile_preview`,
     `runtime_codex_status`, `runtime_claude_status`, `runtime_codex_auth` and
     `runtime_claude_auth` (PD-2), and add matching `skillFlagSet` cases.
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
   - `workflow.select` metadata stays unchanged; its preview with no
     authority inputs is the Project's read-only active-selection inspection
     (PD-1).
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
   an offline unittest (PD-4). It maps scenarios A–H to existing Go tests,
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

   The report carries every Report H field:
   - Axiom version and target/observed revisions;
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
  keeps its anchored-root, ownership and conflict checks. `--file` currently
  lacks a regular-file/symlink check (PD-9).
- **Credentials and disclosure.** No credential, environment value or host
  path appears in skill text, Evidence or runner output. Readiness `auth`
  runs only the existing read-only vendor status observation.
- **Fail-closed.** Wrong-skill intents are redirected. #276/#277 operations
  and G-1/G-2 are reported as unavailable. There is no Runtime, Profile or
  model fallback, and a synthetic result never fills R-2/R-3.
- **External effects.** None. The runner runs local `go test` only, with fakes
  and controlled observations.
- **Negative tests.** T07 (stale routes, unsupported operations), T09
  (stale/unauthorized apply), T10 scenario F, T11 (HD-006/credential).

## 6. Testing (R-1 deterministic/synthetic)

| Area | Tests (task) |
|---|---|
| Skill registration | manifest-driven `skill_inspect_test.go:41`; `cmd/lingo/skill_inspect_test.go`; runtime tests (T02, T04) |
| Operation metadata | `TestSkillDiscoveryExactCommands`, `TestSkillDiscoveryMatchesParserAndAcceptedForms` (T02) |
| Routing | `TestCanonicalRoutingContract`, `TestCanonicalSkillRoutingTableConvergesWithInspection`, `TestExplicitOperationRoutingIsDeterministic`, `TestSemanticResolutionIsBoundedAndNeverMutatesOnAmbiguity` (T02, T07) |
| Draft handling, validation, preview/apply | new draft journey test (T09); existing #273 authoring tests |
| Project selection | `select` preview read-only test (T08) |
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
  | R1 | v0.15.0 `axiom upgrade` refuses a three-skill archive | Supported path is the new release installer (already exercised by CI journeys); test refusal with zero effects; release note (PD-3) | Yes, until PD-3 is decided |
  | R2 | v0.15.0 set missing from history makes v0.15.0 installs look foreign | T05 before release | No |
  | R3 | Drift between table, inspect and matrix | Atomic T02; existing tests | No |
  | R4 | Wrong inspect flags for runtime actions | `skillFlagSet` cases plus parser-parity test | No |
  | R5 | `--file` accepts non-regular files/symlinks | PD-9 | No (drafts are Runtime-owned) |
  | R6 | Runner map goes stale | Missing test gives `blocked`; unittest pins the map | No |
  | R9 | Runner Evidence attributed to an untested revision | Provenance gate: target = `HEAD`, clean tree, toolchain; otherwise `blocked`; unittest covers divergence | No |
  | R7 | PR #304 not on `main` | T01 gate | Yes |
  | R8 | Claude converges only via `first-run` after `axiom upgrade` | Assert in T06; document in T12 | No |

- **Documentation:**
  - `docs/commands.md` skill sections (`:125-128`, `:521-529`, `:1256`) and the Project workflow authoring section;
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

## 8. Planning decisions for approval with this document

All fall within the accepted contract; none reopens HD-005–HD-008.

| ID | Decision | Recommendation | Alternative |
|---|---|---|---|
| PD-1 | Project active-selection inspection | `workflow.select` preview with no authority inputs | Keep `list` in Project (duplicates an action, conflicts with HD-007) |
| PD-2 | `configuration.readiness` scope | `runtime profile validate/preview` plus `runtime codex|claude status/auth` (read-only) | Exclude `auth` |
| PD-3 | Upgrade from v0.15.0 | New release installer is the supported path; v0.15.0 `axiom upgrade` refuses with zero effects (tested, documented) | Version the archive skill manifest (`skillSetVersion=2`) for an explicit error, changing the installer/metadata contract |
| PD-4 | R-1 runner form | python3 runner + JSON schema + offline unittest; durable; owner `maintainer-acceptance`; manual trigger | Go tests only |
| PD-5 | Lane L in #303 | Not built (optional per amendment) | Build sandbox CLI checks |
| PD-6 | Skill-set constants | Keep `SkillSetVersion`/`BinaryCompatibility` "2"; append history *(confirm in T05)* | Bump |
| PD-7 | v0.15.0 receipt fixture | Add from published v0.15.0 assets (read-only download) | Skip |
| PD-8 | Work Item acceptance surface | SKILL.md procedure section; no new inspect operation | New operation (out of scope) |
| PD-9 | `--file` regular-file/symlink hardening | Separate bounded follow-up Issue | Include in #303 with explicit approval |

---

# Part II — Tasks

Common rules for every task:
- No CLI, application or schema change unless the task names it.
- No vendor inference, Provider effect or network access beyond the Go module
  cache, except PD-7 if approved.
- Existing tests change only where they enumerate skill names or catalog rows.
- Each task ends with `go build ./...`, `go vet ./...` and its focused tests.

### T01 — Contract baseline gate (human)
- **Objective:** implementation starts from a base containing the accepted contract.
- **Scope:** PR #304 merged, or explicit authorization to stack on its head; then the implementation branch is created.
- **Files:** none.
- **Dependencies:** maintainer acceptance of this document, including PD-1 to PD-9.
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
  - add `configuration.readiness` (PD-2) and its `skillFlagSet` cases;
  - no action in two skills;
  - `axiom-work-item` inspect byte-identical to before.
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
  - Work Item acceptance procedure (PD-8) with Plan-confirmation rules;
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
- **Scope:** shared history, published-revision list, optional receipt fixture.
- **Files:**
  - `internal/codexruntime/manifest.go` (append v0.15.0: `axiom-project d80cd1d1…`, `axiom-work-item e47c9256…`);
  - `internal/codexruntime/shared_history_test.go:22`;
  - optional `testdata/published-receipts/v0.15.0/` (PD-7);
  - `testdata/published-skills/PROVENANCE.md`.
- **Dependencies:** T04 (final bytes).
- **Requirements:** append only; Codex-only history and existing fixtures stay frozen; confirm PD-6.
- **Acceptance criteria:** the five history/receipt tests in §6 pass.
- **Verification:** `go test ./internal/codexruntime`.
- **Evidence:** test output; recorded digests.
- **Definition of Done:** green.
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
- **Objective:** HD-005 boundaries hold and the CLI is unchanged.
- **Scope:** one new Project-skill test, plus a check that existing tests are untouched.
- **Files:** a new test in `internal/cli` or `cmd/lingo`; existing `internal/cli/workflow_authoring_test.go`, `cmd/lingo/{workflow_authoring,workflow_binding,workflow_stage_plan}_test.go`.
- **Dependencies:** T02.
- **Requirements:**
  - `workflow.select` metadata unchanged;
  - its no-authority preview is read-only and reports `selection` (PD-1);
  - Work Item `run`/`status`/`plan` metadata unchanged;
  - CLI help tree, flags and results unmodified.
- **Acceptance criteria:** the #273/#274/#275 tests pass unmodified.
- **Verification:** `git diff` shows those tests untouched; test output.
- **Evidence:** diff and output.
- **Definition of Done:** green.
- **Parallel:** with T07.

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
  - PD-9 outcome applied.
- **Acceptance criteria:** the categories returned today; no state change on refusals.
- **Verification:** focused test.
- **Evidence:** test output.
- **Definition of Done:** green.
- **Parallel:** with T07, T10 and T11.

### T10 — Work Item-owned #275 R-1 runner
- **Objective:** deterministic synthetic Evidence for scenarios A–H (Lane S), bound to the exact tested revision.
- **Scope:** runner, schema, offline unittest, registry entry.
- **Files:** `scripts/acceptance/stage-plan-r1.py`, `scripts/acceptance/stage-plan-r1.schema.json`, `scripts/acceptance/test_stage_plan_r1.py`, `scripts/automation-registry.json`; wiring into `scripts/validate-repository.sh` only if it matches how existing acceptance unittests are run.
- **Dependencies:** T01.
- **Requirements:**
  1. **Inputs:** `--target-revision <40-hex>`, `--target-version <version>`, `--source <dir>`, `--output <new-path>`.
  2. **Provenance gate, before any test runs:**
     - `git -C <source> rev-parse HEAD` must equal the target revision;
     - `git status --porcelain` must be empty;
     - `go version` must succeed.

     Otherwise every scenario and `r1` are `blocked`, with reason `revision_mismatch`, `dirty_source`, `target_unverifiable` or `toolchain_missing`. No test results are recorded as `passed`.
  3. **Scenario map:**
     - A, C, D, E: `TestWorkflowStagePlanPublicJourneySingleAndMixedRuntimeGraph`;
     - B: #273 authoring tests;
     - F: the journey negatives, `TestWorkflowStagePlanExecutableFailsClosedAndNeverMutates`, `TestReadPlanDocumentRefusesNonRegularFiles`, `TestCompileRejectsInvalidTopologyControlsAndReferences`, `TestCompileSequentialConcurrencyAndUnsafeOverlap`, `TestPlanStageFailsClosedWithSpecificationCategories`;
     - G: the `snapshotTrees` assertions.

     The runner uses `go test -json -run`, with bounded output and a timeout.
  4. **Report H fields** (closed schema, `additionalProperties: false`):
     - `axiomVersion`, `targetRevision`, `observedRevision`, `sourceClean`, `goVersion`;
     - `runtimeObservation: "controlled"`;
     - `skillSet` (version and per-skill SHA-256 from `axiom --json runtime codex status` against an empty temporary root);
     - `sandboxId` (random, no host path);
     - `lane: "S"`, `class: "synthetic"`;
     - per-scenario `{result: passed|failed|blocked, tests[], refusalCategories[]}`;
     - `r1`;
     - `r2`/`r3` fixed to `deferred_to_278`;
     - `limitations` (G-1–G-5) and `handoff` (#278/AXM-12).
  5. **Classification:** a missing or renamed test gives `blocked`; a timeout or exceeded bound gives `failed`. The runner writes only to a new output path; no network, vendor process, credential, environment value or host path appears in the output.
- **Acceptance criteria:**
  - the schema rejects `passed`/`accepted` for R-2/R-3, missing H fields and unknown fields;
  - the unittest proves that a revision mismatch, dirty source, unverifiable target or missing toolchain each yields `blocked` with no `passed` scenario;
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

  Also confirm the T03 text has no credential, path or tier assumption, and record the PD-9 decision.
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
- **Dependencies:** T02–T12.
- **Requirements:**
  - `go test ./...`, `go vet ./...`, `go build ./...`, `go mod verify`;
  - `scripts/check-go-quality.sh all`;
  - `scripts/validate-repository.sh .`;
  - the T10 runner;
  - a bounded independent review;
  - Draft PR with `Related-Issues: #303`, `Completes-Issues: #303` (only if every R-1 AC is met) and `Relates to AXM-7`;
  - no merge, release, closure, AXM-7 acceptance or AXM-8 unblock.
- **Acceptance criteria:**
  - every check green, or NOT_RUN with a reason;
  - no unresolved Blocker/Major;
  - R-2/R-3 `deferred_to_278`.
- **Verification:** CI on head; review record.
- **Evidence:** final implementation record.
- **Definition of Done:** Draft PR open, pushed under separate authorization.
- **Parallel:** no.

## Task dependency graph

```text
T01 -> T02 -> T03 -> T04 -> T05 -> T06 -> T07 -> T13   (critical path)
T02 -> T08 -> T13
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
- T09 after T03 and T08;
- T12 drafting after T03.

T02–T05 run sequentially: they change the same embedded bytes and digests.

## AC-015 traceability

| AC-015 | Plan § | Task(s) | Test(s) | Expected Evidence |
|---|---|---|---|---|
| 1 Accepted contract, annotated | Status | T01, T12 | ADR governance; Spec index | Contract bytes on base; annotations present |
| 2 Embedded/installed/discoverable; v0.15.0 upgrade; reinstall; history; no duplicate active routes | 3.7, 4 | T04, T05, T06, T07 | runtime/install/upgrade tests; upgrade journeys; inspect tests | Converged three-skill inventories; owned v0.15.0 set |
| 3 Configuration routes = canonical commands; Project routes absent; select/run/status/plan kept; CLI unchanged | 3.1, 3.3, 3.4 | T02, T03, T07, T08 | routing/catalog/inspect/help tests | Parity; negative discovery |
| 4 NL configuration via drafts; Project selects; start resolves and binds | 3.2, 3.4 | T03, T08, T09 | content tests; draft journey; `TestConfiguredExecutableRevisionIsolationAndAuthority` | Journey output |
| 5 Work Item-owned #275 R-1 runner; classified scenarios; R-1/R-2/R-3 separated | 3.4, 3.8 | T10 | runner + unittest (incl. revision mismatch, dirty source → `blocked`) | Evidence JSON A–H with the full Report H fields, bound to the tested revision |
| 6 No real sessions/inference/dispatch/Provider/human E2E; #278 handoff | 1, 6 | T10, T12 | schema rule; review | `deferred_to_278`; handoff checklist |
| 7 R-1 + review + authorization; G-2/G-4 not waived | 7 | T13 | full validation; review | Review record; blockers listed |
| 8 CLI/result/digest/preview/security unchanged | 3, 5 | T08, T13 | unmodified existing tests | Diff review |
| 9 HD-006 boundaries; #305/#306 linked; no tiers/conflation/unproven capability/credential mutation | 3.5, 5 | T03, T11 | profile/resolver tests; content tests | Cited tests |
| 10 Independent review, no unresolved Blocker/Major | 7 | T13 | independent review | Review record |

**Unresolved technical dependencies:**
- *blocking implementation:* R1/PD-3 and R7/T01;
- *non-blocking:* PD-1, PD-2 and PD-4 to PD-9, decided with this document;
- *outside #303:* G-1 #305, G-2 #306, G-3 #272, G-4 consent, G-5 #276.

R-2 and R-3 rows always read `deferred_to_278`. A synthetic PASS is never
real Runtime proof.

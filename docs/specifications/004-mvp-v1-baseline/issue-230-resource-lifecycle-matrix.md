# Issue #230 — Resource lifecycle capability matrix (I230-T01)

| Field | Value |
|---|---|
| Matrix version | 1 |
| Task | I230-T01 — Freeze lifecycle, ownership and authority matrix |
| Issue | [#230](https://github.com/rgomids/axiom/issues/230) |
| Baseline inspected | `main` at `419523c9315afbe70ac6e244dc9908327628e241` (v0.7.0) |
| Status | Technically complete, pending human review. Does not authorize I230-T02 or any later Task. |

Approved sources, in precedence order after current code:

1. [Specification proposal](https://github.com/rgomids/axiom/issues/230#issuecomment-6042966439)
   (FR-068–FR-084, AC-50–AC-61);
2. [Resolved Clarification](https://github.com/rgomids/axiom/issues/230#issuecomment-6043928322)
   (C230-01 Project archive, C230-02 Integration disable/enable/remove);
3. [Implementation Plan](https://github.com/rgomids/axiom/issues/230#issuecomment-6043970620)
   (P230-01–P230-07);
4. [Tasks](https://github.com/rgomids/axiom/issues/230#issuecomment-6044007476)
   (I230-T01–I230-T08);
5. [Specification 004](spec.md) and [Issue #229 Evidence](evidence-229.md).

The FR-068–FR-084 / AC-50–AC-61 amendment is approved in the Issue but is not
yet versioned in `spec.md`; I230-T08 owns that reconciliation. Source
references below are `path:line` at the baseline revision.

## 0. Reading this matrix

**Status labels.** *Current* means executable at the baseline. *#230 target*
means approved by the Clarification/Plan and frozen here for I230-T02–T08; it is
not delivered. Nothing labelled target exists in the binary.

**Effect classes** (never collapsed into read/write/delete):

| Code | Effect class | Meaning |
|---|---|---|
| `R` | read-only | Reads protected local state or portable configuration. No write. |
| `R-X` | read-only Provider | Reads through a Provider adapter (for example GitHub `GET`). No mutation, but it *uses* the Integration. |
| `M-L` | administrative local mutation | Writes machine-local Project operational/installation state only. Never portable, never Provider. |
| `M-P` | portable Project mutation | Writes portable Project intent (`axiom.yaml`) and its coherent local installation record. No Git commit/sync. |
| `M-O` | workflow/local operational mutation | Writes local Execution, Work Item link or projection-ledger state that evolves delivery. |
| `M-X` | external Provider mutation | Mutates a Provider resource (GitHub Issue, label, comment). |

**Admission classes** (I230-T05 guard input, from C230-01 and P230-06):

| Code | Admission | Archived Project | Locally disabled Integration |
|---|---|---|---|
| `I` | inspection | allowed | allowed when it uses no Provider/Transport; `R-X` through the disabled Integration is denied |
| `A` | administrative maintenance | allowed | allowed |
| `E` | operational evolution | denied, `project_archived` | denied before any Provider/Transport call, `integration_disabled`, when it uses that Integration |

The guard classifies an *operation*, including its preview step: a preview
exists only to authorize that operation, so an `E` operation's preview is denied
too.

**Gap statuses:** Delivered · Partial · Missing · Unsupported by design ·
Needs reconciliation.

## 1. Scope and invariants

Preserved without change: Project ≠ Repository; Execution ≠ Agent; Provider ≠
Transport; Integration ≠ MCP; Skill ≠ workflow source of truth; portable Project
intent ≠ machine-local state; Runtime skill → Lingo/application → Axiom
contracts; Provider effects are adapter-owned; operation selection never grants
authority; unsupported lifecycle operations fail explicitly; no universal CRUD.

Approved #230 terminal semantics (C230-01, C230-02):

- **Project:** `active -archive-> archived -reactivate-> active`. Machine-local
  only. It never changes `axiom.yaml` or another machine, and never deletes a
  Project, Repository, Work Item, Execution or Evidence. Rule: *archive blocks
  evolution, not inspection or administrative maintenance.*
- **Integration, local:** `enabled -disable-> disabled -enable-> enabled`.
  Machine-local only. It blocks operational use and keeps inspect, validate and
  admin.
- **Integration, portable:** `remove` drops the declaration from portable
  intent. It resolves dependent references explicitly and never revokes
  credentials, logs out of a Provider, uninstalls MCP/Runtime configuration or
  deletes Provider resources. Rule: *disable/enable controls whether this
  machine may use the Integration; remove controls whether the Project declares
  it at all.*
- **Ownership:** archive and disable state are machine-local Project operational
  state, in a versioned record beside `installation.json` v1, which is not
  extended (P230-02).
- **Not in #230:** Project delete, Repository working-copy or remote deletion,
  Work Item/Provider deletion, Execution update/delete or history mutation,
  credential revocation, Provider logout, MCP/Runtime uninstall.

## 2. Resource classification matrix

### 2.1 Classification of every MVP concept

| Concept | Class | Owner of its lifecycle | Basis |
|---|---|---|---|
| Project | 1 directly user-managed | itself | `internal/project/project.go:79`; `installation.json` |
| Repository association | 1 directly user-managed | Project | portable `repositories[]` (`project.go:29-32`) and local binding (`internal/projectapp/edit.go:47-55`) |
| Work Item | 1 directly user-managed | itself (GitHub adapter for effects) | local link `internal/local/work_item_store.go:26-34` |
| Execution (sequential workflow) | 1 directly user-managed | itself | `internal/workflow/service.go:76-80`; `internal/local/workflow_store.go:143-154` |
| Integration | 1 directly user-managed | Project (portable) and the Project's local operational state | `project.go:40-46`; `internal/manifest/dto.go:74-80` |
| Provider declaration (`providers[]`) | 2 managed through Integration/Project | Project | `project.go:35`; paired with the `work-items` Integration (`edit.go:364-425`) |
| Transport declaration | 2 managed through Integration | Project | `project.go:36-39` |
| Credential reference (portable) / credential binding (local) | 2 managed through Project/Integration | Project / installation | `project.go:56-59`; `internal/projectapp/ports.go:85-87`. Secrets never portable. GitHub auth is the existing `gh` session (`internal/githubissues/adapter.go:34-46`) |
| Runtime / Model Profile / Runtime preference policy | 2 managed through Project | Project (portable), Runtime Profile store (local) | `internal/project/runtime_policy.go:10-81`; `<state-root>/runtime-profiles/v1/configuration.json` (`internal/local/runtime_profile_store.go:60`). No independent managed identity |
| Local operational state (archive, disabled Integrations) | 2 managed through Project/Integration | Project installation boundary | #230 target, I230-T02 |
| Work Item create-attempt fence | 2 managed through Work Item create | Work Item service | `work_item_store.go:403-406`; `internal/workitem/service.go:335-345` |
| Execution transition history and Evidence references | 3 derived/read-only | Execution transitions only | append-only by validation, `service.go:896-938` |
| Workflow lifecycle projection (GitHub labels/comments) and projection ledger | 3 derived/read-only | `workflow reconcile` | `service.go:484-574`, `:794-816` |
| Evidence/diagnostic artifacts | 4 outside #230 | S7 `artifact retire` / `artifact cleanup` | `internal/cli/maintenance.go:43-44`; `docs/commands.md:1452-1475` |
| Execution Graph, attempts, coordination streams | 4 outside #230 user surface | S8 internals; no CLI or Runtime surface | `scripts/acceptance/s9-graph-runner.go:1-6`; `cmd/lingo` imports neither package |
| Runtime skill integration (`runtime codex\|claude install\|status`, `first-run`) | 4 outside #230 | install/upgrade contract | `internal/codexruntime/integration.go:11-40`. A different concept from a Project-declared Integration |
| Release installation and maintenance (`upgrade`, `compatibility inspect\|backup\|export`, `recovery inspect\|apply`) | 4 outside #230 | S7/S9 maintenance | `internal/cli/maintenance.go:40-48` |
| Read-only global surfaces (`version`, `help`, `skill inspect`, `runtime profile preview\|validate`) | 4 outside #230 | their own contracts | `cmd/lingo/main.go:42-48`; `internal/cli/skill_inspect.go:252-256`; `internal/cli/cli.go:269-291`. They manage no user resource |
| Other Provider objects (any GitHub resource beyond the operations below) | 4 outside #230 | none | no generic Provider CRUD |

### 2.2 Lifecycle rows

| # | Dimension | Project | Repository association | Work Item | Execution | Integration |
|---|---|---|---|---|---|---|
| 1 | Resource | Configured, installed Project | Project-scoped Repository key plus this machine's working-copy binding | GitHub Issue linked to a Project Repository | One sequential workflow Execution per Project + Repository + Work Item | Project-declared capability `integrations[]` plus local eligibility |
| 2 | Domain/application owner | `internal/project`, `internal/projectapp` (setup, edit, catalog, lifecycle) | `internal/projectapp` edit/setup (Project-owned, no separate model) | `internal/workitem` | `internal/workflow` | `internal/project` (declaration), `internal/projectapp` (edit, capability); local state from I230-T02 |
| 3 | Persistence / source of truth | Portable `<projects-root>/<slug>/axiom.yaml`; local `<state-root>/projects/<id>/installation.json` v1 (`installation_store.go:91,200-216`); target `operational.json` | Portable `repositories[].key`; local `installation.json` `repositories[]` path and identity | Provider Issue is truth for title, body and state; local link `<state-root>/work-items/<project-id>/<repo>-<sha>.json` v1 | `<state-root>/executions/v1/<project-id>/<sha(scope)>.json`, whole-record CAS (`workflow_store.go:86-94`) | Portable `integrations[]`, `providers[]`; target local disable set in `operational.json` |
| 4 | Classification | Portable + machine-local; archive is machine-local | Key portable; path and identity machine-local | Provider-owned, plus a machine-local link | Machine-local; reconcile projects to the Provider | Declaration portable; enable/disable machine-local; effects are Provider-owned through adapters |
| 5 | Canonical identity / selectors | UUID or installation-unique slug: `--selector` (show/resolve), `--project` (configure edit, work-item, workflow), legacy `--slug` (POC) | `--project` + Repository key | `github:<owner>/<repo>#<n>` (`--work-item`), or `--number` [+ `--provider-repository`]; always `--project` + `--repository` | Scope (Project + Repository + Work Item); `--execution <id>` is a cross-check only (`service.go:613,629-630`) | `--project` + Integration `key`. Only `work-items` is written by Lingo today (`edit.go:22,364-425`) |
| 6 | Discovery / list | Current `project list`, archived not yet hidden. Target `project list` hides archived; `--include-archived` shows them with status | Current `project show` (keys and paths) | Current: none. Target `work-item list`, reading local links only | Current: none (store has Create/Load/Save only, `service.go:139-143`). Target `workflow list` | Current: none (visible only inside the configure edit preview manifest). Target `integration list` |
| 7 | Show / read | Current `project show`, `project resolve` (read-only); target adds archive status | Current `project show`; fails `repository_unavailable` on a broken binding (`cmd/lingo/main.go:464`) | Current `work-item show`, local only (`service.go:479-499`) | Current `workflow status`, `workflow evidence` | Target `integration show` (declaration, readiness, local eligibility) |
| 8 | Administrative mutations | Current `project configure` create. Edit is preview-only. Legacy `project init`, `update`, `install`. Target: edit publication, `archive`, `reactivate` | Current attach at create. Edit upsert/remove preview-only. Target: publish attach, binding update, detach | none | none | Current: `--work-item-provider` at create; edit set/remove preview-only. Target: `update`, `disable`, `enable`, `remove` |
| 9 | Operational / evolution operations | (consumes) Work Item and Execution evolution are gated by archive | none | Current `create`, `select`, legacy `comment`, `complete`. Target `update`, `comment`, `close`, `reopen` | Current `start`, `advance`, `fact`, `resume`, `reconcile` | (consumed) the `work-item` capability by Work Item `R-X`/`M-X` ops and by `workflow reconcile` |
| 10 | Terminal semantic | archive/reactivate (local, reversible). No delete | detach (association only). No Repository delete | close/reopen (Provider state). No delete | `completed` by a `pass` at the `completion` gate (`service.go:408-411`). No cancel and no delete | Local disable/enable; portable remove. No credential or Provider cleanup |
| 11 | States / transitions | Target `active ⇄ archived`; missing record means `active` | attached → detached; re-attach with the same key restores reachability | Provider `OPEN ⇄ CLOSED` (link `state`, `work_item_store.go:268-275`) | `active → interrupted → active` (fail/resume), `active → completed`. No `failed` or `cancelled` state | Target `declared+enabled ⇄ declared+disabled`; `declared → undeclared` by remove |
| 12 | CLI surface | §4 | §4 (reuses `project configure --project`) | §4 | §4 | §4 (target top-level `integration` group) |
| 13 | Canonical Runtime skill | `axiom-project` | `axiom-project` (configure edit) | `axiom-work-item` | `axiom-work-item` | `axiom-project` (target) |
| 14 | Application use case | `projectapp.PrepareSetup` / `Lifecycle.PublishConfigured`; `PreviewEdit` (`edit.go:212`); `ProjectCatalog.List` (`list.go:58`); `InstallationStore.Resolve`; target archive/reactivate service (T03) on the T02 store | `projectapp.PreviewEdit` repository upsert/removal (`edit.go:330-356,438-500`); target edit publication (T03) | `workitem.Service` `Prepare`, `Create`, `PreviewSelect`, `Select`, `Show`, `Comment`, `Complete`; target `List`, `Update`, `Close`, `Reopen` (T06) | `workflow.Service` `Start`, `Transition`, `RecordLifecycleFact`, `Resume`, `Status`, `PrepareProjection`, `Project`; target `List` (T07) | `projectapp` edit (`setWorkItemProvider`, `removeWorkItemProvider`), `GitHubWorkItemCapability` (`edit.go:551-574`); target Integration service (T04) and admission guard (T05) |
| 15 | Provider adapter effect | none | none. Detach never touches the working copy or the remote | `internal/githubissues` via `gh api`: GET issue, GET labels, search, POST issue, POST comment, PATCH `state=closed` (`adapter.go:129-383`); target PATCH title/body, PATCH `state=open` | reconcile: GET labels/issue/comments, POST/DELETE labels, POST comment (`adapter.go:215-309`) | none from lifecycle operations. Gated operations reach GitHub only through Work Item/workflow adapters |
| 16 | Required authority class | `R` none. `M-P`/`M-L` need `--preview-digest` + `--authorize-local` | `M-P` + `M-L` with `--preview-digest` + `--authorize-local` | `M-X` with `--preview-digest` + `--authorize-external`. `select` (`M-O`) needs `--preview-digest` + `--authorize-local` | `M-O` needs `--expected-revision` (`fact` also `--authorize-local`; `start` needs `--runtime-preview`). Reconcile `M-X` needs `--preview-digest` + `--authorize-external` | `M-L` (disable/enable) and `M-P` (remove/update) need `--preview-digest` + `--authorize-local` |
| 17 | Review / preview | Digest-bound complete-candidate preview (`edit.go:141-156,586`) | same Project edit preview, with per-key `added`/`updated`/`removed`/`preserved` (`edit.go:112-118`) | create: draft preview and digest; select: preview with expected local revision. Target: every `M-X` previewed | start: Runtime preview digest; reconcile: projection preview digest; transitions: revision CAS | Target preview names the exact portable/local effect and Integration key |
| 18 | No-op | create equivalent: `already_configured`; edit with no change has empty effects; target archive-when-archived and reactivate-when-active: deterministic no-op | upsert of an unchanged binding is `unchanged` | select of an equal link: `work_item_already_linked`, no authority; create fence `confirmed`: `work_item_already_linked`. Target close-when-closed and reopen-when-open: no-op without a Provider call | start equivalent: `execution_already_started`; transition replay: `workflow_transition_replayed`; resume when not interrupted: `execution_not_interrupted`; reconcile: `projection_already_converged` | Target disable-when-disabled, enable-when-enabled: no-op; remove of an undeclared key: `integration_not_found`, never a silent no-op |
| 19 | Conflict / stale | `coherentSelection` fails closed (`edit.go:279-311`); a stale digest denies; slug/ID collision on create (`main.go:502-521`) | unknown key `EditUnknownRepository`; set+remove of one key rejected (`cli.go:714-728`) | link CAS gives `local_work_item_conflict`; ambiguous number gives `work_item_ambiguous`; ambiguous create gives `provider_create_ambiguous` | `stale_execution_revision`, `execution_scope_conflict`, `execution_selector_conflict` | Target: stale digest denies; contradictory local entries fail closed (T02) |
| 20 | Unsupported operations | delete, uninstall, portable archive; edit replay fails `unsupported_edit_authority` until T03 | delete working copy or remote; second Repository ownership model | delete; arbitrary Provider field update; Provider-side discovery of unlinked Issues | update, delete, history rewrite, cancel (§2.3) | credential revocation, Provider logout, MCP/Runtime uninstall, Provider resource deletion |
| 21 | Recovery implications | Local publication follows existing recovery (`recovery inspect\|apply`); a missing operational record defaults to active | A detached key's local Work Item links and Executions are preserved and become unreachable until re-attach (§6 F-03) | Partial `provider_confirmed_local_*` reports the Provider effect truthfully; create fence prevents a re-POST | Interrupted → resume; projection Partial/Retryable keep confirmed effects (`service.go:830-848`) | A stale local disable entry for an undeclared key is inert (§6 F-04); remove never touches local or external state |
| 22 | Evidence / test owner | Current: `internal/projectapp/{setup,edit,list,lifecycle}_test.go`, `cmd/lingo/configure_edit_test.go`. Target: I230-T02, T03 | Current: `cmd/lingo/{configure_edit,install_bindings}_test.go`. Target: I230-T03 | Current: `internal/workitem/service_test.go`, `internal/githubissues/adapter_test.go`. Target: I230-T06 | Current: `internal/workflow/*_test.go`, `internal/local/workflow_store_test.go`. Target: I230-T07 | Current: `internal/project` validation tests, `edit_test.go`. Target: I230-T02, T04, T05 |
| 23 | FR / AC | FR-068–070, 071–075, 082–083; AC-50–52, 57–58 | FR-070–074, 076; AC-53 | FR-069–074, 077, 084; AC-54 | FR-069–070, 074, 078; AC-55 | FR-069–074, 079, 082; AC-56 |

### 2.3 Execution cancellation (current truth)

| Execution type | User reachable | Cancellation in current contract | #230 consequence |
|---|---|---|---|
| Sequential workflow Execution | yes (`workflow *`) | **None.** Statuses are only `active`, `interrupted`, `completed` (`service.go:76-80`). `workflow_cancelled` means the *invocation's* context ended, and it writes nothing (`service.go:268-269`) | `cancel` is **Unsupported by design**. It returns an explicit unsupported result and is never emulated as delete, rollback or `interrupted` |
| Execution Graph child/attempt | **no** (no CLI or skill; `scripts/acceptance/s9-graph-runner.go:4-6`) | `CancelRequested` marks undispatched children `cancelled_before_dispatch` without persisting (`internal/executiongraph/scheduler.go:95-101`); a running attempt becomes `cancelled` only when the process is confirmed stopped, otherwise `unknown` (`scheduler.go:163-170,336-339`) | Outside the #230 user surface. Exposing a graph surface belongs to S8 and is not inferred here |

## 3. Operation / effect / authority matrix

`operation → effect → admission → authority → owner → adapter`. *Current* rows
are executable; *target* rows are frozen for the named Task.

| Resource | Operation | State | Effect | Admission | Authority | Owner | Adapter |
|---|---|---|---|---|---|---|---|
| Project | configure (create) | current | `R` preview; then `M-P`+`M-L` | A | `--preview-digest` + `--authorize-local` | `projectapp.PrepareSetup`, `Lifecycle.PublishConfigured` | local portable + installation stores |
| Project | configure edit (`--project`) | current preview-only; publication target T03 | `R` now; target `M-P`+`M-L` | A | target `--preview-digest` + `--authorize-local` (now rejected, `cli.go:704-708`) | `projectapp.PreviewEdit` | local stores |
| Project | list | current; `--include-archived` target T03 | `R` | I | none | `projectapp.ProjectCatalog.List` | installation store |
| Project | show / resolve | current; archive status target T03 | `R` | I | none | `local.InstallationStore.Resolve` | installation + portable stores |
| Project | validate (`--slug`, POC) | current | `R` | I | none | `projectapp.Lifecycle.Validate` (projects root by slug) | portable store |
| Project | validate (`--project`) | target T03 | `R` | I | none | target validation over the recorded source | stores |
| Project | archive / reactivate | target T02+T03 | `M-L` | A | `--preview-digest` + `--authorize-local` | target operational-state service | target local operational store |
| Project | init / update / install / reopen (POC) | current, historical | `M-P` (init, update); `M-L` (install); `R` (reopen, `installation_store.go:281-311`) | A | none beyond the invocation (historical event contract) | `projectapp.Lifecycle`, `local.InstallationStore` | local stores |
| Repository association | attach at create | current | `M-P`+`M-L` | A | as configure create | `projectapp.PrepareSetup` | local stores |
| Repository association | attach / binding update / detach (edit) | current preview-only; publication target T03 | target `M-P`+`M-L` | A | `--preview-digest` + `--authorize-local` | `projectapp.PreviewEdit` + target publication | local stores; never the working copy or remote |
| Work Item | create | current | `R-X` preview; `M-X`+`M-O` | E | `--preview-digest` + `--authorize-external` | `workitem.Service.Prepare/Create` | GitHub POST issue |
| Work Item | select | current | `R-X` preview; `M-O` link | E | `--preview-digest` + `--authorize-local` | `PreviewSelect/Select` | GitHub GET issue |
| Work Item | show | current | `R` (local link) | I | none | `Show` | none |
| Work Item | list | target T06 | `R` (local links) | I | none | target `List` | none |
| Work Item | update (title, Axiom-authored sections) | target T06 | `R-X` preview; `M-X` | E | `--preview-digest` + `--authorize-external` | target `Update` | GitHub PATCH issue |
| Work Item | comment | current, historical; reviewed form target T06 | `M-X` | E | now `--authorize-external` only, with no preview (`service.go:520-534`); target adds the reviewed digest (§6 F-01) | `Comment` | GitHub POST comment |
| Work Item | close | target T06 | `M-X`+`M-O` (link state) | E | `--preview-digest` + `--authorize-external` | target `Close` | GitHub PATCH `state=closed` |
| Work Item | complete | current, historical; compatibility of `close` (§6 F-01) | `M-X`+`M-O` | E | now `--authorize-external` only | `Complete` (`service.go:536-554`) | GitHub PATCH `state=closed` (`adapter.go:382-383`) |
| Work Item | reopen | target T06 | `M-X`+`M-O` | E | `--preview-digest` + `--authorize-external` | target `Reopen` | GitHub PATCH `state=open` |
| Execution | start | current | `R` preview; `M-O` | E | `--runtime-preview` digest | `workflow.Service.Start` | none |
| Execution | advance / resume | current | `M-O` | E | `--expected-revision` | `Transition`, `Resume` | none |
| Execution | fact | current | `M-O` | E | `--expected-revision` + `--authorize-local` | `RecordLifecycleFact` | none |
| Execution | reconcile | current | `R-X` preview; `M-X`+`M-O` ledger | E | `--expected-revision` + `--preview-digest` + `--authorize-external` | `PrepareProjection`, `Project` | GitHub labels and comments |
| Execution | status / evidence | current | `R` | I | none | `Status` | none |
| Execution | list | target T07 | `R` | I | none | target `List` (Project-scoped enumeration) | none |
| Execution | cancel | unsupported by design (§2.3) | — | — | — | explicit unsupported result (T07) | none |
| Integration | list / show | target T04 | `R` | I | none | target Integration service | none |
| Integration | validate | target T04 | `R` (declaration, references, static readiness; no Provider call) | I | none | target Integration service | none |
| Integration | configure / update | current at create; edit target T03/T04 | `M-P` | A | `--preview-digest` + `--authorize-local` | `setWorkItemProvider` | none |
| Integration | disable / enable | target T02+T04 | `M-L` | A | `--preview-digest` + `--authorize-local` | target operational-state service | target local operational store |
| Integration | remove | target T04 (preview-only predecessor: `--remove-work-item-provider`) | `M-P` | A | `--preview-digest` + `--authorize-local` | `removeWorkItemProvider` generalized | none; zero credential, MCP, Runtime or Provider effect |

## 4. CLI / Runtime routing matrix

`domain operation → CLI → canonical Runtime skill → application use case`.
Runtime routing is from `internal/cli/skill_inspect.go:82-164` and the
`## Operation routing` tables. The compatibility skills `axiom-project-configure`,
`-list`, `-show`, `axiom-work-item-create`, `-run` and `-status` map to the same
modes (`skill_inspect.go:140-154`). Target CLI spellings are frozen here for
I230-T08 and follow the existing two-level grammar and strict parser.

| Domain operation | CLI | Runtime skill / operation / mode | Application use case | State |
|---|---|---|---|---|
| Project create | `project configure --slug … [--repository k=p] [--work-item-provider github] [--project-id --preview-digest --authorize-local]` | `axiom-project` / configure / create | `PrepareSetup` → `PublishConfigured` | current |
| Project edit (name, provider, repositories) | `project configure --project <sel> …` | `axiom-project` / configure / edit (preview-only) | `PreviewEdit` | current preview-only; publish target T03 |
| Project list | `project list [--include-archived]` | `axiom-project` / list | `ProjectCatalog.List` | current; flag target T03 |
| Project show | `project show --selector <sel>` | `axiom-project` / show | `InstallationStore.Resolve` | current |
| Project resolve | `project resolve --selector <sel>` | — (not routed) | `InstallationStore.Resolve` | current |
| Project validate | `project validate --slug` (POC); `project validate --project <sel>` | target `axiom-project` / validate | `Lifecycle.Validate`; target recorded-source validation | current POC; target T03/T08 |
| Project archive / reactivate | `project archive --project <sel>`, `project reactivate --project <sel>` [`--preview-digest --authorize-local`] | target `axiom-project` / archive, reactivate | target operational-state service | target T03/T08 |
| Project init / update / install / reopen | `project init\|update\|install\|reopen` | — (not routed; historical) | `Lifecycle`, `InstallationStore` | current, historical |
| Repository attach / update / detach | `project configure --project <sel> --repository k=p` / `--remove-repository k` | `axiom-project` / configure / edit | `PreviewEdit` + target publication | current preview-only; publish target T03. No separate `repository` group |
| Work Item create | `work-item create …` | `axiom-work-item` / create / new | `Prepare` → `Create` | current |
| Work Item select | `work-item select …` | `axiom-work-item` / create / existing | `PreviewSelect` → `Select` | current |
| Work Item show | `work-item show …` | — (not routed); target `axiom-work-item` / show | `Show` | current CLI; Runtime target T08 |
| Work Item list | `work-item list --project <sel> [--repository <key>]` | target `axiom-work-item` / list | target `List` | target T06/T08 |
| Work Item update | `work-item update <selectors> [--title] [section flags] [--preview-digest --authorize-external]` | target `axiom-work-item` / update | target `Update` | target T06/T08 |
| Work Item comment | `work-item comment … --message` | — (not routed); target `axiom-work-item` / comment | `Comment` | current, historical; target T06/T08 |
| Work Item close / reopen | `work-item close …`, `work-item reopen …` | target `axiom-work-item` / close, reopen | target `Close`, `Reopen` | target T06/T08 |
| Work Item complete | `work-item complete …` | — (not routed) | `Complete`; target compatibility path to `Close` | current, historical (§6 F-01) |
| Execution start / advance / resume | `workflow start\|advance\|resume …` | `axiom-work-item` / run / transition | `Start`, `Transition`, `Resume` | current |
| Execution fact | `workflow fact …` | `axiom-work-item` / run / fact | `RecordLifecycleFact` | current |
| Execution reconcile | `workflow reconcile …` | `axiom-work-item` / run / reconcile | `PrepareProjection`, `Project` | current |
| Execution status / evidence | `workflow status\|evidence …` | `axiom-work-item` / status | `Status` | current |
| Execution list | `workflow list --project <sel> [--repository <key>]` | target `axiom-work-item` / status / list | target `List` | target T07/T08 |
| Execution cancel | none; an invented `workflow cancel` stays `invalid_command` | none; semantic intent "cancel" is unsupported | — | unsupported by design |
| Integration list / show / validate | `integration list\|show\|validate --project <sel> [--integration <key>]` | target `axiom-project` / integration / list, show, validate | target Integration service | target T04/T08 |
| Integration update | `project configure --project <sel> --work-item-provider <id>` (the only Lingo-writable Integration) | `axiom-project` / configure / edit | `setWorkItemProvider` | current preview-only; publish target T03 |
| Integration disable / enable / remove | `integration disable\|enable\|remove --project <sel> --integration <key>` [`--preview-digest --authorize-local`] | target `axiom-project` / integration / disable, enable, remove | target Integration + operational-state services | target T04/T08 |

Runtime constraints for all target rows: explicit operations dispatch
deterministically; semantic resolution is `allowed` only for `R` modes and
`unambiguous-only` for every mutating mode (`skill_inspect.go:159-164`);
ambiguous intent never selects archive, disable, remove, detach, close or
reopen; authority inputs are never supplied by routing.

## 5. Current vs target gap matrix

This is the implementation input for I230-T02–T08.

| Resource | Operation | Gap | Evidence | Task |
|---|---|---|---|---|
| Project | configure create | Delivered | `cmd/lingo/main.go:480-560` | — |
| Project | edit publication | Partial (preview only) | `cli.go:704-708`; `edit.go:33-34` | T03 |
| Project | list | Partial (no archive filter or status) | `main.go:423-436` | T03 |
| Project | show / resolve | Partial (no archive status; show fails on an unavailable binding) | `main.go:409-421,464` | T03 |
| Project | validate | Needs reconciliation (POC `--slug` reads the projects root, not the recorded source) | `lifecycle.go:180-195` vs the #147 rule in `spec.md:43-60` | T03 |
| Project | archive / reactivate | Missing | no symbol | T02, T03 |
| Project | `update --name` (POC) | Needs reconciliation (writes portable state by slug and does not update the installation record's portable revision, which installed consumers require (`main.go:262-264`); inferred from code, not executed) | `lifecycle.go:205-235` | T03 |
| Project | `reopen` (POC) | Needs reconciliation, naming only (read-only verification, not reactivate) | `installation_store.go:281-311` | T03/T08 docs |
| Project | delete / uninstall | Unsupported by design | C230-01 | — |
| Repository | attach at create | Delivered | `setup.go` | — |
| Repository | attach, update, detach on existing Project | Partial (preview only) | `edit.go:330-356,438-500` | T03 |
| Repository | list / show | Partial (via `project show`; no portable/local status split; broken-binding show fails) | `main.go:409-421` | T03 |
| Repository | delete working copy or remote | Unsupported by design | FR-076 | — |
| Work Item | create, select | Delivered | `service.go:269-477` | — |
| Work Item | show | Delivered (local link; CLI only) | `service.go:479-499` | T08 routing |
| Work Item | list | Missing | no service, adapter or CLI verb | T06 |
| Work Item | update | Missing | no adapter PATCH of title/body | T06 |
| Work Item | comment | Needs reconciliation (historical, unreviewed authority, not idempotent) | `service.go:520-534` | T06 (§6 F-01) |
| Work Item | close | Partial (exists only as `complete`) | `adapter.go:382-383` | T06 |
| Work Item | complete | Needs reconciliation (bare close under another name) | `service.go:536-554`; `main.go:991`; `docs/commands.md:1287-1289` | T06 (§6 F-01) |
| Work Item | reopen | Missing | no adapter call | T06 |
| Work Item | delete | Unsupported by design | FR-077 | — |
| Execution | start, advance, fact, resume, reconcile, status, evidence | Delivered | `service.go:267-574` | T05 gating |
| Execution | list | Missing | `service.go:139-143` | T07 |
| Execution | cancel (sequential) | Unsupported by design | §2.3 | T07 explicit result |
| Execution | cancel (graph) | Outside #230 user surface | §2.3 | — |
| Execution | update / delete | Unsupported by design | FR-078 | — |
| Integration | list / show | Missing | no surface | T04 |
| Integration | validate | Partial (only inside whole-manifest POC validation) | `validation.go:119-154` | T04 |
| Integration | configure / update | Partial (create delivered; edit preview-only; `work-items` only) | `edit.go:364-393` | T03/T04 |
| Integration | disable / enable | Missing | no concept; the Runtime Profile `enabled` flag is unrelated (`internal/runtimeprofile/runtimeprofile.go:44`) | T02, T04 |
| Integration | remove | Partial (preview-only `--remove-work-item-provider`) | `edit.go:395-425` | T04 |
| Integration | credential revoke / logout / MCP uninstall | Unsupported by design | C230-02 | — |
| Cross-cutting | archive and disable admission guard | Missing; `workflowResolver` does not check the `work-item` capability, unlike `workItemResolver` (`main.go:266-268` vs `276-285`) | `cmd/lingo/main.go` | T05 |
| Cross-cutting | Runtime routing for new operations | Missing | `axiom-project` supports only configure/list/show; `axiom-work-item` only create/run/status (`SKILL.md:14-16` each) | T08 |
| Cross-cutting | FR-068–084 in `spec.md` | Needs reconciliation | not yet versioned | T08 |

## 6. Open issues

**Human decision required: none.** Every row above follows from repository
truth, the approved Specification, C230-01/C230-02 and the Plan. The items below
are consequences frozen by this matrix. They need reviewer attention because
they change or confirm visible behavior. None introduces a new ownership,
destructive or authority boundary. A reviewer who disagrees should amend the
row before authorizing the dependent Task.

- **F-01 — `comment` and `complete` converge on the reviewed external-mutation
  contract (T06).** Evidence: `complete` is one GitHub `PATCH {"state":"closed"}`
  plus a local link update. It has no `state_reason`, no workflow precondition,
  no preview and no use of `--authorize-local` (`service.go:536-554`,
  `adapter.go:382-383`). Code and docs call both commands historical POC
  surfaces "until the later Slice replaces it" (`main.go:989-992`,
  `docs/commands.md:1287-1289`). Neither is routed by a Runtime skill.
  - Frozen: canonical `close` is the terminal operation. `complete` remains a
    CLI compatibility spelling that calls the same `Close` use case. Neither is
    workflow progress or human acceptance.
  - FR-073 requires reviewed effects for every external mutation, so `close`,
    `complete` and `comment` all require `--preview-digest` +
    `--authorize-external`.
  - Compatibility impact: a single-step `--authorize-external` call that
    succeeds today will return `external_authority_denied` with a preview.
- **F-02 — No user-invocable Execution cancel in #230 (T07).** The only
  user-reachable Execution type has no cancellation contract (§2.3).
  - T07 returns an explicit unsupported result.
  - Adding cancellation to sequential workflow, or a graph surface, would be a
    new semantic outside #230.
- **F-03 — Repository detach preserves dependent local history (T03).** No
  portable declaration references a Repository key (`validation.go:70-154`).
  Local Work Item links and Executions are addressed by that key
  (`work_item_store.go:394-397`, `workflow_store.go:83`).
  - Detach is administrative and not blocked by that history.
  - The preview lists the preserved records as effects, and nothing is
    deleted.
  - Those records stay unreachable for evolution until the same key is
    re-attached.
- **F-04 — Integration remove and dependent references (T04).** No portable
  declaration references an Integration key. An Integration references a
  Provider and a CredentialReference.
  - Removing the canonical `work-items` Integration also removes its paired
    `providers[work-items]` in the same reviewed effect, as
    `removeWorkItemProvider` already does. Other Integrations remove only the
    declaration; still-valid Provider and Credential declarations are kept and
    named in the preview.
  - Afterwards Work Item operations fail `work_item_capability_unavailable`,
    as specified for a Project without a Work Item Provider
    (`spec.md:515-518`). Existing links and Executions are preserved.
  - `remove` never writes local operational state. A local disable entry for a
    key that is no longer declared, here or after synchronization from another
    machine, is reported as stale and inert. It never blocks other operations
    and is cleared only by a local `enable`.
- **F-05 — Integration disable scope (T05).** "Use" means any Provider or
  Transport call through that Integration, reads included (P230-06: "zero
  Provider/Transport effects").
  - For `work-items`, this denies Work Item `create`, `select`, `update`,
    `comment`, `close`, `reopen` and `complete`, plus `workflow reconcile`.
  - Work Item `show`/`list` and Execution `status`/`evidence`/`list` stay local
    and allowed.
  - Integration `validate` makes no Provider call. A live connectivity check is
    not part of #230.
- **F-06 — Work Item scope bounds (T06).**
  - `list` enumerates Axiom-linked Work Items only. An unlinked GitHub Issue
    is not an Axiom-managed resource until `select`, which already takes its
    Provider identity.
  - `update` is limited to title and the Axiom-authored sections. Type and
    classification labels stay create-time contracts (#136). There is no
    arbitrary field bag.
- **F-07 — Name collisions (T02/T03/T08 docs).** Avoid two confusions:
  - `project reopen` (POC read-only verification) is not `project reactivate`
    and not Work Item `reopen`.
  - The S7 preservation namespace `AXIOM_ARCHIVE_ROOT` (`cmd/lingo/main.go:100-116`)
    is unrelated to Project archive state. Archive must not write there.

## 7. Traceability

| Requirement | Where established | Remaining owner |
|---|---|---|
| FR-068 lifecycle inventory | §2.1, §2.2 rows 1–23 | T08 final Evidence |
| FR-069 resource-specific lifecycle | §2.2 rows 10–11, 20; §3 | T03, T04, T06, T07 |
| FR-070 discovery before identity | §2.2 rows 5–7; §5 list gaps | T03, T04, T06, T07 |
| FR-071 safe update | §2.2 row 17; §3 edit rows | T02–T04, T06 |
| FR-072 no-op / conflict | §2.2 rows 18–19 | T02–T04 |
| FR-073 reviewed effects and authority | §0 effect classes; §2.2 row 16; §3; F-01 | T02, T05–T08 |
| FR-074 terminal semantics | §1; §2.2 rows 10–11; §2.3 | T03, T04, T06, T07 |
| FR-075 Project lifecycle | §2.2 Project; §5 | T02, T03, T05 |
| FR-076 Repository association | §2.2 Repository; F-03 | T03 |
| FR-077 Work Item | §2.2 Work Item; F-01, F-06 | T06 |
| FR-078 Execution | §2.2 Execution; §2.3; F-02 | T07 |
| FR-079 Integration | §2.2 Integration; F-04, F-05 | T02, T04, T05 |
| FR-080 CLI/Runtime convergence | §4 | T05–T08 |
| FR-081 skill thinness | §4 Runtime constraints | T08 |
| FR-082 portable/local/credential separation | §0 `M-L`/`M-P`; §2.1 credentials | T02–T05, T08 |
| FR-083 fail-closed unsupported | §2.2 row 20; §5 unsupported rows | T02–T08 |
| FR-084 partial external effects | §2.2 row 21 | T05–T08 |
| AC-50 | this document | T08 |
| AC-51–AC-53 | §2.2 Project/Repository; §5 | T02, T03 |
| AC-54 | §2.2 Work Item; F-01, F-06 | T06 |
| AC-55 | §2.2 Execution; §2.3 | T07 |
| AC-56 | §2.2 Integration; F-04, F-05 | T02, T04 |
| AC-57–AC-60 | §0 admission; §2.2 rows 18–20 | T02–T08 |
| AC-61 | §2–§4 matrices | T08 |

## 8. I230-T01 Evidence

- **Revision inspected:** `419523c9315afbe70ac6e244dc9908327628e241`, also
  `origin/main`, on 2026-10-07.
- **Resources inventoried:** Project, Repository association, Work Item,
  Execution and Integration, plus 13 managed, derived or out-of-scope concepts
  (§2.1).
- **Executable surfaces inspected:**
  - CLI actions `internal/cli/cli.go:438-465` and
    `internal/cli/maintenance.go:40-48`;
  - service composition `cmd/lingo/main.go`;
  - application services in `internal/projectapp`, `internal/workitem` and
    `internal/workflow`, plus `internal/executiongraph` and
    `internal/graphapplication`;
  - local stores `internal/local/{installation,portable,work_item,workflow,graph,coordination,runtime_profile}_store.go`;
  - the GitHub adapter `internal/githubissues/adapter.go`;
  - Runtime routing in `internal/cli/skill_inspect.go` and
    `internal/codexruntime/skills/axiom-{project,work-item}/SKILL.md`.
- **Method:** the Orchestrator inspected Project and Repository directly.
  Read-only investigations covered Work Item, Execution, Integration and
  Runtime routing. Decisive claims were re-checked against the source,
  including `complete` (`service.go:536-554`, `adapter.go:378-386`), workflow
  statuses and the store interface (`service.go:74-80,139-143`), graph
  reachability and cancellation (`scheduler.go:50-60,92-102`), and capability
  gating (`main.go:243-285`).
- **Effect classes used:** `R`, `R-X`, `M-L`, `M-P`, `M-O`, `M-X`.
  **Admission classes:** `I`, `A`, `E`.
- **Unsupported by design:** Project delete and uninstall; Repository
  working-copy or remote deletion; Work Item delete; Execution
  update/delete/cancel (sequential); credential revocation, Provider logout,
  MCP/Runtime uninstall; Provider resource deletion.
- **Validation**, run on 2026-10-07 at the baseline plus this change:

  | Check | Command | Exit |
  |---|---|---|
  | Matrix completeness | ephemeral script: every `action` constant in `internal/cli/cli.go` and `internal/cli/maintenance.go` (34) must appear with its command group in this document | 0 (34/34 covered) |
  | No target shown as current | same script: no `archive`, `reactivate`, `include-archived`, `work-item list\|update\|close\|reopen`, `workflow list` or `integration *` row may be labelled `current` | 0 (0 rows) |
  | Repository validation | `scripts/validate-repository.sh` (includes `check-claude-bootstrap.sh --skill-adapters`) | 0 |
  | Sensitive files | `scripts/check-sensitive-files.sh` | 0 |
  | ADR governance | `python3 scripts/check-adr-governance.py` | 0 |
  | Whitespace | `git diff --check` (new file added intent-to-add) | 0 |

  Go tests were not run: the change touches only Markdown.
- **Limitations:**
  - No product code was executed against real or fake Providers in this Task.
  - The POC `project update` revision-coherence gap (§5) is inferred from
    code, not reproduced.
  - Whether S7 `artifact cleanup` can remove `executions/` records was not
    traced. It is outside #230 either way.

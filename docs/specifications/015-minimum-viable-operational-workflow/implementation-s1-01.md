# S1-01 implementation receipt — AXM-42 (PARTIAL)

Attempt `S1-01`, stage `implement`, on [PR #366](https://github.com/rgomids/axiom/pull/366), branch `spec/axm-42-minimum-viable-workflow`, from verified base `8e5977ec43a9874410830d743d3d2845e2f96567`.
Authority: S1 implementation as stated in the AXM-42 dispatch (Linear comment `32f02fa5`, read-back **not verified** in this session: Linear read permission was denied); architecture per [ADR-0037](../../decisions/0037-one-canonical-execution-for-new-runs.md).
Classification: deterministic unit and isolated local-stub evidence only. No vendor CLI, Provider, remote or real installed state was touched.

## Task results

| Task | Result | Evidence |
| --- | --- | --- |
| S1.1 | DONE | ADR-0037 records the approved decision; ADR-0018 annotations in ADR-0009/0020/0024/0025/0026/0027/0032, index notes, prospective notes on Spec 007, #276, #306, Spec 004 #272 |
| S1.2 | DONE (new owner) | `internal/workflow/execution*.go`, `internal/local/execution_store.go`: format 4 under `executions/v2`, strict codec, legacy refusal, rooted/no-symlink paths, revision CAS, lease pinned through the ADR-0007 commit point, crash-window tests |
| S1.3 | DONE (resolver + reader entry) | `runtimeapplication/invocation.go` precedence/provenance; `runtimeadapter/frozen_invocation.go` reuses the single bounded effort-source reader, no auth preflight. Old Profile-v4 selection/admission consumers **not yet removed** (S1.7) |
| S1.4 | DONE (new engine) | `stageexecution/engine.go` over the ADR-0028 `OSProcessRunner`; old graph/single supervisors **not yet removed** (S1.7) |
| S1.5 | PARTIAL | `stageexecution/run_start.go` + minimal default Workflow: one start action, linked repository identity check, automatic sequential progression, no Provider access. **Public CLI verb and installed-skill wiring not done** |
| S1.6 | DONE (domain + engine ingest) | `workflow/execution_messages.go`, `Engine.Ingest`. Child-process transport for the private ingress is not wired (consumed by S2.2) |
| S1.7 | NOT_STARTED | No legacy producer/consumer deleted yet; see measured delta |
| S1.8 | PARTIAL | `local/cutover.go` isolated inventory/backup/decision + fixtures; checklist below. Architecture/commands/skills docs not updated because no new public command ships yet |

## Checks (exact commands)

- PASS: `go build ./...`; `go vet ./internal/... ./cmd/...`.
- PASS: targeted new tests `go test ./internal/workflow ./internal/local ./internal/stageexecution ./internal/runtimeapplication ./internal/runtimeadapter -count=1` (full packages, including pre-existing suites); engine tests repeated `-count=3`.
- `go test ./... -count=1`: 34 packages PASS; `cmd/lingo` FAIL by the default 10m timeout while running in parallel (`TestInstalledGraphOperationalDiagnosticsBothAdapters` still progressing; package unchanged by S1). Rerun alone: PASS, `go test ./cmd/lingo -count=1 -timeout=30m` (572.7s). The native suite already uses `-timeout=25m`.
- NOT_RUN: `-race` (no C compiler), repository Python/shell validators (approval unavailable), gitleaks.

## Measured delta (vs `8e5977e`, `internal/` + `cmd/`)

Added production 2,476 lines in 10 files; added tests 1,506 lines in 7 files; deleted 0. The replacement exists beside the legacy path; **no reduction is claimed**. The baseline envelope (88 production files / 21,872 lines) is untouched. The end-of-S3 budget (net −15 files / −4,000 lines / −40 exported types) therefore depends entirely on S1.7–S3.4 retirement.

## Breaking-cutover operator checklist (prepared, NOT_RUN)

Run only under separate authority, on the exact installed source/binary:

1. Stop new starts/writers. Run `local.InspectCutover(stateRoot)`; record sanitized counts and blockers, never state dumps in Git.
2. Any blocker — held runner lease, interrupted publication marker, symlink, unknown top-level entry, unreadable family — **BLOCKS** cutover.
3. `local.BackupLegacyState(stateRoot, emptyPrivateDirOutsideState, inventory)` copies raw bytes and verifies byte identity; it never decodes, converts or deletes. If infeasible, record why.
4. Stop only verified owned processes with existing controls; unknown/escaped work or uncertain remote mutation stays BLOCKED pending explicit disposition. Never replay.
5. `local.CutoverDecision(inventory, disposition)` permits switching only with a `human:` actor, the exact inventory digest, a verified (or explicitly infeasible) backup and disposed legacy attempts.
6. After switching, only `executions/v2` format 4 runs; old bytes report `execution_legacy_format_refused`. Rollback never points an old binary at new state or resumes old attempts.

## technical_decisions[]

1. **Format 4 kept despite an existing unkinded legacy format 5** in `executions/v1`: the `kind: execution` discriminator plus the separate `executions/v2` namespace make it unambiguous; any record without that kind/version pair reports the cutover refusal. Plan TD-01 named version 4.
2. **Runner lease reused** (`local.RunLeaseStore`, `run-leases/v1`) as the single runner fence; its epoch becomes the Execution runner generation. No new lock type.
3. **Engine depends on a store interface** (`workflow.RunnerFence` alias) instead of importing `local`, keeping storage below the application layer and avoiding a test import cycle with a legacy test.
4. **Only the runner classifies cancelled-before-spawn**: cancel intent never rewrites attempts, because only the runner knows whether a spawn happened.
5. **Unauthorized reported effects become `unknown`**, not failed: an effect outside authority may have happened.
6. **Process primitives reused in place** (`executiongraph.OSProcessRunner`, `ObserveRecordedProcess`); relocation into `stageexecution` belongs to S1.7 and does not count as deletion.
7. **Messages accepted only from running attempts of the current generation**; exact redelivery is idempotent; rejected candidates are not stored.

## deferred_findings[]

- Historical Model Advisor synthetic failure and builtin SDD r1/r2 drift: not reproduced or touched (deferred as before).
- `-race` NOT_RUN: no C compiler in this environment (`CGO_ENABLED=1` fails: gcc not found).

## Limitations / NOT_RUN

Live Codex/Claude, real E2E, Provider effects, real state inventory/backup/stop/cutover, Product Accepted: NOT_RUN by scope. Repository Python/shell validators (`check-adr-governance.py`, `check-sensitive-files.sh`) were NOT_RUN: execution needed approval that this non-interactive session lacked; a manual `git grep` secret/private-path scan of changed docs found nothing.

## Remaining work (resume_implement)

1. S1.5: `workflow run` CLI verb on `internal/cli`/`cmd/lingo`, resolving Project context, stored Work Item link and local Runtime defaults into `StartRequest`/`InvocationRequest`; a real Codex/Claude `Launcher` applying `CheckFrozenSelectors`; status/cancel/retry/reconcile verbs.
2. S1.7: port surviving invariant tests, then delete A–H/K producers and consumers (validatorpolicy, workflowcompiler stage plan, policy_v4 selection, preview/authority DTOs, graph/single stores and supervisors, authpreflight composition), retired CLI verbs and retired-only tests; relocate process primitives; measure against `deletion-baseline.json`.
3. S1.8: reconcile `docs/architecture`, `docs/commands.md`, installed docs/skills once the verb ships.

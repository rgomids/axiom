# AXM-12 — self-correcting E2E validation from v0.24.1

Sanitized, incremental record of the 2026-11-14 E2E mission (America/Sao_Paulo
dates, UTC timestamps). Raw JSON outputs stay on the test host, outside Git.
This record is technical Evidence only: it is not Product acceptance, review
approval or Work Item closure, and it changes no accepted Specification or ADR.

## Environment

- Host: macOS 27.2.1, arm64. Go 1.26.1. codex-cli 0.166.0 (ChatGPT login),
  Claude Code 2.1.323 (claude.ai login). No `ANTHROPIC_API_KEY`/`OPENAI_API_KEY`
  in the environment; no paid API used.
- Project `axiom-mvp-e2e-20261106` (`e4e2b26f-ca22-4258-96a8-fbe3d2c650ad`),
  Repositories `library` (`rgomids/axiom-e2e-library`) and `cli`
  (`rgomids/axiom-e2e-cli`); Work Items `axiom-e2e-library#1`, `axiom-e2e-cli#1`.
- Models: Claude `claude-haiku-5-5`. Codex `gpt-6.1-luna` was refused by the
  vendor for ChatGPT auth before inference ("not supported when using Codex
  with a ChatGPT account"); per the prior recorded authorization
  ([test-battery policy](../../operational/test-battery-simplest-model.md),
  AXM-41 amendment, and the 10/11 human grant on AXM-12) Codex uses
  `gpt-6.1-sol`. New local Profile `codex-gpt-6-1-sol-low` (effort `low`).
- The pre-existing `~/.local/bin/axiom` is a development build (`f5709fcbacf7`),
  not the release; it was preserved and not used for cycle 1.

## Cycle 1 — release v0.24.1

Build under test: release archive `axiom-0.24.1-macos-arm64.tar.gz`, archive
SHA-256 `188441a0…c74e` matches `SHA256SUMS`; `MANIFEST.sha256` all OK; binary
SHA-256 `a524d2e5f6c1a600030c180a8b814416772932087e23ef2ce0580713f89af9df`
= `release-metadata.txt`; `axiom version` → `0.24.1`, revision `5814b85b711b`,
sourceState clean. Invocations: 12/12 (3 availability probes, 9 Runtime
attempts); 0 retries.

| Scenario | Execution | Result | Evidence |
|---|---|---|---|
| T01 readiness | — | PASS | `project validate`: structure valid, effective ready, both Repositories bound/available |
| E2E-C same Repository, Codex + Claude writers, real `go test` post-merge check (F-3/F-4 retest) | `dadc9e76-a231-48a3-b055-e9c2ab2479fb` (`e2e-same-repo` r1) | PASS | Codex sandbox refused `git add`; writer left files uncommitted; principal recorded `09a9abb` ("axiom: record attempt … changes", 2 files). Serial merges `c05e1b9 → da46422 → 1cc267f`, each `post-merge` `go test -count=1 ./...` passed (exit 0). Reader attempt based on integrated `1cc267f`. Operator re-ran `go vet`/`go test` on a `git archive` of `1cc267f`: ok. Operator checkout untouched. Review gate left closed |
| E2E-F same-file conflict | `08bea680-2153-4142-b468-323152c9cd4e` (`e2e-same-repo` r3) | PASS | First writer integrated (`be451e2`, check passed); second `not_integrated` / `merge_conflict`; target kept at `be451e2`; dependent reader never launched; no replay; branches retained |
| E2E-B parallel Codex ∥ Claude with typed Q→A (F-6 retest) | `32acc430-91e7-46d6-9582-94973b9ba3d9` (`e2e-parallel` r8) | PASS | `cli-q1` question cli-claude→lib-codex (14:52:39Z); `lib-answer-cli-q1` answer, `replyTo: cli-q1` (14:52:59Z); sender attempts equal the recorded attempt IDs; answer matches `FormatLabel` source; integrator quotes the exchange |
| E2E-D one Execution, two Repositories | `32acc430…` | PASS (bounded) | Separate `repositoryOutcomes` (`library`: lib-result; `cli`: result, cli-result, result); no file change needed (contracts already satisfied) |
| Authority negatives | none created | PASS | No `--authorize-commands` → `commands_unauthorized`, next names the flag (F-8 fixed); no `--authorize-external` → `authority_required`; Execution count unchanged |
| After-gate + enrolled dependent | `735d9b2b-1324-4031-a341-8540b5ccd884` (`e2e-same-repo-gate` r1) | FAIL (F-2) | Refusal itself correct (`integration_refused`, no attempt/ref/worktree/spawn), but the Execution persists `ready` and status `next` is `workflow run --execution … --authorize-external`, which can only refuse again (verified twice) |
| Recovery, runner lost | `8d1c1b2c-0c79-4441-b63d-00f8445f5d11` (prior) | PASS (observe only) | `stage reconcile --authorize-local` kept attempt `unknown`, no signal/replay; human abandon not authorized here |
| Distribution | — | PASS | Isolated `install.sh` with pinned verifier: `docs inspect` verified (binary/commit/inventory digests match); installed Claude and Codex skills byte-identical to the release `skills-manifest.txt` |
| F-1 flaky test | — | not reproduced | 6/6 pass; `go vet` + full `go test ./...` on `5814b85`: pass |

Not run in cycle 1 (budget or authority): T04 `default-sdd`, T08 native
initiation, T11 reject/rework and T12 delivery (both need a human gate or
rejection decision, not delegated), cancellation, retry refusal after unknown.

### Findings

| ID | Class | Summary | Disposition |
|---|---|---|---|
| F-2 | BUG_CONFIRMED (minor, recovery guidance) | Refused after-gate Execution advertises a resume that can never succeed; resume hint also omits `--authorize-commands` | Fixed in this branch: `stageexecution.AfterGateIntegrationDependency` (pure check shared by preflight and status) and `executionNext` |
| F-12 | OPTIONAL_IMPROVEMENT | Unknown agent in `--select` reports `recovery_required` "Execution record failed validation" (nothing written); pinned by an existing test | Not changed |
| F-13 | OPTIONAL_IMPROVEMENT | `runtime profile create --help` lists `--enabled`, which is always refused with a precise next step | Not changed |
| F-14 | OPTIONAL_IMPROVEMENT | `workflow run --execution` with `--repository/--work-item` reports `selections_frozen` | Not changed |
| F-15 | OPTIONAL_IMPROVEMENT | After a successful observe-only reconcile, `next` repeats the reconcile step before the abandon step | Not changed |
| F-16 | EVIDENCE_GAP (pre-existing) | Retained artifact contents are not readable through the public CLI; the operator read them read-only from the local artifact store | Recorded |

## Cycle 2 — development build `f21748d`

Build: `scripts/install-axiom.sh` with `AXIOM_BIN_DIR`/`AXIOM_INSTALL_STATE_ROOT`
in an isolated scratch directory; `axiom version` → `development`, revision
`f21748db407c`, sourceState clean; SHA-256
`4de07de30ddc0bf974b0fdfa282099a4a04d0407ca5d1b9861fac225395e7f56`. The release
v0.24.1 extraction and the pre-existing `~/.local/bin/axiom` were left
untouched. Invocations: 10/12 (cumulative 22/40); 0 retries.

| Scenario | Execution | Result | Evidence |
|---|---|---|---|
| F-2 retest | new `46d662ae-8856-4d99-89d8-ca846ab84ba8`; historical `735d9b2b…` and `fa7efdb6…` | PASS | Refusal unchanged (`integration_refused`, no spawn); status `next` now says the Execution can never start and to start a new one with the gate on a later Stage |
| E2E-C regression (preflight touched) | `be72675f-a7c9-4baa-a426-5946586b49ab` | PASS | Codex capture `0cdadf7` by the principal; merges `c05e1b9 → 08c23df → fd2bfe6`, both `go test` checks passed; reader on `fd2bfe6`; operator re-ran `go vet`/`go test` on the archived tree: ok |
| Retry after unknown (no blind replay) | `8d1c1b2c…` | PASS | `stage retry --authorize-external` refused `recovery_required`; still one `unknown` attempt; nothing spawned |
| Cancellation | `866d6476-a3c3-4c0e-a06a-347a8f3f470c` | PASS | `stage cancel --authorize-local` → `cancel_requested`; attempt `cancelled`, `group_stop_confirmed`; process gone; Repository unchanged |
| T08 initiation from Claude Code (installed `axiom-work-item` skill, dev build first on PATH) | `ac2177ea-958e-4a35-97cb-501e87841983` | PASS (bounded) | Natural-language prompt; skill composed `workflow run --workflow minimal` with explicit Profile and `--authorize-external`; `completed`; Repository unchanged. The operator AI could not read the result body (F-16) |
| T08 initiation from Codex | none | BLOCKED (F-17) | See findings. The first attempt resolved `axiom` to the stale `~/.local/bin/axiom` (`f5709fc`) through Codex's login shell, which also cannot read records written by newer builds (ENVIRONMENT_FAILURE, not counted as product Evidence). Deterministic reproduction without inference: under `codex sandbox -P :workspace`, the dev build reads state but every state write (`stage reconcile --authorize-local`) fails |
| T04 built-in `default-sdd` r2 | `0256a738-a045-4814-84e6-a905e77426e1` | PASS (bounded) | intake → specification → clarification succeeded (Claude haiku via Profile); stopped at the `plan` `planning_authority` gate; later stages need human authority |

### Findings added in cycle 2

| ID | Class | Summary | Disposition |
|---|---|---|---|
| F-17 | MATERIAL_DECISION_REQUIRED | Codex-native initiation cannot write Axiom state from Codex `workspace-write` (state root outside writable roots). Options: (a) operator adds the state root as a Codex writable root (sandbox widening, owner decision); (b) relocate or broker state writes (architecture change); (c) document Codex initiation as unsupported until the #387-class sandbox decision. Recommendation: (c) now, decide (a)/(b) with #387 | Not implemented; no sandbox widened |
| F-18 | BUG_CONFIRMED (observability) | A denied state write surfaced as generic `execution_operation_failed`, so the operator AI reported "effects unknown" with no cause | Fixed after the cycle-2 build: `os.ErrPermission` maps to `state_permission_denied` with a sandbox-aware next step; regression test `TestPublicWorkflowRunUnwritableStateNamesPermission`; troubleshooting guide updated. Validated in cycle 3 |
| F-19 | ENVIRONMENT_FAILURE | Stale `~/.local/bin/axiom` (`f5709fc`, older than v0.24.1) is first on login-shell PATH and fails `workflow list` with `recovery_required` on newer records | Preserved; operator decision to replace it |

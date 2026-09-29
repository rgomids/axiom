# T24 against `v0.1.2-rc.2` — preparation checkpoint (2026-09-29)

```text
RC: v0.1.2-rc.2
revision: 859969a07f3807822580431a05b6c78b07691fb1
T24: IN PROGRESS / BLOCKED (no acceptance journey executed)
T25: BLOCKED
human acceptance: PENDING
```

This checkpoint prepares T24 for the corrected RC. It executes no acceptance
journey, no Runtime invocation, no Provider mutation and no installation. It
records read-only observations, harness changes, offline tooling tests and six
authority envelopes (A–F) awaiting separate human authorization. The
[`rc.1` checkpoint](../evidence-s9-t24.md) and [`evidence-s9-rc/`](../evidence-s9-rc/)
are preserved unchanged as history.

## Candidate identity (read-only, this session)

| Fact | Observation | Source |
|---|---|---|
| Tag → commit | `v0.1.2-rc.2` → `859969a07f3807822580431a05b6c78b07691fb1` (lightweight; equals `main`) | [remote-refs.txt](remote-refs.txt), `gh api …/git/ref/tags/v0.1.2-rc.2` |
| Release | ID `399403900`, `draft=false`, `prerelease=true`, `immutable=true` | [release-metadata.json](release-metadata.json) |
| `latest` | `v0.1.1` (ID `398763823`, `prerelease=false`) — stable unchanged | [latest-release.json](latest-release.json) |
| Assets | exactly four; digests equal the expected values below | [release-metadata.json](release-metadata.json) |
| Publication run | [36609926942](https://github.com/rgomids/axiom/actions/runs/36609926942), `success`, head `859969a…` | [publish-run.json](publish-run.json) |
| Official verification | `./scripts/release.sh verify --tag v0.1.2-rc.2 --download` exit 0, `artifacts_verified=pass`, `result=pass` | [release-verification.txt](release-verification.txt) |
| Orphan release `399339376` | `GET /releases/399339376` → 404; [Issue #122](https://github.com/rgomids/axiom/issues/122) closed | this session |
| Remote bootstrap | `scripts/install.sh` SHA-256 `3bf39863…58dc7`, unchanged since `rc.1` | [candidate](candidate-v0.1.2-rc.2.json) |

| Asset | SHA-256 |
|---|---|
| `axiom-0.1.2-rc.2-linux-amd64.tar.gz` | `2e33522fa25c944855457a86dae222237d9293c47f90edd9f347ec7ed1d94dcb` |
| `axiom-0.1.2-rc.2-linux-arm64.tar.gz` | `cc30bc528e2a16fe8b5c55de2a7d6cf924a41d366e37783ee30c0e13819185dc` |
| `axiom-0.1.2-rc.2-macos-27-arm64.tar.gz` | `5a18fbbb6db0c2479758ee8e1d912247f8bdb3f05134609611c79d5e243127bb` |
| `SHA256SUMS` | `372baf0587515c8dcb079066ba0e6dfc92544d1291347aed6217068fe14f82ea` |

Nothing diverged. The reviewed descriptor
[candidate-v0.1.2-rc.2.json](candidate-v0.1.2-rc.2.json) binds these values;
[prior-candidate-v0.1.2-rc.1.json](prior-candidate-v0.1.2-rc.1.json) binds the
published `rc.1` (re-read this session, unchanged) only as the owned-upgrade and
downgrade-refusal precondition.

**Documentation gap (not fixed here):** [tasks.md](../tasks.md),
[evidence-s9.md](../evidence-s9.md) and the
[rc.2 incident record](../evidence-s9-t23-rc2-incident.md) still describe
`rc.2` as BLOCKED/orphan. The successful publication above has no T23 record in
the repository yet; it belongs to the release flow's own Evidence.

## Reconciliation of the `rc.1` checkpoint

| `rc.1` artifact | Disposition for `rc.2` |
|---|---|
| Bounded capture (64 KiB streaming cap, full-stream SHA-256, process-group kill, timeout, interruption ledger) | **Reused** unchanged in behavior; extracted as `bounded_run` so both tools share it |
| Plan-only default, fresh private Evidence dir, envelope digest gate, drift refusal | **Reused** |
| Hardcoded `rc.1` tag/revision/asset digests in the collector | **Replaced** by reviewed candidate descriptors; a test asserts no candidate identity remains in the collector |
| Single stage (install, version, first-run neither, reinstall) | **Extended** to the full local matrix (below) |
| `rc.1` plan/envelope `5807ece1…` and earlier plan dirs | **Superseded, preserved**; cannot authorize the new collector (collector bytes are hash-bound) |
| [coverage.json](../evidence-s9-rc/coverage.json) (190 rows) | **Reusable as the row list**; its candidate binding is `rc.1`, so every row still needs `rc.2` Evidence. Not rewritten |
| T36 lab runner (`axiom-e2e-lab/artifacts/runner/main.go`, SHA-256 `7761179b…c467`, outside the repo) | **Structure reused**: ported into the reviewable [s9-graph-runner.go](../../../../scripts/s9-graph-runner.go); T36 Evidence itself is another revision and is not reused |
| `rc.1` dogfood (Issue #117 / PR #118) | Context only; it proved the `rc.1` defects and does not count for `rc.2` |
| Finding: `test-s7-native.sh` required `ubuntu:26.04:ext4` | **Fixed** (tooling): any Linux on native ext4, distribution recorded, not filtered (Specification platform reconciliation) |
| Finding: primary checkout `.claude` directory fails repository validation | Not re-observed in this worktree; unchanged |

## Harness (maintainer tooling; no product code changed)

- [s9-rc-evidence.py](../../../../scripts/s9-rc-evidence.py) (via
  [test-s9-rc-acceptance.sh](../../../../scripts/test-s9-rc-acceptance.sh)):
  `--version vX.Y.Z-rc.N` exact pin only (floating/channel values refused),
  `--candidate`/`--prior-candidate` descriptors, every process materialized in
  the envelope as ordered steps. Stage A covers: native filesystem fact
  (APFS + case-insensitive / ext4), published release/latest/tag/prior read-back,
  bootstrap digest, clean install with archive/receipt/binary identity,
  provenance, first-run **neither** (+ rerun), idempotent reinstall with
  unchanged inventory, first-run **Codex only / Claude only / both** (+ idempotent
  rerun with unchanged skill digests) under the isolated HOME's default
  user-global roots, fail-closed **foreign / modified / unsafe** installer states
  with unchanged inventory, **owned upgrade** `rc.1 → rc.2` and **downgrade
  refusal**. Runtime executables are placed on a private PATH by symlink for
  detection and never executed (first-run resolves PATH only).
- [s9-rc-envelope.py](../../../../scripts/s9-rc-envelope.py): executes exactly one
  phase of a reviewed envelope when the authorized envelope digest matches, the
  executor/collector bytes match the envelope, and the human supplies exactly
  the parameters that phase declares (preview digests, Issue number, revisions).
  No environment inheritance; credential-shaped names/values refused; one
  automatic attempt; a later attempt requires the failed ledger and is refused
  while any declared external effect is `unknown` (timeout/capture failure).
  Never marks T24/T25 complete; human decision stays PENDING.
- [s9-graph-runner.go](../../../../scripts/s9-graph-runner.go) (`//go:build ignore`):
  the published CLI has no Execution Graph surface (T36 finding 6), so T24 drives
  `internal/graphapplication` exactly as T36 did, built only from a checkout whose
  `cmd/ internal/ go.mod go.sum` equal `859969a` (checked in D1). Modes
  `prepare` / `run` / child-side `question|answer|wait-answer`; persisted Evidence
  is write-once.
- [test-s7-native.sh](../../../../scripts/test-s7-native.sh): Linux rows are
  `linux-<arch>-ext4`; optional `AXIOM_NATIVE_CANDIDATE_REVISION` blocks (exit 78)
  unless the checkout's product tree equals the candidate.

Offline tests (fixtures are tooling tests, never acceptance Evidence):
`python3 -B scripts/test_s9_rc_evidence.py` (17 pass),
`python3 -B scripts/test_s9_rc_envelope.py` (9 pass).

**Graph runner rehearsal (tooling only):** with fake `codex`/`claude` scripts
that call the helper and edit the two allowed files, `prepare` → `run`
completed with coordination, dependency-gated Integration, all ten real combined
validators passing (`go test`, `-race`, `vet`, `build`, `mod verify`, bootstrap
suite, repository, sensitive files, `diff --check`, gitleaks) and
`BuildEvidence`. The product labels that rehearsal `real_run_recorded` because
it cannot tell a fake executable from a Runtime — it is **not** Evidence. It
found and fixed three runner defects before any authorized run: Runtime banner →
version token (as T36), a shell validator refused by S8 (removed), and child
helper failures on momentary store-lock contention (bounded retry on
`local.ErrConflict`).

## Platform matrix

| Row | Status | Notes |
|---|---|---|
| macOS 27.0 / arm64 / APFS | **available** (this host: 27.0 build 26A428, arm64, APFS data volume) | Envelope A ready; B–F planned on this host |
| Linux / amd64 / ext4 | **unavailable** | No Linux host, VM or approved remote endpoint in this environment. VMware Fusion is installed with no VM; on Apple silicon it runs arm64 guests only. No other endpoint was provided for T24. |
| Linux / arm64 / ext4 | **unavailable** | Same; no VM exists. |

Cross-compilation, containers without native ext4 semantics, synthetic rows and
archive inspection are not substitutes. Smallest technically valid mechanisms
(each needs a human decision; none was provisioned):

1. **Linux arm64/ext4:** a local VMware Fusion arm64 Linux VM with an ext4 root
   (native arm64 kernel and filesystem on this Mac). Needs an ISO download and VM
   creation authority.
2. **Linux amd64/ext4:** a native x86_64 host — an operator-owned machine or
   cloud VM, or a GitHub-hosted `ubuntu-24.04` runner (x86_64, ext4 root) via a
   new reviewed `workflow_dispatch` workflow. Emulated amd64 on Apple silicon is
   not native.
3. On each row: run the collector (row-local envelope), `AXIOM_NATIVE_CANDIDATE_REVISION=859969a… ./scripts/test-s7-native.sh`
   from a tooling checkout, and the Runtime/Provider rows the operator chooses
   to cover there. GitHub-hosted runners have no Codex/Claude login; that choice
   is part of the decision.

## Authority envelopes

Digests are SHA-256 of the envelope file bytes. Each provider phase is its own
per-run authority: envelope digest + the exact values shown by the preceding
local phase. The lab is `/Users/rgomids/Projects/axiom-t24-rc2-lab` (outside the
repository; nothing there is committed until sanitized).

| Id | Digest | Purpose | Authority class |
|---|---|---|---|
| A | `cb926b83be06ebe20903c56ab608d555528b043e939a403448773dc909f6a070` | macOS local install/first-run/installer-state matrix | local only |
| B | `ae5f0a2f89bdcd0e6aca0b1f43e6c00c5fa54c3d338b79ebd19c05121846f55c` | real Codex invocation through installed rc.2 skills | local + Runtime |
| C | `ce6efbd8f7ff65b044af2520e7af049cc37328596bbd9e23ab6b66d16944e7ea` | real Claude invocation through installed rc.2 skills | local + Runtime |
| D | `4f18e034d45d8f29bee713b297d6ba2a5b091f6e528c2a29811681be5a7bdcc1` | Codex + Claude S8 graph, Integration/Reconciliation, Evidence | local + Runtime |
| E | `986dc44fb61109ff07db78c20b2a62884068d19317af9e219b638ec4735a6370` | GitHub Work Item, projections, branch + PR (dogfood) | local + Provider |
| F | `ea9061f1fd58a81c7a9b5088d7cdaedc549d5e9ca80eecb324a46755b21a65ee` | interruption/resume, projection drift/recovery, failure suites | local + Provider (F2) |

Files: [A](envelopes/A.json) (identical bytes to the plan at
`<lab>/evidence/A-macos-arm64/envelope.json`), [B](envelopes/B.json),
[C](envelopes/C.json), [D](envelopes/D.json) with
[graph spec](envelopes/D-graph-spec.json) (SHA-256 `83ed14ac…693b`, checked by
D1), [E](envelopes/E.json), [F](envelopes/F.json).

Execution order: **A → B → C → D1, D2 → E1–E8 → D3, D4 → E9–E11 → F**.
Grouping is limited to steps with the same authority class inside one phase
(for example E9 commits locally, pushes one new branch and opens one PR: a PR
cannot exist without its branch, and a failed PR creation leaves only a branch
the human deletes).

Dogfood activity (E/D): **retire the obsolete synthetic Ubuntu row** from
`scripts/test-install-bootstrap.sh` (Codex child) and its paragraph in
`docs/commands.md` (Claude child). The installer has no distribution filter
since the 2026-09-28 reconciliation, so that mount-namespace row no longer
exercises any installer decision. Small, test-tooling/docs only, verifiable by
the bootstrap suite plus the combined validators, disjoint files, real
question/answer coordination. The human may substitute another activity; that
changes D/E digests.

How an authorized envelope runs (only after an explicit human yes for that
digest; `$EV` is this directory, `$LAB` the lab):

```bash
./scripts/test-s9-rc-acceptance.sh --version v0.1.2-rc.2 \
  --candidate "$EV/candidate-v0.1.2-rc.2.json" --prior-candidate "$EV/prior-candidate-v0.1.2-rc.1.json" \
  --evidence-dir "$LAB/evidence/A-macos-arm64" --execute-install \
  --approved-envelope-sha256 cb926b83be06ebe20903c56ab608d555528b043e939a403448773dc909f6a070

python3 scripts/s9-rc-envelope.py --envelope "$EV/envelopes/B.json" \
  --candidate "$EV/candidate-v0.1.2-rc.2.json" --phase B1-setup \
  --approved-envelope-sha256 <B digest> --evidence-dir "$LAB/evidence/B-B1-setup-attempt-1"
```

`--plan` prints a phase's steps and the approvals it needs without running
anything. The Envelope A plan directory must stay untouched (three private
files) until execution; Runtime upgrades change its bound executable digests
and require a fresh plan.

Hypotheses recorded as stop conditions (never fallbacks):
H-B1 Codex finds user skills in `$HOME/.agents/skills` with an isolated HOME and
the existing `CODEX_HOME`; H-C1 Claude authenticates with an isolated
`CLAUDE_CONFIG_DIR`; H-C2 Claude loads user skills without T36's `--safe-mode`.

## Findings

- **Decision (scope):** the S8 graph is reachable only through internal Go
  packages, not the published `axiom` binary. D exercises the exact RC product
  tree through a source-built runner (as T36). Human confirmation that this
  satisfies "against the exact RC" for AC-32–AC-43/AC-49 is required.
- **Blocker (environment):** Linux amd64/ext4 and arm64/ext4 unavailable.
- **Triage (Minor, not on the acceptance path):** at the exact tree,
  `cmd/lingo` `TestConfigureReportsCommittedPortableStateWhenLocalPublicationFails`
  passes with `TMPDIR` of ≤146 characters and fails deterministically at ≥150
  on this host (reports `application_unavailable` instead of `partial`). Not
  investigated further; the lab uses short paths. It may be a test-fixture limit
  or a truthfulness gap for long state roots; needs human triage before T25.
- **Risk:** product stores take non-blocking locks and fail closed with
  `project conflict` on contention; the runner's child helper retries, but the
  scheduler's own persistence does not. A real run could surface this as a
  failed attempt; it would be recorded, not retried automatically.
- **Documentation gap:** `rc.2` publication not yet reconciled in tasks/S9/incident records.

## Deterministic validation (this worktree, exact product tree + tooling changes)

See [checks](checks.md). No command was run against host Axiom roots,
host skill roots or GitHub state other than the reads listed above.

## Next

Human reviews and authorizes the first bounded envelope (A). Provide or approve
a mechanism for both Linux rows.

# Evidence — MVP Slice S7: Compatibility, Recovery, Cleanup and Upgrade Hardening

## Claim and authority boundary

This record covers Specification 004 Slice S7 only: T16–T22, tracked by
[Issue #80](https://github.com/rgomids/axiom/issues/80). S7 was explicitly
authorized on 2026-09-25 (recorded on Issue #80). The implementation is commit
`11f2b7decbe4ddef428d4cb3b7e962680100e1db` plus the upgrade-resume fix
`408a2b51f20e797744f9f6ba6eaa00e0061582f9` (tree
`607eb25e94660b1be786c46b2b120fb002ea7d64`) on branch
`feat/80-s7-compatibility-recovery`, based on `main` at `8971255` (S6 merge,
PR #96). The final Evidence runs executed with `HEAD` at `408a2b5`; the only
tracked differences were the uncommitted status and Evidence documents recorded
in the following documentation commit, so the exercised code equals `408a2b5`.

Every destructive, recovery, and upgrade case ran against isolated temporary
roots created by the tests or scripts. No real user state, installed binary,
Codex skill root, or Provider was mutated. No GitHub mutation, prerelease,
release, merge, Issue closure, or human acceptance occurred. Passing tests, CI,
commits, or PR state do not constitute human acceptance.

**Result:** T16–T21 are technically complete with the Evidence below. T22 has
passing native Evidence for the macOS 27.0/arm64/APFS row only. The Ubuntu
26.04 amd64/ext4 and arm64/ext4 rows were **not executed** and remain explicit
blockers for T22 completion.

## Handoff recovery

The work continued an interrupted Codex session. The handoff state was
untracked files only, on `main`, with no tracked-file changes, no CLI wiring,
no documentation, and no Evidence. It compiled and its own tests passed, but
executable probes against real formats showed that the core behavior was wrong:

| Handoff defect | Consequence | Resolution |
|---|---|---|
| `HistoricalPOCRevision` named a commit absent from the repository | POC provenance unverifiable | Pinned to tag `v0.1.0-poc.1` = `242d67c4cf2d4c3efe534dd894cb56a05558e139` |
| POC skill digests came from pre-tag commit `5961f7f` (#33), not the tag | A real POC skill set was not recognized | Skills are classified through the v1 runtime's own legacy table |
| Classifier expected `formatVersion: 1` in `axiom.yaml`; the real manifest uses `schemaVersion: 1` | Every real v1 Project root was `malformed` | Real `manifest.Decode` is used |
| POC and v1 Work Item and installation formats are byte-identical; any Work Item was a "POC signature" | Valid v1 state with a Work Item was `malformed`, blocking upgrade | POC recognition requires the POC-only `workflows/` record; shared formats are v1-readable |
| Any non-Axiom skill in the shared skill root made state `malformed` | Upgrade blocked for users with other skills | Only Axiom-owned skill names are inspected |
| Recovery accepted an arbitrary operator path | Mutation not confined to owned roots | Recovery enumerates only owned protocol directories |
| Cleanup references were caller-supplied; Evidence became eligible 365 days after creation | Non-authoritative eligibility; early Evidence removal possible | Execution references read under the state-root lock; Evidence preserved (see limitations) |
| Upgrade had no archive verification, CLI, or interruption marker, and used a lock distinct from the installer | Not usable; installer/upgrade not mutually exclusive | Installer-equivalent verification, shared lock and marker |

The package boundaries, preview/authorize/apply shape, exact-digest authority,
ordered upgrade ledger, cleanup revalidation, and test seams were preserved.
No S8 multi-agent or multi-runtime concept was present or introduced.

## Environment and identity

| Fact | Observation |
|---|---|
| Repository | `rgomids/axiom` |
| Branch | `feat/80-s7-compatibility-recovery` |
| Implementation commits | `11f2b7decbe4ddef428d4cb3b7e962680100e1db`, `408a2b51f20e797744f9f6ba6eaa00e0061582f9` |
| Exercised code | `408a2b5` (tracked differences: documentation only; one untracked operator-owned `CLAUDE.md`, not part of S7) |
| Go | `go1.26.1 darwin/arm64` |
| macOS | 27.0, build `26A428`, kernel `27.0.0` |
| Architecture | `arm64` |
| Filesystem | APFS, case-insensitive (probed) |
| Hard-link primitive | available (probed) |
| Gitleaks | 8.30.1 |

Content digests at `408a2b5`:

| Artifact | SHA-256 |
|---|---|
| `internal/compatibility/classification.go` | `b0623ad72327160c8b80aa0af57f08cb9c6ab86d8c3c1c75ea18ee0d5397fca7` |
| `internal/compatibility/transfer.go` | `03d0377455b1d54a67c1df3ad848a22649b59d67e41b7b6781a969916abced76` |
| `internal/local/inventory.go` | `eefdd290c0863935c0ad7eb8fba052026c51afa6ba68261f22fab4b6b27c9e7a` |
| `internal/local/artifact_cleanup.go` | `d37b33e6050a283fbf42615bf09539d60a838d1fc6091c3b4d370b51ccc2374b` |
| `internal/local/recovery.go` | `97b38004b3dc6948678918a6ec13fa90cd58d4826c745a97e7e28820e73e1f82` |
| `internal/detailartifact/cleanup.go` | `0a91bbe691a7b11c247626b6056caa836395350a2e0fbb4ae0436349f1b396df` |
| `internal/install/upgrade.go` | `c807d29bd29f596efd3a63176ba31d6b4ded0eaf0629bdedd16c93f02692521f` |
| `internal/install/archive.go` | `87898a3d6f8dccc719af4a64009b1a09b471a4076e0794d3470783148c8e4fef` |
| `internal/cli/maintenance.go` | `f2a335c1e9cafb971665f297926f59b4fac496fc3bb4a873bbbda05493139fcd` |
| `cmd/lingo/maintenance.go` | `76ad198b7c69302f1df5fb2c186599a968bb8aa9cddf5bc8719ebb9267e81b34` |
| `scripts/test-s7-security.sh` | `7977f714d5bb45ad92d0f473d3f23ff8c3b67dc273f29984af044b95a8d3c933` |
| `scripts/test-s7-native.sh` | `3004e79a34b12e8721f043d18e56ad0551098338478b4d0638cda54ad6826354` |
| `scripts/generate-poc-fixture.sh` | `88c8ebd5d0d99ebcfc8e3f4450e3cba9b3c8cd693bfbfbebfa9236ce791b0ad3` |

## T16 — Compatibility inspection and POC classification

`lingo compatibility inspect` is strictly read-only: no root creation, no lock,
no write. Directories are opened as anchored private roots; ownership, mode,
ACL, link count, and file type are checked before any read, and non-regular
entries (FIFOs, devices) are never opened. Every file is decoded by the real v1
decoder, or by the frozen historical POC workflow validator reproduced from the
tag's `internal/local/workflow_store.go`. The report is content-free: categories,
kinds, owned relative names, and digests, with at most 32 findings.

**Fixture provenance.** `scripts/generate-poc-fixture.sh` extracts
`v0.1.0-poc.1` with `git archive`, builds it, and drives the historical binary
through `runtime codex install`, `project configure`, `work-item create`,
`workflow start`, and two `workflow advance` gates, using fake `git`/`gh`
executables and no network. Only the machine-specific temporary prefix is
normalized to `/axiom-poc-fixture`.

| Fixture object | SHA-256 |
|---|---|
| `projects/poc-fixture/axiom.yaml` | `e585ed925ae8ec296abc0433b2867c732e9508108fd1ae0faf5e8f3f03a7947a` |
| `state/projects/<id>/installation.json` | `11b3e3b288c07b5a3d8a6b80dbdd863338e9c638dc537bab8d266fc763e617f4` |
| `state/work-items/<id>/main-7.json` | `2027ac1f17125148754be09dd8e0b099ea678c752b8430b28142ad3ff7b5cc24` |
| `state/workflows/<id>/main-7.json` (POC signature) | `4b589983dc623e33a68e384f331d313c7bb20906ccc1ebc5e3e2bc114adadaf0` |
| five POC skills | tag digests `b9d55306…`, `a80b3038…`, `6750abfe…`, `a75f2168…`, `4fbb6fda…` |

**Classification matrix** (`TestClassificationMatrix`,
`TestRecognizedPOCFromHistoricalBinaryOutput`, `TestUnsafeFilesystemFactsFailClosed`):

| Case | Classification / reason |
|---|---|
| Historical POC binary output | `recognized_poc` / `complete_poc_workflow_signature`, tag and revision reported |
| Absent roots; empty private roots | `absent_v1` |
| Shared formats without POC workflow; Work Item only; v1 artifact | `valid_v1` |
| POC workflow plus v1-only artifact | `malformed` / `mixed_poc_and_v1_state` |
| POC workflow with an invalid gate (partial signature) | `malformed` |
| Unknown entry; foreign Axiom skill | `malformed` |
| Unrelated third-party skill in the shared root | ignored (`valid_v1`) |
| `formatVersion: 2`; `executions/v2`; `schemaVersion: 2` | `unsupported_newer` |
| `formatVersion: 0` | `unsupported_older` |
| `.axiom-recovery-*` alongside a newer record; S2 `.lingo-attempt-*` | `recovery_required` (recovery wins) |
| Symlink, `0644` file, `0755` directory or root, hard link, oversized record, FIFO, permissive macOS ACL | `malformed` |

Zero-write: tree hashes before and after inspection are equal for the POC tree
and for every unsafe case. The inspection digest is deterministic across runs.
A content sentinel placed in a recovery marker never appears in the report.
Relative, `/`, and empty-string roots are refused before any read.

## T17 — Authorized POC backup, export, and reconfiguration

`lingo compatibility backup|export --target <absent path>` previews the source
digest, per-object source/target/digest/bytes, omitted categories, required
and observed free space, and a preview digest. Backup and export produce
distinct digests, and each authority is bound to its own kind, target, and
effect set (`TestBackupAndExportUseSeparateExactAuthorities`).

| Case | Observation |
|---|---|
| Backup of the POC fixture | 9 objects plus `manifest.json`; every file `0600`, directory `0700`, no links |
| Export | 1 object (`projects/poc-fixture/axiom.yaml`); 8 omitted; no `state/` |
| Post-export v1 validation | `local.PortableStore.Inspect` reads the exported Project with its original identity |
| Source after both transfers | tree hash unchanged |
| Backup authority applied to export | denied (`ErrTransferStale`) |
| Equivalent retry on a complete target | `complete` no-op; no second authority |
| Occupied target; relative target; `/`; target inside a source root; source inside target; hidden basename | refused at preview; occupied content unchanged |
| Valid v1 source | refused (`ErrTransferSource`) |
| Source changed after authorization | denied; target never created |
| Observed free space below requirement | refused before any write |
| `ENOSPC` injected at the third object | `partial`, exactly two objects reported, no manifest, source unchanged; the incomplete target is never adopted; retry into a new target completes |

Cross-filesystem targets are refused by comparing device identity with every
present source root. Reconfiguration is explicit: the export next-action points
`LINGO_PROJECTS_ROOT` at `<target>/projects` with a separate clean
`LINGO_STATE_ROOT`. No POC workflow or Execution history is converted.

## T18 — Reference-aware artifact cleanup and capacity recovery

Eligibility is computed under the state-root lock, which excludes every store
writer, from authoritative facts only:
- transition references in every Execution record under `executions/v1`;
- artifacts owned by a non-completed Execution;
- artifact-to-artifact and supersession references;
- metadata live references.

Any unreadable, corrupt, or pending Execution makes reference state uncertain,
and preview fails closed (`TestArtifactCleanupFailsClosedOnUncertainReferenceState`).

| Class / fact (fake clock) | Outcome |
|---|---|
| `diagnostic`, unreferenced, 31 days and exactly 30 days | eligible |
| `diagnostic`, 30 days minus 1 second | preserved `diagnostic_within_30_days` |
| `active`; `preserved_review` | preserved |
| `evidence`, 400 days | preserved `evidence_retirement_unrecorded` |
| metadata live reference; cross-artifact reference; Execution transition reference; artifact of an interrupted Execution | preserved `referenced` |
| confirmed cleanup record, 90 days minus 1 minute / plus 1 minute | preserved / eligible |

- Authority binds the exact effect and preserved sets, not the raw clock.
- A missing, stale, or replayed digest, a concurrent root lock, or a newly
  committed Execution reference each deny authority, with the artifact
  preserved.
- The audit record is published through the ADR-0007 protocol before any
  removal; confirmed records are at most 64 KiB and never contain artifact
  payload.
- An injected interruption after the first of two removals is `partial`: the
  record lists exactly the removed identity.
- Batches are capped at 128 effects, and the remainder is reported.
- Capacity exhaustion never evicts; explicit cleanup then succeeds.
- Content outside the artifact root is unchanged.

**Measured cost** (`BenchmarkMaintenanceScanCost`, added with this record,
1,000 live artifacts, native row, measured on the exercised source): cleanup preview ≈157 ms/op (23 MB allocated); state inventory ≈131 ms/op.
Cost is linear in object count, so roughly 1.6 s is expected at the
10,000-artifact guardrail. No retention default was changed.

## T19 — Guided local recovery

`lingo recovery inspect` enumerates only owned protocol directories: state
`projects/<id>`, `work-items/<id>`, `executions/v1/<id>`,
`artifacts/v1/objects/<shard>`, `artifacts/v1/cleanup`, and the portable root
and `<slug>` directories. It takes the owning stores' top-down directory locks.
`recovery apply` re-inspects the one directory under exclusive locks and acts
only if the fresh plan digest is identical.

| Interrupted state (real `publishFile` or store fault) | Plan | After apply |
|---|---|---|
| Update at F3, F4, F5 | `restore_prior` | prior bytes canonical; stage and marker removed |
| Create at F3, F5 | `restore_prior` | no canonical object |
| Update at F6, F7, F8; create at F6 | `finalize_committed` | new bytes canonical; marker removed |
| Pre-commit stage already removed, marker left | `restore_prior` | marker removed |
| Artifact directory create at F3, F5 / F6 | `restore_prior` / `finalize_committed` | absent / readable |
| Portable Project create before commit | `restore_prior` | projects root empty |
| Committed marker with a third canonical revision; corrupt marker; second marker; unknown protocol object; staging replaced by a symlink; S2 attempt marker | `preserved_review` | unauthorizable; directory bytes unchanged |

- A second inspection after every apply finds nothing.
- A concurrent writer lock and a changed stage file both deny authority with
  zero effects.
- Marker objects must be plain non-hidden names.
- Staging names must carry a store-owned prefix.
- Directory generations are removed only by their known file names; nothing is
  deleted recursively.
- A committed generation is never rolled back.

## T20 — Owned install upgrade and resumable partial state

`lingo upgrade` verifies the candidate exactly as `install-release.sh` does
before anything else: the single exact `SHA256SUMS` entry; a single bundle
root; regular files only; a complete `MANIFEST.sha256`; closed release metadata;
and the approved skill manifest. It then previews the source and target
versions, archive digest, ordered effects, state classification, skill
compatibility, and free space.

| Case (`internal/install`) | Outcome |
|---|---|
| Checksum mismatch / missing / duplicate entry | refused |
| Symlink entry; unlisted file; dirty release metadata; shell metacharacters in version; skewed skill manifest | refused; nothing executed |
| 1.0.0 → 1.1.0 | ledger: binary then receipt, each re-read and confirmed; `installedAt` preserved; modes `0700`/`0600`; lock and marker removed |
| Equivalent candidate after upgrade | no effects; no authority issued |
| Installed skills not matching the candidate | `partial` with next action `lingo runtime codex install` |
| Interruption after binary publication | `partial` with ledger `[binary]`; marker `stage=binary_committed` with the exact archive digest; a different archive is `recovery_required`; the same archive resumes with the receipt effect only |
| Interruption after both effects, before marker removal | resumed preview has no publication effect but is authorizable to clear only the owned marker; afterwards a plain no-op (fixed in `408a2b5` after the first Evidence run exposed the stuck state) |
| Downgrade; divergent same version; modified binary; unsupported host row; concurrent lock; installer marker; hard-linked binary; symlinked receipt; POC state; permissive target | refused before any effect; owned tree unchanged |
| Receipt changed after review; insufficient space | denied / refused with zero effects |
| SemVer precedence including pre-release and build metadata | verified |

The upgrade shares the installer's `.axiom-install.lock` and
`.axiom-install-operation` names, so each refuses the other's interrupted state.
There is no cross-root transaction, automatic rollback, downgrade, or automatic
update.

## T21 — Cross-slice security and bounded-I/O regression

`scripts/test-s7-security.sh` names exact tests per requirement alias and fails
when a name no longer exists, so an empty match cannot pass silently. The run
at `408a2b5` reported `failures=0 result=pass`:

| Requirement aliases | Owning tests (package) |
|---|---|
| MVP-SEC-01, SEC-002, SEC-004 | transfer, cleanup, recovery, upgrade authority (`compatibility`, `local`, `install`); `projectapp`, `workitem`, `workflow` authority and stale-revision tests |
| MVP-SEC-02, SEC-001 | `manifest` structural secret exclusion; `detailartifact` sanitization and credential URL rejection; `workitem` sensitive-text rejection; `cli` rejected-input non-exposure; compatibility content sentinel |
| MVP-SEC-03, MVP-SEC-05 | `githubissues` stdin-only untrusted body and authorship; black-box strict flags and shell-metacharacter targets kept as data; selector ambiguity before fallback |
| MVP-SEC-04, MVP-NFR-03 | `githubissues` bounded timeout/output, strict response, ambiguous create |
| MVP-SEC-06, MVP-SEC-08, SEC-005 | compatibility unsafe-fact matrix; `codexruntime` symlink, hard-link, ACL, and permission refusals; composition symlink and overlap refusals |
| MVP-SEC-07, FR-030 | export allowlist and absent `state/` |
| MVP-SEC-09, FR-025, FR-033 | cleanup eligibility, capacity, retention, and partial tests; recovery F-stage, preserved, and directory tests |
| MVP-NFR-01 | completion status classification matrix |
| MVP-NFR-03 | manifest byte/depth/node limits; inventory entry bound |
| FR-023, SEC-004 | upgrade interruption and resume; skill-incompatible partial; Work Item confirmed-effect partial |
| FR-036 | historical POC skill set is known legacy and upgraded by the authorized flow |

Also in that run: `go test -race` on the S7 packages passed;
`check-sensitive-files.sh` passed; `gitleaks detect --source . --no-git`
found no leaks.

**Review findings.** A security and correctness self-review of the S7 diff
found no open blocker, critical, or major issue. Items fixed before the final
Evidence run:
- the upgrade marker rewrite now uses `O_EXCL|O_NOFOLLOW`;
- recovery accepts only store-owned staging prefixes;
- an upgrade interrupted after both effects but before marker removal can now
  be finalized. Previously it left the installer permanently
  `recovery_required`. Scanner success is
supplemental, not proof of absence.

## T22 — Exact-target native Evidence

`scripts/test-s7-native.sh` detects the exact row, probes case behavior and
hard links, runs named test groups, then drives real archives through the
published installer and the owned upgrade on the native filesystem. Result on
`macos-27.0-arm64-apfs` at `408a2b5`: 28 passing steps, `failures=0 result=pass`.

| Step | Result |
|---|---|
| F0–F8 publication; two-process barriers; traversal/link/replacement; ownership/mode/ACL | pass |
| Artifact capacity/cleanup; guided recovery; compatibility/transfer; upgrade units; race | pass |
| Existing release-archive suite | pass |
| Build 1.0.0 and 1.1.0 archives (development, see limitations) | pass |
| Clean install; equivalent reinstall `unchanged`; installer refuses owned upgrade | pass |
| Upgrade preview read-only; stale digest `denied_authority`; apply; receipt version and binary digest verified | pass |
| Equivalent upgrade no-op; installer accepts the upgraded receipt as `unchanged`; downgrade refused | pass |
| Native resume: binary committed, receipt pending; installer refuses; preview shows receipt only; apply succeeds; marker cleared | pass |

| Row | Status |
|---|---|
| macOS 27.0 / arm64 / APFS (case-insensitive) | **pass** at `11f2b7d` and again at `408a2b5` |
| Ubuntu 26.04 / amd64 / ext4 | **not executed**: no environment available; release blocker |
| Ubuntu 26.04 / arm64 / ext4 | **not executed**: no environment available; release blocker |

Cross-compilation is not substituted for native Evidence. The existing
`macos-15`/`ubuntu-24.04` workflow remains historical and was not changed;
adding exact-target runners is a separately reviewed workflow change.

## Discovered pre-existing defects fixed in S7

1. **POC skill legacy omission (S3).** When `axiom-work-item-create` changed in
   `e0bd2f1`, the replaced tag digest `6750abfe…` was not added to the legacy
   table, while the other four tag digests were. `lingo runtime codex install`
   therefore refused a real POC skill set as foreign. The digest was added, with
   `TestHistoricalPOCSkillSetIsKnownLegacy` proving detection and an authorized
   upgrade to `current`.
2. **AppleDouble entries in macOS-built archives (S2).** On macOS 27, `tar`
   embedded `._*` entries for the `com.apple.provenance` attribute. They are
   hidden by `bsdtar -t`, absent from `MANIFEST.sha256`, and would be extracted
   as files by GNU tar on Ubuntu, failing the installer there.
   `build-release-archives.sh` now sets `COPYFILE_DISABLE=1`; the strict Go
   archive reader rejects such entries.

## Limitations and unexecuted cases

- **T22 Ubuntu rows were not executed.** They remain blocking for T22 and for
  release; no Ubuntu behavior is claimed.
- **Development archives.** Native archives were built with `--development`
  because the untracked operator file makes the checkout dirty for the release
  builder (`release=false`, development provenance). Clean release-mode
  archives remain RC Evidence work.
- **Evidence retention.** Evidence-class artifacts are never age-eligible:
  metadata format 1 records no reference-retirement time, so the Plan's "365
  days after retirement" cannot be computed without risking early removal.
  Enabling it needs a recorded retirement time, which is a persisted-format
  change requiring a reviewed decision.
- **Interrupted S2 attempt markers** (`.lingo-install-*`,
  `.lingo-attempt-*`) have no prior/new generation facts and are always
  `preserved_review`.
- **Skill upgrade.** `lingo upgrade` does not write skill files. It reports
  skill compatibility, and the known-legacy skill set is upgraded by the
  existing authorized `lingo runtime codex install` of the new binary.
- **Stale upgrade lock.** A process killed while holding
  `.axiom-install.lock` leaves it in place, as the installer does; both report
  busy until an operator confirms no operation is running.
- **Skill manifest version skew.** The release skill manifest declares
  `skillSetVersion=1`/`binaryCompatibility=1` while the runtime receipt uses
  version 2. This is pre-existing and was observed but not changed.
- **Excluded by the approved threat model:** arbitrary malicious same-UID
  interleavings, physical power loss, and media durability.
- **Repository validators and `CLAUDE.md`.** At `408a2b5`,
  `validate-repository.sh` and `validate-agent-package.sh` failed only on the
  untracked operator `CLAUDE.md`. After the operator asked to commit it, both
  validators were changed to allow only a root `CLAUDE.md` whose entire content
  is `@AGENTS.md`. Any other Claude artifact is still rejected.

## Commands and results

| Command | Result |
|---|---|
| `go build ./...`, `go vet ./...`, `go mod verify` | pass |
| `go test ./... -count=1` | pass (all packages) |
| `go test -race` (compatibility, install, local, detailartifact, codexruntime, cli, cmd/lingo) | pass |
| `./scripts/test-s7-security.sh` | `failures=0 result=pass` |
| `./scripts/test-s7-native.sh` | `native_row=macos-27.0-arm64-apfs failures=0 result=pass` |
| `./scripts/check-sensitive-files.sh --staged .`, `gitleaks` (staged and worktree) | pass / no leaks |
| `./scripts/validate-repository.sh .`, `./scripts/validate-agent-package.sh .` | at `408a2b5`: fail only on the untracked operator `CLAUDE.md`, pass without it; after the pointer allowance: pass |
| `git diff --check` | clean |

Human acceptance of S7 is not inferred from this record.

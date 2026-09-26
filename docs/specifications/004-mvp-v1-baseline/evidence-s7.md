# Evidence — MVP Slice S7: Compatibility, Recovery, Cleanup and Upgrade Hardening

## Claim and authority boundary

This record covers Specification 004 Slice S7 only: T16–T22, tracked by
[Issue #80](https://github.com/rgomids/axiom/issues/80). S7 was explicitly
authorized on 2026-09-25 (recorded on Issue #80). The implementation is on
branch `feat/80-s7-compatibility-recovery`, based on `main` at `8971255` (S6
merge, PR #96):

| Commit | Content |
|---|---|
| `11f2b7decbe4ddef428d4cb3b7e962680100e1db` | S7 delivery (T16–T22) |
| `408a2b51f20e797744f9f6ba6eaa00e0061582f9` | upgrade resume after all effects |
| `3eb4427` | maintainer-runtime bootstrap contract and closed `CLAUDE.md` validator (review follow-up) |
| `b69a93fa66d3aa0bb99c8e77541ed3c240ed503a` | T20: skill files published inside `lingo upgrade` (review follow-up) |
| `d76a3b3`, `604bd10c29c03606fa948f18937da5c413d51c50` | native-suite script defects exposed by a clean checkout |

**The current Evidence was executed at `604bd10` (tree
`b9d53a266fdc8dd036623ed162faf13a3243fad2`) with a clean worktree and no
untracked entries.** The commit that adds this record changes documentation
only. Results recorded at `408a2b5` are kept below as history only; they do
not cover later code.

Every destructive, recovery, and upgrade case ran against isolated temporary
roots created by the tests or scripts. No real user state, installed binary,
Codex skill root, or Provider was mutated. The only GitHub mutations were
explicitly authorized updates to PR #100 (branch push, body, and metadata). No
prerelease, release, merge, Issue closure, or human acceptance occurred. Passing tests, CI,
commits, or PR state do not constitute human acceptance.

**Result at `604bd10`:**

| Task | State |
|---|---|
| T16, T17, T19, T21 | technically complete |
| T18 | technically complete except the 365-day Evidence boundary: **Human decision required** (see [Human decisions required](#human-decisions-required)) |
| T20 | technically complete for the Plan ordering (binary, receipt, skill files, verification); the Codex skill-set receipt refresh needs a release-format decision (**Human decision required**) |
| T22 | native pass for macOS 27.0/arm64/APFS only; the Ubuntu 26.04 amd64/ext4 and arm64/ext4 rows were **not executed** and block T22 |

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
| Implementation commits | see the table above |
| Exercised code | `604bd10`, clean worktree, `untracked_entries=0` |
| Go | `go1.26.1 darwin/arm64` |
| macOS | 27.0, build `26A428`, kernel `27.0.0` |
| Architecture | `arm64` |
| Filesystem | APFS, case-insensitive (probed) |
| Hard-link primitive | available (probed) |
| Gitleaks | 8.30.1 |

Content digests at `604bd10` (unchanged files keep their `408a2b5` digest):

| Artifact | SHA-256 |
|---|---|
| `internal/compatibility/classification.go` | `b0623ad72327160c8b80aa0af57f08cb9c6ab86d8c3c1c75ea18ee0d5397fca7` |
| `internal/compatibility/transfer.go` | `03d0377455b1d54a67c1df3ad848a22649b59d67e41b7b6781a969916abced76` |
| `internal/local/inventory.go` | `eefdd290c0863935c0ad7eb8fba052026c51afa6ba68261f22fab4b6b27c9e7a` |
| `internal/local/artifact_cleanup.go` | `d37b33e6050a283fbf42615bf09539d60a838d1fc6091c3b4d370b51ccc2374b` |
| `internal/local/recovery.go` | `97b38004b3dc6948678918a6ec13fa90cd58d4826c745a97e7e28820e73e1f82` |
| `internal/detailartifact/cleanup.go` | `0a91bbe691a7b11c247626b6056caa836395350a2e0fbb4ae0436349f1b396df` |
| `internal/install/upgrade.go` | `71712db48b2aaee31cb61fe37b479e2b3ee5b20c5c2e07b10f38c840e03488b7` |
| `internal/codexruntime/upgrade.go` | `9f031ad8e3200f4a64d179083142ead843b772f94a30e409d0bfabdf62d926d6` |
| `internal/install/archive.go` | `96aa5482476bcd76196847cac7cb2ebf5600db7cf728e662c633e2fffe1b1184` |
| `internal/cli/maintenance.go` | `f2a335c1e9cafb971665f297926f59b4fac496fc3bb4a873bbbda05493139fcd` |
| `cmd/lingo/maintenance.go` | `9089019482db59e4e232d1e815c90bc200ac31e3af6f1f11b6bbb27be3756760` |
| `scripts/test-s7-security.sh` | `5b6aa63e7248aa852e9a3bddf1731dda5383c36e415c927e457613b8ebe71f6d` |
| `scripts/test-s7-native.sh` | `eef9bb929140a515a242940e5f0e04dc8480006a8dcfb43ec8f1f49301fb80ed` |
| `scripts/check-claude-bootstrap.sh` | `3cca0c2ef943160095d02eafe7ae1ece43631df23936eb032d31fd77fb8942ac` |
| `CLAUDE.md` | `336cc4fbf19beaada7ccf9986414fa91851a8d7a07dfb3ccbe800a69eed0ab49` (`@AGENTS.md` + newline) |
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
| `evidence`, 400 days, no reference | preserved `evidence_retirement_unrecorded` (the 365-day boundary is not implementable; see [Human decisions required](#human-decisions-required)) |
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
versions, archive digest, ordered effects, state classification, skill state,
and free space in both the binary directory and the Codex skill root.

**Ordering (Plan §12).** Read-only preflight → exact preview → authority bound
to the preview digest → re-validation under the installer lock and the Codex
skill-set lock → binary (published, re-read) → installation receipt → each
Axiom skill file with its expected digest → compatibility and skill
verification. Each effect is confirmed individually; there is no cross-root
transaction.

**Skill ownership.** A present Axiom skill is replaced only when one of these
holds:
- the whole installed set reproduces the installation receipt's
  `skillManifestSha256`;
- the running binary knows the content as its own or as a legacy digest;
- a resumed operation recorded the skill's expected revision in the shared
  `.axiom-install-operation` marker before its first effect.

A missing skill in a configured set is created. Modified, foreign, unsafe, or
unexpected entries refuse before any effect.

| Case (`internal/install`, `internal/codexruntime`) | Outcome |
|---|---|
| Checksum mismatch / missing / duplicate entry | refused |
| Symlink entry; unlisted file; dirty release metadata; shell metacharacters in version; skewed skill manifest | refused; nothing executed |
| 1.0.0 → 1.1.0 with skills already equal to the candidate | ledger binary, receipt; `success`, `skillReceipt=unchanged`; `installedAt` preserved; modes `0700`/`0600`; lock and marker removed |
| 1.0.0 → 1.1.0 with the owned 1.0.0 skill set | ledger binary, receipt, five skills in order, each confirmed; skill files equal the candidate (`0600` in `0700` directories); `partial` with `skillReceipt=refresh_required`; next preview is a no-op |
| Known-legacy (historical POC) skills, one skill missing | four replaced, one created (`expected=absent`) |
| Modified skill; unexpected entry; `0644` skill; symlinked skill directory; interrupted runtime install stage | `skill_conflict`; owned tree unchanged |
| Upgrade staging leftover without an operation marker | `recovery_required` |
| Codex skill-set lock held by a concurrent install | `skill_set_busy_or_interrupted`; zero effects |
| Skill changed after review | `authority_denied`; zero effects |
| Interruption after each of the 7 ordered effects | `partial`; ledger lists exactly the confirmed prefix; marker records all five expected skill revisions; a different archive is `recovery_required`; the same archive previews exactly the remaining effects (plus a crashed skill stage as a leftover), resumes, removes the leftover and marker, and ends with every skill equal to the candidate |
| Skill modified after an interruption | resume refused `recovery_required`; zero effects |
| Interruption after all effects, before marker removal | resumed preview has no publication effect but is authorizable to clear only the owned marker |
| Downgrade; divergent same version; modified binary; unsupported host row; concurrent lock; installer marker; hard-linked binary; symlinked receipt; POC state; permissive target | refused before any effect; owned tree unchanged |
| Receipt changed after review; insufficient space | denied / refused with zero effects |
| `PublishUpgradeSkill` with a mismatched or absent expectation; non-Axiom name | refused; no staging left |
| SemVer precedence including pre-release and build metadata | verified |

**Codex skill-set receipt.** `.axiom-skill-set.receipt` is derived from the
installing binary's runtime `skillSetVersion`/`binaryCompatibility` (currently
`2`/`2`). Release archive format 1 does not carry the candidate's values; its
skill manifest declares `1`/`1`. The upgrade therefore never writes that
receipt. When skill files change, it reports `partial` with
`skillReceipt=refresh_required` and the next action
`lingo runtime codex install` with the upgraded binary. The native row shows
that this action makes the runtime compatible. See
[Human decisions required](#human-decisions-required).

The upgrade shares the installer's `.axiom-install.lock` and
`.axiom-install-operation` names, so each refuses the other's interrupted state.
It also takes the Codex runtime's `.axiom-skill-set.lock`. There is no automatic
rollback, downgrade, or automatic update.

## T21 — Cross-slice security and bounded-I/O regression

`scripts/test-s7-security.sh` names exact tests per requirement alias and fails
when a name no longer exists, so an empty match cannot pass silently. The run
at `604bd10` reported `failures=0 result=pass`:

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
| FR-023, SEC-004 | upgrade interruption at every ordered effect and resume; skill changed after interruption; Work Item confirmed-effect partial |
| FR-036, MVP-SEC-06, SEC-005 | historical POC skill set is known legacy; owned skill publication, legacy replacement, skill conflicts, expected-revision publication, staging leftovers |
| MVP-SEC-01, MVP-SEC-05 | skill stale authority and concurrent Codex install |

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
`macos-27.0-arm64-apfs` at `604bd10`: `source_state=clean`,
`untracked_entries=0`, `archive_kind=release`, 34 passing steps,
`failures=0 result=pass`.

| Step | Result |
|---|---|
| F0–F8 publication; two-process barriers; traversal/link/replacement; ownership/mode/ACL | pass |
| Artifact capacity/cleanup; guided recovery; compatibility/transfer; upgrade units (including skill publication, interruption at every effect, skill conflicts); race | pass |
| Existing release-archive suite | pass |
| Build 1.0.0 and 1.1.0 **release-mode** archives | pass |
| Clean install; equivalent reinstall `unchanged`; installer refuses owned upgrade | pass |
| Upgrade preview read-only; stale digest `denied_authority`; apply; receipt version and binary digest verified | pass |
| Equivalent upgrade no-op; installer accepts the upgraded receipt as `unchanged`; downgrade refused | pass |
| Native resume: binary committed, receipt pending; installer refuses; preview shows receipt only; apply succeeds; marker cleared | pass |
| Native skills: historical POC skill set; preview has five skill effects; apply reports `skillReceipt=refresh_required`; published files equal the 1.1.0 archive skills; no staging or marker left; upgraded binary's `runtime codex install` then reports a compatible runtime | pass |

| Row | Status |
|---|---|
| macOS 27.0 / arm64 / APFS (case-insensitive) | **pass** at `604bd10` (history: also at `11f2b7d` and `408a2b5` with development archives) |
| Ubuntu 26.04 / amd64 / ext4 | **not executed**: no matching environment is available (no VM, container, or remote runner on this host); blocks T22 and release |
| Ubuntu 26.04 / arm64 / ext4 | **not executed**: same reason; blocks T22 and release |

Cross-compilation is not substituted for native Evidence. The existing
`macos-15`/`ubuntu-24.04` workflow is a regression check, not T22 Evidence. It
was not changed; adding exact-target runners is a separately reviewed workflow
change.

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
3. **Native suite on a clean checkout (S7).** With no untracked files,
   `test-s7-native.sh` expanded an empty array under `set -u`, which macOS bash
   3.2 rejects, so it aborted before building archives. Earlier runs only
   passed because an untracked file forced development mode. Fixed in
   `d76a3b3`.

## Limitations and unexecuted cases

- **T22 Ubuntu rows were not executed.** They remain blocking for T22 and for
  release; no Ubuntu behavior is claimed.
- **Native archives** at `604bd10` are release-mode builds from a clean
  checkout of an unpublished branch. They are not identified RC archives
  (T23/S8 work).
- **Evidence retention.** Evidence-class artifacts are never age-eligible; see
  [Human decisions required](#human-decisions-required).
- **Interrupted S2 attempt markers** (`.lingo-install-*`,
  `.lingo-attempt-*`) have no prior/new generation facts and are always
  `preserved_review`.
- **Skill-set receipt.** `lingo upgrade` publishes skill files but not the
  Codex skill-set receipt; see [Human decisions required](#human-decisions-required).
- **Upgrade skill ownership by set proof** reconstructs the release skill
  manifest in the format `build-release-archives.sh` writes today. A future
  manifest format change must keep this reconstruction in step, or the
  set proof fails closed to per-skill ownership.
- **Stale upgrade lock.** A process killed while holding
  `.axiom-install.lock` leaves it in place, as the installer does; both report
  busy until an operator confirms no operation is running.
- **Skill manifest version skew.** The release skill manifest declares
  `skillSetVersion=1`/`binaryCompatibility=1` while the runtime receipt uses
  version 2. This is pre-existing and is the root cause of the skill-set
  receipt decision below. It was not changed.
- **Excluded by the approved threat model:** arbitrary malicious same-UID
  interleavings, physical power loss, and media durability.
- **Local `.claude/` directories fail validation by design.** A maintainer
  whose Claude client creates project-level configuration must keep it outside
  the checkout, or first obtain a recorded decision.

## Maintainer runtime bootstrap (not S8)

By human decision (2026-09-26, PR #100), Codex and Claude are both maintainer
runtimes for this repository. `AGENTS.md` records the contract ("Maintainer
runtimes"):
- `AGENTS.md` remains the single canonical agent policy;
- the only runtime-specific file is a regular root `CLAUDE.md` whose entire
  content is `@AGENTS.md`;
- anything else (other content or target, nested or differently cased
  `CLAUDE.md`, `CLAUDE.local.md`, `.claude/`, symlinks) requires an explicit
  recorded decision.

`scripts/check-claude-bootstrap.sh` enforces this for both
`validate-repository.sh` and `validate-agent-package.sh`.
`scripts/test-check-claude-bootstrap.sh` covers 20 cases:
- pass: absent file, exact pointer, exact pointer without a final newline;
- fail: own instructions, extra blank line, CRLF, other targets, prose,
  nested, differently cased, `CLAUDE.local.md`, `.claude` directory, file, or
  nested directory, symlinks inside and outside the root, directory,
  missing or symlinked `AGENTS.md`.

`test-validate-agent-package.sh` adds end-to-end different-target and nested
cases. The root `CLAUDE.md` is therefore not a finding under this contract.

This is **maintainer/runtime bootstrap support** for developing this
repository. It is not **Axiom product multi-runtime orchestration** (Runtime
and model resolution, Execution Graph, child Executions, cross-runtime
coordination, budgets). That remains Slice S8, tracked by Issue #97, and is
neither implemented nor evidenced here.

## Human decisions required

### T18 — Evidence 365-day post-retirement retention

**Human decision required.** The approved rule makes an Evidence artifact
eligible "365 days after every Evidence reference is explicitly retired". It
cannot be implemented correctly on the current persisted model.

1. **Missing information.** No authoritative record exists that an Evidence
   reference was explicitly retired, or when. `metadata.json` format 1 records
   `createdAt`, `references`, `liveReferences`, `supersededBy`, and
   `cleanupState` (always `retained`). Execution records hold transition
   references. No command or store operation retires a reference.
2. **Why it is impossible today.**
   - The window starts at an explicit act that is never persisted.
   - `createdAt` is not retirement.
   - An unreferenced artifact may never have been referenced, or may have lost
     its reference in a way Axiom cannot attribute.
   - The metadata decoder is closed (`DisallowUnknownFields`, `formatVersion`
     1), so no field can be added compatibly.

   Any inference could delete Evidence early. The current fail-closed
   behavior (`evidence_retirement_unrecorded`, never eligible) is the only
   correct one. The 30-day (`diagnostic`) and 90-day (cleanup record)
   boundaries are implemented and tested with a fake clock.
3. **Smallest evolution.** Add an explicit, separately authorized "retire
   Evidence references" operation, published under the state-root lock
   through the ADR-0007 protocol. It writes a bounded owned retirement record
   per artifact (artifact ID, digest, retired reference identities, retirement
   time), for example under `artifacts/v1/retirements/`. Artifact metadata v1
   stays unchanged. Cleanup counts 365 days from the recorded time and only
   when no live reference remains.
4. **Compatibility and migration.** No backfill: an absent record means "not
   retired", which preserves today's behavior for all existing artifacts. The
   compatibility inspector must learn the new record type; older binaries
   would classify it as unknown content (`malformed`), which blocks mutation
   rather than losing data. That requires a reviewed state-format decision.
5. **Security and recovery impact.**
   - Retirement becomes a precursor to deletion authority, so it needs exact
     preview/authority, audit, lock, and stale-reference refusal.
   - Recovery must classify interrupted retirement publication like other
     ADR-0007 objects.
   - A wrong local clock could shorten the window, exactly as it can for the
     30/90-day rules.
6. **Recommendation.** Keep the current fail-closed behavior for the MVP.
   Record in the Plan that Evidence post-retirement cleanup is deferred until
   an explicit retirement operation is specified and approved. Specify that
   operation as a separate reviewed change rather than inside S7.

Until decided, T18's "30/365/90-day" boundary obligation is met for 30 and 90
days only.

### T20 — Codex skill-set receipt after an upgrade

**Human decision required** (it does not block the Plan ordering, which is
implemented):

- **Problem.** The upgrade publishes skill files but cannot publish the
  candidate's `.axiom-skill-set.receipt`, because release archive format 1
  does not carry the candidate binary's runtime
  `skillSetVersion`/`binaryCompatibility`. Its manifest says `1`/`1`; the
  runtime uses `2`/`2`.
- **Options.**
  - (a) Extend the release skill manifest with the runtime identity, so the
    upgrade can publish a self-verifying skill-set receipt. This is a release
    format, installer, and `LoadCandidate` change.
  - (b) Keep the current explicit `partial` + `refresh_required` + next action.
  - (c) Execute the upgraded binary after publication. This is a new trust
    topology.
- **Recommendation.** (b) for S7, and (a) as a reviewed release-format
  decision before RC.

## Commands and results

All at `604bd10`, clean worktree:

| Command | Result |
|---|---|
| `go build ./...`, `go vet ./...`, `go mod verify` | pass; all modules verified |
| `go test ./... -count=1` | pass (all packages) |
| `go test -race ./internal/compatibility ./internal/install ./internal/local ./internal/detailartifact ./internal/codexruntime ./internal/cli ./cmd/lingo -count=1` | pass |
| `./scripts/test-s7-security.sh` | 28 case groups pass; `failures=0 result=pass` (includes race, sensitive files, gitleaks worktree) |
| `./scripts/test-s7-native.sh` | `native_row=macos-27.0-arm64-apfs archive_kind=release failures=0 result=pass` (34 steps) |
| `./scripts/validate-repository.sh .` | pass (runs `check-claude-bootstrap.sh` and its test) |
| `./scripts/validate-agent-package.sh .` | pass |
| `./scripts/check-sensitive-files.sh .`; `--staged` before each commit | pass |
| `./scripts/test-check-claude-bootstrap.sh` | pass (20 cases) |
| `./scripts/test-validate-agent-package.sh` | pass |
| `gitleaks detect --source .` (209 commits) and `--no-git`; `gitleaks protect --staged` before each commit | no leaks |
| `git diff --check` | clean |

History: at `408a2b5` the same suite passed except the repository validators,
which then failed only on the untracked `CLAUDE.md`.

Human acceptance of S7 is not inferred from this record.

# S1 Evidence — Result, provenance and protected local substrate

## Scope and claim

This record covers the authorized remainder of S1 in GitHub Issue #75: T02 and
T03 after accepted/merged T01. It supports the claim that Axiom can create and
read one bounded, durable, machine-local Markdown detail artifact by opaque ID,
and that delivered local stores preserve old-or-new commit truth and fail closed
when publication state is uncertain.

This is implementation Evidence, not human acceptance. It does not authorize or
claim S2–S7, cleanup/recovery mutation, Provider effects, Git remote mutation,
Runtime installation, migration, release, or physical power-loss durability.

## Delivered contracts

- `internal/detailartifact` owns closed metadata v1, initial Plan guardrails,
  structural sensitive-value rejection across Markdown, `Reference.Value`, and
  `LiveReferences`, correlation, SHA-256 integrity,
  retention classification, and required/optional completion materialization.
- `internal/local.ArtifactStore` owns
  `<state-root>/artifacts/v1/objects/<prefix>/<uuid>/` and publishes exactly
  `metadata.json` plus `details.md` as one protected directory.
- `internal/local` owns the shared F0–F8 vocabulary, versioned bounded recovery
  marker, private staging, protected create/update, canonical confirmation,
  committed-state reporting, and fail-closed reader checks.
- Work Item and workflow records now carry exact observed byte revisions for
  stale-authority rejection. Project create/update uses the shared versioned
  recovery marker with object, operation, stage, prior/new presence and digest,
  staging object, and rename commit protocol. Installation retains its bounded
  versioned attempt marker within the same observable publication invariants.
- Coordination order is managed state root, store namespace, Project/object
  scope. Locks are kernel-backed and acquired broad-to-narrow. A conflict does
  not wait, retry, or infer lock ownership from PID/time.

Commit point: successful protected rename of the complete canonical file or
directory. Before that point, the prior complete generation remains authority.
After canonical bytes are confirmed, later acknowledgment/secondary/cleanup
failure reports `Committed=true`; it never claims rollback. Any surviving stage
or marker makes the relevant reader return `recovery_required`.

## Bounds and capacity

| Guardrail | Enforced value | Failure behavior |
|---|---:|---|
| Markdown per artifact | 1 MiB | reject before publication |
| Metadata per artifact | 64 KiB | reject; no field truncation |
| Captured output represented per artifact | 256 KiB | reject before publication |
| Live artifacts per state root | 10,000 | explicit capacity error; no eviction |
| Aggregate artifact content | 1 GiB | explicit capacity error; no eviction |
| Directory read batch | 64 entries | incremental enumeration; no unbounded read |
| Artifact object directory | 2 entries | reject on the third entry |
| Artifact shard namespace | 256 entries | reject on the 257th entry |
| Artifact IDs across shards | 10,000 entries | stop and fail closed on the 10,001st entry |
| Local protocol scan | 10,002 entries | stop and fail closed when exceeded |

Lookup accepts only a lowercase UUID v4 and returns `artifact:<uuid>` after
validating object identity, closed metadata, content size/digest, owner-only
mode/ACL, regular-file type, and link count. Paths are never lookup authority.

## Per-store fault and reader Evidence

The tables below distinguish a tested protocol checkpoint from a stage that the
store cannot reach. `N/A` is not coverage: it means the store has no corresponding
effect in S1. Test names are exact Go test identifiers under `internal/local`.

### Project store (`PortableStore`)

Mechanism: anchored `os.Root` handles, non-blocking kernel advisory locks, private
same-filesystem directory staging for create, the shared single-file publisher
for update, protected no-replace directory rename or atomic file rename, and the
shared bounded versioned recovery marker. The marker identifies the affected
object, operation ID, protocol stage, prior/new presence and SHA-256, staging
object, and rename commit protocol. Create commits when the staged Project
directory is renamed to the canonical slug; update commits when the staged
manifest replaces `axiom.yaml`. Update authority is the exact previously observed
manifest bytes; their SHA-256 is the observable portable revision above this
store boundary.

| Stage | Applicable mechanism | Pre/post-commit and reader outcome | `recovery_required` | Test / limitation |
|---|---|---|---|---|
| F0 | Pre-cancel after entry, before staging | Prior bytes remain readable; no digest change | No | `TestPortableCancellationBeforeStagingPreservesOldBytes` |
| F1 | Complete-write loop; ENOSPC/EDQUOT seam | Create stays absent or update retains exact expected bytes | No after controlled cleanup | `TestPortableDiskFullBeforePublicationPreservesPriorState`; `TestWriteCompleteHandlesShortWritesAndStorageFaults` |
| F2 | Prepared manifest identity/mode/link verification | Unsafe stage is not published; prior/absence remains authority | No when controlled cleanup succeeds; surviving stage fails closed | `TestPortableRejectsStagedHardLinkBeforePublication`; `TestPortableStoreReportsInterruptedCreate`; `TestPortableStoreReportsInterruptedUpdate` |
| F3 | Expected-byte recheck, cancellation, root/Project identity recheck | Prior exact bytes remain authority; marker identifies both possible generations | No after controlled cancellation cleanup; Yes after process death or uncertain cleanup with marker | `TestPortableUpdateCancellationBeforePublicationPreservesOldBytes`; `TestPortableRecoveryMarkerIdentifiesPriorAndNewGeneration`; replacement and crash tests |
| F4 | No separate prior-generation rename | Atomic file replacement retains old bytes until F5; create has no prior generation | N/A as a distinct Project-store stage | ADR-0007 permits replaceable mechanisms; no separate prior object exists |
| F5 | Protected directory create or atomic manifest replacement | One winner; loser conflicts; reader returns winner only | No after controlled collision cleanup | `TestPortableConflictingWritersHaveOneWinner` |
| F6 | Post-rename directory sync/ack boundary | New complete bytes exist; reader blocks while marker survives | Yes | `TestPortablePostPublicationSyncFailureRequiresRecovery`; `TestPortableCreateCrashBoundaryRequiresRecovery`; `TestPortableUpdateCrashBoundaryRequiresRecovery` |
| F7 | No secondary Provider/artifact effect in Project publication | Canonical Project commit is the sole primary effect | N/A | S1 Project store exposes no secondary effect |
| F8 | Attempt-marker removal and directory sync | Canonical bytes remain committed; ordinary reader blocks; a failed sync after successful removal restores the owned marker | Yes | `TestPortableCleanupFailurePreservesCommittedBytesAndRequiresRecovery`; `TestPortableCreateRestoresMarkerWhenRemovalSyncFailsAfterCommit` |

### Installation store

Mechanism: anchored state/Project roots, broad-to-narrow advisory locking, strict
source re-observation, private file staging, protected no-replace rename, directory
sync, and bounded versioned attempt marker. It is create-only in S1. Commit is the
rename to `installation.json`. The record embeds the exact portable revision;
there is no expected local revision for create. Equivalent existing bytes are a
no-op and different existing bytes are an explicit conflict.

| Stage | Applicable mechanism | Pre/post-commit and reader outcome | `recovery_required` | Test / limitation |
|---|---|---|---|---|
| F0 | Pre-cancel before source/stage mutation | No local record; reopen reports absent local state | No | `TestInstallationCancellationBeforeStagingLeavesNoRecord` |
| F1 | Complete-write loop; ENOSPC/EDQUOT seam | No canonical record | No after controlled cleanup | `TestInstallationDiskFullBeforePublicationLeavesNoRecord`; shared short/zero-write test |
| F2 | Prepared record identity/link/content verification | Unsafe staged record is not published | No when controlled cleanup succeeds | `TestInstallationRejectsStagedHardLinkBeforePublication` |
| F3 | Source revision, cancellation, target/root identity recheck | No record is committed; replacement target is not followed | No after controlled cleanup; Yes after process death with marker | `TestInstallationCancellationBeforePublicationLeavesNoRecord`; `TestInstallationRejectsTargetReplacementBeforePublication`; crash test |
| F4 | No prior-generation preservation | Installation is create-only and cannot replace an existing record | N/A | Existing divergent record conflicts before staging |
| F5 | Protected no-replace rename | Complete concurrent winner remains readable; attempted publisher returns conflict | No after controlled collision cleanup | `TestInstallationPublicationCollisionPreservesCompleteWinner` |
| F6 | Post-rename target/projects sync and acknowledgment | Complete record exists; normal reopen blocks while marker survives | Yes | `TestInstallationPostPublicationSyncFailureRequiresRecovery`; `TestInstallationCrashBoundaryRequiresRecovery` |
| F7 | No secondary effect in local installation-record publication | Canonical record is the sole effect | N/A | Runtime installation belongs to T04 and was not introduced |
| F8 | Attempt-marker removal and sync | Canonical record remains committed; reopen blocks; a failed sync after successful removal restores the owned marker | Yes | `TestInstallationCleanupFailurePreservesCommittedRecordAndRequiresRecovery`; `TestInstallationRestoresAttemptWhenRemovalSyncFailsAfterCommit` |

### Work Item store

Mechanism: shared single-file publisher with anchored roots, ordered advisory
locks, private stage, bounded JSON recovery marker containing prior/new SHA-256,
atomic create/update rename, confirmation, and owned cleanup. Commit is canonical
file rename. Create uses the zero revision only when the record is absent; update
requires SHA-256 of the exact observed canonical bytes. Repeating a zero-revision
create, even with identical bytes, conflicts without replacement
(`TestWorkItemStoreCreateCollisionConflictsWithoutReplacement`).

| Stage | Applicable mechanism | Pre/post-commit and reader outcome | `recovery_required` | Test / limitation |
|---|---|---|---|---|
| F0 | Injected after locks/before staging | Exact prior revision remains readable | No | `TestWorkItemStoreRequiresExpectedRevisionAndFailsClosedAcrossF0F8/F0` |
| F1 | Injected before complete staged write | Exact prior revision remains readable | No | same matrix `/F1`; short/zero-write primitive test |
| F2 | Injected after stage/before validation | No commit; surviving private stage blocks reader | Yes | same matrix `/F2` |
| F3 | Expected SHA-256 and target recheck | Stale update conflicts; interruption retains prior authority; failed marker/stage removal or cleanup sync reports uncommitted recovery uncertainty | Yes when injected state or uncertain cleanup survives | same matrix `/F3`; `TestWorkItemStoreRejectsStaleRevision`; `TestPublishFileCleanupFailuresRequireRecoveryBeforeCommit` |
| F4 | Marker checkpoint before publication; no separate prior file for atomic replacement | Prior canonical revision remains; marker records prior/new digests | Yes | same matrix `/F4` |
| F5 | Atomic no-replace/create or replacement rename | Old or new complete JSON only | Yes for injected uncertainty | same matrix `/F5` |
| F6 | Post-rename marker/confirmation/sync | New SHA-256 is committed; reader blocks pending acknowledgment | Yes | same matrix `/F6` |
| F7 | Protocol post-primary checkpoint; no Provider call inside store | Canonical new bytes stay committed | Yes for injected interruption | same matrix `/F7`; no external secondary effect claimed |
| F8 | Marker cleanup/sync | Canonical new bytes stay committed; failed sync after successful marker removal restores fail-closed state | Yes | same matrix `/F8`; `TestPublishFileRestoresMarkerWhenRemovalSyncFailsAfterCommit` |

`Service.Select` separately proves reconciliation: it loads the local link,
then reads the Provider, and passes the previously observed revision to `Save`.
A deterministic barrier test updates local state during the Provider call and
proves the stale observation conflicts without replacing the concurrent writer
(`TestSelectRejectsStaleProviderObservationAfterConcurrentLocalUpdate`).

### Workflow store

Mechanism and commit point are the shared single-file protocol above. Create is
explicit; save requires SHA-256 of exact observed workflow JSON bytes. Workflow
format/status checks run before publication. An existing canonical workflow,
including byte-identical content, conflicts at the protected create boundary;
the application then loads and reports the existing lineage. This composition is
covered by `TestWorkflowServiceStartReturnsExistingWorkflowWithRealStore`, while
`TestWorkflowStoreCreateCollisionNeverReplacesCanonicalState` covers identical
and divergent create collisions without replacement.

| Stage | Applicable mechanism | Pre/post-commit and reader outcome | `recovery_required` | Test / limitation |
|---|---|---|---|---|
| F0 | After locks/before staging | Prior workflow revision readable | No | `TestWorkflowStoreFailsClosedAcrossF0F8/F0` |
| F1 | Before complete staged write | Prior workflow revision readable | No | same matrix `/F1`; shared short/zero-write primitive test |
| F2 | After stage/before validation | No commit; surviving stage blocks reader | Yes | same matrix `/F2` |
| F3 | Expected SHA-256 and target recheck | Stale save conflicts; prior remains authority | Yes only when injected state survives | same matrix `/F3`; `TestWorkflowStoreRequiresExpectedRevision` |
| F4 | Marker checkpoint before atomic replacement | Prior canonical revision remains; prior/new digests retained | Yes | same matrix `/F4` |
| F5 | Atomic create/replacement rename | Old or new complete workflow only | Yes for injected uncertainty | same matrix `/F5` |
| F6 | Post-rename confirmation/sync | New workflow revision committed; reader blocks | Yes | same matrix `/F6` |
| F7 | Protocol post-primary checkpoint; no Provider projection in T03 | Canonical workflow remains committed | Yes for injected interruption | same matrix `/F7`; Provider projection belongs to T12 |
| F8 | Marker cleanup/sync | Canonical workflow remains committed; leftover blocks reader | Yes | same matrix `/F8` |

### Artifact store

Mechanism: shared vocabulary implemented for one multi-file object using an
anchored shard root, private directory staging, strict reread/digest validation,
bounded JSON recovery marker, protected no-replace directory rename, canonical
confirmation, and owned cleanup. Commit is rename to the UUID v4 canonical
directory. Create authority has no prior object revision; metadata records the
SHA-256 content digest and creation checks the exact capacity observation.

| Stage | Applicable mechanism | Pre/post-commit and reader outcome | `recovery_required` | Test / limitation |
|---|---|---|---|---|
| F0 | After locks/capacity, before stage | Artifact absent | No | `TestArtifactStoreFaultStagesPreserveCommitTruthAndFailClosed/F0` |
| F1 | Stage creation/write | Artifact absent if controlled cleanup completes | Yes only if private stage survives | same matrix `/F1`; shared short/zero-write test |
| F2 | Complete metadata/content/digest/mode/link validation | No canonical object; surviving stage blocks lookup | Yes | same matrix `/F2`; `TestArtifactStoreRejectsDigestModeLinkAndTypeChanges` |
| F3 | Capacity/identity/context recheck and marker | No canonical object | Yes for injected interruption | fault matrix `/F3`; capacity/collision test |
| F4 | Marker checkpoint; create has no prior generation | Artifact remains absent; intended digest retained in marker | Yes | fault matrix `/F4`; no prior artifact exists |
| F5 | Protected no-replace directory rename | Complete object or absence; collision cannot replace | Yes for injected uncertainty | fault matrix `/F5`; `TestArtifactStoreCapacityAndCollisionFailWithoutEviction` |
| F6 | Post-rename marker, canonical reread/digest and sync | Complete digest-verified artifact committed; lookup blocks | Yes | fault matrix `/F6` |
| F7 | Protocol post-primary checkpoint; no secondary effect in store | Complete artifact remains committed | Yes for injected interruption | fault matrix `/F7`; completion-service partial classification is tested separately |
| F8 | Marker cleanup/sync | Complete artifact remains committed; failed sync after successful marker removal restores fail-closed state | Yes | fault matrix `/F8`; `TestArtifactStoreRestoresMarkerWhenRemovalSyncFailsAfterCommit` |

### Process, confinement, and mechanism rationale

| Store | Process/concurrency observation | Outside-root observation |
|---|---|---|
| Project | Create/update helpers are killed after pipe barriers before and after commit; readers then require recovery. Separate two-process lock tests admit one owner and reject the contender. | Ancestor-replacement tests prove redirected trees unchanged. Shared sentinel SHA-256 remains `aaa8d3c8d74ad3e8f6b1772aa9c7e0eaa528cb42fc93599ce2f125b00d4c424c`. |
| Installation | Helper is killed after pre/post-commit barriers; reopen requires recovery and observes absent/complete canonical record according to commit point. | Target-replacement test proves redirected tree unchanged; shared sentinel hash remains the value above. |
| Work Item | No separate process-kill helper: the adapter uses the same `publishFile` and ordered kernel-lock implementation tested by its full F0–F8 matrix and process-lock suites. | Shared per-store confinement test retains the sentinel hash above. |
| Workflow | No separate process-kill helper: same shared publisher/lock implementation, plus its own full adapter F0–F8 matrix. | Shared per-store confinement test retains the sentinel hash above. |
| Artifact | Helper is killed after a staged barrier; a concurrent writer conflicts and subsequent access requires recovery. | Artifact-specific portable/Repository sentinels and shared per-store sentinel retain the hash above. |

`TestEveryT03StorePreservesOutsideRootHash` runs successful publication through
all five adapters and asserts the exact pre/post sentinel digest. Attack-specific
replacement/link/type tests remain separate so this normal-path hash does not
stand in for confinement validation.

The selected mechanisms are standard-library anchored filesystem handles,
kernel locks, same-filesystem private staging, rename publication, bounded
versioned markers, exact byte digests, and fail-closed reads. They preserve the
ADR-0007 observable invariants without making a persisted syscall/library choice.
Assumptions remain: user-owned local roots; supported regular-file/directory
identity, link-count, restrictive mode/ACL, rename and advisory-lock semantics;
stage and canonical target on one filesystem. Exclusions remain malicious
arbitrary same-UID interleaving, physical power loss/media guarantees, mutating
recovery, and native Ubuntu 26.04 evidence. Exact-target native evidence remains
obligatory at T22.

## Security and confinement observations

- Synthetic token/private-key/raw-chat sentinels are rejected from Markdown and
  free reference metadata on both creation and decode; unsafe input is neither
  truncated nor persisted.
- Structurally valid URLs in `Reference.Value` and `LiveReferences` reject any
  user-info and case-insensitive sensitive query names `token`, `access_token`,
  `refresh_token`, `api_key`, `client_secret`, and `password`. Creation and decode
  return generic errors without the sensitive value; a safe GitHub issue URL is
  accepted (`TestArtifactRejectsCredentialURLsInReferenceMetadataOnCreateAndDecode`).
- Invalid UTF-8, control characters, oversized content/metadata, unknown or
  duplicate metadata fields, invalid correlation, and digest mismatch fail.
- Symlink, hard-link, wrong type, permissive mode/ACL, collision, replacement,
  unsafe root, and arbitrary-path lookup cases fail closed.
- Isolated portable and Repository sentinel trees remain byte-identical after
  artifact creation; production code has no Provider, Git, Runtime, network, or
  process-execution port.
- Capacity failures preserve existing objects and perform no automatic cleanup.

## Review-finding remediation Evidence

The PR #83 review remediation adds deterministic coverage for all three reported
findings without changing the ADR-0007 commit point, authority, marker schema, or
persisted recovery meaning:

- URL credential/user-info and sensitive-query rejection is covered across
  `References`, `LiveReferences`, `New`, and `Decode`, including case-insensitive
  query-name matching and a safe URL acceptance case. The synthetic sensitive
  value is absent from returned errors, accepted artifact state, and this Evidence.
- `TestPublishFileRestoresMarkerWhenRemovalSyncFailsAfterCommit`,
  `TestArtifactStoreRestoresMarkerWhenRemovalSyncFailsAfterCommit`,
  `TestPortableCreateRestoresMarkerWhenRemovalSyncFailsAfterCommit`, and
  `TestInstallationRestoresAttemptWhenRemovalSyncFailsAfterCommit` inject `EIO`
  only after marker removal succeeds. Each proves committed canonical bytes remain,
  the result is truthful, the next reader returns `recovery_required`, and unrelated
  objects remain unchanged.
- `TestProtocolStateScanIsBoundedAndFailClosed`,
  `TestDirectoryEnumerationStopsAtExplicitLimit`,
  `TestArtifactObjectEnumerationRejectsAdditionalEntriesBoundedly`, and
  `TestArtifactStoreCapacityBoundariesAndInvalidEntriesDoNotEvict` cover empty,
  safe, marker, exact-limit, over-limit, large-unknown-entry, artifact-object, and
  capacity paths. Enumeration uses batches, never truncates, and never evicts or
  cleans unknown entries.

The remediation run executed every command in the Reproduction section with exit
0. The focused race/shuffle command passed three repetitions for all six requested
packages; the full suite, vet, build, module verification, install test, dogfood,
repository validation, sensitive-file checks, both Gitleaks modes, shell syntax,
and `git diff --check` also passed. This remains technical Evidence only, not Human
Acceptance, S2 authority, guided recovery, cleanup authority, migration, or release.

## Reproduction

Run from repository root without Makefile indirection:

```bash
go test -race -shuffle=on -count=3 \
  ./internal/detailartifact ./internal/local ./internal/completion \
  ./internal/provenance ./internal/workitem ./internal/workflow
go test ./... -count=1
go vet ./...
go build ./...
go mod verify
./scripts/test-install-axiom.sh
./scripts/dogfood-poc.sh
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
./scripts/check-sensitive-files.sh --staged .
gitleaks dir . --no-banner --redact
gitleaks git --staged --no-banner --redact
for script in scripts/*.sh; do bash -n "$script" || exit 1; done
git diff --check
```

The final S1 run recorded every command above with exit 0. The focused race run
was repeated three times with shuffled test order. No remote Provider or Git
mutation occurs in these commands; dogfood uses synthetic local/fake boundaries.

## Native observation and deferred matrix

Native execution for this delivery:

| Field | Observation |
|---|---|
| OS | macOS 27.0, build 26A428 |
| Kernel / architecture | Darwin 27.0.0, arm64 |
| Filesystem | APFS (`/dev/disk3s3s1`) |
| Go | go1.26.1 darwin/arm64 |
| baseline `main` | `5d4ca4babe8b63a33da3d8d20cfd2b19415c36a5` |

T22 retains the approved obligation for native Ubuntu 26.04 amd64/arm64 ext4
and final exact-target reruns. This S1 delivery does not convert current CI or
cross-compilation into those native claims.

## Remaining limitations

- Explicit artifact cleanup is T18; guided mutation of recognized recovery state
  is T19. S1 preserves and diagnoses uncertainty but does not auto-repair/delete.
- The guardrails are approved initial defaults, not benchmark-derived production
  capacity promises. Later dogfood/RC review may propose changes without changing
  artifact identity or ownership semantics.
- Same-UID arbitrary hostile interleavings, physical power loss, and media
  durability remain excluded by ADR-0005.
- S2 was not started.

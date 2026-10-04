# Issue #153 Evidence — RecognizedPOC transition and N -> N+1 upgrade acceptance

Issue: [#153](https://github.com/rgomids/axiom/issues/153). Tasks I153-T02 and
I153-T03 ([tasks.md](tasks.md)); Plan §21; decision record
[ADR-0017](../../decisions/0017-recognized-poc-preservation-archive.md).
Executed 2026-10-04 on macOS 27.0.1 (`26A434`), arm64, APFS, `go1.26.1`.
This is new Evidence for the new flow. Historical T17 and S7/S9 Evidence is
unchanged and is not reused as proof here.

## Baseline (before)

Real artifacts: POC state created by the tagged `v0.1.0-poc.1` source build
(the POC shipped no binaries; this is how its fixture is generated), the
published v0.1.0 installed through its own bundle facade, then the published
v0.4.1 facade:

```text
compatibility inspect (v0.1.0): recognized_poc / complete_poc_workflow_signature
install_error: owned upgrade refused: Upgrade blocked before any effect: state_transition_unavailable
install_next: Existing Axiom state needs an automatic transition this release does not perform yet; ...
```

Rerunning stays blocked: the product dead end of #153.

## After

Same journey into a candidate built from this branch (`0.4.2-local.2`, clean
release archive, its own facade):

| Check | Result |
|---|---|
| Installer | `install_preserved=<home>/.local/state/axiom/archive/recognized-poc-<digest>`, `install_status=upgraded`, exit 0 |
| Archive | `manifest.json` + 4 `objects/<sha256>`; policy `recognized-poc-preservation/v1`; retired: `state/work-items/<id>/main-7.json`, `state/workflows/<id>/main-7.json`; kept: `projects/poc-project/axiom.yaml`, `state/projects/<id>/installation.json` |
| Active state | only `projects/<id>/installation.json`; `compatibility inspect`: `valid_v1` / `v1_readable_state` |
| Reconfigured intent | `axiom project list`: `project: poc-project … name="POC Project"` |
| `first-run` | Codex `already_configured`, Claude `configured`, exit 0 |
| Rerun installer | `install_status=unchanged`, exit 0, archive unchanged |
| Archive on a second APFS volume (`AXIOM_ARCHIVE_ROOT`, distinct device) | `upgraded`, exit 0; 5 files on the second volume |
| Archive on a full HFS+ volume (0 KiB free) | `insufficient_space` before any effect; POC state and archive untouched |

## Automated upgrade acceptance (I153-T03)

`scripts/test-upgrade-journeys.sh --candidate <0.4.2-local.2> --previous <v0.4.1>
--previous <v0.1.1> --poc-binary <v0.1.0-poc.1 build>`: **43 checks,
`failures=0 result=pass`**.

- **v1 state, N=v0.4.1 → candidate:** state created by v0.4.1's own CLI
  (Project, GitHub Work Item and create attempt through a fake `gh`, Execution
  advanced through two gates) plus the frozen stable-v1 corpus (graph,
  coordination, runtime profile, artifacts, cleanup/retirement records,
  another Execution); v0.4.1 classifies it `valid_v1`; after upgrade: version,
  state and portable trees byte-identical, `valid_v1`, `project show` and
  `workflow status` read the Project and Execution, Codex ready, `first-run`
  converges and is a no-op on rerun, installer rerun `unchanged`.
- **v0.1.1 → candidate:** `upgraded` (no `partial`), Codex ready without
  `first-run`, Claude ready after `first-run`, rerun `unchanged` (#186).
- **POC → v0.4.1 → candidate:** v0.4.1 classifies `recognized_poc`; upgrade
  preserves, the manifest equals an inventory computed independently by the
  script (`find` + `sha256`, category/path/digest/bytes), every archived
  object verifies, workflow history is archived and absent from active state,
  rebuilt state `valid_v1`, Project listed, `first-run` passes, rerun
  `unchanged`, archive byte-identical afterwards.

**Negative control:** the same suite with the published v0.4.1 as candidate
(`--previous v0.4.0 --previous v0.1.1 --poc-binary …`) fails 16 checks: the
v0.1.1 journey (`partial`, Codex not ready, Claude conflict) and every POC
transition check (`state_transition_unavailable`). It also shows v0.4.0
misclassifying the state it wrote itself, the defect fixed by #172.

CI runs the suite as `upgrade-journeys (linux)` and `upgrade-journeys (macos)`
on every pull request, `main` push and Release PR, against the newest
published stable tag and v0.1.1.

## Plan §21.6 regression matrix

`go test ./internal/install -run RecognizedPOC` and
`go test ./internal/compatibility -run 'Preservation|Retirement|Resume'`:

| Required regression | Test | Result |
|---|---|---|
| Complete source inventory <-> final manifest (category, path, digest, bytes) before retirement, checked independently | `TestRecognizedPOCUpgradePreservesRebuildsAndInstalls` (`assertPreserved` walks the fixture itself) | pass |
| Portable intent kept only as validated by the current contract; workflow truth absent from active state, inspectable in the archive | same, `assertRebuilt` | pass |
| Interruption during copy, after preservation before activation, during retirement, after activation before install, after binary; resume under fresh authority; no duplicated preservation; equivalent retry no-op | `TestRecognizedPOCInterruptionAtEachBoundaryResumes` | pass (5 boundaries) |
| Partial archive / stage leftover recovered only under the operation marker; unmarked leftover refused | `TestRecognizedPOCArchiveStageLeftovers` | pass |
| Missing, unexpected, changed, unverifiable archived object; manifest missing an inventory object although copies pass; manifest listing an extra object; manifest bytes changed | `TestRecognizedPOCCorrespondenceBlocksRetirement` | pass; source never retired |
| Manifest replaced after the first retirement | `TestResumeAfterRetirementRequiresTheRecordedManifest` | pass |
| After activation: v1 state written meanwhile resumes; archive tampered meanwhile is `recovery_required` | `TestRecognizedPOCResumeAfterActivationVerifiesOnlyTheArchive` | pass |
| Source digest drift after preservation; leaf replacement; ancestor replacement; stale authority | `TestRecognizedPOCDriftAndStaleAuthority` | pass; zero effects |
| Path traversal, absolute escape, top-level, kept kind, wrong category, changed digest at retirement | `TestRetirementIsConfinedToHistoricalStateObjects` | pass |
| Archive inside State, containing Projects, relative, symlinked, `0777`, occupied with foreign content, holding another source's object; foreign entry, hard-linked and symlinked POC records; no archive location | `TestRecognizedPOCUnsafeDestinationsAndSourcesHaveZeroEffects` | pass; zero effects |
| Insufficient space (preflight) | `TestRecognizedPOCInsufficientArchiveSpaceHasZeroEffects`; real full volume above | pass |
| ENOSPC and EDQUOT during copy | `TestPreservationSpaceErrorsKeepSourceAndResume` (injected errno) | pass on POSIX; EDQUOT not defined on Windows (reported skipped) |
| Cross-filesystem target | `TestRecognizedPOCCrossFilesystemArchive` with `AXIOM_TEST_OTHER_FILESYSTEM` on a second mounted APFS volume; Linux uses `/dev/shm` when it is a distinct filesystem | pass (macOS); Linux in CI |
| Secret/local-data boundary | archive `0700`/`0600`, outside the portable root, manifest carries no record content; installation records keep local paths only in machine-local state | pass |
| Foreign/modified/unsafe/ambiguous/corrupt/newer/out-of-window refusal | `TestUpgradeResolvesForwardTransitionPolicy` (unchanged, still passing) | pass |

Platform applicability: a real mid-copy ENOSPC/EDQUOT cannot be produced
deterministically on a real filesystem without racing the preflight, so it is
proven by errno injection; the real full volume proves the preflight refusal.
Windows execution of these tests is by CI (`verify (windows)`), not local.

## Commands

| Command | Result |
|---|---|
All on this branch, clean worktree:

| Command | Result |
|---|---|
| `gofmt -l .` | empty |
| `go vet ./...` (darwin, `GOOS=linux`, `GOOS=windows`) | pass |
| `go test ./... -count=1`; `go test -race ./... -count=1` | pass |
| `AXIOM_TEST_OTHER_FILESYSTEM=<second APFS volume> go test ./internal/install -run RecognizedPOC -v` | pass, cross-filesystem case executed |
| `scripts/test-upgrade-journeys.sh` (candidate `0.4.2-local.3`, previous v0.4.1 and v0.1.1, POC build) | 43 checks, `failures=0 result=pass` |
| `test-install-bootstrap.sh` | pass; its historical-POC case now asserts the transition (`install_status=upgraded`, `install_preserved=`, workflow archived, Project kept, rerun `unchanged`) instead of the former `state_transition_unavailable` dead end |
| `test-install-posix-facade.sh`, `test-install-axiom.sh`, `test-release-archives.sh`, `test-codex-skills.sh`, `test-release-pipeline.sh`, `test-release-flow.sh`, `test-check-automation-registry.py`, `validate-repository.sh` | pass |
| `test-s7-security.sh` | `failures=0 result=pass` |
| `test-s7-native.sh` with a disclosed `sw_vers` shim reporting 27.0 (its historical row gate; host is 27.0.1) | `failures=0 result=pass` |
| `gitleaks git` over the branch | no leaks |

## PR #188 technical review corrections — 2026-10-04

New Evidence against reviewed head `e0464be593a4d7610bb7b3b4d100a77d9a99d02f`.
Historical results above remain unchanged. Corrections preserve accepted
ADR-0017, ADR-0005 and ADR-0007; no POC history is activated, and no Issue
closure, merge, release or remote ruleset mutation is performed.

Executed locally on macOS 27.0.1 / arm64 / Go 1.26.1:
`go test ./internal/install ./internal/compatibility ./internal/local ./cmd/lingo
-run 'Test(RecognizedPOC|Preservation|HistoricalRetirement|WindowsInstallRelease)'
-count=1` — exit 0.

| Finding / regression | New proof |
|---|---|
| CR-001: `TestHistoricalRetirementCoordinatesConcurrentWorkItemWriter`, `TestHistoricalRetirementConflictsWithWorkItemPublicationInProgress`, `TestHistoricalRetirementCannotRemoveConfirmedWorkItemRevision` | Deterministic pre-commit channel barriers exercise both lock acquisition orders using the real WorkItem store. Retirement-held coordination refuses the writer; writer-held coordination refuses retirement and the successful new generation remains readable. A previously confirmed changed revision also refuses stale retirement. |
| CR-001: `TestHistoricalRetirementRejectsLeafAndParentReplacement` | Same-byte replacement of a leaf and replacement of its parent at the controlled commit boundary refuse removal; replacement bytes remain. |
| CR-002: `TestRecognizedPOCResumeRequiresExactActiveInventory` | After confirmed Work Item retirement, removal of portable `axiom.yaml`, installation record or pending workflow refuses resume; new or modified active objects refuse. Only previously confirmed absence resumes successfully. |
| CR-002: `TestRecognizedPOCRevalidatesKeptObjectsBeforeNextRetirement` | Kept-object loss after one confirmed retirement within the same run refuses before the pending workflow can be removed. |
| CR-002: `TestRecognizedPOCResumeRejectsAmbiguousMarkers`, `TestRecognizedPOCActivatedResumeRequiresCompleteRetirementProgress` | Old transition format, missing/negative/excessive progress, duplicate/unknown fields and activation without manifest refuse; activated progress must exactly equal the archive's retirement cardinality. |
| CR-002: existing `TestRecognizedPOCInterruptionAtEachBoundaryResumes` and `TestRecognizedPOCResumeAfterActivationVerifiesOnlyTheArchive` re-executed | All five supported interruption boundaries still resume; ordinary v1 changes after explicit activation remain allowed, while archive tampering refuses. |
| CR-003: `TestRecognizedPOCResumeRejectsSemanticManifestTampering` | After manifest publication and before first retirement, independent tampering of POC tag, POC revision, retired set and kept set refuses preview with every retirement target still present. |
| CR-003: `TestPreservationManifestRejectsAdditionalJSON` | A second JSON value or trailing non-JSON content refuses; canonical re-encoding additionally binds every policy field and derived set and rejects unknown fields. |
| CR-004: `TestWindowsInstallReleasePreservationOutput` | Exact Windows output contains preserved path then upgraded status for a successful transition; ordinary upgrades, fresh installs and unchanged results retain their output without a preservation line. |

The Windows-only `TestWindowsInstallReleasePropagatesRecognizedPOCPreservation`
exercises the complete backend transition and archive-path return. Local Windows
cross-compilation verifies buildability; native execution belongs to the new
head's `verify (windows)` CI job.

Recovery implementation: transition marker format 2 binds archive/manifest,
confirmed deterministic retirement prefix and explicit activation. Before
activation, kept and pending objects must be present and byte-exact. A deletion
without durable progress confirmation is ambiguous and refuses automatic resume;
absence alone never authorizes recovery. After activation, complete progress and
the canonical archive still verify, while active v1 state can evolve normally.
Retirement reuses local store locks (state -> family -> Project), held through
exact-generation checks, deletion and confined empty-container cleanup.

Directed adversarial re-review checked ownership, expected revisions, process
coordination, preview digest binding, recovery truth and interruption boundaries.
An inconsistent activated retirement count discovered during review was fixed
and receives its own regression above. Windows presentation was reviewed
independently. No remaining finding in CR-001–CR-004 was identified by that
review; remote CI and human re-review remain separate evidence/gates.

### Correction validation

Final executable correction source:
`ae11becf1c4d361201d542a6ab610ec035108906` (following correction commit
`9e22f0c047cc79d606d56dffd945dddff47d239e`). Later Evidence-only edits do
not change that executable source. Results below are newly executed, not copied
from historical Evidence.

| Check | Result |
|---|---|
| `gofmt -l .` | exit 0, empty |
| `go test ./...` | exit 0 |
| `go test -race ./...` | exit 0 |
| `go test ./internal/compatibility ./internal/install ./internal/local ./cmd/lingo -count=1` | exit 0 on final executable source |
| `go vet ./...`; `go build ./...`; `go mod verify` | exit 0; all modules verified |
| `./scripts/validate-repository.sh .` | exit 0; runtime behavioral scenarios explicitly skipped by this validator, not claimed |
| `./scripts/test-release-flow.sh` | exit 0; release preflight, state machine, authority and workflow contracts pass |
| `./scripts/test-install-bootstrap.sh` | exit 0 at the initial correction source; includes POC transition and interrupted-upgrade resume |
| Windows test cross-compilation of `internal/install` and `cmd/lingo`; `GOOS=windows go vet ./internal/local ./internal/compatibility ./internal/install ./cmd/lingo` | exit 0; native Windows execution not performed locally |
| `./scripts/build-release-archives.sh --version 0.4.2-review.188.1 --output <private temporary candidate>` | exit 0, clean correction source `ae11bec` |
| `./scripts/test-upgrade-journeys.sh --candidate <0.4.2-review.188.1> --previous <v0.4.1> --previous <v0.1.1> --poc-binary <historical POC build>` | exit 0, 43 checks, `failures=0 result=pass`; isolated local homes, release facades and cached public assets; no release publication |
| `gitleaks dir . --no-banner --redact`; staged Gitleaks and sensitive-file checks before correction commits | exit 0, no leaks |
| `git diff --check` | exit 0 |

CI for the final pushed PR head remains separate from these local results.
Required contexts remain unchanged: verify Linux/macOS/Windows, release-contract,
delivery-metadata and upgrade-journeys Linux/macOS. Historical cross-filesystem,
full-volume and real Runtime results above were not rerun by this correction
validation and are not claimed as new Evidence.

## CR-005 — identity-aware POC Work Item retirement (2026-10-04)

New executable correction source:
`a72f5f069873898cb668c3b987a9b54e365661b0`. Subsequent test/journey refinement
and this Evidence preserve that production implementation. Previous reviewed PR
head: `ad87fe6398ce9d47808042006e8f7fd96a18f545`.

The frozen POC workflow persists Project ID, repository key and Work Item
number, but no provider/resource. Local decoding exposes these validated facts
and binds workflow/link payloads to their real store locations. Compatibility
policy requires exactly one validated link with that complete historical
reference. Zero matches, multiple provider resources or conflicting locations
refuse with `state_changed` before preservation/retirement. Archive identity
failures refuse with `preservation_conflict`. No timestamp, filename heuristic,
create-attempt inference or general classification change is used.

The original retirement set is calculated from source bytes before copying,
and recalculated from verified archived bytes during resume and final archive
verification. Canonical manifest generation and exact active inventory use that
same set; confirmed progress covers only historical objects. All other validated
Work Item links are kept, without changing coordination, pruning or marker v2.

### New deterministic regressions

All five tests below were executed in `internal/install`:

- `TestRecognizedPOCUpgradeKeepsUnrelatedWorkItemLinks`: real
  `WorkItemStore.Save` writes #42 in the same Project/repository as POC #7,
  without a create attempt. Classification remains `RecognizedPOC` before
  preview/authorize/apply. Only workflow/link #7 retire; #42 stays byte-exact,
  `WorkItemStore.Load` succeeds, and the nonempty `work-items/<project>`
  directory survives cleanup. Final classification is `ValidV1`. Every original
  object remains in the verified archive; retired/kept arrays exactly match
  policy. Its second variant adds workflow #8 and a real #8 link: both historical
  pairs retire while #42 stays. The frozen store supports multiple workflows
  addressed by repository key/number; no single-workflow restriction is invented.
- `TestRecognizedPOCUpgradeRefusesUnprovedWorkItemMembership`: workflow #7
  with only unrelated #42 refuses with no effects and unchanged active bytes;
  two #7 links across provider resources also refuse without mutation.
- `TestRecognizedPOCUpgradeRefusesIdentityLocationMismatch`: decodable workflow
  and link payloads under wrong filenames or Project directories cannot authorize
  retirement; all four cases leave the complete installation snapshot unchanged.
- `TestRecognizedPOCResumeRefusesMissingKeptWorkItemLink`: after the first
  confirmed historical retirement, disappearance of kept #42 blocks resume;
  no further retirement or installation effect occurs.
- `TestRecognizedPOCResumeRefusesKeptWorkItemReclassifiedAsRetired`: swapping
  #42 from kept to retired and #7 from retired to kept in the archived manifest
  fails canonical verification before retirement; #42 remains active.

Baseline refutation used `go test -overlay <temporary-overlay.json>
./internal/install -run 'TestRecognizedPOC(UpgradeKeepsUnrelatedWorkItemLinks|
UpgradeRefusesUnprovedWorkItemMembership|ResumeRefusesMissingKeptWorkItemLink|
UpgradeRefusesIdentityLocationMismatch|ResumeRefusesKeptWorkItemReclassifiedAsRetired)$'
-count=1`, replacing only `preservation.go` with the previous reviewed head's
version. Baseline failed: it retired #42, accepted missing/ambiguous membership
and misplaced payloads, and removed #42 before the interrupted resume. The
manifest-swap test's later stronger precondition was not part of that baseline
capture. The final tests pass without the overlay.

### Published-version journey

The updated `scripts/test-upgrade-journeys.sh` ran actual historical POC source
through published v0.4.1 to candidate `0.4.2-review.188.2`, plus the existing v1
and v0.1.1 skill-convergence journeys: **48 passing checks, failures=0**.
Providers and Runtime executables remain deterministic stand-ins; no external
GitHub operation or release publication occurs.

v0.4.1 cannot publish an edit adding a capability to the capability-free POC
Project. Therefore the CLI journey creates a separate v1 Project in the same
state root, selects #42 through real `work-item select` preview/authority, and
proves classification stays `RecognizedPOC`. Candidate upgrade proves #42's exact
bytes, `work-item show`, manifest kept membership, two historical retirements,
valid v1 active state, complete archive and unchanged idempotent rerun. The
stricter same-Project/repository case is proved by the real-store Go regression
above, rather than fabricating an unsupported old CLI edit.

### CR-005 local validation

| Check | Result |
|---|---|
| `gofmt -l .` | exit 0, empty |
| `go test ./...` | exit 0, including final #42-only regression refinement |
| `go test -race ./...` | exit 0, including final regression refinement |
| `go test ./internal/compatibility ./internal/install ./internal/local ./internal/workitem -count=1` | exit 0 |
| `go test ./internal/local ./internal/compatibility ./internal/install ./cmd/lingo -run 'HistoricalRetirement\|RecognizedPOC\|Preservation\|WindowsInstallRelease' -count=1` | exit 0; affected CR-001–CR-004 locking, exact inventory, canonical manifest and Windows installer regressions reexecuted |
| `go test ./internal/install -run TestRecognizedPOCUpgradeRefusesUnprovedWorkItemMembership -count=1` | exit 0 after explicitly adding #42-only case |
| `go vet ./...`; `go build ./...`; `go mod verify` | exit 0, all modules verified |
| `./scripts/validate-repository.sh .` | exit 0; real Runtime behavioral scenarios explicitly skipped, not claimed |
| `./scripts/test-release-flow.sh` | exit 0; preflight, state machine, authority and workflow contracts pass |
| `./scripts/test-install-bootstrap.sh` | exit 0 on committed correction source `a72f5f0`; failures=0; POC transition, concurrent install and interrupted-upgrade resume pass |
| `bash -n scripts/test-upgrade-journeys.sh` | exit 0 |
| `./scripts/build-release-archives.sh --version 0.4.2-review.188.2 --output <private temporary candidate>` | exit 0 on clean source `a72f5f0` |
| `./scripts/test-upgrade-journeys.sh --candidate <0.4.2-review.188.2> --previous <v0.4.1> --previous <v0.1.1> --poc-binary <historical POC build>` | exit 0; 48 passing checks, failures=0, with final journey refinement |
| `gitleaks dir . --no-banner --redact`; staged Gitleaks and sensitive-file checks before the executable correction commit | exit 0, no leaks |
| `git diff --check` | exit 0 |

Delegation isolated regression implementation and independent adversarial review;
main session integrated and validated results. Review identified one missing
#42-only test precondition, corrected and reexecuted above; no remaining
production finding was identified. Final pushed-head CI and human re-review
remain separate gates. This correction does not claim new native Windows,
real Provider/Runtime, full-volume or cross-filesystem Evidence beyond the
checks actually executed here.

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

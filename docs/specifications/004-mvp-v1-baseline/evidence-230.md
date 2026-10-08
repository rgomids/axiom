# Issue #230 Evidence — User-managed resource lifecycle

> Historical Evidence: this records the original delivered revision. The
> 2026-10-08 [canonical consolidation decision](issue-230-canonical-consolidation.md)
> supersedes retention of operation-specific Runtime entrypoints; earlier
> tests and observations below remain historical, not current acceptance.


| Field | Value |
|---|---|
| Issue | [#230](https://github.com/rgomids/axiom/issues/230) (Epic [#15](https://github.com/rgomids/axiom/issues/15)) |
| Tasks | I230-T02–I230-T08 (I230-T01 delivered by PR #254) |
| Base | `main` at `e47890731e8dd070baa5397b643a9ca6c07f5629` (v0.8.0); `main` at `b69fa83` (v0.9.0: #247, #259) merged in after the independent review; validation re-run on the merge |
| Branch | `feat/230-resource-lifecycle` |
| Contract | [Lifecycle matrix](issue-230-resource-lifecycle-matrix.md) (frozen v1 plus §9 delivery reconciliation); [Specification 004](spec.md) FR-068–FR-084, AC-50–AC-61, J11–J16 |
| Status | Technically complete, pending human review. Not merged, not released, not accepted. |

Statements below are **confirmed** by a named test or command unless marked
*inference* or *not validated*.

## 1. Delivered Tasks

| Task | Delivered | Main code |
|---|---|---|
| I230-T02 | Machine-local `operational.json` (format 1): Project `active ⇄ archived`, locally disabled Integration keys; strict canonical codec; ADR-0007 publication with replacement checks; reviewed preview, exact local authority, revision CAS, deterministic no-op | `internal/projectapp/operational.go`, `internal/local/operational_store.go` |
| I230-T03 | Project archive/reactivate, `project list --include-archived` with per-Project status, `project show` with Repository availability, `project validate --project`, refusal of by-slug `project update` for installed Projects; authorized EDIT publication (Project edit, Repository attach/update/detach) with cross-store recovery | `cmd/lingo/project_lifecycle.go`, `internal/projectapp/{list,project_state,edit,edit_publish}.go`, `internal/local/{edit_publication,edit_recovery,recovery}.go` |
| I230-T04 | `integration list|show|validate|disable|enable|remove`; remove through the EDIT engine; stale local disable entries reported and clearable | `internal/projectapp/integration.go`, `cmd/lingo/integration.go` |
| I230-T05 | One admission guard (inspection / administrative / evolution) composed before #231 readiness in one gate; `project_archived`, `integration_disabled`; previews gated | `internal/projectapp/admission.go`, `cmd/lingo/admission.go` |
| I230-T06 | `work-item list|update|close|reopen`; reviewed `comment`; `complete` as compatibility spelling of close; GitHub PATCH title/body and state; truthful partial results | `internal/workitem/lifecycle.go`, `internal/githubissues/lifecycle.go` |
| I230-T07 | `workflow list` (Execution discovery without identity); cancel stays unsupported | `internal/workflow/service.go`, `internal/local/workflow_store.go` |
| I230-T08 | Runtime routing (`axiom-project`, `axiom-work-item`, `axiom-work-item-status`), `skill inspect` metadata from the same FlagSets, help, docs, Specification 004 amendment, end-to-end journey | `internal/cli/skill_inspect.go`, `internal/codexruntime/skills/*`, `docs/commands.md` |

## 2. Operation / effect / authority matrix (delivered)

| Resource | Operation | CLI | Runtime skill / operation / mode | Effect | Admission | Authority |
|---|---|---|---|---|---|---|
| Project | create | `project configure --slug …` | `axiom-project` / configure / create | M-P+M-L | A | `--preview-digest` + `--authorize-local` |
| Project | edit; Repository attach/update/detach | `project configure --project <sel> …` | `axiom-project` / configure / edit | M-P+M-L | A | `--project-id` + `--preview-digest` + `--authorize-local` |
| Project | list / show / validate | `project list [--include-archived]`, `project show --selector`, `project validate --project` | `axiom-project` / list, show, validate | R | I | none |
| Project | archive / reactivate | `project archive|reactivate --project <sel>` | `axiom-project` / archive, reactivate | M-L | A | `--preview-digest` + `--authorize-local` |
| Integration | list / show / validate | `integration list|show|validate --project <sel>` | `axiom-project` / integration / list, show, validate | R | I | none |
| Integration | disable / enable | `integration disable|enable --project <sel> --integration <key>` | `axiom-project` / integration / disable, enable | M-L | A | `--preview-digest` + `--authorize-local` |
| Integration | remove | `integration remove --project <sel> --integration <key>` | `axiom-project` / integration / remove | M-P+M-L | A | `--project-id` + `--preview-digest` + `--authorize-local` |
| Work Item | list / show | `work-item list|show` | `axiom-work-item` / list, show | R | I | none |
| Work Item | create / select | `work-item create|select` | `axiom-work-item` / create | M-X / M-O | E (uses `work-items`) | digest + `--authorize-external` / `--authorize-local` |
| Work Item | update / comment / close / reopen | `work-item update|comment|close|reopen` | `axiom-work-item` / update, comment, close, reopen | M-X(+M-O) | E (uses `work-items`) | `--preview-digest` + `--authorize-external` |
| Work Item | complete | `work-item complete` (compatibility of close) | not routed | M-X+M-O | E | `--preview-digest` + `--authorize-external` |
| Execution | list / status / evidence | `workflow list|status|evidence` | `axiom-work-item` / status (default, list) | R | I | none |
| Execution | start / advance / fact / resume | `workflow start|advance|fact|resume` | `axiom-work-item` / run | M-O | E | existing revision / Runtime preview / `--authorize-local` |
| Execution | reconcile | `workflow reconcile` | `axiom-work-item` / run / reconcile | M-X+M-O | E (uses `work-items`) | `--preview-digest` + `--authorize-external` |
| Execution | cancel | `workflow cancel` → `invalid_command` | unsupported | — | — | — |
| any | delete, Provider/credential cleanup, logout, MCP/Runtime uninstall | none | unsupported | — | — | — |

Semantic Runtime resolution is `allowed` for read-only modes and
`unambiguous-only` otherwise; ambiguous intent never selects archive,
reactivate, detach, disable, enable, remove, update, comment, close or reopen.
Routing never supplies authority (`TestRoutingAloneNeverGrantsAuthority`).

## 3. Acceptance traceability

| AC | Expected | Evidence (tests) | Result |
|---|---|---|---|
| AC-50 | Versioned lifecycle matrix | `issue-230-resource-lifecycle-matrix.md` §2–§4, §9; `TestEveryLifecycleOperationHasOneClassification` | Confirmed |
| AC-51 | Project maintainable without editing files | `TestResourceLifecycleJourneyWithoutPersistedFileEditing`; `TestProjectArchiveAndReactivateAreReviewedLocalAndReversible`; `TestProjectValidateBySelectorReadsTheRecordedSource`; `TestEditReplayPublishesNameProviderAndRepositoriesPreservingEverythingElse` | Confirmed |
| AC-52 | Preservation, conflict, no-op | `TestEditReplayPublishesNameProviderAndRepositoriesPreservingEverythingElse`; `TestEditReplayPreservesDocumentationBindings`; `TestEditReplayNoOpStaleAndDriftFailWithZeroWrites`; `TestEditPublicationRefusesConcurrentWritersWithoutOverwrite`; `TestApplyOperationalNoOpNeedsNoAuthorityAndWritesNothing`; `TestOperationalStaleRevisionConflictsWithoutWrite` | Confirmed |
| AC-53 | Repository associations without deleting data | `TestRepositoryDetachPreservesWorkingCopyAndDependentHistory`; `TestProjectShowReportsUnavailableBindingsWithoutFailingAndKeepsResolveStrict`; journey (sentinel unchanged) | Confirmed |
| AC-54 | GitHub Work Item lifecycle, no delete | `TestWorkItemLifecycleThroughTheCLIWithAStatefulProvider`; `TestLifecycleAdapterRequestAndResponseFixtures`; `TestCloseAndReopenWhenAlreadyThereAreNoOpsWithoutMutation`; `TestCommentAndCompleteRequireTheReviewedDigest`; `TestWorkItemDeleteIsNotACommand` | Confirmed (Provider faked) |
| AC-55 | Execution discovery; resume/cancel reuse contracts | `TestWorkflowListEnumeratesDeterministicallyAndFilters`; `TestWorkflowListIsInspectionUnderArchiveAndDisabledIntegration`; `TestWorkflowCancelStaysInvalidCommandWithZeroEffects`; existing workflow suites unchanged | Confirmed |
| AC-56 | Integration inspect/validate/update; disable/remove where supported | `TestIntegrationInspectionOnConfiguredUnconfiguredAndV1Projects`; `TestIntegrationDisableEnableLifecycleIsLocalAndBlocksOnlyProviderUse`; `TestIntegrationDisableIsIndependentPerStateRoot`; `TestIntegrationRemoveWorkItemsPreviewIsCompleteAndZeroWrite`; `TestIntegrationInventoryNamesTransportKindAndCredentialReferenceOnly` | Confirmed |
| AC-57 | Explicit failure before mutation | `TestAdmissionFailsClosedWhenTheIntegrationCannotBeAttributed`; `TestLifecycleTargetFailuresHappenBeforeAnyProviderCall`; `TestRemovingIntegrationFailsClosed`; `TestOperationalMalformedRecordIsNeverTreatedAsAbsent`; `TestUnsupportedDomainOperationsFailSafely` | Confirmed |
| AC-58 | Reviewed effects, exact authority; routing grants none | `TestApplyOperationalRequiresExactReviewedLocalAuthority`; `TestApplyEditPreviewsThenPublishesOnlyExactAuthority`; `TestUpdatePreviewsCurrentProviderDocumentAndRequiresReviewedAuthority`; `TestRoutingAloneNeverGrantsAuthority`; `TestProjectEditPublicationRequiresTheCompleteReplayTuple` | Confirmed |
| AC-59 | CLI and Runtime reach the same use case | `TestCanonicalSkillRoutingTableConvergesWithInspection`; `TestSkillOperationMetadataMatchesExecutableParser`; `TestCompatibilitySkillsConvergeWithCanonicalOperations`; `TestAdmissionDenialIsTheSameThroughTheCLIEntrypoint` | Confirmed (Runtime skills route to the same `axiom --json` commands; no real Runtime run) |
| AC-60 | No-op, preservation, missing, stale, unsupported, destructive denial, Provider mismatch, partial effect | rows above plus `TestProviderConfirmedCloseWithLocalFailureIsTruthfulPartial`; `TestTransientProviderFailureIsRetryableWithoutLocalWrite`; `TestEditPartialPublicationRequiresRecoveryAndFinalizes` | Confirmed |
| AC-61 | Matrices, fixtures, adapter tests, persistence/recovery | this document §2, §4; `TestOperationalInterruptedPublicationFailsClosedUntilRecovery`; `TestProjectEditPortableCommittedLocalFailedFinalizes`; `TestEveryV1WriterIsRecognizedByInventory` | Confirmed |

FR-068–FR-084 are covered by the same rows: FR-068 AC-50; FR-069/FR-074 AC-54,
AC-55, AC-57; FR-070 AC-51, AC-53–AC-56; FR-071/FR-072 AC-52; FR-073 AC-58;
FR-075 AC-51; FR-076 AC-53; FR-077 AC-54; FR-078 AC-55; FR-079 AC-56; FR-080/FR-081
AC-59; FR-082 journey portable-intent assertions and
`TestIntegrationInventoryNamesTransportKindAndCredentialReferenceOnly`; FR-083 AC-57;
FR-084 AC-60.

## 4. State, denial and recovery evidence

- **Wire fixture:** `{"formatVersion":1,"projectId":"<uuid>","projectStatus":"archived","disabledIntegrations":["docs","work-items"]}` (`TestOperationalRoundTripIsStrictAndCanonical`); revision = SHA-256 of exact wire; missing record = `absent`.
- **Fail closed:** unknown version/field, duplicate field, null, unsorted or duplicate keys, foreign Project, symlink, hard link, directory replacement before commit, interrupted publication (`TestOperationalDecoderFailsClosed`, `TestOperationalRejectsLinkedRecordAndDirectory`, `TestOperationalRefusesInstallationDirectoryReplacementBeforeCommit`).
- **Two machines:** archive and disable on one state root are invisible to another (`TestOperationalStateIsIsolatedBetweenStateRoots`, `TestProjectArchiveIsIndependentPerMachineStateRoot`, `TestIntegrationDisableIsIndependentPerStateRoot`).
- **Zero effects on denial:** byte-identical portable/state/Repository trees and an unused `gh` spy for every evolution entrypoint, preview included (`TestArchivedProjectDeniesEvolutionBeforeAnyEffect`, `TestDisabledIntegrationDeniesOnlyItsProviderUse`, `TestArchivedProjectAllowsWorkItemInspectionAndDeniesEveryEffect`).
- **Recovery:** operational interruption at F3/F5 → `restore_prior`, F6/F8 → `finalize_committed`; EDIT cross-store state prior/prior → restore, new/prior → finalize (writes the recorded local record), new/new → finalize, otherwise preserved review; readers return `recovery_required` until applied (`TestOperationalInterruptedPublicationFailsClosedUntilRecovery`, `TestExecutableEditReplayAndCrossStoreRecovery`).
- **Compatibility:** `installation.json` is not extended; `operational.json` is an additive v1 kind, supported and writer-tested; the stable-corpus freeze and the FR-026 compatibility-window declaration are owed at the first stable release that ships it. The canonical skill set changed (`axiom-project`, `axiom-work-item`, `axiom-work-item-status`); the replaced revision is in `sharedSkillHistory`, so owned upgrades converge (`TestEveryPublishedSharedRevisionStaysOwned`, `TestUpgradeFromSixSkillReleaseAddsOnlyDomainSkills`).

## 5. End-to-end journey

`TestResourceLifecycleJourneyWithoutPersistedFileEditing` drives, only through
the executable CLI: Project create (with locally authored Runtime policy) →
list/show/validate → edit + Repository attach → detach (working-copy sentinel
unchanged) → Integration list/validate → Work Item select/list/update → reviewed
Execution start → `workflow list`/status → archive (hidden by default, denials
for update/advance/close, inspection allowed) → reactivate → disable
(`integration_disabled`, no Provider call) → enable → close. The `gh` stub
records exactly two PATCHes (`title`, `state=closed`); portable intent never
holds operational state. Setup outside the journey: the machine-local Runtime
Profile configuration and the Provider stub.

## 6. Validation

Run on 2026-10-08, macOS darwin/arm64, Go 1.26.1, at the PR head:

| Check | Command | Exit |
|---|---|---|
| Formatting | `gofmt -l cmd internal` (empty) | 0 |
| Build | `go build ./...`; `GOOS=windows go build ./...`; `GOOS=linux go build ./...` | 0 |
| Static | `go vet ./...` | 0 |
| Modules | `go mod verify` | 0 |
| Tests | `go test ./... -count=1` | 0 |
| Race | `go test -race ./... -count=1` | 0 |
| Repository | `./scripts/validate-repository.sh .` | 0 |
| Sensitive files | `./scripts/check-sensitive-files.sh .` | 0 |
| ADR governance | `python3 scripts/check-adr-governance.py .` | 0 |
| Runtime skills | `bash scripts/test-codex-skills.sh` | 0 |
| Whitespace | `git diff --check e478907 HEAD` | 0 |
| Secrets | `gitleaks dir .` | 0 |
| After merging v0.9.0 | gofmt, native and Windows build, vet, `go test ./...`, `go test -race ./...`, `validate-repository.sh` | 0 |

Not executed: Windows/Linux test runs, real GitHub, real Codex/Claude Runtime.

## 7. Independent review

Three independent read-only reviewers (architecture/contracts,
security/state integrity, test/acceptance) reviewed `e478907..HEAD`. None
reported a Blocker.

| Finding | Severity | Resolution |
|---|---|---|
| A-1 Integration remove preview did not name retained Provider/Credential declarations (F-04) | Major | Fixed: `preserve_portable_provider` / `preserve_portable_credential` effects, digest-bound, tested |
| C-1 `--repository` filter assertions could not fail | Major | Fixed: cross-repository Work Item and Execution list tests |
| A-3 Work Item and Execution list treated detached keys differently | Minor | Fixed: both hide preserved history of a detached key until re-attach |
| A-4 Routing rows omitted `--integration <key>`; skill restated domain rules | Minor | Fixed |
| A-6 Comment replay posts again | Minor | Documented (skill, docs); no fence added |
| A-2 / B-2 No-op or partial close/reopen leaves a stale link | Minor | Next actions name `work-item select`, which refreshes the link under reviewed local authority; no new local-only effect |
| B-1 EDIT recovery finalize lacked the canonical-chain check | Minor | Fixed, tested (`TestEditRecoveryFinalizeRefusesReplacedChain`) |
| C-2 No `workflow.Service.List` unit test | Minor | Fixed (`TestListDiscoversExecutionsAndFailsClosed`) |
| C-3 No positive control for the Provider spy | Minor | Fixed (`TestProviderSpyRecordsAdmittedProviderCalls`) |
| A-5 Show/validate view assembly in `cmd/lingo` | Minor | Open: presentation assembly kept in composition; rules (status, availability probe) come from projectapp/local |
| C-4 Matrix document not tied to code by a test | Minor | Open: matrix reviewed manually; operation catalog is tested |
| C-5 `workflow list` human/oversize output untested | Minor | Open |
| A-7, C-6, C-7 | Nit | Catalog annotated; static ambiguity rules are the pre-existing pattern; reopen partial shares close's tested path |

## 8. Behavior changes and limitations

- `work-item comment` and `complete` now need the reviewed digest (F-01); a
  single-step `--authorize-external` returns `external_authority_denied` with the
  preview.
- `project update --slug` refuses installed Projects (`explicit_edit_required`).
- `project show` no longer fails on one unavailable Repository binding.
- `workflow reconcile` now requires an attributable `work-items` Integration.
- EDIT publication is limited to Projects whose recorded source is
  `<projects-root>/<slug>`; others can still preview.
- A GitHub Issue PATCH has no compare-and-set; Axiom re-reads after authority,
  leaving a small window (*inference*: no Provider support exists to close it).
- `work-item comment` is not idempotent; an uncertain failure is reported
  ambiguous and never retried automatically.
- An interrupted file in any Project directory fails the whole `project list`
  with `recovery_required` (existing all-or-nothing selection).
- *Not validated:* real GitHub, real Codex/Claude Runtime invocation, Windows
  and Linux execution (Windows build only), release acceptance.


## 9. PR #268 review corrections (2026-10-08)

Corrections against reviewed head `aad2508a65f198d4662bbfd2a1cf10bbb6330492`;
preceding review and validation tables remain historical Evidence.

- Smoke: the old repository JSON assertion failed at line 170. Structural JSON
  checks now cover `available`, `unavailable` and restored `available`; each
  `project show` succeeds and preserves the configured repository identity/path.
- Operational store: replace the POSIX-only round-trip assertion with the existing
  platform-aware `testfs.PrivateMode`. POSIX still requires mode 0600; Windows
  verifies owner and private DACL through `windowsfs.Check`. A shared-file
  regression checks refusal of both inspection and commit. Production security
  mechanisms are unchanged.
- Close/reopen: regression tests failed before correction when the Provider was
  already in the desired state but the local link was stale. Preview now names
  `update_local_work_item_link`; replay still requires the exact digest and
  existing explicit authority, and retains local revision CAS. No Provider
  mutation occurs during repair. Both states aligned remain a no-op.
- Regressions cover close/reopen authority denial, changed local revision, local
  conflict, and recovery after Provider success followed by local failure.

Validation on the local macOS host: `go test ./internal/local -run TestOperational
-count=1`, `go test ./internal/workitem -count=1`, `./scripts/dogfood-poc.sh`,
`go vet ./...`, `go build ./...`, `go mod verify`, and
`./scripts/validate-repository.sh .`, `go test -race ./...`, and
`go test -race ./internal/workitem ./internal/local -count=1` passed. Native
Windows/Linux execution is
left to the correction PR CI; no real Provider/Runtime invocation or human
acceptance is claimed.

# Issue #274 — Bound sequential workflow execution

Implements the accepted Specification 007 / ADR-0020 contract after #273's
authoring delivery (PR #294, v0.13.0). Baseline: `b87edb6`. The maintainer's
development request authorizes this bounded implementation. Review, publication,
merge, release and human acceptance remain separate decisions.

## Binding and execution

New production Executions require explicit Project selection. Admission resolves
the exact published identity, validates the canonical JCS/SHA-256 definition and
binds the complete Project/local observation to the reviewed Runtime start
preview. No Runtime or Provider dispatch is introduced.

Format 2 retains the canonical definition and required context bytes inside the
same private, fenced Execution record. `snapshotRef` addresses that retained
record. Atomic publication and read-back cover both snapshot and initial state,
avoiding an independently published snapshot dependency. Resume uses these bytes
without resolving the mutable Project source. Retirement inventory includes bound
definitions even after terminal completion. Artifact cleanup preserves every
stage ledger reference.

Execution follows the ordered stage array. Required inputs, prior-output/context
identities, typed outputs, every acceptance criterion and registered validator
must pass before a technical transition commits. The current registered policy
is `builtin-sdd-v1`; unknown policy implementations fail closed. Validators cannot
name shell commands. Human-review additionally binds an explicit authorized human
fact to the exact StageResult digest. Protected phase gates remain human facts;
technical completion and human acceptance remain distinct.

CR-001 reconciliation: configured `human-acceptance` fails closed with
`delivery_packet_required` until #277 supplies the exact validated delivery packet
and revision/digest-bound decision. Technical completion projects to `reviewed`;
`accepted` remains reserved in the existing vocabulary. Valid but unrelated
Evidence, final technical artifacts, or a forged acceptance event cannot grant
acceptance. Status/Evidence report the blocker and omit an unsupported acceptance
command. Format-1 acceptance remains unchanged.

Custom stage IDs map through declared phases/checkpoints to the existing ten
lifecycle values. Provider labels remain a projection of canonical state and
cannot authorize advancement. Status/Evidence expose identity, binding revisions,
stage contract/inputs, ledger, criteria, human facts, blockers and next command;
retained context bodies are not included in those DTOs.

## Compatibility and recovery

The existing `executions/v1` location now has separate strict format-1 and format-2
codecs. Legacy bytes, fixed progression and automatic Intake retain their existing
meaning; no migration or history rewrite occurs. Schema-1 WorkflowDefinition
revisions are retained for the configured executor's compatibility window.
Unknown future codecs/schema, missing snapshots, digest mismatch, incomplete
required context, forged ledger/history and changed immutable bindings fail
closed. Existing recovery inspection/publication protocols remain authoritative.

Configured records are bounded to 4 MiB, including a definition up to 1 MiB and
aggregate context content up to 1 MiB (64 sources). Stage-result files are bounded
to 64 KiB. Replay remains bounded; typed output content stays in ArtifactStore.
The CLI's configured completion window is 8 MiB so it can report the complete
bounded stage ledger. Legacy codec/output limits remain unchanged.

## Verification map

| Issue acceptance criterion | Evidence |
|---|---|
| A remains R1 after selection changes; B starts R2 | Domain revision-isolation test and `TestConfiguredExecutableRevisionIsolationAndAuthority` |
| Missing prerequisite/validator/human authority blocks before effects | Domain stage-result and exact human-review tests; executable missing-result and denied-authority checks |
| Status/Evidence identify contract, scope, stage and next step | Executable status/evidence assertions, binding and stage-ledger DTOs |
| Legacy resumes; missing/corrupt/future bindings fail closed | Existing legacy corpus/tests, new frozen format-2 corpus, codec/inventory/corruption tests and offline resume |
| Custom stages preserve the ten-value lifecycle vocabulary | Configured stages cover intake through reviewed; packet-free acceptance is denied pending #277. Legacy acceptance and canonical Provider projection tests remain unchanged. |

The registered output adapter tests also reject unknown policies, foreign stage/
output correlation and incorrect artifact categories. Local persistence tests
cover concurrent create convergence and F0–F8 interrupted
publication without a partial binding. Immutable-binding and transition-prefix
checks reject local history rewrites. Tests use private isolated homes, repositories,
skills and Runtime probes; no live Provider mutation or Runtime dispatch is needed.

Validation commands: `go test ./...`, `go vet ./...`,
`bash scripts/validate-repository.sh .`, and the repository sensitive-file check.
Their observed results are reported with the delivery; this document is a mapping,
not an assertion of external acceptance. #275/#276 own planning/dispatch,
#277 owns delivery/rework; those capabilities are not claimed here.

Windows verification uses a private temporary directory outside user Runtime
configuration ancestors. The full Go suite, `go vet`, sensitive-file checker and
its behavior tests pass. Automation registry coverage, issue-label policy and ADR
structure checks pass. `validate-repository.sh` cannot finish on this host: Git
materializes the tracked Claude symlinks as files and native symlink creation is
denied (WinError 1314). The Python validator unit suites also hit that restriction;
the automation registry's executable-bit negative test requires POSIX semantics.
The complete repository bootstrap therefore still needs its Linux CI run. These
checks were not weakened or skipped to manufacture a successful bootstrap result.

The first PR head's Linux/macOS CI exposed two POSIX-only fixtures that started
without explicit workflow selection. Their preparation now selects through the
public preview/apply operation. The executable lifecycle also supplies validated
typed stage results and human actors, then asserts packet-free acceptance denial.
The cross-platform executable test independently covers technical completion,
status/Evidence blockers and byte-preserving denial of unrelated valid Evidence.

Follow-up CI confirmed the full Go race suites and repository validators on Linux
and macOS, and the complete Windows job. The remaining failure was the dogfood
smoke script's historical implicit workflow and untyped advancement. It now uses
public selection preview/apply in its managed portable root, a registered
synthetic fixture that publishes through the production artifact store, typed
StageResults and explicit human actors. Its final assertion requires `reviewed`
and `delivery_packet_required`, with no revision increment on acceptance denial.
The complete smoke script passed in an isolated Linux WSL checkout with Go 1.26.0;
the fixture build/vet and automation-registry validation also passed.

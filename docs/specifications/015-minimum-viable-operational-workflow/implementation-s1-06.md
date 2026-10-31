# S1-06 final recovery — PARTIAL, not review-ready

Attempt `S1-06-final-recovery` on [AXM-42](https://linear.app/rgomids/issue/AXM-42),
under [human S1 authority](https://linear.app/rgomids/issue/AXM-42#comment-32f02fa5),
on the existing [draft PR #366](https://github.com/rgomids/axiom/pull/366).
This recovers interrupted work; it does not count the interrupted attempt as
completed. No S1-07, S2/S3, acceptance, merge, release or installed-state cutover.
Final-attempt disposition: `HUMAN_INTERVENTION_REQUIRED / NO_FURTHER_S1-06_ATTEMPTS`.
The single Linear completion receipt carries the actual final publication SHA
and its CI read-back; an Evidence commit cannot embed its own SHA.

## Recovery preflight and preservation

Verified host `ubuntu-arm-01`, physical worktree
`/home/grokbot/axiom-wt/axm-42`, owner `grokbot`, registered Git worktree,
origin `https://github.com/rgomids/axiom.git`, branch
`spec/axm-42-minimum-viable-workflow`. Initial local HEAD, remote branch and
`refs/pull/366/head` all matched `0ce0adc34d07ec56abfe949adff7e6034d45a201`.
The active implementation process command was `codex exec -m gpt-6.1-sol`.
User process and `/proc/*/cwd` inspection found this executor, its launcher and
owned commands; no separate agent writer was identified. This is an observation,
not a machine-wide exclusion guarantee. No runtime substitution or delegation.
Quota/reset assertions in the dispatch remain operator reports; no availability
conclusion is drawn from them. `rg` was unavailable; standard tools were used.

Before tests or edits, created a private snapshot outside the worktree with
0700 directory and 0600 top-level files: index and working-tree binary patches,
NUL-delimited complete status, original source/untracked/ignored file archive,
mode/size/hash manifest and private restoration scratch. Original index patch
was empty. No deletion intent. Snapshot manifest SHA-256:
`77676143552474fd27df81fe3699d8c48a77cb8b75ed26d184e2eb861360ab0d`.
The private path and archive contents are not published.

Restored archive bytes and modes into private scratch and compared every hash.
Separately seeded scratch with initial-HEAD tracked bytes, ran
`git apply --no-index --check` and applied the saved patches; restored tracked
hashes match the original dirty manifest. The original dirty tree was preserved.
No reset, clean, stash, checkout overwrite, force push or worktree replacement.

[Recovery inventory](recovery-s1-06.json) records all 8 modified tracked files,
3 untracked source files, ignored log and 2 ignored binary caches, with hashes,
modes, classifications and safety-scan limits. Classification:

- **KEEP:** recovered tests, bounded retained-artifact adapter, output subject
  and documentation; reconcile against final behavior.
- **FIX:** mapping, launcher and codec: input lowering independent of validators,
  completed-stage input correlation, output-check bounds/categories and retained
  identity; canonical lowercase definition digest. Existing store also needed
  immutable-history enforcement beyond revision CAS.
- **KEEP privately / exclude from publication:** inherited ignored log and binary
  caches. Original ignored files were neither read as instructions nor deleted.
  Source-pattern triage found no high-confidence secret; this is not gitleaks.
- **DROP:** none. No original work silently discarded.

## Delivered increments and safety review

`66181151dc3ce3125dadead428ccebf8e4d4e556` recovers typed output validation.
`3baa3118e26b5944f45253143fc1093a83da16a0` protects immutable attempt history.
Documentation repair: `2e6c4dd3b46c611c4021ece613925492436e02a7`.
Source increments were explicitly staged, diff-reviewed and sensitive-scanned, then pushed
normally after remote-ancestry verification. Remote branch and PR ref were read
back at `3baa3118e26b5944f45253143fc1093a83da16a0`; the PR ref initially lagged
the push and converged on the next read. No metadata/title/body mutation.

The installed `builtin-sdd-v1` artifact-schema/evidence subset reads a strict
bounded artifact-reference object, then the existing protected ArtifactStore.
It verifies complete typed metadata/content digest, successful outcome/category,
and exactly one matching Execution/stage/definition/output/attempt/repository
correlation. Evidence additionally requires evidence retention. A file's
existence or process exit zero cannot substitute for these checks. Missing
validator adapters refuse. Invalid collection preserves conservative unknown
effect semantics and stops successors; it cannot trigger automatic replay.

Workflow mapping now retains Work Item inputs from `target` regardless of
validators. Earlier-stage output inputs reach the child only from a completed
predecessor's successful latest attempt, with an unambiguous output and the same
repository. Missing required inputs, unknown/pending results, duplicate output
producers and wrong repositories refuse. Unimplemented project-context/artifact
inputs refuse before creation. No new Provider read/write or agent authority.

The aggregate checks duplicate artifact names, retained identity for configured
checks, exact check/output correspondence, category and input bounds. Rooted
no-symlink reads, bounded regular-file collection and protected publication are
reused. No artifact-content copy enters portable configuration or diagnostics.

Store CAS now additionally refuses changes to the selected Workflow, linkage,
frozen invocation/provenance/authority, prior attempt identity or terminal facts,
and refuses removing prior attempts. Process identity/start facts cannot be
replaced once observed. Existing gate/cancel facts cannot be rewritten. A human
may append abandonment to an unresolved unknown without changing its effects or
observations. Ordinary retry continues to use the frozen invocation and current
authority; new selection needs another explicitly selected attempt.

Focused tests cover wrong correlation/category/digest/outcome, duplicate and
missing references, invalid JSON/unknown keys/size/cancellation, missing adapter,
unresolved context, prior-stage result ambiguity and persistence-history edits.
Store negatives use internally valid candidate records at the current revision
and assert rejected saves preserve exact stored bytes. Human unknown disposition
still succeeds without rewriting the unknown result. Tests use isolated local
stubs, never actual Codex/Claude inference.

Review finding **major / blocking**: S1.5 exact result-bound human review,
authorized command checks, complete schema-2/context lowering and S1.7 retirement
remain incomplete. Refusal is safer than omission, but is not delivery of those
acceptance criteria. Required architecture/repository checks also remain FAIL.
No independent review or review readiness is claimed.

## S1 task matrix

| Task | Actual status | Remaining acceptance |
| --- | --- | --- |
| S1.1 | Accepted ADR-0037 inherited, preserved | No new material architecture decision |
| S1.2 | Canonical codec/store retained; validation references and immutable-history checks added | Structural old-owner retirement still open |
| S1.3 | Existing precedence, optional Profiles/defaults and frozen retry preserved and tested | Legacy predictive policy/auth consumers remain |
| S1.4 | Existing fence/process/cancel/unknown/retry engine preserved | Old operational ownership not fully retired |
| S1.5 | PARTIAL: typed artifact/Evidence subset and target/prior-stage inputs | Human review, authorized argv, optional validated outputs, project context/artifact inputs, full schema-2 limits |
| S1.6 | Existing private ingress preserved | S2 routing/parallelism not implemented |
| S1.7 | NOT COMPLETE | Legacy writers/readers/supervisors remain; no additional family retired here |
| S1.8 | Existing isolated cutover negatives PASS | Cannot certify structural cutover while S1.7 remains open; actual cutover NOT_RUN |

## Validation ledger

Source bytes are exactly `3baa3118e26b5944f45253143fc1093a83da16a0` for final
Go validation. Binary source patch SHA-256 against initial HEAD, restricted to
`internal cmd`: `d7d32041acc0d1d37c0dfbc2f52aa0f7811d2e2b0c66a574567ef6b5b93cc83e`.
Documentation/inventory reconciliation afterward does not change these bytes.

| Exact command / check | Result |
| --- | --- |
| `go test ./internal/local ./internal/workflow ./internal/stageexecution ./cmd/lingo -run 'TestExecutionOutput\|TestRuntimeLauncher\|TestMapWorkflowDefinition\|TestPublicWorkflowRun' -count=1 -timeout=5m` | PASS: local 1.466s, workflow 0.103s, stageexecution 0.101s, cmd 19.027s |
| `go test ./internal/local -run 'TestExecutionOutput\|TestExecutionStore' -count=1 -timeout=5m` | PASS, 2.877s; same final store source |
| `go test ./internal/local -run 'TestCutover' -count=1 -timeout=5m` | PASS, 0.508s; isolated state only |
| `go build ./...` | PASS, exit 0 |
| `go vet ./internal/... ./cmd/...` | PASS, exit 0 |
| `go test ./... -count=1 -timeout=30m` | PASS, exit 0: cmd 275.413s, local 140.109s, workflow 114.496s, stageexecution 12.548s, codexruntime 17.241s |
| `bash scripts/test-docs-offline.sh` | PASS after installed-link repair, isolated linux-arm64 archive; corpus links/integrity/negative tamper checks passed. Qualification PARTIAL: network/source denial and native Runtime NOT_RUN; pinned v1 installer UNVERIFIED |
| `bash scripts/check-go-quality.sh format` | PASS, Go 1.27.3 linux/arm64 |
| `go run scripts/check-architecture.go domain` | PASS, 12 files |
| `go run scripts/check-architecture.go application` | FAIL, `non-allowlisted application import: strconv`, exit 1 |
| `bash scripts/test-check-project-domain.sh` | PASS |
| `bash scripts/test-check-projectapp.sh` | FAIL, `domain profile accepted an application-only allowance`, exit 1 |
| `python3 scripts/check-adr-governance.py` | PASS, 38 ADRs |
| `python3 scripts/check-automation-registry.py` | PASS, 99 governed surfaces; no durable automation added |
| `bash scripts/check-claude-bootstrap.sh --skill-adapters .` | PASS, exit 0 |
| `bash scripts/check-sensitive-files.sh .` | PASS |
| `bash scripts/check-sensitive-files.sh --staged .` | PASS before source commits; Evidence staged scan recorded at publication |
| `bash scripts/validate-repository.sh .` | FAIL, inherited ignored `.s1-03-lingo-test.log`; preserved unchanged |
| `git diff --check`; JSON parse of three inventories and Evidence local links | PASS |
| Race / local Staticcheck / gitleaks | NOT_RUN: gcc, staticcheck and gitleaks absent; nothing installed |
| Source CI at `3baa3118e26b5944f45253143fc1093a83da16a0` | FAIL: [run 41237484569](https://github.com/rgomids/axiom/actions/runs/41237484569), solely installed documentation link; Go tests, vet, pinned Staticcheck and build passed |
| Final publication-SHA CI | Separate actual-SHA read-back in the final Linear receipt, never inferred from source CI |
| Live vendor tests, actual installed cutover, external Provider effects, merge/release/deployment/acceptance | NOT_RUN, outside authority |

Earlier runs are diagnostic: the first pattern omitted the local validator test
(`no tests to run` for that package); later focused commands actually ran it.
After stage-output support, the old public negative expected all validators to
refuse and failed; it now explicitly configures unsupported human-review and
asserts zero Execution writes. A new context-negative fixture used schema 1,
where project-context is not admitted; it failed fixture schema validation.
Changed only that fixture to schema 2 and reran successfully. No production
assertion or checker allowlist was weakened. Source CI exposed an introduced
installed-bundle link failure: `docs/text/workflow.md: unresolved link ../commands.md`.
Corrected that recovered link to the established generated `cli.md` reference;
no product source or test was changed by the documentation repair. Overlapping intermediate diagnostic
runs do not establish final-source PASS. No vendor CLI test invocation occurred;
the required Haiku/Luna pins were therefore not exercised or presumed available.

## Deletion and consumer evidence

[Actual deletion measurement](deletion-s1-06.json) uses the approved candidate
union plus every added Go file, including replacements anywhere under internal
and cmd. Current production: 81 files / 20,603 physical lines / 254 exported
types / 319 exported functions. Baseline: 88 / 21,872 / 287 / 351. Delta:
**-7 files, -1,269 lines, -33 types, -32 functions**. Tests: 76 files / 15,537
lines / 411 functions, versus 84 / 18,320 / 481; delta -8 / -2,783 / -70.
Expanded union including modified existing files outside the envelope:
production -11 files / -1,678 lines / -37 types / -51 functions; tests -11 files /
-3,113 lines / -77 functions. Physical lines include comments/blanks; declaration
counts are not reachability. S3 numerical goals are not S1 gates.

This recovery **adds** 2 production files / 319 production lines / 3 exported
types / 1 exported function, and 1 test file / 387 test lines / 8 tests versus
S1-05. It makes no new structural-deletion claim.
[Consumer inventory](consumers-s1-06.json) refreshes the prior exact regex patterns
at source SHA: 253 production matches remain. Its 91 declaration subset is
defined explicitly and is narrower than S1-05's 117 relevant declarations;
that difference is not a claimed reduction.

Remaining operational chains: lifecycle command dispatch → workflow.Service →
WorkflowStore; delivery readiness → Graph/Single stores and authority; child
publisher → persisted Graph/Single/Workflow resolver/leases; shared stageStores
composition; predictive runtime policy/auth consumers; old codecs and stores.
These are unfinished mandatory S1.7 work, not justified permanent survivors.
The reused OSProcessRunner is an indispensable low-level survivor for process
identity, cancellation and unknown-stop limits, not a second execution owner.
Independent delivery/rework effects remain S3-owned and were not deleted early.

## technical_decisions[]

- **TECHNICAL_LOCAL / recovery:** private patches + original byte archive rather
  than stash/reset. More disk usage, preserves staged/unstaged/untracked/ignored
  distinctions and demonstrates restoration without modifying the source tree.
- **TECHNICAL_LOCAL / validators:** reuse strict retained ArtifactStore validation
  and exact correlation rather than invent a new validator/policy store or accept
  file existence. Unsupported controls refuse; that refusal remains incomplete AC.
- **TECHNICAL_LOCAL / inputs:** freeze and deliver supported target/prior-output
  references; refuse unresolved context rather than silently omit it. This narrows
  executable input support until the remaining mapper is implemented.
- **TECHNICAL_LOCAL / provenance:** add history comparison at the store CAS
  boundary rather than rely only on callers/codec validity. Whole-record compare
  cost is bounded; human abandonment appends without rewriting uncertain effects.
- **TECHNICAL_LOCAL / review scope:** preserve old safety mechanisms until their
  consumers are ported; report mandatory retirement as unfinished, never count
  hiding commands as deletion. No material baseline change, no new ADR.
- **Delegation:** direct implementation; no verified strictly lower capability
  hierarchy used. DESIGN NOT_APPLICABLE for this CLI-only scope.

## deferred_findings[]

- Application architecture failure is inherited: `active_workflow.go` and both
  checker/fixture files are unchanged from initial HEAD. Its additional
  workflowdefinition dependency is also outside the current application allowlist.
  No allowlist widening. The required check remains FAIL; remediation must
  reconcile the bounded application read model and accepted inward boundaries.
- Baseline-only extraction of `check-architecture.go` and `test-check-projectapp.sh`
  into temporary scratch reproduces the UTF-8 fixture failure (exit 1). An isolated
  pure `encoding/hex.EncodeToString` domain fixture likewise reproduces the
  inherited hex rejection (exit 1); the actual current domain tree passes.
  No claim that the historical hex issue is a current product failure.
- Ignored inherited log: 36,764 bytes, raw SHA-256
  `d6ee23dd4c98846202ec0a08f20bfb2b95aa17ea4453af2d551474f63c73cf28`.
  Repository validation refuses its path; this is not proof of a secret in it.
  It and the binary caches remain original and privately preserved.
- PR metadata still says documentation-only. No separately verified metadata
  authority was established, so title/body remain untouched. Proposal below.
- Historical Model Advisor synthetic failure and builtin SDD drift remain prior
  deferred findings; neither repaired nor reclassified by this attempt.

## remaining_work and rollback boundary

1. Complete exact result-bound human review with retained Evidence, optional
   authorized argv checks, optional output handling, context/artifact inputs and
   schema-2 instruction/aggregate bounds. Gates are not a review substitute.
2. Port canonical readiness/lifecycle/child consumers and retire A–H/K writers,
   readers, supervisors, stores/codecs and retired-only tests. Preserve CAS/fence,
   identity/cancel, unknown effects, no replay, containment and actual authority.
3. Reconcile CLI/help/installed contract after that retirement; certify S1.8 only
   after S1.7. Existing isolated negatives cover active/unknown/incomplete
   disposition, held lease, links/unreadable/unexpected entries, interrupted
   publication, exact-byte backup and old-format refusal. No real switch occurred.
4. Resolve required failing checks and review the final source/CI Evidence.
   This is the last permitted S1-06 attempt; no executor retry is scheduled.

Before a real switch, separately authorized operators must stop new writers,
inventory actual process/effect uncertainty, verify private raw backup and safely
stop owned processes or BLOCK. Backup does not establish stopped work. Before
switching, restoring a binary/state also needs safe-stop/disposition authority.
After new effects exist, preserve both namespaces and prefer a forward fix: never
point an old binary at new state, resume old attempts or replay unknown effects.
Recovery snapshots restore source bytes; they do not reverse external effects.

Proposed title: `feat(workflow): AXM-42 canonical Execution — partial S1 recovery`.
Proposed body: `Recovers interrupted retained artifact/Evidence validation,
correlated target/prior-stage inputs, and immutable invocation/result history
at the canonical store CAS boundary. S1 remains partial: result-bound human
review, authorized commands, full schema/context mapping and A–H/K retirement
are unfinished. See implementation-s1-06.md and its inventories. No real cutover,
vendor E2E, S2/S3, acceptance or merge.` Preserve `Related-Issues: none` and
`Completes-Issues: none`.

Callback is one structured same-AXM-42 event after final publication/CI read-back.
No continuing orchestration or background monitoring is claimed. Final checkpoint
is `AWAITING_EXECUTION_REPORT` with human intervention and no further S1-06 attempt.

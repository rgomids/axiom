# Evidence — MVP Slice S4: Local Execution and GitHub Projection

## Claim and authority boundary

This record covers the explicitly authorized Specification 004 Slice S4 only:
T10–T13. Work started from `main` revision `c37a297`, the PR #85 merge. T14/S5+
implementation, Issue closure, prerelease/release publication, merge, and final
MVP acceptance were not authorized and were not performed.

The deterministic implementation and fake-Provider observations below are
complete. The mandatory bounded real GitHub label/comment observation remains a
**Human decision required** gate. No real Provider projection is claimed here,
and technical checks do not imply human acceptance.

## Delivered behavior

### T10–T11 — local Execution truth

- One opaque stable Execution ID binds exact Project, Repository, linked Work
  Item, Runtime, workflow/format versions, provenance, stage, revision, bounded
  transition history, references, projection ledger, and terminal facts.
- Protected machine-local state lives under `executions/v1`; closed decoding,
  strict identity checks, private modes, expected storage revision, and the shared
  publication protocol reject malformed, newer, stale, mixed, or uncertain state.
- Duplicate equivalent start returns the same lineage. Conflicting scope/version
  fails. Every transition requires its exact Execution revision and current stage.
- The fixed path is intake → specification → clarification → plan → tasks →
  implementation → review → evidence → reconciliation → completion. Failed
  transitions stay at the current stage and require exact-revision resume.
- Artifact references bind artifact ID, Execution ID, and SHA-256. Evidence
  references bind a confined repository-relative regular file and SHA-256, with
  no symlink traversal and a 1 MiB read bound.
- Completion is local truth only. It does not close or otherwise infer state from
  the GitHub Issue.

### T12–T13 — post-commit projection and convergence

- `workflow reconcile` first reads bounded repository labels, exact Issue state,
  Issue labels, and comments, then returns a digest-bound preview. Preparing the
  preview performs no mutation.
- Exact authority binds Work Item, committed Execution revision, stable projection
  key, current stage label, provenance-marked bounded comment, observation, and
  ordered effect set.
- Projection creates/adds the current `axiom:stage:<stage>` label, removes only
  obsolete Axiom stage labels, preserves foreign labels/content, and posts at
  most one comment for the Execution revision. It never advances local truth.
- Intended effects are persisted before Provider calls. Every successful or
  ambiguous mutation is reinspected before confirmation is recorded. Replays
  reconcile the same stable key; confirmed Provider effect followed by local
  bookkeeping failure is canonical `partial`.
- Provider execution remains bounded by the existing absolute `gh` process,
  15-second deadline, 256 KiB captured-output limit, JSON stdin, strict response
  identity/state/content checks, and closed error taxonomy.
- Deterministic process barriers prove one winning local revision without using
  sleeps as the race oracle. F0–F8 injection preserves old-or-new truth or
  `recovery_required` according to the shared publication protocol.

## Deterministic observations

| Condition | Evidence seam | Observed truth |
|---|---|---|
| duplicate equivalent/conflicting start | application store spy | one stable lineage; conflict cannot replace it |
| skipped, stale, duplicate, failed, resumed transition | workflow contract matrix | fixed order; exact revision; replay convergence; failure does not advance |
| malformed/newer/incoherent history | closed codec/state validator | rejected before use |
| artifact/Evidence reference mismatch or traversal | real local stores/files | transition denied; no state change |
| two writers from one revision | two executable processes and barrier | one save; one conflict; one authoritative revision |
| crash at staged/committed publication boundaries | killed helper process | reader fails closed with `recovery_required` |
| F0–F8 local-store faults | deterministic publication hooks | prior truth or preserved uncertain marker; no mixed reader |
| missing/stale projection authority | preview digest tests | zero Provider mutation |
| foreign and obsolete stage labels | fake Provider ledger | foreign label preserved; only Axiom-owned obsolete label removed |
| ambiguous Provider response | apply-then-error fake plus reinspection | effect confirmed once; no duplicate mutation |
| success response without observable effect | apply fake plus bounded read | retryable reconcile; effect not falsely confirmed |
| Provider unavailable | inspect failure | local transition remains authoritative and unchanged |
| confirmed comment plus local save failure | Provider/store fault seams | canonical `partial` with preserved intended ledger |
| installed-binary dogfood | isolated Project/state/fake `gh` | complete local workflow plus one authorized fake projection converges |

## Commands executed

Deterministic validation ran on macOS 27.0/arm64 with Go 1.26.1 from branch
`agent/s4-workflow-provider-projection`, based on `c37a297cb3ee698ca7da0cce51ea98acf2f776f6`.
Every command below exited `0`:

```bash
go test ./internal/workflow ./internal/local ./internal/githubissues ./cmd/lingo
bash scripts/dogfood-poc.sh
go test ./...
go test -race ./...
go vet ./...
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
git diff --check
```

`gitleaks detect --source . --no-git --redact --no-banner` also exited `0` and
reported no leaks. `go mod verify` reported `all modules verified`.

The final isolated installed-binary dogfood observation recorded these fields;
the projection key is written separately below to keep its digest classification
explicit during secret scanning:

```json
{"evidenceVersion":1,"evidence":"axiom_e2e_dogfood","cwdIndependent":true,"globalSkillCount":5,"workItem":7,"executionId":"5aeddfc3-8de9-4cba-bb0f-aef1ce3ac07d","revision":13,"workflow":"completed","projectionKey":"recorded below","projectionDigest":"8896312d16d71f82c4d21e4815233e3fc060d96ead08d63d8524e0d477aa7860","binarySha256":"0020fc807e99c1acf540b0984d33f5563e18bba4cdaa07e015b04c7503fe1502","workflowSha256":"185b61ae5ff7558b90ba28b016de6386b90f272e50419074947e0aeea9152190","result":"pass"}
```

That run used this exact local transition matrix:

| Revision | From | Outcome | To / status |
|---:|---|---|---|
| 1 | start | — | intake / active |
| 2 | intake | pass | specification / active |
| 3 | specification | pass | clarification / active |
| 4 | clarification | pass | plan / active |
| 5 | plan | pass | tasks / active |
| 6 | tasks | pass | implementation / active |
| 7 | implementation | fail | implementation / interrupted |
| 8 | implementation | resumed | implementation / active |
| 9 | implementation | pass | review / active |
| 10 | review | pass | evidence / active |
| 11 | evidence | pass | reconciliation / active |
| 12 | reconciliation | pass | completion / active |
| 13 | completion | pass | completion / completed |

At revision 6 the fake Provider before-state contained no stage label and no
projection comment. Exact fake-Provider authority converged to one
`axiom:stage:implementation` label and one comment carrying projection marker
`a721ff5eef76fa618e9032eaeeff5b7fc0137ad291798b9556ecacb68d90a93d`.
The persisted local projection record contained the ordered intended effects and
the same confirmed set. Reinspection after every fake effect and a final replay
preview observed zero remaining effects. These are deterministic fake-Provider
facts, not the pending real GitHub observation.

## Remaining mandatory observation

No real GitHub mutation has been executed for S4. Before one bounded observation,
the operator must select the exact Issue target and review a fresh
`workflow reconcile` preview containing repository, Issue, Execution revision,
projection key, exact label/comment effects, comment text, cleanup ownership,
and retry behavior. Separate exact authority is required for that preview digest.
The observation must prove before/after state and replay convergence without
closing the Issue or removing non-Axiom content. Until then, T12/T13 real-provider
Evidence and the S4 PR remain pending.

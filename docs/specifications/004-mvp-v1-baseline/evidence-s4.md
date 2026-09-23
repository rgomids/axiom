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

- `workflow reconcile` first reads and binds exact Provider, repository/resource,
  Issue external ID, URL, state, bounded repository labels, Issue labels, and
  projection-comment presence, then returns a digest-bound preview. Preparing
  the preview performs no mutation. An external OPEN/CLOSED state change alters
  the observation and preview digests, invalidating stale mutation authority.
- Exact authority binds Work Item, committed Execution revision, stable projection
  key, current stage label, provenance-marked bounded comment, observation, and
  ordered effect set.
- Projection creates/adds the current `axiom:stage:<stage>` label, removes only
  obsolete Axiom stage labels, preserves foreign labels/content, and posts at
  most one comment for the Execution revision. It never advances local truth.
- Intended effects are persisted before Provider calls. Every successful or
  ambiguous mutation is reinspected before confirmation is recorded. Replays
  first reconcile previously Intended but unconfirmed effects from the current
  Provider observation, persist Confirmed/Complete, and only then determine
  whether new Provider effects remain. A confirmed effect followed by local
  bookkeeping failure is canonical `partial`; retry converges the ledger without
  repeating the mutation or duplicating the projection comment.
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
| Issue OPEN after preview becomes CLOSED | exact fake observation identity/state | digest changes; old authority denied; zero Provider effects |
| foreign and obsolete stage labels | fake Provider ledger | foreign label preserved; only Axiom-owned obsolete label removed |
| ambiguous Provider response | apply-then-error fake plus reinspection | effect confirmed once; no duplicate mutation |
| success response without observable effect | apply fake plus bounded read | retryable reconcile; effect not falsely confirmed |
| Provider unavailable | inspect failure | local transition remains authoritative and unchanged |
| confirmed comment plus local save failure | Provider/store fault seams | canonical `partial`; retry records Confirmed, sets Complete, and does not repost comment |
| installed-binary dogfood | isolated Project/state/fake `gh` | complete local workflow plus one authorized fake projection converges |

## Commands executed

Deterministic validation ran on macOS 27.0/arm64 with Go 1.26.1 from branch
`agent/s4-workflow-provider-projection`. Initial S4 delivery was based on
`c37a297cb3ee698ca7da0cce51ea98acf2f776f6`; review remediation was validated
from PR head `02facfb0e6532751578451236c589346521cc38d` plus the focused working-tree
changes recorded by this Evidence. Every command below exited `0`:

```bash
go test ./internal/workflow ./internal/local ./internal/githubissues ./cmd/lingo
bash scripts/dogfood-poc.sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
go mod verify
go test -shuffle=on -count=10 ./internal/workflow ./internal/local ./internal/githubissues
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh .
./scripts/check-sensitive-files.sh --staged .
git diff --check
```

`gitleaks detect --source . --no-git --redact --no-banner` also exited `0` and
reported no leaks. `go mod verify` reported `all modules verified`.

The final isolated installed-binary dogfood observation recorded these fields;
the projection key is written separately below to keep its digest classification
explicit during secret scanning:

```json
{"evidenceVersion":1,"evidence":"axiom_e2e_dogfood","cwdIndependent":true,"globalSkillCount":5,"workItem":7,"executionId":"307c7248-4d99-447d-902c-a98de408a4e6","revision":13,"workflow":"completed","projectionKey":"recorded below","projectionDigest":"8d2b3da0fa3845894c165e79a271e047e3c7b0a201fed46413841b242503f41f","binarySha256":"259e647e00a102aef8602fe3bd2e68dcaa48da3924031769778b2c1a8b6383dd","workflowSha256":"fa73975a6480ab691f0a83be2cc44ba4d11aad1979545032951cb46696e9cf08","result":"pass"}
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
`d0d66d7194019d8a82c9512cbb9675b9e54f0b57a3aea9713f6a156ad598eca4`.
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

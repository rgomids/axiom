# Evidence — AXM-9 / #277, CR-001

Date: 2026-10-17. Scope: IMPLEMENT correction on existing Draft
[PR #323](https://github.com/rgomids/axiom/pull/323), branch
`docs/axm-9-delivery-plan`, for
[CR-001](https://github.com/rgomids/axiom/pull/323#discussion_r4302811430).

## Baseline and contract

The clean local HEAD, fetched branch and PR HEAD all matched reviewed commit
`1ad5d80ca865ff67ed199e0d8ff5125b8c5dcc38`; no subsequent commits existed.
[WF-011 / AC-011](spec.md) and [T1](tasks-277-reviewed-delivery.md#t1--produzir-e-inspecionar-packet-com-readiness-verificável)
require correlated execution evidence. Specification 007 requires 1–32 agents
per stage; neither its retained definition nor the accepted Plan/Tasks provides
a manual stage without an agent. ADR-0027 separates workflow checkpoints from
single-stage execution evidence. A checkpoint alone cannot replace that run.

## Correction and regression evidence

- The CLI evidence reader blocks an absent single-stage run. Incomplete attempts,
  unverified authority and missing qualifying coordination remain inspectable
  blockers. The retained run digest and attempt remain visible when available.
- The reader treats `SingleStageReadiness` as dispatch readiness: its terminal
  `succeeded` result permits delivery verification rather than incorrectly
  blocking it. Pending retry/cancellation and non-success remain blocked.
- The existing canonical classifier selector verifies the latest qualifying
  record against Execution/attempt/Workflow revision, digest and success.
- The packet composer independently requires a single-run digest and successful,
  identified, result-qualified attempt for a configured single-agent stage.
  Existing graph integration, validators, waivers and packet codecs are retained.
- `TestDeliveryValidatedOutputsWithoutAttemptsRemainInspectable` and
  `TestDeliverySingleStageRequiresQualifiedAttempt` cover valid outputs/checkpoints
  without execution, absent digest/attempt, failed/unqualified attempt and absent
  coordination reference.
- `TestDeliveryWithoutAttemptsRefusesAllPublicationEffects` verifies zero
  publication-store writes, pushes and PR creations for local and GitHub targets.
- `TestDeliveryCLIValidatedCheckpointsWithoutRuns` advances valid checkpoints
  through the installed CLI with no Runtime attempts, inspects `Ready=false`,
  refuses `delivery apply` and verifies no publication record or Runtime call.
- `TestInstalledSyntheticSingleStageJourney` verifies the real protected stores
  after fake Runtime dispatch, then removes/restores authority and coordination
  to verify their respective inspectable blockers.
- `TestInstalledSyntheticGraphStageJourney` now dispatches each single-agent
  stage through the fake Runtime before advancing checkpoints. With the existing
  real synthetic multiagent integration, all gates produce `Ready=true`; packet
  digest retention, local publication/replay, rejection and linked rework remain
  exercised. Each single-stage test manager has its own authorized workspace
  root. Unit port projections are explicitly synthetic; the installed journey
  supplies the independent store/authority/coordination verification.

## Validation

Environment: Linux, Go 1.26.0 selected with `GOTOOLCHAIN=auto` (host Go 1.24.4).

- PASS: `GOTOOLCHAIN=auto go test ./internal/workflow ./cmd/lingo -run 'TestDelivery|TestInstalledSynthetic(SingleStageJourney|GraphStageJourney)$' -count=1`.
- PASS: `GOTOOLCHAIN=auto go test ./...` (complete suite, including the installed CLI journeys).
- PASS: `GOTOOLCHAIN=auto go vet ./...` and `GOTOOLCHAIN=auto go build ./...`.
- PASS: `./scripts/validate-repository.sh .` and `git diff --check`.
- PASS: `./scripts/check-go-quality.sh format`, `./scripts/check-go-quality.sh tidy`
  with the selected Go 1.26.0 toolchain directory prepended to PATH; and
  `GOTOOLCHAIN=auto go mod verify`.

- PASS: `./scripts/check-go-quality.sh staticcheck` and
  `python3 scripts/test-go-quality.py` (10 tests), using CI's pinned
  `staticcheck@v0.8.1`, installed only under `/tmp/axm9-quality-bin`, and Go 1.26.0
  on PATH. The initial quality regression run failed because the host PATH used
  Go 1.24.4 and had no Staticcheck; the corrected environment passed.
- PASS: `./scripts/test-delivery.sh`. Its initial historical-commit check failed
  because the checkout was shallow. `git fetch --unshallow origin` supplied the
  existing history; the unchanged test then passed.

The remote CI outcome for the pushed commit is recorded in the correction
handoff. The repository validator explicitly skips maintainer Runtime
behavioral scenarios unless requested. Tests use synthetic Runtime/Provider
fixtures; no real vendor, publication, production or operational rework was run.
Plan, Tasks, Specification and accepted ADRs are preserved. This correction
records neither human acceptance nor merge/release authority.

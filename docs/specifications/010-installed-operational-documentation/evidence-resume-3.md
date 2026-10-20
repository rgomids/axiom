# Implement Evidence 010 — resumption 3 checkpoint

**P1 PARTIAL; G1 submitted BLOCKED; D2 candidate unqualified.**
[Gate report / C2 / recoverable handoff](gate-g1-resume-3.md) is the current fan-in.
No Implement completion or independent review claimed. T05–T16 BLOCKED/NOT_RUN;
G2/G3, AC-01–AC-08 and full V1–V9 NOT_RUN.

## Preflight and authority

- Same branch/Draft PR #335, clean entry worktree and matching remote/local head
  `790b1a2e648146700a63b42c70e18669861f0c8c`. No duplicate Execution/PR/dispatch.
  Canonical Execution remains unknown; existing records do not reconstruct it.
- Read full human D1/D3 and Tasks approval comments, checkpoint 6301868275 and
  explicit resumption-3 C1 in this task. Latest PR comment at recheck remained that
  checkpoint, without conflicting new authority. Linear re-read In Review,
  blockedBy empty, blocks AXM-39/AXM-12 unchanged. No private provider prose copied.
- Entry-head CI 39500006601 and delivery metadata SUCCESS verified live. Experimental
  head CI is separate from native jobs; fa59969 CI cancelled, inconclusive. Final
  head/CI read-back belongs to the delivery comment, never inferred from old runs.
- Read applicable implementation/orchestration/security/automation policies,
  affected owners and ADR-0033/0005/0016/0018/0021. Runtime/skill versions and
  historical ownership preserved. No product package, dependency, public command,
  installer, builder or release workflow change. No secrets/infrastructure effects.
- No delegation: no demonstrable strictly inferior model ordering in this runtime;
  Orchestrator performed probes/integration. No independent review claimed.

## E01 — C1 and development contract/oracles

[C1 literal acceptance](acceptance-resume-3.md) and appended Plan/Tasks amendment
preserve accepted history. Initial quotas remain development/security budgets;
public freeze requires full T05/T06 corpus/reference measurement and coverage.
Incomplete coverage/excess blocks T07/packaging; no trimming or silent increases.
v0.1.0 remains a migration-floor objective, never qualified support. Prior measured
105 leaves/549 flags, 160,546 help bytes, 25 source baselines, 11 retained receipt
fixture tags and passive three-archive v0.1.0 integrity remain qualified historical
Evidence, not final corpus/native migration proof.

[Contract closure](contracts-p1-resume-3.md) and [machine oracles](contract-oracles-resume-3.json):
14 exact hash/DAG/v1-pin/coverage/state checks PASS; 27 inclusive quota-boundary
specifications and 148 per-key KV negative mutation specifications. Mutation rows
are **not executed future decoder tests**. N−1/N are within each quota, N+1 refuses;
other invariants independently apply. Canonical syntax remains leading
`axiom --json docs inspect` / `axiom --human docs inspect`.

Additional scratch exact-record dispatch checks: pinned v1 Go LoadCandidate accepts
synthetic v1, rejects exact proposed v2 record and unknown version. Pinned v1 shell
rejects those invalid records with exit 1 and untouched candidate sentinels. Archives
have synthetic old-layout members, not a complete positive v2 package. The Go test
was removed. Future v2 shell/Go/archive/migration qualification stays NOT_RUN.
Executed envelope hashes identify transient gzip bytes, not a reproducible archive
claim; exact metadata/binding/control hashes and replay algorithms are retained.

## E02 — native positive suites and decisive limit

[Per-host results and supplemental counterexample](probe-results-resume-3.json).
Initial cgo bridge passed 22 cases + fresh B on all three native rows. Sources and
results remain immutable at fa59969/5c8b4a0. Current builder forces CGO_ENABLED=0
and builds all archives on Ubuntu, so a further bounded prototype preserved that
setting instead of changing release preparation.

CGO-disabled prototype passed **24 cases + fresh B per native host**, same source/
harness hashes: macOS arm64/APFS local, Linux amd64 and arm64 GitHub-hosted. Native
[run 39505487190](https://github.com/rgomids/axiom/actions/runs/39505487190), source
`1206ba9b02767ed23d52fd6dc4df3fb429e44384`; both jobs SUCCESS. Linux filesystem
reported `ext2/ext3` via stat. Native image APIs, source/toolchain/compiler/binary
hashes, barrier phases, actual consumed hashes and refusals are in JSON. PATH was
changed inside the launched process, with CWD changed to `/`; this supersedes the
older launch-only PATH primitive. API-error control uses an invalid read-only
Darwin query flavor / unavailable fixed Linux proc path; it is not a claim that
the actual supported API was absent. Stale descriptor control closes the real pin.

The no-cgo Darwin prototype used measured SDK record size/offsets 1272/96/104 and
stdlib Syscall6; Linux used its independently opened proc image descriptor. Primary
Apple source references and ABI/build trade-offs are in the gate report. This is
not adoption of a raw ABI or new dependency. Local `go version -m` confirmed
CGO_ENABLED=0/darwin/arm64. Both alternatives still need material review.

**Subsequent macOS in-place-before-pin oracle FAIL:** append to the executing file
was allowed; original mapped device/inode stayed equal to the modified file. With
a coherent synthetic changed binding, live tag A consumed all B content hashes.
Reader process exit 0 is not oracle PASS. Full replay, original/modified binary
hashes and A/B generations are retained. This is a bounded prototype counterexample,
not a production vulnerability or proof all OS mechanisms impossible. Do not
silently exclude it as authenticity or arbitrary continuous same-UID defense.
**No algorithm adopted, no further dependent execution after this finding.**

Docker not rerun: prior arm64 primitive PASS complementary; prior amd64 emulated
image oracle FAIL remains negative, not native proof. No old result relabelled.

## E03 — installation lock and transition model

[Sanitized three-host results](transition-results-resume-3.json) bind source
`5c8b4a0afa318bcde6abb3a5b1dd39ea2cdf6753`, exact model hash and jobs from
[39504536442](https://github.com/rgomids/axiom/actions/runs/39504536442).
Both subsequent CGO-disabled native jobs repeated the same model successfully;
these were already launched before the supplemental macOS finding.

Each host passed 21 transition rows; two-process same-target/different-receipt-root
contention, distinct-installation independence, physical alias identity and stale
binary/lock revisions; three foreign/link lock refusals; lock-path replacement
refusal at authority recheck. Both child process exit codes 0, busy contender did
not commit. The lock namespace/path/record remain proposals for human review.

Actual model operations prepare/read back complete three-output generations and
bindings before binary rename; readers observe complete A/B or refusal. Confirmed
binary effects survive later errors. Ambiguous marker/rename effects remain unknown;
inspection makes no mutation or automatic retry/rollback/GC. Existing generations,
conflicting residue and foreign sentinels retained. Same-parent stage publication
passed; cross-parent rename after 0500 failed EACCES on macOS and was not adopted.
Counts of retained directories do not certify all as valid generations.

Space/quota/permission/sync/rename failures are deterministic injected outcomes,
not physical storage exhaustion. Receipt/skills/ack are synthetic model effects,
not real candidate/Runtime receipt authority or convergence. No production lock,
layout or recovery implementation. Since D2 remains unqualified, these passing
fixtures do not complete/adopt T03; reconcile them after mechanism qualification.

## Harness corrections and validation

Early local harness errors retained as limitations: binding key count corrected
16→accepted 15; controlled mutations chmod only disposable fixture parents;
Python multiprocessing main guard added; new-lock descriptor rewound and private
namespace creation fixed. No false native failure/passing product claim inferred
from these harness revisions. The cross-parent EACCES and final in-place oracle
FAIL are separate substantive observations, not silently erased corrections.

Executed exit 0:

- Go candidate builds and shell syntax validation before native candidate runs;
  source parity checks and native experiment commands recorded at source commits.
- `go test ./internal/install ./internal/cli ./internal/local ./internal/codexruntime`.
- `go test ./internal/install -run '^TestAXM37PinnedV1RecordDispatch$' -count=1 -v`
  (temporary test, three cases; removed after collection).
- `bash -n scripts/install-release.sh scripts/test-install-posix-facade.sh`;
  `bash scripts/test-install-posix-facade.sh` (failures=0).
- Repository validator, automation registry, sensitive-file checks, gitleaks and
  diff checks ran during preparation; final delivery records final-head checks.

Workflow was temporary and branch-only, read-only token, pinned actions,
persist-credentials=false; removed with its registry entry after all expected
native jobs completed. Source commits remain replayable in Git history. No net
workflow/registry or production-code change against entry head. Final CI is not
native or functional acceptance. [G1 report](gate-g1-resume-3.md) names the exact
blocker, human material assessment, remaining unit statuses and next sequence.

# AXM-37 — bounded-v2-gzip-r2

Same Implement and Draft [PR #335](https://github.com/rgomids/axiom/pull/335).
Entry: `8c02c1d4bab3696ee9fcc23b2605aeeb7dc3185b`, clean workspace and matching
remote branch. Entry CI/metadata PASS is historical only. Decisions 1B/2A/3A/4A/5A
and explicit bounded-patch authority were verified against the named Linear
receipts on 2026-10-21; the user's task reproduces the permitted scope. No raw
provider content is imported. Canonical Execution identity remains unproven;
no new Execution, Work Item, PR or dispatch was created. No delegation: a
strictly lower model hierarchy could not be established.

## Delta and traceability

| Task / AC | Bounded delta / evidence | Remaining state |
| --- | --- | --- |
| T02 / AC-02,05 | Existing association resolver unchanged; local package regression and v2 development installation re-check intact binding/content | PARTIAL; full native matrix absent |
| T03 / AC-03,05 | Existing publication/lock tests rerun through `internal/install`; no lock/recovery/GC change | PARTIAL; no process-kill or native qualification |
| T07 / AC-05,06 | Regression-first gzip seam: one tar stream may cross legacy v1 members at header/body/metadata; v2 single-member only. Reject gzip CRC/truncation/trailing data, complete-tar concatenation, duplicate paths/types and file bounds. Bootstrap selects a closed three-row checksum inventory; producer/verifier/smoke keep default 1 and explicit format 2 | PARTIAL; selected differential, not exhaustive shell/Go equivalence |
| T09 / AC-03,05 | Bootstrap selects macos-27 v1 versus macos v2 without URL fallback. Source shell verifier exposes verify-only without candidate execution/destination publication; existing installer handles receipt 2/generations | PARTIAL; native migrations and v0.1.0 floor unqualified |
| T12 / AC-02,07 | Skills/man and expected/observed/loaded semantics unchanged; offline harness reads installed corpus, checks links and association, safe candidate sentinels for documentary refusals | PARTIAL; actual man selection and Q6 absent |
| T14 / AC-01 | Local development rehearsal, env-i/outside checkout, corpus integrity/links and refusals | PARTIAL; no source/network denial or observers |
| AC-06 | Existing reference/parity/determinism tests rerun; no coverage/budget changes | Prior scoped PASS retained; not native acceptance |

T13/T15 NOT_RUN, T14/T16 PARTIAL, AC-04/08 NOT_RUN and v0.1.0 UNVERIFIED.
T04 is the accepted bounded internal checkpoint, not independent review.
No default activation, native hosts/sessions, recovery, merge, release, acceptance,
status/relationship change or AXM-39/AXM-12 unblocking occurred.

## Validation record

- Regression red: `go test ./internal/install -run 'Test(Legacy.*Gzip|GzipCorruption|V2SplitGzip)' -count=1` failed on all three valid legacy splits and legacy trailing garbage, before the parser patch.
- PASS: `go test ./internal/install -run 'Test(Legacy.*Gzip|Gzip.*|V2SplitGzip|LoadCandidate)' -count=1` after the patch, including complete-tar/duplicate-directory refusal.
- PASS: `go test ./internal/install ./internal/operationaldocs ./internal/cli ./internal/codexruntime` (development macOS arm64).
- PASS: `python3 scripts/test-prepared-artifact-smoke.py` (13 tests; stubbed subprocesses, including v2 compression refusal before any execution).
- PASS: `./scripts/test-docs-offline.sh`, repeated after adding safe execution sentinels. Installs/reads 26 outputs; generated development artifacts, not release or native qualification. Re-sealed extra output, swapped content, extra root file and stale metadata refuse without invoking the sentinel candidate.
- PASS: `./scripts/check-go-quality.sh format`, `./scripts/validate-repository.sh .`, `python3 scripts/check-adr-governance.py .` (35 ADRs), `git diff --check`, `gitleaks dir . --no-banner --redact --exit-code 1`.
- PASS: `go vet ./... && go build ./...`.
- PASS: `./scripts/check-sensitive-files.sh --staged .` and `gitleaks git --staged --no-banner --redact` before commit `f1896d138311d91480aff1f7e9ead9daa2877d69`.
- PASS: `./scripts/test-release-pipeline.sh`, clean committed fixture source `f1896d138311d91480aff1f7e9ead9daa2877d69`. Builds/verifies full v1 and explicit v2 sets; rejects either set under the other format, preserves no-publication workflow checks. Its Python stub tests use the working-tree test script (13 tests), not the clone.
- PASS: producer rejects `--only unsupported` before output creation (expected exit 1, `unsupported row filter`).
- Bootstrap first clean-commit run: all 78 cases PASS / `failures=0`, including v2 install/reinstall and `loaded:unknown`; overall command FAIL because its old cleanup trap could not remove immutable v2 generation directories. The test-only trap now grants owner write permission inside its own temporary fixture before removal. Clean-commit rerun pending; no production permission/lock behavior changed.
- FAIL: `GOPROXY=off GOTOOLCHAIN=local go test ./internal/executiongraph -run '^TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation$' -count=3 -timeout=30s -v`: FAIL/FAIL/FAIL; exit -1, empty output, Stopped/OutcomeUnknown true. No file under executiongraph changed. Timing/race remains an unconfirmed hypothesis.
- PASS: release-pipeline repeated at clean source `5aadca9cca9da6c8c78a5a8549e6a2d1336d0b7d`, including the updated checksum refusal and all 13 Python tests.
- PASS: final repository validator and ADR governance after Evidence/routing updates.
- PASS: `./scripts/test-install-bootstrap.sh` at clean committed source `5aadca9cca9da6c8c78a5a8549e6a2d1336d0b7d`: 78 cases, `failures=0`, command exit 0, including fixture cleanup. All three row selections for both generations are simulated without foreign candidate execution; local development lifecycle uses macOS arm64 only.
- FAIL: `go test ./...`: every other package PASS (including `cmd/lingo`, 487.001s; `internal/install`, 48.183s), only `internal/executiongraph/TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation` FAIL with the same lost process result. Go sources are unchanged since this run; later commits adjust Python test safety/checksum selection and documentation. This is a failing full suite, not a product/native PASS.
- Final staged security/diff checks and final-head CI are reported in the same-attempt Linear completion receipt. Source test commit and final documentation commit are distinct; no old SHA's CI PASS is reused.

## Limits and next gate

The Go v2 loader and Python prepared smoke reject extra gzip members and trailing
bytes. The existing dependency-free shell reader still uses system tar/gzip;
no new parser prerequisite was introduced. Exhaustive compression/framing
parity before the shell candidate version probe remains unproven. Do not promote
T07/AC-05/06 to full PASS or activate the default on this evidence alone.
The shell verify-only path does not establish compiled descriptor association;
it is packaging validation without execution. Actual association stays with
the installed binary/resolver. Release workflows still invoke generation 1.

`internal/executiongraph` is outside this patch. Earlier FAIL/FAIL/PASS isolated
reproductions remain historical; timing/race is an unconfirmed hypothesis, not
a diagnosed cause. A later full-suite PASS cannot erase those failures.

Next: integrate clean-commit test receipts and exact-head CI, then independent
T16 technical review. Native T13–T15/Q6 and default activation require their
separate human authority. Preserve fail-closed residual locks.

## Author review and security note

The delta preserves installed identity and skill/man content, fixed budgets,
ADR-0016 candidate proof and fail-closed locks. No runtime or module dependency
was added. Selected Go/shell/Python security cases are evidence, not a claim of
full codec equivalence or independent review. The bootstrap deliberately requires
the published complete checksum set; single-row development output is not a
publishable bootstrap release. Existing release workflows keep their default
invocations and legacy macOS token.

During a new Python test, mocking the shared subprocess module also intercepted
Python host-description discovery; a failing mock assertion printed inherited
environment data into the local tool output. The test now stubs that discovery
and all 13 cases pass. No environment dump was written to repository artifacts
or published to Git/Linear. The user was notified to rotate the exposed token;
rotation/credential operations were not performed or authorized. No credential
value is retained in this Evidence.

## Handoff

Bounded producer/bootstrap/release-consumer and legacy-gzip patches are committed
on the original branch. Default remains generation 1; no workflow dispatch or
publication took place. Delivery is **PARTIAL**: exact-head remote CI must be
read after push, T16 independent review remains required, and the shell framing
differential above must be resolved/qualified before any separate default
activation authority. T13/T15 remain NOT_RUN, T14/T16 PARTIAL, AC-04/08 NOT_RUN,
v0.1.0 UNVERIFIED. No historical PASS or FAIL was erased.

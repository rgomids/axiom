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
- PASS: `python3 scripts/test-prepared-artifact-smoke.py` (12 tests; stubbed subprocesses, including v2 compression refusal before any execution).
- PASS: `./scripts/test-docs-offline.sh`, repeated after adding safe execution sentinels. Installs/reads 26 outputs; generated development artifacts, not release or native qualification. Re-sealed extra output, swapped content, extra root file and stale metadata refuse without invoking the sentinel candidate.
- PASS: `./scripts/check-go-quality.sh format`, `./scripts/validate-repository.sh .`, `python3 scripts/check-adr-governance.py .` (35 ADRs), `git diff --check`, `gitleaks dir . --no-banner --redact --exit-code 1`.
- Pending at this checkpoint: full Go suite, final vet/build, clean-commit bootstrap/release-pipeline fixtures, staged sensitive-files, final-head CI. Results will be appended before final handoff.

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

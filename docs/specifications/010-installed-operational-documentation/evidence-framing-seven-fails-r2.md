# AXM-37 — seven framing failures, round 2

**PARTIAL: four failures repaired; three v2 member-count failures remain.**
Default v1 is unchanged. This is the same Implement on Draft PR #335.

## Authority and continuity

Entry local HEAD, remote branch and OPEN/Draft PR head all matched
`e6fbfe13810bff91ffb8939234415a37ecd1dee5`; workspace was clean.
Linear comments were read completely (one page, no next page), including
`framing-seven-fails-r2` and the previous stage.partial. The
[bounded patch/commit/push decision](https://linear.app/rgomids/issue/AXM-37#comment-e16ecc09)
was revalidated. No incompatible newer decision or competing AXM-37 executor
was identified in bounded local process/worktree/provider inspection; this
is not a global lease or a claim about other hosts. AXM-36 was separate.
No new Execution, PR, work item or delegation was created. A strictly lower
model hierarchy was not established, so work remained sequential.

Repair commit: `8992c444dc082e91457758c1efab7a239437f868`.
No provider payload was imported; this document contains sanitized technical
results authorized by the current task.

## Reproduction and case matrix

The exact inert [r1 experiment](framing-preflight-r1.patch) was extracted into
a temporary Go test and supplied through a Go overlay, without changing that
historical patch. `TestPreflightShellGoFraming`, `-v -count=1`, reproduced all
seven reported failures separately on the entry source: shell verify-only
accepted, Go refused, and the normal entrypoint reached a safe sentinel that
wrote a temporary marker and exited 91. SHA256SUMS was recomputed for every
mutation. No candidate installation was delegated. Host: development macOS
arm64 with system gzip/bsdtar; no Linux/native qualification is inferred.

| Exact checksum-valid case | Entry shell / Go | Repaired shell / Go | Status |
| --- | --- | --- | --- |
| v1 complete tar followed by another gzip-compressed complete tar | accept / refuse | refuse / refuse | PASS |
| v1 nonzero bytes after tar EOF inside gzip | accept / refuse | refuse / refuse | PASS |
| v2 complete tar followed by another gzip-compressed complete tar | accept / refuse | refuse / refuse | PASS |
| v2 nonzero bytes after tar EOF inside gzip | accept / refuse | refuse / refuse | PASS |
| v2 tar split across two gzip members at byte 17 | accept / refuse | accept / refuse | FAIL |
| v2 additional empty gzip member | accept / refuse | accept / refuse | FAIL |
| v2 prefixed empty gzip member | accept / refuse | accept / refuse | FAIL |

The original 34-subcase experiment was rerun after repair: exactly three FAIL
above remain; the original v2 symlink mutation remains explicitly SKIP. This
is reproducible partial progress, not a nonreproducible case or seven-case PASS.

## Control change and remaining boundary

Before system tar listing/extraction, the shell now walks bounded physical tar
headers in the already bounded, private decompressed staging file. Header size
(octal or positive base-256) determines the next 512-byte boundary. The first
zero header requires a second zero block and an entirely zero remaining tail.
Malformed framing, oversized advances, excessive headers and nonzero tails
are refused before candidate execution or persistent destination creation.
System tar still checks archive validity; existing paths, links, modes, CRC,
checksums, quotas, binary/docs/skills identity and locks are unchanged.
The original compressed bytes and checksum remain delegation inputs.
No new tool or prerequisite was introduced for this repair.

The three v2 failures require a reliable gzip-member boundary oracle.
Existing gzip list/verbose summaries do not prove absence of an empty prefix;
magic-byte counting and canonical recompression are not sound substitutes.
A human decision was requested before adding an explicit `python3` bootstrap
prerequisite, using stdlib `zlib` for bounded member framing. That host-contract
change is **NOT_AUTHORIZED / NOT_IMPLEMENTED** pending a response. No tool was
installed. The four independent tar repairs proceed under existing authority;
no failing v2 assertion was silently relabeled PASS or included as green CI.
The inert original experiment remains the reproduction for these three cases.

`TestShellArchiveIntegrityBeforeExecution` now contains 31 active cases:
existing 20 plus the four repaired negatives, zero tar-tail and compressed
zero-tail controls for both formats, and three historical v1 multistream
controls (split/empty suffix/empty prefix). Expected Go acceptance is asserted
explicitly. Shell verify-only is compared with Go, never executes the sentinel
and creates no persistent destination. Normal entrypoint negatives assert no
sentinel and no destinations; valid controls prove the sentinel is executable.

## Validation

Validation receipts are completed below before delivery. No older-head CI or
previous local suite result is represented as final-head PASS.

- PASS: targeted four repaired negatives and four valid/zero-tail controls.
- PASS: `GOPROXY=off GOTOOLCHAIN=local go test ./internal/install -count=1`.
- PASS: affected archive/v2 Go tests, `-count=1`.
- PASS: `go vet ./...`; `go build ./...` (GOPROXY=off, GOTOOLCHAIN=local).
- PASS: `scripts/test-install-posix-facade.sh` (14 cases).
- PASS: `python3 scripts/test-prepared-artifact-smoke.py` (13 tests).
- PASS: `scripts/test-docs-offline.sh` (development synthetic only).
- PASS: formatting, repository validator, staged sensitive-file scan, gitleaks,
  complete scoped diff review and `git diff --check`.
- FAIL: original adversarial experiment, three v2 member cases remaining.
- FAIL: isolated `go test ./internal/executiongraph -run
  '^TestOSProcessRunnerRetainsFastRejectionBeforeCorrelation$' -count=3
  -timeout=30s` (GOPROXY=off, GOTOOLCHAIN=local), all three iterations lose
  the process result (`ExitCode=-1`, empty output, Stopped/OutcomeUnknown).
  The executiongraph diff from entry is empty; cause is not established.
- PASS: `scripts/test-release-pipeline.sh` at clean repair commit
  `8992c444dc082e91457758c1efab7a239437f868`, default-v1 and explicit-v2
  three-row artifact sets, exact-byte smoke and no-publication controls.
- PASS: `scripts/test-install-bootstrap.sh` at clean repair commit
  `8992c444dc082e91457758c1efab7a239437f868` (78 cases, fake network).
- PASS: `GOPROXY=off GOTOOLCHAIN=local go test ./...` on the repaired
  code/test tree (cache where applicable). This does not erase the isolated
  executiongraph FAIL above.
- UNVERIFIED: remote CI of the delivery HEAD until final push/read-back. The
  same-attempt Linear receipt records its exact SHA, run URLs and observed
  states; no entry-head CI is reused as final-head evidence.

T13–T15/Q6 native qualification stays deferred to AXM-12; T16 remains partial.
No v2 default activation or activation preparation, host/session provisioning,
real Runtime inference, credential/cost change, lock recovery, workflow dispatch,
merge, release, acceptance, Done, status or blocker mutation occurred.
Next gate: incremental review of the four repaired cases on the actual PR head,
plus the explicit host-prerequisite decision before the remaining three repairs.

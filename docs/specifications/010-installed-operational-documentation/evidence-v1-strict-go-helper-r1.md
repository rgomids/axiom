# AXM-37 — Strict v1 Go helper r1: distribution-contract blocker

## Scope and reconciliation

Attempt `v1-strict-go-helper-r1`, same Draft [PR #335](https://github.com/rgomids/axiom/pull/335),
branch `docs/axm-37-offline-operational-documentation`. Inspected source HEAD:
`ea45f441c45dbf722e17404d13b5dd293e504691`; local and PR HEAD matched, working
tree clean before this documentary change. No independent executor was observed.
The operator reconciled the accepted dispatch as belonging to this same executor;
the initial concurrency stop is superseded. No second dispatch or execution.

The delegated choice authorizes Go stdlib as the verifier technique, but explicitly
requires a stop if trustworthy distribution needs a material contract change.
This inspection reaches that stop. No helper, parser, production change, dependency,
permanent regression suite, ADR or new acquisition mechanism is introduced.

## Exact blocker

Go `compress/gzip` is already used by `internal/install/archive.go`; its availability
in source does not supply a trusted executable on an end-user host. Existing
mechanisms provide no independently identified pre-candidate helper artifact:

| Existing boundary | Executable source fact | Missing contract for a helper |
| --- | --- | --- |
| Remote acquisition, `scripts/install.sh` | `archive_format` accepts exactly three checksum entries, all release archives. The bootstrap extracts only `install.sh`, `release-metadata.txt` and `MANIFEST.sha256` after checking the selected archive SHA-256 | No helper asset selector, identity binding, acquisition or handoff |
| Producer, `scripts/build-release-archives.sh` | v1 builds `./cmd/lingo` as `axiom`, copies the shell installer and packages metadata, license and skills | No separate verifier build/package identity |
| Release verification, `scripts/verify-release-artifacts.sh` | Top-level artifact set is exactly SHA256SUMS plus three archives; archive members are compared with a closed expected bundle inventory | An external helper asset or an additional packaged executable changes these accepted sets |
| Publication, `scripts/publish-release.sh` | Published checksum rows are archive names, with the exact verified asset set | A new helper asset cannot be added without changing release-set validation |
| Local facade, `scripts/install-release.sh` | Accepts archive/checksums/destinations and verify-only; stages and hashes original compressed bytes, then uses host gzip before executing the candidate | No trusted helper input, pin, provenance check or acquisition path; local mode cannot assume networking or an installed Go compiler |
| Go archive reader, `internal/install/archive.go` | Per-file allowance is 1 MiB except `axiom`; Go code is part of the candidate product executable | Candidate execution cannot bootstrap the verifier; a separate executable must also fit an explicitly approved packaging budget |

The closed release-verifier inventory is distinct from the Go v1 reader's manifest
completeness check; this Evidence does not claim that Go rejects every extra
manifest-listed v1 member.

A helper extracted from a caller-controlled archive plus its matching internal
manifest is not independently trusted. The existing Plan acquisition boundary
requires previously trusted local archive/checksum provenance, or authorized exact
repository/tag/assets over HTTPS. A verified helper in that trust chain could be
designed, but its package identity, distribution and offline invocation contract
are currently absent. Selecting a PATH executable, using the candidate itself,
compiling on the end-user host, or silently adding an asset/member does not close
that gap within the unchanged contracts.

ADR-0033/0034 preserve trusted acquisition separately from local hash integrity;
neither adopts a new verifier artifact. The Specification 010 Plan's
producer-consumer matrix likewise requires explicit versioned evolution rather
than silently expanding v1. Technical choice of Go is therefore insufficient
authority to change this distribution boundary.

## Validation and reproducibility

Environment: macOS 27.0.1 arm64, Apple gzip 487.0.1, Go 1.26.1 darwin/arm64.
No strictly inferior model hierarchy was verified; all work performed directly.

```sh
git status --short
git branch --show-current
git rev-parse HEAD
gh pr view 335 --json headRefOid,headRefName,state,isDraft
go version
sw_vers
gzip --version
go test ./internal/install -run 'Test(LegacyGzipMembers|GzipCorruptionRefused|V2SplitGzipRefused|LegacySplitGzipRetainsArchiveRefusals|GzipAmbiguousTarRefused|ShellArchiveIntegrityBeforeExecution)$' -count=1
gh run view 39690256790 --json headSha,status,conclusion,url,jobs
./scripts/validate-repository.sh .
./scripts/check-sensitive-files.sh --staged .
gitleaks git --staged --redact --no-banner
git diff --cached --check
```

Focused macOS suite: **PASS**, `ok github.com/rgomids/axiom/internal/install 27.840s`
at inspected HEAD (before documentation edits). This existing suite
includes valid split-member controls, corruption and shell/Go comparisons, candidate
sentinel and persistent-destination assertions. It is not the requested expanded
GNU/Apple/Go regression suite for a corrected helper: that work is blocked.

Baseline exact-SHA CI was read back as terminal **FAIL** at inspected HEAD:
[run 39690256790](https://github.com/rgomids/axiom/actions/runs/39690256790),
Tests failed; Installed documentation journey and Repository and sensitive-file
checks skipped. This is historical baseline evidence, not a successful check of
this documentary continuation. Final pushed HEAD, new CI run IDs and actual
terminal outcomes must be read back in the same attempt's completion receipt;
no exploratory or earlier PASS certifies that final HEAD.

GNU gzip 1.12 and Linux host regressions: **NOT_RUN in this continuation**.
Local repository validator, staged sensitive-file scan, gitleaks staged scan and
documentary diff review: **PASS**. Repository validator is a governance check,
not native Runtime or host qualification.
Native/Q6, new helper provenance/order tests and independent correction review:
**NOT_RUN**. Prior failures remain in
[strict v1 no-Python Evidence](evidence-v1-strict-no-python-r1.md).

## Next gate

**BLOCKED: authorize a concrete verifier distribution-contract amendment**, covering
exact helper version/source/host identity, trusted remote and offline acquisition,
checksum binding before execution, private staging/handoff, artifact/member sets,
size budgets and historical v1 compatibility. Prefer extending existing release
trust and packaging mechanisms; no separate trust service is proposed or adopted.
Only after that decision can implementation and permanent differential tests
proceed, followed by full final-HEAD CI and independent review of the correction.

Default v1 and opt-in v2, current bounds/tar/path/hash/link checks, accepted ADRs,
historical Evidence, statuses and AXM-12/39 dependencies remain unchanged. No merge,
release, human acceptance or native qualification is claimed. T16 remains partial.

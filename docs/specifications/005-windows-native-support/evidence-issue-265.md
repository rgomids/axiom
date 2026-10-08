# Issue #265 — Default Windows onboarding Evidence

Validation date: 2026-10-08. Scope: the issue-265 amendment and ADR-0019,
implemented for PR #266. Evidence is sanitized; no personal paths, account SIDs
or raw security descriptors are recorded here.

## Environment and boundaries

Native Windows client amd64, Windows PowerShell 5.1 and Go 1.26. Fixture profiles,
Runtime executables and ACLs are temporary objects created by the tests. Real
profile/AppData/Runtime ACLs and the user's registry PATH were not changed.
Fake Runtime command shims establish discovery; they are never executed.
The production bootstrap uses an in-memory HTTP transport and mocked registry
boundary, with a real checksummed native bundle and real `first-run` execution.

## Results

| Check | Result / established behavior |
|---|---|
| `go test ./... -timeout 5m` | Passed on native Windows after the recovery correction; all packages passed. |
| `go test ./internal/windowsfs -count=1` | Passed: exact authority, no effects on backup failure, bounded targets, drift refusal, recovery, no descendant propagation and rollback after a later object drifts. |
| `go test ./cmd/lingo -run WindowsOnboarding -count=1` | Passed: denied repair publishes nothing; approved repair prepares standard roots and a backup; repeat requires no repair. |
| `scripts/test-windows-install.ps1` | Passed: checksum refusal, isolated install, unsafe AppData left intact, standard Codex root repair with consent, unrelated skill ACL/content preservation, version, first-run, default reinstall, unsafe storage diagnostics, locked binary refusal and upgrade. |
| `scripts/test-windows-bootstrap.ps1` | Passed: production default command without root overrides, automatic first-run, standard Codex and Claude discovery, session PATH, persistent user PATH and reinstall. Registry cases cover missing PATH, expandable duplicate, unsupported type, length limit and concurrent change. Raw placeholders, separators and registry kind are preserved; reinstall writes only once. |
| Linux/darwin amd64 `go build ./cmd/lingo` | Passed cross-compilation; native Unix behavior still requires platform CI. |
| `scripts/validate-landing-page.sh .` | Passed; remote asset checks disabled. |
| `scripts/check-adr-governance.py .` | Passed. |
| `scripts/check-automation-registry.py .` | Passed. |
| `git diff --check` | Passed. |

The initial whole Go suite exposed a recovery handle lacking `WRITE_DAC` after
the least-privilege inspection change. Recovery now opens write access only for
the exact approved, identity-checked pinned target; the recovery and Windows
filesystem suites passed after the correction.

## Review and remaining acceptance

Security review checked target allowlisting, current ownership, NTFS/reparse
confinement, non-delete-sharing ancestor handles, descriptor/object drift,
backup-before-effect ordering, rollback, recovery and user-only PATH writes.
Repair subtracts rejected permissions; it adds no trusted principals and never
walks or updates existing descendants. No blocking finding remains in this
bounded implementation review.

`scripts/validate-repository.sh .` stops at an unchanged Claude skill adapter
materialized as a regular file instead of a symlink in this Windows checkout.
The complete repository validator remains unverified locally; CI must validate
the canonical adapters. Gitleaks is unavailable locally; the required staged
sensitive-file check is recorded in the PR.

The affected real profile has not been repaired or installed with this branch;
that requires approval of its concrete repair preview. Runtime UI discovery,
interactive registry notification/new-terminal inheritance, native Unix tests,
interruption at every repair boundary and upstream CI remain acceptance work.
The public install command requires a published native release containing this
implementation; existing immutable releases are unchanged. Merge and release
publication were not performed by this validation.

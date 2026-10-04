# Specification 006 — Installer Host Eligibility and Platform Support Policy

## Status

**Approved, 2026-10-04.** This applies the human decision recorded in
[Issue #183](https://github.com/rgomids/axiom/issues/183) and accepted as
[ADR-0015](../../decisions/0015-installer-host-eligibility-os-family-architecture.md#decision).
The implementation is the bounded #183 hotfix in PR #184. There is no separate
Plan or Tasks file: the work is a single installer-eligibility correction.
Merge, human acceptance and release publication remain separate decisions.

## Problem

Exact OS-version gates prevented otherwise compatible supported hosts from
installing Axiom. On Darwin/arm64 with macOS `27.0.1`, the installer refused
the host because only `27.0` was allowlisted, although the darwin/arm64 build
runs there. Windows applied a similar numeric gate (Windows 10 build `17763`
or later, major version `10`). Linux was already version- and
distribution-agnostic.

## Observable behavior

Host selection uses only the OS family, the architecture and real
prerequisites:

```text
Darwin  + arm64                                   -> macOS arm64 release row (macos-27-arm64)
Linux   + x86_64                                  -> linux-amd64
Linux   + aarch64                                 -> linux-arm64
Windows client edition + AMD64 + 64-bit PowerShell -> windows-amd64
anything else                                     -> refused before any download or effect
```

Numeric OS versions do not participate in this selection. `macos-27-arm64`
is legacy release-row naming, not a version requirement
([ADR-0015, Artifact naming](../../decisions/0015-installer-host-eligibility-os-family-architecture.md#artifact-naming)).

## Functional requirements

- **FR-E01:** Every installer layer (the POSIX bootstrap `scripts/install.sh`,
  the verified POSIX facade `scripts/install-release.sh`, the Windows bootstrap
  `scripts/install.ps1`, and the Go installer owner) MUST validate the OS family.
- **FR-E02:** Every installer layer MUST validate the architecture against the
  supported rows above.
- **FR-E03:** Windows eligibility MUST keep the client/server distinction.
  Windows Server and domain controllers MUST be refused.
- **FR-E04:** Real runtime and tool prerequisites MUST still be validated, with
  explicit diagnostics when missing. Examples are `curl`, `tar`, `bash`, `awk`,
  `grep`, `mktemp`, `sha256sum` or `shasum`, `Windows_NT`, 64-bit PowerShell
  and `tar.exe`.
- **FR-E05:** The numeric OS version MUST NOT be used as an eligibility filter.
  No installer layer may keep an OS-version allowlist or minimum, and none may
  need `sw_vers` (or an equivalent version probe) to establish eligibility.
- **FR-E06:** Unsupported OS families MUST fail closed before download or
  effect.
- **FR-E07:** Unsupported architectures MUST fail closed before download or
  effect.
- **FR-E08:** Existing checksum, archive, manifest, ownership, filesystem,
  authority and HTTPS protections MUST remain unchanged.
- **FR-E09:** Refusal diagnostics MUST NOT claim that a specific OS version is
  required.

## Support lifecycle

Axiom's support commitment follows vendor-maintained operating systems. An OS
version that its vendor no longer maintains is outside Axiom's maintenance
commitment.

Installer eligibility does not enforce vendor lifecycle. The installer does
not calculate, detect or enforce vendor EOL, and automatic EOL detection is
not implemented. If a newer maintained OS exposes a real incompatibility, it
is handled as a concrete bug rather than blocked in advance.

## Evidence

The exact OS version remains valuable Evidence. Evidence records identify the
validated environment precisely, for example:

```text
macOS 27.0.1 (build) / arm64 / APFS
Windows <exact build> / amd64 / NTFS
Linux <distribution, kernel> / <architecture> / ext4
```

Evidence records what was validated. It does not define installer
eligibility. Historical Evidence that names a specific OS version is preserved
unchanged.

## Non-goals

- renaming `macos-27-*` artifacts or `platform=macos-27` metadata;
- new CPU architectures;
- new OS families;
- Windows Server support;
- vendor-EOL detection;
- package-manager changes;
- persisted-state compatibility (Issue #153);
- CI/CD redesign (Issue #154).

## Acceptance criteria

These criteria come from Issue #183 and are implemented by PR #184:

- **AC-E01:** Darwin/arm64 is not rejected because `sw_vers` reports `27.0.1`,
  `27.1`, `28.x` or another version, and eligibility does not invoke `sw_vers`.
- **AC-E02:** The POSIX second-stage verified installation (facade and Go
  installer owner) does not reintroduce a macOS version gate after bootstrap
  acceptance.
- **AC-E03:** Windows client amd64 eligibility with 64-bit PowerShell does not
  depend on the numeric Windows version.
- **AC-E04:** Windows Server behavior is unchanged: it is refused.
- **AC-E05:** Linux remains version- and distribution-agnostic for
  `x86_64`/`aarch64`.
- **AC-E06:** Unsupported OS families and architectures still fail before
  download or effects.
- **AC-E07:** Missing required tools or runtime prerequisites still fail with
  explicit diagnostics.
- **AC-E08:** Installer safety, checksum, ownership, archive, filesystem and
  authority checks are unchanged.
- **AC-E09:** Regression tests prove that OS-version changes alone do not
  alter eligibility.
- **AC-E10:** Existing installer and release validation suites pass.

## Supersedes

These fragments of earlier contracts are superseded. Each is preserved, struck
through and annotated in place. The rest of each artifact remains in force.

| Artifact | Superseded fragment |
|---|---|
| [ADR-0010](../../decisions/0010-windows-native-filesystem-boundary.md#decision) | Windows support bounded to "Windows 10 (1809+) and Windows 11" (superseded by ADR-0015) |
| [Specification 004 — Platform reconciliation](../004-mvp-v1-baseline/spec.md#platform-reconciliation--2026-09-28) | "The macOS installer still requires 27.0/arm64" and the "executable compatibility constraint" |
| [Specification 004 Plan — Platform reconciliation](../004-mvp-v1-baseline/plan.md#platform-reconciliation--2026-09-28) | "macOS retains the current executable 27.0/arm64 constraint" |
| [Specification 004 Plan — Supported release matrix](../004-mvp-v1-baseline/plan.md#supported-release-matrix-and-assumptions) | Table cell "Current installer requires macOS 27.0" |
| [Specification 004 Tasks — Platform reconciliation](../004-mvp-v1-baseline/tasks.md#platform-reconciliation--2026-09-28) | "macOS retains the current executable 27.0/arm64 constraint" |
| [Specification 005 — Outcome](../005-windows-native-support/spec.md#outcome) | "Windows 10 version 1809 or later" as the eligible host |

Generic requirements that the Plan or release contract declare "the exact
supported OS/architecture combinations" (Specification 004, HD-1) remain
current. Those combinations are the OS-family/architecture rows above.

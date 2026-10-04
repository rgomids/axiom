# ADR-0015 — Installer host eligibility is OS-family and architecture based

## Status

Accepted on 2026-10-04 by the human decision recorded in
[Issue #183](https://github.com/rgomids/axiom/issues/183) ("Human decision
recorded — 2026-10-04"). It is implemented in PR #184; upstream adoption
remains subject to PR review. This is not release-publication authority.

This ADR supersedes only the OS-version fragment of
[ADR-0010](0010-windows-native-filesystem-boundary.md#decision) (see
[Supersedes](#supersedes)). The rest of ADR-0010 stays in force.

## Context

Earlier contracts put exact OS versions into installer host eligibility:

- **macOS:** only `27.0` on arm64 was eligible. The POSIX bootstrap
  (`scripts/install.sh`), the verified release facade
  (`scripts/install-release.sh`) and the Go installer owner
  (`internal/install`) each compared the product version with `27.0`
  ([Specification 004](../specifications/004-mvp-v1-baseline/spec.md#platform-reconciliation--2026-09-28)).
- **Windows:** only Windows 10 version 1809 or later and Windows 11 on amd64
  were eligible ([ADR-0010](0010-windows-native-filesystem-boundary.md#decision),
  [Specification 005](../specifications/005-windows-native-support/spec.md#outcome)).
  The PowerShell bootstrap required `Win32_OperatingSystem.Version >= 10.0.17763`
  with major version `10`, and the Go installer required major `10` and build
  `>= 17763`.
- **Linux:** already independent of version and distribution. Any Linux on
  x86_64 or aarch64 selected the static `linux-amd64` or `linux-arm64` build.

Issue #183 reproduced the resulting regression on Darwin/arm64 with macOS
`27.0.1`. The public bootstrap refused the host with
`unsupported host Darwin/arm64; supported rows are exactly macOS 27.0/arm64, ...`,
although the published darwin/arm64 binary runs there.

A version allowlist ties installation to the moment the allowlist was written.
Every vendor patch, minor or major release locks users out until Axiom ships a
new installer, even when nothing is actually incompatible. Each layer that
repeats the allowlist must also change in lockstep. The allowlist also does
not encode anything the installer verifies: it is a proxy for compatibility,
not a check of it.

## Decision

1. Installer host eligibility SHALL depend on the supported OS family, the
   supported architecture, and real runtime and platform prerequisites.
2. Numeric OS version SHALL NOT be an installer eligibility gate. No layer
   (bootstrap, verified release facade, Go installer owner) keeps an OS-version
   allowlist or minimum.
3. The eligibility contract is:

   | Host | Eligible release row |
   |---|---|
   | Darwin + arm64 | macOS arm64 row (`macos-27-arm64`) |
   | Linux + x86_64 | `linux-amd64` |
   | Linux + aarch64 | `linux-arm64` |
   | Windows client edition + AMD64 + 64-bit PowerShell | `windows-amd64` |

4. These checks remain in force:
   - supported architecture restrictions;
   - the Windows client/server distinction (`ProductType` workstation only;
     Windows Server and domain controllers stay refused);
   - required tools and runtime prerequisites (for example `curl`, `tar`,
     `sha256sum`/`shasum`, `Windows_NT`, 64-bit PowerShell, `tar.exe`);
   - filesystem and security constraints (ADR-0005, ADR-0010);
   - checksum, archive, manifest, ownership and authority protections.
5. Unsupported OS families and architectures still fail closed before any
   download or effect.

### Eligibility, support and Evidence are separate

```text
installation eligibility != support commitment != acceptance Evidence
```

- **Installation eligibility:** the deterministic host check above. It decides
  whether the installer proceeds.
- **Support commitment (support lifecycle):** Axiom's maintenance commitment
  covers operating-system versions that the vendor still maintains. Axiom does
  not promise maintenance for vendor-EOL versions. Vendor lifecycle is a
  support policy, not an installation gate: the installer does not calculate,
  detect or enforce vendor EOL.
- **Native acceptance Evidence:** a record of the exact environment that was
  validated (for example macOS `27.0.1` build, Windows build, Linux
  distribution and kernel). Evidence records what was validated. It does not
  define eligibility.

A newer OS release may install. If it exposes a real incompatibility, that
incompatibility is handled as a concrete bug rather than preemptively blocked.

### Artifact naming

The `macos-27-*` artifact identifier and the `platform=macos-27` release
metadata are legacy release-row naming. They identify the macOS arm64 row and
do not imply an OS-version installation gate. Renaming them is outside this
decision's hotfix and would require its own release-format change.

## Supersedes

| Artifact | Superseded fragment | Still in force |
|---|---|---|
| [ADR-0010](0010-windows-native-filesystem-boundary.md#decision) | "on Windows 10 (1809+) and Windows 11" as the boundary of Windows support | local NTFS user storage, amd64, client-only (no Windows Server), and the rest of the ADR |

No other ADR encoded an OS-version eligibility boundary. Specification and Plan
fragments superseded by this decision are listed in
[Specification 006](../specifications/006-installer-host-eligibility/spec.md#supersedes).
Superseded text is preserved, struck through and annotated in place.

## Alternatives considered

- **Keep exact-version allowlists and refresh them each OS release.** This
  keeps the current fragility, adds release churn for every vendor update, and
  still refuses compatible hosts between updates. Rejected.
- **Minimum-version floors (for example macOS `>= 27.0`, Windows build
  `>= 17763`).** Fewer refusals, but it is still a proxy for real
  prerequisites, and an old floor drifts into an implicit EOL policy. Rejected
  for eligibility. A real prerequisite is checked directly, as with `tar.exe`.
- **Vendor-EOL detection in the installer.** This needs a vendor-lifecycle
  data source, network or embedded data, and frequent updates, and it turns a
  support policy into a gate. Rejected.
- **Warn on unvalidated versions instead of refusing.** Possible later as a
  separate UX decision. It is not needed to fix eligibility and not adopted
  here.

## Consequences

### Positive

- Patch, minor and future OS releases install without a new Axiom release.
- Eligibility lives in one rule, consistent across bootstrap, facade and Go
  installer layers, and matches the existing Linux behavior.
- Support policy and Evidence remain explicit instead of being implied by a
  version gate.

### Negative / trade-offs

- A future OS release can install and then fail at runtime. That failure
  surfaces as a bug report rather than an up-front refusal.
- Installation on a vendor-EOL OS is not blocked, although Axiom does not
  commit to maintaining it.
- Releases published before this decision carry the old gate inside their
  bundled second-stage installer. Affected hosts need a release that includes
  this change.

## Revisit when

- an OS release introduces an incompatibility that cannot be detected as a
  concrete prerequisite;
- a new OS family or architecture, or Windows Server, is proposed;
- release-row naming (`macos-27-*`) is redesigned;
- a support-lifecycle warning or EOL policy surface is proposed.

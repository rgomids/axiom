# ADR-0021 — Windows Server AMD64 native acceptance

## Status

Accepted on 2026-10-09 by the maintainer's explicit instruction to include
Windows Server AMD64 support and acceptance in the Issue #256 PR and record
the result on the Issue. Review, merge and release publication remain separate.

## Context

Issue #256 initially preserved client-only Windows support. Standard hosted
`windows-2022` runners are Server AMD64, so they could prove refusal but could
not accept the supported client installation lifecycle. The maintainer explicitly
authorized Server support after reviewing that limitation.

## Decision

Support Windows workstation (`ProductType=1`) and member Server (`ProductType=3`)
on AMD64 using the existing `windows-amd64` release artifact. Domain controllers
(`ProductType=2`), unknown products and Windows ARM64 remain refused. Server uses
the same local NTFS, current-token ownership, protected DACL, reparse rejection,
locking, atomic publication and explicit recovery boundaries as workstation.
No OS-version allowlist, service installation, network filesystem support or
multi-user administration is introduced.

Require native Server acceptance on `windows-2022` during release preparation,
using the exact prepared artifact without rebuilding it. Native Server regression
tests must execute their positive lifecycle rather than stopping after host refusal.
Client acceptance remains a separate coverage gap; Server execution proves Server.

There is no previous supported Server release. A genuine prior-supported-release
Server upgrade is not applicable to the first supported candidate. Native fixture
upgrade tests validate mechanisms but are explicitly not historical-release Evidence.
After the first Server release, add that genuine baseline to native acceptance.

## Alternatives considered

- Keep client-only support: smallest product scope but leaves native Windows
  acceptance blocked on available hosted infrastructure.
- Add member Server AMD64: reuses the existing adapter and artifact, with native
  hosted validation and bounded maintenance cost. Selected by the maintainer.
- Include domain controllers or Windows ARM64: changes security/deployment or
  artifact boundaries without a validated requirement. Outside this decision.

## Consequences

### Positive

Available hosted Server runners execute real installation and CLI lifecycles;
required acceptance failures prevent publication without adding publication authority.

### Negative / trade-offs

Server becomes a supported product environment. Native evidence is limited to
the runner build actually tested; it does not certify every Server version or
client edition. The first supported Server release becomes a future upgrade baseline.

## Revisit when

A supported Server release is published, client AMD64 hosted runners become
available, or services, domain controllers, Windows ARM64 or network storage are required.

## Supersedes

- [ADR-0010](0010-windows-native-filesystem-boundary.md#decision): only the
  historical Windows Server exclusion; filesystem/security boundaries remain.
- [ADR-0015](0015-installer-host-eligibility-os-family-architecture.md#decision):
  only client-only Windows eligibility and Server refusal; OS-family/architecture
  eligibility, domain-controller refusal and runtime prerequisites remain.

# Native Windows implementation evidence

## Scope

Implementation for PR #149 (superseding #128), requested and authorized on 2026-09-30. This is
implementation/test evidence, not permission to merge or publish a release.

Delivered code includes Windows security-descriptor and handle checks, NTFS and
reparse-point refusal, stable identity, kernel-object directory locks, native
no-replace publication, free-space checks, state-root selection, Runtime skill
and Git-workspace adapters, verified fresh install/reinstall/upgrade, a PowerShell
bootstrap and bundle installer, a fourth release artifact row, and native CI.
Both READMEs document the post-merge interface, not a future specification-only
promise. Older releases without the Windows asset remain uninstallable on Windows.

## Local deterministic results

Native Windows amd64, Go 1.26.0, local NTFS, user-private validation directory:

- `go vet ./...`: passed.
- `GOOS=linux GOARCH=amd64 go build ./...`: passed cross-build; not Linux runtime evidence.
- `go test ./internal/local -count=1 -timeout=3m`: passed, including native ACL,
  junction, hardlink, lock/publication and state-root tests, plus existing
  publication/recovery/concurrency workflows.
- `go test ./internal/install -run TestWindows -count=1 -timeout=60s`: passed
  fresh install, exact reinstall, owned upgrade, foreign binary preservation,
  and interrupted receipt finalization tests.
- Native executable Project configuration and the complete Claude/GitHub-fake
  Work Item/workflow projection regression passed during implementation.
- Execution Graph tests passed after converting synthetic absolute path fixtures
  to host paths.
- PowerShell 5.1 parser accepted the bootstrap.

## CI regression validation — 2026-10-01

These are historical results for the pre-review implementation, not validation
of the review corrections below. The same four jobs also passed on `80f0f02` in
[CI run 36816437388](https://github.com/rgomids/axiom/actions/runs/36816437388),
as read from PR #149 before applying the corrections.

[CI run 36816149591](https://github.com/rgomids/axiom/actions/runs/36816149591)
passed on commit `5b3d5f474f17a506a7d59c1207c030c26f7b65e4`:

- Windows Server 2022 amd64: complete native Go suite, static/build/module
  checks, and PowerShell 5.1 offline fresh-install/reinstall/upgrade contract.
- Linux and macOS: race-enabled Go suites, static/build/module checks,
  repository validators, and bounded dogfooding.
- Release artifact and publication-flow contracts: passed.

The Server installation result above exercised a host that the approved
Specification excludes. It is regression evidence only, not Windows 10/11
acceptance or authority to expand the support boundary.

Integration with the newer Project edit/list and recorded-source changes keeps
host-absolute fixture paths, JSON path escaping, and actual Windows DACL tests.
Release checks retain all four platforms and the six current Runtime skills.

A local development build and `go vet ./...` also passed. Executing that build
for `version`/`help` was blocked by Smart App Control; local interactive use is
not claimed as validated. Signing/trust compatibility with that policy is not
delivered by this port. No security policy was changed.

## Validation limitations / release gates

The local machine's Application Control policy blocks some newly built Go test
executables (Windows error 4551). The policy was not disabled or bypassed. Such
runs are not reported as passes. The corrected offline PowerShell installer
contract did execute successfully on Windows 11; the current results are below.
Windows Server CI runs the complete Go suite and verifies executable installer
host refusal; Linux/macOS CI retains race and release checks. Final CI results
must be reviewed before merge and before publication.

POSIX chmod/UID and shell-fixture cases remain POSIX-specific. Windows permission
tests inspect real DACLs instead of interpreting Go's synthetic mode bits.
Symlink tests skip only when the OS denies the creation privilege; native junction
refusal is tested independently. Windows may itself prevent an open ancestor from
being renamed, so those POSIX attack fixtures report that prevention explicitly.

No live release was downloaded, published, tagged, or installed into the user's
normal binary directory. Clean-account latest-release acceptance is a release
gate once a release containing the new Windows asset is available. Physical
power-loss durability, remote filesystems, Windows ARM64 and code signing are
outside the approved initial support boundary.

## Review corrections — 2026-10-01

- **Host boundary:** Go checks `VER_NT_WORKSTATION`, major version 10 and build
  17763 or newer. The PowerShell bootstrap checks the same client-only boundary
  before staging/network activity. Server and domain-controller rows are refused.
  `TestSupportedWindowsVersion` covers client minimum/newer builds, older clients,
  Server 2022, domain controllers and unknown product types.
- **CI meaning:** `windows-2022` remains a native regression runner. Its executable
  installer contract now requires `unsupported_host` and no target mutation;
  it explicitly reports client installation acceptance as not run. Internal
  install tests inject a client row only within the test process to retain the
  filesystem/reinstall/upgrade matrix. Production has no host-policy override.
  Actual Windows 10/11 installation acceptance remains required separately.
- **Private creation:** Go supplies the final protected security descriptor to
  `NtCreateFile(FILE_CREATE)` relative to the anchored parent handle, with
  reparse traversal refused. It does not reopen a path to harden it afterward.
  Tests cover the resulting protected/private DACL, exclusive creation, nested
  relative names and refusal of traversal/absolute/ADS names. PowerShell uses
  the directory-creation security-descriptor overload (Framework), or
  `FileSystemAclExtensions.CreateDirectory` (Core), before storing bytes.
  See the [Windows object attributes contract](https://learn.microsoft.com/en-us/windows/win32/api/ntdef/ns-ntdef-_object_attributes)
  and [.NET creation API](https://learn.microsoft.com/en-us/dotnet/api/system.io.filesystemaclextensions.createdirectory).
- **SDD state:** the Specification and ADR indexes now match implementation
  authorization / acceptance for the requested implementation dated 2026-09-30.
  They do not claim merge, human product acceptance or release authority.

### AC-W02 executable matrix

`scripts/test-windows-bootstrap.ps1` executes the production bootstrap with an
in-memory HTTP transport and test-only host facts. Validation, hashing, tar
inspection, private staging and cleanup are production code; no release network
or installer executable is invoked. Every refusal checks the expected error,
absence of both target directories and absence of leftover staging directories.

| Requirement | Fixture / assertion |
|---|---|
| HTTPS → HTTP redirect | HTTPS response redirects to HTTP; only one transport request occurs, no HTTP request reaches transport |
| Missing / duplicate checksum | Empty checksum set / duplicate exact asset entries refused before extraction |
| Tampered archive | Wrong SHA-256 refused |
| Malicious archive paths | Parent traversal, absolute Unix/drive paths, nested traversal, ADS and backslash paths; valid outer checksum |
| Invalid release metadata | Invalid tag, draft/prerelease, missing booleans, wrong boolean type and non-object JSON |
| Unsupported architecture / host | ARM64 and Server refused before downloads |

The existing `internal/install` candidate-validation tests additionally reject
dirty bundle metadata, injected versions, skill-manifest skew and links before
installation. The bootstrap matrix is a new dedicated Windows CI step.

### Results and remaining work for this correction

- `powershell -NoProfile -File scripts/test-windows-bootstrap.ps1`: passed the
  complete matrix above, including strict metadata-shape cases, on Windows 11
  build 26200, PowerShell 5.1, local NTFS and an isolated generated test profile.
- `powershell -NoProfile -File scripts/test-windows-install.ps1`: passed on the
  same Windows 11 host with Go 1.26.0 and `TEMP`/`TMP` set to a user-private
  validation directory. Fresh install, unchanged reinstall, owned upgrade,
  in-use binary preservation, checksum refusal and invalid selectors passed.
- `gofmt` applied to the changed Go files. `go vet ./...`, `go build ./...` and
  `go mod verify`: passed. `go test ./internal/install -count=1 -timeout=3m` passed
  as part of the targeted three-package run.
- The targeted `go test ./internal/windowsfs ./internal/install ./internal/local
  -count=1 -timeout=3m` run failed overall: Application Control refused to launch
  the `windowsfs` and `local` test executables. No compile error was reported;
  those two runtime results require CI evidence.
- PowerShell 5.1 parser: passed for the final bootstrap and both Windows test
  scripts. `git diff --check`: passed.
- `bash --login scripts/check-sensitive-files.sh .`: passed for the worktree.
- `bash --login scripts/validate-repository.sh .` reached the sensitive-file and
  package-structure checks (passed), then failed the existing negative fixture:
  `package containing a symlink should fail validation`. The repository validator
  is not reported as passing on this Windows session; POSIX CI must
  execute the full validator as configured.
- The temporary session sandbox/network restriction was removed by the user.
  Go was downloaded from the official distribution and its SHA-256 verified;
  GitHub access was restored. No Windows security policy was changed.
- A new four-job CI result for this correction remains required before review;
  it will be recorded after the branch is pushed and the workflow completes.

The PR remains **not ready for merge** until the current correction has executable
validation. Earlier green runs do not satisfy that gate.

## Security review

- No implicit elevation, policy bypass, persistent PATH edits, credentials or
  Runtime installation.
- Checksummed HTTPS downloads with bounded responses and cancellation, private
  staging, closed Go bundle validation, and foreign-target preservation.
- Atomic staged no-replace fresh publication; existing owned upgrade protocol is
  reused. Interrupted/ambiguous state is preserved rather than adopted silently.
- User/SYSTEM/Administrators trust is explicit; reparse points, alternate streams,
  non-NTFS storage, unsafe DACLs and changed object identities fail closed.
- Native `.exe` Runtime names are supported; shell executable extension aliases
  do not bypass the existing direct-executable restriction.
- LF checkout attributes protect byte-level fixture and embedded-skill digests.

# Windows installer — issue #189

## Scope and acceptance

Requested bug correction against `main` at `4f99958`, validated on 2026-10-05.
The installation command must work on supported Windows PowerShell hosts,
including the original `Invoke-Expression` case. Default destinations must work
when their storage is eligible; refusals must identify the checked path, rule,
and an actionable alternative. Explicit destinations and `first-run` remain
supported. HTTPS, SHA-256, archive checks, private storage and ancestor checks
must stay intact. No global permission changes, project mutations, release,
merge, or external acceptance are part of this delivery.

Plan: isolate PowerShell parameter binding; preserve handle-based filesystem
diagnostics through the local adapter and native installer; extend existing
bootstrap/installation regressions; validate real Windows commands. This keeps
ADR-0010's storage policy and introduces no new trusted principal or dependency.

## Confirmed causes

- PowerShell 7.6 reproduces the original `Channel` attribute error with
  `Get-Content scripts/install.ps1 -Raw | Invoke-Expression` on the baseline.
  `Invoke-Expression` executes the parameter declaration in the caller's scope;
  even an omitted selector fails its validation. A child scriptblock fixes the
  binding and prevents overwriting a caller's `Channel`. Tests cover absent,
  empty and invalid caller values. Historical caller state remains unknown.
- On the validation machine, the first rejected ancestor is the profile's
  `AppData` directory. Its DACL grants `FullControl` (mask `0x001f01ff`) to an
  additional capability SID outside the trusted set. `AppData/Local` also has
  that grant. The handle-based checker correctly refuses replacement access.
  The historical issue did not retain ACL snapshots, so this establishes the
  current machine's cause, not proof of its exact historical ACL state.
- A pre-existing default installation now takes the upgrade preview path, which
  previously reduced the cause to `upgrade: unsafe_target`. The category remains
  stable; the Windows installer explicitly displays the retained cause.
- `os.Root`-relative file names can describe a parent instead of the checked
  object. Diagnostics resolve the path from the checked Windows handle.

## Changes and regressions

- `scripts/install.ps1`: execute in a child scope and forward explicit arguments.
- Windows filesystem checks: retain the sentinel, quoted path, rule and, when
  applicable, refusing SID/mask. No permission decision changes.
- Local ancestor/private checks: preserve the cause; POSIX refusals retain their
  existing sentinel behavior. Native installer prints storage recovery guidance.
- Offline bootstrap suite: actual IEX invocation and existing HTTPS downgrade,
  checksum, malformed archive, metadata and host refusal coverage, under both
  Windows PowerShell 5.1 and PowerShell 7.6. CI now exercises both shells.
- Native install suite: safe isolated defaults, paths with spaces, same-release
  reinstall, upgrade, occupied executable and checksum refusal; unsafe ancestor
  diagnostics for fresh and existing installs with preserved files/permissions.
- Go regression: exact rejected ancestor/private-object path, rule, SID,
  `errors.Is` compatibility and absence of newly created unsafe targets.

## Native validation

Host: Windows client amd64, PowerShell 7.6 and Windows PowerShell 5.1,
Go 1.26.0. Test temporary storage was explicitly placed beneath the user profile
because this machine's default Temp is below the refused AppData ancestor.
No existing profile ACL was changed. Test ACL grants affect isolated fixtures only.

Real published release installation used the corrected local bootstrap:

```powershell
& .\scripts\install.ps1 -Version v0.4.2 `
  -BinDir "$env:USERPROFILE\AxiomInstall\bin" `
  -ReceiptDir "$env:USERPROFILE\AxiomInstall\install"
$env:PATH = "$env:USERPROFILE\AxiomInstall\bin;$env:PATH"
axiom version
axiom help
axiom first-run
```

Results: `install_status=installed`; Axiom `0.4.2`, revision `df7eb1ab65ca`,
source `clean`; help succeeded; `first-run` succeeded with Codex and Claude
`already_configured`, all six skills per runtime `equivalent`. PATH was set only
for the validation process. The previous installation was preserved.

The locally built native installer was also run against the actual default
destinations with the published v0.4.2 archive and checksums: expected refusal,
reporting `AppData`, the ancestor replacement rule, SID/mask, and `install_next`.
Default-path success was validated in an isolated safe LOCALAPPDATA; success in
this machine's real AppData is neither claimed nor forced.

The exact published scriptblock command from README was also executed against
the public `main` URL with LOCALAPPDATA scoped to an isolated eligible directory:
`install_status=installed`, v0.4.2 confirmed by `version`. This exercises actual
HTTPS downloads and release checksums, beyond the offline fixtures.

Completed checks:

- `scripts/test-windows-bootstrap.ps1`: pass on PowerShell 5.1 and 7.6.
- `scripts/test-windows-install.ps1`: pass on PowerShell 5.1 and 7.6, including
  `windows_default_paths`, `windows_storage_diagnostic`, and
  `windows_upgrade_storage_diagnostic`.
- `go test ./... -timeout 10m -skip '^TestUpgradeResolvesForwardTransitionPolicy/unsafe_symlink_entry$'`:
  pass across all packages; exactly the privilege-blocked subtest is excluded.
- `go vet ./...`, `go build ./...`, `go mod verify`: pass.
- Cross-compilation with `go build ./...`: Linux amd64 and macOS arm64 pass.
- `python scripts/check-automation-registry.py .`, sensitive-file checks, and
  `git diff --check`: pass.

Review: permission masks, trusted identities, checksum/HTTPS verification and
archive extraction rules are unchanged. Negative tests verify refusal before
publication and preservation of owned binaries/receipts. No blocking finding
was identified in the scoped diff; the environment limits below remain explicit.

## Verification limits

- The complete `go test ./... -timeout 10m` run failed only at
  `TestUpgradeResolvesForwardTransitionPolicy/unsafe_symlink_entry`: Windows
  denied the privilege required to create its symlink fixture. No policy or
  developer-mode setting was changed to bypass this limitation.
- Repository validation was attempted with Git Bash. It refuses this checkout's
  `.claude/skills` adapters because Git materialized their symlinks as files.
  This is a checkout limitation; those unrelated files were not changed.
- No Linux/macOS runtime execution or native race-detector run is claimed.
- v0.4.2 does not contain the new native diagnostics. Local validation of the
  correction is not a release, remote CI result, or maintainer acceptance.

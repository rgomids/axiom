# Installation

Install a published Axiom binary without a checkout or Go toolchain. For the
short first workflow, see [Getting Started](../README.md#getting-started).
This guide covers installation and diagnosis; [Commands](commands.md) owns
full flags, output contracts, and maintenance operations.

## Supported platforms

| Host | Architecture | Storage / prerequisites |
|---|---|---|
| Linux | amd64 or arm64 | Safe local storage; POSIX utilities below |
| macOS | arm64 | Safe local storage; POSIX utilities below |
| Windows client edition | amd64 | Local NTFS, 64-bit PowerShell 5.1+, Windows-supplied `tar.exe` |

Windows Server and other OS/architecture combinations are refused. Numeric OS
versions are not installer eligibility filters. Maintenance covers
vendor-maintained OS versions; validated environments remain recorded Evidence,
not a promise for every host configuration. See
[host eligibility policy](decisions/0015-installer-host-eligibility-os-family-architecture.md).
Older releases may lack an asset for your platform.

## Prerequisites

- Linux/macOS: `sh`, `bash`, `curl`, `tar`, `awk`, `grep`, `mktemp`, and
  `sha256sum` or `shasum`.
- Windows: the native tools in the platform table; no WSL or Bash required.
- Network access to GitHub release metadata and assets over HTTPS.
- A supported Runtime, installed separately, for Runtime skills. Project setup
  itself does not require a Runtime.
- An authenticated [GitHub CLI](https://cli.github.com/) session for real GitHub
  Work Item operations. Binary installation and `first-run` do not authenticate.

No administrator privileges or `sudo` are needed. Organization application
control and PowerShell policies still apply; Axiom does not bypass them.

## Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh
```

The bootstrap selects latest stable, verifies the release archive's SHA-256,
validates the bundle, and invokes that release's installer. Defaults:

- Binary: `$HOME/.local/bin/axiom`.
- Receipt directory: `${XDG_STATE_HOME:-$HOME/.local/state}/axiom/install`.

The macOS asset name retains `macos-27-arm64`; this is not an OS-version gate.
See [bootstrap reference](commands.md#install-a-published-release-s9t39).

## Windows

```powershell
& ([scriptblock]::Create((Invoke-RestMethod https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.ps1)))
```

The bootstrap verifies the checksum and complete bundle before publishing
`axiom.exe` and its installation receipt. Defaults:

- Binary: `%LOCALAPPDATA%\Axiom\bin\axiom.exe`.
- Receipt directory: `%LOCALAPPDATA%\Axiom\install`.
- Machine-local state: `%LOCALAPPDATA%\Axiom\state`.

To inspect or retain the bootstrap first:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.ps1 -OutFile install-axiom.ps1
.\install-axiom.ps1
```

See [Windows reference](commands.md#windows-native-installation) for offline
installation and the exact filesystem boundary.

## PATH configuration

The installers never change shell profiles or persistent `PATH`. If the summary
reports `PATH setup required`, add the selected binary directory to the current
session:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

```powershell
$env:PATH = "$env:LOCALAPPDATA\Axiom\bin;$env:PATH"
```

Match these paths to your chosen installation directory. For persistence, add
that directory through your shell or Windows user environment configuration.
Then verify the executable before Runtime bootstrap:

```bash
axiom version
axiom help
```

## Installing a specific version

Use an exact **published** tag. These examples prompt for a tag rather than
pinning a release that will age:

```bash
printf 'Published tag (vMAJOR.MINOR.PATCH or vMAJOR.MINOR.PATCH-rc.N): '
read -r AXIOM_TAG
curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh -s -- --version "$AXIOM_TAG"
```

```powershell
$axiomTag = Read-Host 'Published tag (vMAJOR.MINOR.PATCH or vMAJOR.MINOR.PATCH-rc.N)'
& ([scriptblock]::Create((Invoke-RestMethod https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.ps1))) -Version $axiomTag
```

No selector installs latest stable. `--channel stable` / `-Channel stable` are
explicit equivalents. Version and channel selectors are mutually exclusive;
release candidates require an exact tag, with no RC channel. There is no
fallback from stable to a candidate. The selected release must contain your
platform's archive. See [Releases](https://github.com/rgomids/axiom/releases).

## Custom installation directories

Choose absolute, canonical, safely owned local directories:

```bash
curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh -s -- \
  --bin-dir "$HOME/AxiomInstall/bin" \
  --receipt-dir "$HOME/AxiomInstall/install"
export PATH="$HOME/AxiomInstall/bin:$PATH"
```

```powershell
& ([scriptblock]::Create((Invoke-RestMethod https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.ps1))) `
  -BinDir "$env:USERPROFILE\AxiomInstall\bin" `
  -ReceiptDir "$env:USERPROFILE\AxiomInstall\install"
$env:PATH = "$env:USERPROFILE\AxiomInstall\bin;$env:PATH"
```

These examples must pass the same checks as defaults, including ancestor
checks; they are not guaranteed eligible on every machine. Binary and receipt
options do not relocate Project state or Runtime skills. `LINGO_STATE_ROOT`
selects a separate absolute state root, subject to its own safety checks.

## Isolated Windows fresh-install test

Use this diagnostic test when the default Windows paths fail permission checks.
It selects new binary, receipt, state, Project and Codex skill locations without
deleting existing installations or changing their ACLs. Run the complete block
in one PowerShell window, from any working directory:

```powershell
$axiomRoot = Join-Path $env:USERPROFILE ('AxiomFresh-' + [guid]::NewGuid().ToString('N'))
$env:LINGO_PROJECTS_ROOT = "$axiomRoot\projects"
$env:LINGO_STATE_ROOT = "$axiomRoot\state"
$env:AXIOM_CODEX_SKILLS_ROOT = "$axiomRoot\skills"
$axiomDestinations = @{
    BinDir = "$axiomRoot\bin"
    ReceiptDir = "$axiomRoot\install"
}
$axiomInstaller = Invoke-RestMethod https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.ps1
& ([scriptblock]::Create($axiomInstaller)) @axiomDestinations
if ($LASTEXITCODE -ne 0) { throw 'Installation failed; stop before verification.' }
$env:PATH = "$axiomRoot\bin;$env:PATH"
& "$axiomRoot\bin\axiom.exe" version
```

Success verifies binary installation and version reporting only. These locations
and their ancestors must still pass the same filesystem checks. The variables
and PATH apply only to this window; a new window returns to the configured
defaults. Reuse the same destinations to test reinstall; running this block
again creates another fresh installation.

The Codex skill override is for isolated validation: Codex does not automatically
discover skills at this custom location. This test does not configure Claude's
skill root or validate either Runtime integration. For normal use, resolve the
reported safety issue in the standard Runtime roots before running
[`first-run`](#runtime-bootstrap). Do not copy skills into a rejected root or
remove security checks to make integration succeed.

On Windows, path names are case-insensitive: `$env:USERPROFILE\Axiom` can be
the same directory as an `axiom` repository checkout. Keep installation targets
separate from source repositories. The version command is `axiom version`;
`axiom --version` is not supported.

## Upgrade / reinstall

Rerun the installer to converge an Axiom-owned installation to latest stable.
Equivalent reinstall is a no-op. Modified or foreign targets are preserved;
downgrades are refused. Close running Axiom processes before Windows upgrades.
After successful installation, verify `axiom version` and rerun `axiom first-run`
to converge integrations for every detected Runtime.

Compatibility is checked before replacing the binary. Supported v1 state stays
active. A recognized historical POC transition preserves and verifies an archive
before retiring historical workflow material and rebuilding compatible active
state; it does not reinterpret POC history as current Executions. Success reports
`install_preserved=<archive>` when preservation occurred. Keep that archive until
you decide it is no longer needed.

An interrupted or partial upgrade may have confirmed some effects. Preserve its
receipt, operation marker, and archive; follow the reported next action. Resume
requires the same candidate archive and valid authority. Do not delete markers
or retry with a different candidate. There is no automatic rollback or update.
See [compatibility, recovery, and upgrade](commands.md#compatibility-cleanup-recovery-and-upgrade)
for the exact preview/authorization protocol.

## Runtime bootstrap

```bash
axiom first-run
```

Axiom detects `codex` and `claude` executables on `PATH` without running them.
It installs or upgrades six user-global thin skills, independently per Runtime:

| Runtime | Skill root |
|---|---|
| Codex | `$HOME/.agents/skills` |
| Claude | `<CLAUDE_CONFIG_DIR or ~/.claude>/skills` |

It never installs a Runtime, signs you in, changes credentials, or infers a
Project. No detected Runtime is success with no effect; make the executable
available and rerun. A detected Runtime that fails configuration makes the
command fail while retaining other successful results. Inspect integrations:

```bash
axiom runtime codex status
axiom runtime claude status
```

Use `$axiom-project-configure` in Codex or `/axiom-project-configure` in Claude.
For the complete skill set and receipt/conflict diagnostics, see
[Runtime reference](commands.md#first-run-and-runtime-integrations).

## Security model

The bootstrap downloads over HTTPS and verifies archives against the selected
release's `SHA256SUMS` before extraction; this relies on trust in the repository,
bootstrap, and release publication source. Bundle metadata and manifests are
validated before publishing installation artifacts.

Ownership, permissions, ACLs, links, and ancestor directories are checked.
On POSIX an existing binary or Runtime skill root can be `0755` when user-owned,
not writable by group/other, and without extended ACLs; private receipt/state
storage has stricter requirements. On Windows storage must be local NTFS with
safe owners/DACLs. Network/device paths, junctions and other reparse points are
refused. Axiom does not weaken permissions to make an installation pass.

Unknown or modified binaries, skills, receipts, and ambiguous compatibility
state fail closed. Checksums do not grant authority to overwrite foreign data.
See [repository security](security/repository-security.md) and the
[filesystem threat model](decisions/0005-bounded-local-filesystem-threat-model.md)
for guarantees and limits.

## Troubleshooting

| Symptom | Next action |
|---|---|
| `PATH setup required` or command not found | Add the selected binary directory to this session's `PATH`; verify version and help. |
| Unsupported host or missing release asset | Check OS/architecture and prerequisites above; select a published release containing your asset. |
| Checksum or bundle validation failure | Stop using the archive; obtain the exact published archive and matching `SHA256SUMS` again. |
| `unsafe project storage` | Choose eligible local storage; check ownership, ACLs and ancestors. On Windows use local NTFS without junctions/reparse points. Do not disable checks. |
| `upgrade: state_unsafe` | Inspect the reported Project, state or skill location. Moving the binary or setting `LINGO_STATE_ROOT` alone does not repair all roots. Preserve uncertain state. |
| Foreign/modified installed binary or receipt | Preserve the target and inspect ownership; use another eligible installation location when appropriate. |
| Locked Windows executable | Close Axiom processes and resolve the lock before retrying the same installation. |
| No Runtime detected | Install Codex/Claude separately, expose its executable on `PATH`, and rerun `first-run`. Configuration directories alone do not prove availability. |
| `<runtime>_skill_conflict` | Inspect status and listed artifacts/receipt; preserve modified or foreign content. Move it aside only after review, then rerun bootstrap. |
| `<runtime>_skill_root_unavailable` | Check root/ancestor safety and absolute `CLAUDE_CONFIG_DIR`; do not broaden permissions. |
| Partial bootstrap | Inspect each Runtime result; successful integrations remain. Resolve the failed Runtime and rerun. |
| Partial/interrupted upgrade | Preserve marker, receipt and archive; follow the exact diagnostic and [recovery reference](commands.md#compatibility-cleanup-recovery-and-upgrade). Run `first-run` only when the diagnostic or a successful binary installation calls for it. |

For unresolved installation problems, include sanitized version, host, selected
paths, and diagnostic categories in a [support request](../SUPPORT.md).

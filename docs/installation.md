# Installation

Install a published Axiom binary without a checkout or Go toolchain. For the
short first workflow, see [Getting Started](../README.md#getting-started).
This guide covers installation and diagnosis; [Commands](commands.md) owns
full flags, output contracts, and maintenance operations.

## Supported platforms

Supported hosts: macOS arm64 and Linux amd64/arm64.

| Host | Architecture | Storage / prerequisites |
|---|---|---|
| Linux | amd64 or arm64 | Safe local storage; POSIX utilities below |
| macOS | arm64 | Safe local storage; POSIX utilities below |

Other OS/architecture combinations are refused. Numeric OS
versions are not installer eligibility filters. Maintenance covers
vendor-maintained OS versions; validated environments remain recorded Evidence,
not a promise for every host configuration. See
[host eligibility policy](decisions/0015-installer-host-eligibility-os-family-architecture.md).
Older releases may lack an asset for your platform.

## Prerequisites

- `sh`, `bash`, `curl`, `tar`, `awk`, `grep`, `mktemp`, and `sha256sum` or
  `shasum`.
- Network access to GitHub release metadata and assets over HTTPS.
- A supported Runtime, installed separately, for Runtime skills. Project setup
  itself does not require a Runtime.
- An authenticated [GitHub CLI](https://cli.github.com/) session for real GitHub
  Work Item operations. Binary installation and `first-run` do not authenticate.

No administrator privileges or `sudo` are needed. Organization application
control policies still apply; Axiom does not bypass them.

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

## PATH configuration

The installers never change persistent PATH or shell profiles. If the summary
reports `PATH setup required`, add the selected binary directory to the current
session:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Match this path to your chosen installation directory. For persistence, add
that directory through your shell configuration.
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

No selector installs latest stable. `--channel stable` is the explicit
equivalent. Version and channel selectors are mutually exclusive;
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

These examples must pass the same checks as defaults, including ancestor
checks; they are not guaranteed eligible on every machine. Binary and receipt
options do not relocate Project state or Runtime skills. `LINGO_STATE_ROOT`
selects a separate absolute state root, subject to its own safety checks.

## Upgrade / reinstall

Rerun the installer to converge an Axiom-owned installation to latest stable.
Equivalent reinstall is a no-op. Modified or foreign targets are preserved;
downgrades are refused.
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
It installs or upgrades three canonical user-global thin skills, independently per Runtime:

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

The current product skill set is exactly `axiom-project`, `axiom-work-item` and `axiom-workflow`.
Historical releases through v0.10.0 carried six or eight skills. Upgrade removes
obsolete entries only after verifying Axiom ownership and known content. Modified,
foreign, linked or unrecognized entries and receipts are preserved as conflicts;
no complete skill root or unrelated Runtime configuration is deleted. Interrupted
cleanup keeps truthful partial state; rerun the diagnosed install/upgrade path.
Historical ownership metadata remains internal, not an invocation surface.

Project selects with `axiom-project`, Workflow configures with `axiom-workflow`,
and Work Item runs/plans with `axiom-work-item`. Use `$<skill-name>` in Codex or
`/<skill-name>` in Claude; both install the same canonical skill bytes.

For a v0.15.0 installation, use the **new release installer** to upgrade to the
three-skill inventory. The installed v0.15.0 `axiom upgrade --archive <new>` parser
refuses the three-skill archive before effects. After binary upgrade, rerun
`axiom first-run` to converge Claude as well as Codex; reinstall is idempotent.
Foreign or modified skill bytes remain conflicts and are preserved. Downgrade
is not supported. See the [upgrade reference](commands.md#compatibility-cleanup-recovery-and-upgrade).
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
storage has stricter requirements. Axiom does not weaken permissions to make an
installation pass.

Ordinary storage operations never rewrite existing ACLs. Unknown or modified binaries, skills, receipts, and ambiguous compatibility
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
| `unsafe project storage` | Choose eligible local storage; check ownership, ACLs and ancestors. Do not disable checks. |
| `upgrade: state_unsafe` | Inspect the reported Project, state or skill location. Moving the binary or setting `LINGO_STATE_ROOT` alone does not repair all roots. Preserve uncertain state. |
| Foreign/modified installed binary or receipt | Preserve the target and inspect ownership; use another eligible installation location when appropriate. |
| No Runtime detected | Install Codex/Claude separately, expose its executable on `PATH`, and rerun `first-run`. Configuration directories alone do not prove availability. |
| `<runtime>_skill_conflict` | Inspect status and listed artifacts/receipt; preserve modified or foreign content. Move it aside only after review, then rerun bootstrap. |
| `<runtime>_skill_root_unavailable` | Check root/ancestor safety and absolute `CLAUDE_CONFIG_DIR`; do not broaden permissions. |
| Partial bootstrap | Inspect each Runtime result; successful integrations remain. Resolve the failed Runtime and rerun. |
| Partial/interrupted upgrade | Preserve marker, receipt and archive; follow the exact diagnostic and [recovery reference](commands.md#compatibility-cleanup-recovery-and-upgrade). Run `first-run` only when the diagnostic or a successful binary installation calls for it. |

For unresolved installation problems, include sanitized version, host, selected
paths, and diagnostic categories in a [support request](../SUPPORT.md).

# Installation

Install a published release on macOS arm64 or Linux amd64/arm64. Installation
uses HTTPS and verifies the published archive checksum before extraction. It
does not require a checkout, Go toolchain, administrator privileges or a
Runtime. The host needs `sh`, `bash`, `curl`, `tar`, `gzip`, `awk`, `grep`, `mktemp` and
`sha256sum` or `shasum`.

Install the latest stable release:

```sh
curl -fsSL https://raw.githubusercontent.com/rgomids/axiom/main/scripts/install.sh | sh
```

The default executable is `$HOME/.local/bin/axiom`. The receipt is under
`${XDG_STATE_HOME:-$HOME/.local/state}/axiom/install` on both Linux and macOS.
Add the binary directory to the current shell's `PATH` if needed, then check:

```sh
axiom version
axiom help
```

For a specific published tag, pass `--version` to the bootstrap. For custom
locations, pass absolute `--bin-dir` and `--receipt-dir` values. Installation
does not edit shell profiles or move Project data, Runtime skills or local
state roots. These are bootstrap options; they are not Axiom CLI commands.

## Install from a trusted local archive

When network access is unavailable, obtain the archive and its matching
`SHA256SUMS` file through a separately trusted delivery channel. A checksum
detects byte mismatch against that file; it does not establish who supplied the
archive or checksum. Verify that trust before installation. Then invoke the
release installer with absolute paths:

```sh
/absolute/path/unpacked/axiom/install.sh \
  --archive /absolute/path/axiom-<version>-<platform>.tar.gz \
  --checksums /absolute/path/SHA256SUMS \
  --bin-dir /absolute/user-owned/bin \
  --receipt-dir /absolute/user-owned/install-state
```

The archive must match the host and checksum entry. The installer validates the
bundle and candidate before publishing. It uses a private executable temporary
workspace (`TMPDIR` when set) and refuses an unsafe or non-executable workspace.
This flow performs no network fetch and a checksum beside an untrusted archive
does not bootstrap trust.

Rerunning the installer performs an equivalent no-op or a protected owned
upgrade. Modified or foreign targets are preserved; downgrades and divergent
same-version replacements are refused. The CLI command reference does not
document bootstrap flags. An interrupted operation can have
confirmed effects. Preserve its marker, receipt and archive, then follow the
reported next action and [recovery guide](troubleshooting.md). Do not substitute
a different archive. Axiom has no automatic rollback or downgrade.

After verifying the binary, `axiom first-run` can install/update Axiom skills
for detected Codex and Claude executables. It does not install those Runtimes,
authenticate, change credentials or select a Project. See [Runtime](runtime.md).

Continue to [Project](project.md) to start using Axiom.

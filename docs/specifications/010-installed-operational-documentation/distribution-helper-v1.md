# Versioned verifier distribution contract — v1

## Authority and scope

Rafael's Option A, 2026-10-21 20:44 BRT, authorizes this bounded amendment in
[PR #335](https://github.com/rgomids/axiom/pull/335). Applies to distribution,
integrity and bootstrap, independently of archive generation. Extends the
producer/consumer contract in [Plan R1](plan-reconciled.md) and
[P1 contracts](contracts-p1-resume-3.md), preserving ADR-0033/0034's trusted
acquisition boundary, ADR-0005's bounded host model and ADR-0016's candidate
receipt proof. No new trust service, privileged process or archive member.
No default-v2 promotion, release, human acceptance or native/Q6 qualification.

## Closed release inventory and identity

A new release contains exactly seven regular assets: `SHA256SUMS`, the existing
three version/platform archives, and three separate verifier executables named
`axiom-verifier-<version>-<full-source-commit>-<goos>-<arch>`. Targets are
`darwin-arm64`, `linux-amd64`, `linux-arm64`. Source is the exact Axiom release
checkout, entrypoint `cmd/axiom-archive-verify`; static Go stdlib only, CGO off.
Version follows the product release; the helper's contract version remains v1.
All helper names carry the same full commit. SHA256SUMS contains exactly six
unique digest/name rows. Archive member sets and v1/v2 metadata remain unchanged.
Release verification checks the closed set, source VCS revision/cleanliness,
entrypoint, build OS/architecture, binary header, checksum and native embedded
identity. Publication Evidence and envelope include all six digests. No new
asset is published by implementation or CI.

## Authentication before execution

Trusted bootstrap code and host tools are prerequisites. Remote bootstrap uses
HTTPS GitHub API `repos/rgomids/axiom/releases/tags/<tag>` to authenticate the
SHA256SUMS asset's `sha256:` digest, requiring one uploaded checksum asset,
exact non-draft tag and complete closed asset inventory. `jq` parses JSON;
missing tool/digest/API facts refuse. The API is an independently authenticated
control-plane anchor relative to downloaded release asset bytes, not a detached
signature or protection against a compromised GitHub/repository authority.
The resolved tag commit from the API must equal the full commit in helper
names, and the archive revision must match it. HTTPS assets alone, adjacent
checksums, archive-internal manifests and filenames alone do not authenticate a
helper. Redirect protocols remain HTTPS. No credentials or new permission scope.

Only after the helper bytes match this authenticated pin, the binary header
matches the actual host, and its asset name has the expected host suffix, may
bootstrap execute `identity`. Its response must equal the independently pinned
asset name, binding version, full commit and target. Then the helper verifies
compressed bytes and stages tar before bootstrap reads or executes the installer.
The release installer independently restages and repeats pin/header/identity
checks and strict verification before candidate execution. The helper cannot
prove its own authenticity: checksum authentication always precedes execution.

## Offline acquisition and legacy compatibility

An authorized acquisition operator transfers the exact helper, trusted installer,
archive, matching checksums and helper digest/provenance through a trusted offline
handoff. Record repository, release tag, full commit, target, authenticated digest
and acquisition channel in that handoff. Authenticate the pin independently of
untrusted media; copying a helper and a matching adjacent checksum is not proof.
Use explicit `--helper <absolute-file>` and `--helper-sha256 <trusted-pin>`;
`AXIOM_VERIFIER_PATH` and `AXIOM_VERIFIER_SHA256` are equivalent explicit operator
inputs, never PATH discovery. No compiler, Python, download or installed candidate
is used to bootstrap trust offline.

Historical archives and their original three-entry checksum files remain valid
inputs to the new local facade with an explicitly pinned compatible v1 helper.
The helper's own release identity is distinct from the historical archive's
identity: a newer helper can verify older v1/v2 archives. Remote historical tags
without a helper require the same explicit local handoff; absent helper refuses,
never selects another release automatically. Archived old installer bytes cannot
retroactively gain strict framing. v1 permits complete gzip members splitting
the tar stream and empty prefix/suffix members around a valid tar; an empty
archive, invalid split, bad CRC/truncation or any zero/non-gzip compressed tail
refuses. v2 retains exactly one member. No Python prerequisite remains in the
new facade. Default archive generation stays 1.

## Bounds, staging and effects

Helper compressed input <=256 MiB; expanded tar <512 MiB; helper executable
<=16 MiB; checksum inventory <=64 KiB; API response <=1 MiB. v1 <=64 tar
entries; v2 <=192 and <=288 MiB expanded. Binary payload <=256 MiB; other
files <=1 MiB; existing v2 corpus/manifest/quota/mode rules remain enforced by
the facade and Go candidate reader. Helper validates every gzip member's CRC
and framing using stdlib, tar path/root/type/duplicates and explicit two-block
EOF with zero padding, before creating its exclusive 0600 tar staging leaf.
No custom DEFLATE parser. Manifests, skill identity, metadata, permissions,
locks, target confinement and installation quotas retain their existing checks.

Bootstrap and facade stage copies in private 0700 temporary directories, verify
copied bytes, refuse symlink leaves and unsafe ancestors under the existing
root-owned macOS alias allowance. No helper is persisted in destination or
installed on PATH. No candidate or durable destination is touched before full
verification. Temporary files are removed on failure/exit. Existing atomic
publication, lock, truthful partial-effect diagnostics and recovery remain;
there is no automatic rollback or deletion of previous generations. Returning
to prior installation artifacts requires their matching trusted verifier handoff.

Diagnostics distinguish missing trust prerequisite, helper digest, platform,
release linkage, gzip/CRC/framing, tar and existing metadata/manifest mismatches.
Each failure stops the dependent chain; never downgrade verification or retry
through host gzip, Python, network or the candidate. Same-UID arbitrary process
manipulation remains outside ADR-0005; local hashes do not create authenticity.

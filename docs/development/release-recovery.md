# Recover an unpublished release with pinned corrections

Maintainer control-plane recovery; ADR-0010. It does not change product Runtime
contracts, source revision, release version or archive provenance.

This is an exception, not the release path. Since ADR-0011, `$axiom-release`
validates the delivery metadata and Conventional Commit subject of every
commit in the range before the Release PR exists, and a gap is fixed by a
normal reviewed correction on `main` before the release starts. Recovery stays
only for a release commit that already exists with inconsistent history
(v0.3.0) or a partially published state.

## Contract and authority

The maintainer explicitly authorized implementation on 2026-10-02. Review and
merge of the correction/control PR, exact-envelope publication authority and
approval of the GitHub `release` environment remain human gates.

Source SHA stays the merged Release PR commit that introduced the stable version.
Recovery is explicit: select a full correction/control SHA already reviewed,
merged to first-parent main and green in CI, plus the SHA-256 of its committed
`.github/delivery-corrections.txt`. The same pins are required by status, prepare
and publish. They are retained in `release-recovery.txt`, the publication envelope
and published notes; verify discovers published pins and validates them.

New rows may target only commits in the original release range; existing source
rows cannot be changed or removed. Complete grammar, duplicate targets, Issue
relationships, ancestor membership and digest are validated. RCs, floating refs,
unknown/unmerged revisions, stale digests, altered prepared metadata, changed
notes or a different published recovery refuse before effects.

Build and archive verification use the original clean checkout. Recovery scripts
use a separate clean pinned control checkout. Version, changelog, range and
Project configuration stay with original source. Archives and `SHA256SUMS`
retain their existing closed inventory; recovery metadata lives in prepared
metadata and GitHub release notes, not an extra release asset.

## After human merge

Use the full merged correction/control SHA, not the PR branch SHA. Compute the
digest from committed bytes and inspect the resulting facts:

```bash
correction_sha=<full-merged-sha>
correction_digest=$(git show "$correction_sha:.github/delivery-corrections.txt" | shasum -a 256 | awk '{print $1}')
./scripts/release.sh status --tag v0.3.0 \
  --corrections-revision "$correction_sha" --corrections-digest "$correction_digest"
```

Only when status permits preparation:

```bash
./scripts/release.sh prepare --tag v0.3.0 \
  --revision b79d3bf8cbf21247ca30cae06ff000e7f89a5adf \
  --corrections-revision "$correction_sha" --corrections-digest "$correction_digest"
```

Preparation publishes nothing. Show every preview line and digest, including the
two correction pins and Issue effects. Obtain authority for that exact envelope;
then use the existing publish command with the same two correction options.
`release.sh verify --tag v0.3.0 --download` reads pins from published notes.
Never run publication merely because these preparation instructions exist.

Ordinary historical delivery sync and Project reconciliation can discover pins
only from an already published, non-draft stable release matching the original
source SHA. Discovery is isolated from preparation and publication inputs.
Status of a recovered release still requires the explicit pins.

Automatic verification trusts published notes as the provenance record and
checks them against committed main history. It does not independently reconstruct
the originally authorized publication envelope. Manual post-publication editing
is forbidden; use the original explicit pins to detect replacement of that
record when auditing publication authority.

## Acceptance criteria

- #157 is related to #153 and completes none; only #86 is delivered by v0.3.0.
- Source remains `b79d3bf8`; build/verifier cannot switch to correction revision.
- Every consumer uses the same verified SHA/digest; default flow still passes.
- Malformed, duplicate, out-of-range, non-main and changed inputs fail closed.
- Prepared metadata and notes match explicit pins; envelope binds both pins.
- Published read-back verifies notes, pins, tag, archive digests and Issue records.
- No merge/publication/settings mutation occurs during implementation validation.

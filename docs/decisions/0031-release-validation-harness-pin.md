# ADR-0031 — Pin upgrade-journey validation separately from release artifacts

## Status

Proposed. Explicit human acceptance is required before merge or use in release
preparation. This proposal would partially supersede Decision 3–4 of ADR-0029
only for the upgrade-journey harness revision; ADR-0029 remains accepted.

## Context

Release preparation checks out the merged Release PR SHA for product builds and
validation. For v0.20.0, the upgrade journey could not parse current version
output. PR #332 fixed the harness on main, but preparation still ran the old
script from the immutable release source.

## Decision

Proposed: release-artifacts.yml passes its immutable github.sha to the native
suite as optional journey_ref. Dispatch is already restricted to main. Only the
upgrade harness uses this checkout, created after source builds. Candidate
archives, Go suites, contracts and all other product validation remain at the
release SHA. Other callers omit journey_ref and keep the original checkout and
execution behavior.

The preparation run already records its workflow head SHA and workflowRef.
No new dispatch input, artifact metadata requirement, lifecycle state or
publication authority is introduced. Previous preparations remain valid under
the existing verifier. A failed journey still fails the preparation run.

## Alternatives considered

- Explicit operator-supplied validation SHA: redundant with the immutable
  workflow SHA; adds dispatch, ancestry and metadata compatibility rules.
- Move all validation to main: loses release-source test coverage.
- Change the release source: changes the source identity to repair a harness.

## Consequences

One optional checkout lets a corrected harness qualify unchanged release
source. Reviews must distinguish harness code from tested product code. Native
Linux/macOS preparation remains required; green PR CI is insufficient. The
harness can itself change assertions, so review must preserve meaningful
upgrade checks. Publication still uses the verified prepared bytes and exact
human-authorized envelope; it never rebuilds artifacts.

## Revisit when

Upgrade validation gains an independently versioned harness contract.

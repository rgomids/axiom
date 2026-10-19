# ADR-0031 — Pin upgrade-journey validation separately from release artifacts

## Status

Accepted 2026-10-18 by explicit maintainer decision in this conversation.

## Context

Stable release artifacts are built from the exact merged Release PR commit.
The reusable native suite was also checked out at that commit, which meant its
upgrade-journey harness could not receive a later compatibility fix without
changing the release source or rewriting release history. A release preparation
therefore repeated an already diagnosed harness failure after its correction
merged to `main`.

## Decision

Keep the release source revision as the authority for release builds and all
native validation except the upgrade-journey harness. For release preparation,
pin that harness to the exact current `main` SHA after required CI is green.
The workflow verifies the pin is still `main` at execution, on its first-parent
history, and a descendant of the release source. Build the upgrade candidate
from the release source checkout; run only the journey script from the pinned
harness checkout. Record both SHAs in preparation metadata. Other callers of
the reusable native suite omit the harness pin and continue to use their
validation SHA for both checkouts.

The successful preparation run ID continues to bind the retained artifact set
and its metadata in the publication envelope. This decision does not change
artifact bytes, source provenance, publication authority, or the release
environment gate.

## Alternatives considered

| Alternative | Complexity, coupling and operations | Security, cost, portability and reversibility |
| --- | --- | --- |
| Pin the journey harness separately (chosen) | Adds one pin and checkout, but keeps product-source tests tied to the release. | Read-only SHA checkouts and ancestry validation; one extra checkout, standard GitHub Actions; easy to remove. |
| Move all release native validation to current `main` | Fewer pins, but couples release qualification to product code newer than the artifact. | Similar runner cost and portability; reversible, but weakens source-specific evidence while enabled. |
| Cherry-pick or rewrite the release commit | Requires a new source identity and repeated release preflight. | Preserves one-code revision at the cost of immutable provenance; hard to reverse once artifacts are authorized. |
| Do not rerun preparation | No implementation complexity. | No extra runner cost, but the release stays blocked without a verified prepared set. |

## Consequences

### Positive

Harness-only fixes can unblock release validation while the built candidate,
release source tests, and publication remain pinned to the merged Release PR
commit. The harness SHA is captured at dispatch and recorded with the prepared
set.

### Negative / trade-offs

The upgrade job checks out two immutable revisions and must build the candidate
from the source checkout while running the journey script from the harness
checkout. Reviewers must interpret the two pins according to their distinct
roles.

## Revisit when

The upgrade journey becomes part of a separately versioned validation harness,
or repository policy requires every test script to be co-located with the
tested source revision.

## Supersedes

- [ADR-0029 — Minimal PR gate with native qualification before release](0029-minimal-pr-gate-native-qualification.md): only the requirement that the upgrade-journey harness run from the release revision; all other release native-suite checks remain bound to that revision.

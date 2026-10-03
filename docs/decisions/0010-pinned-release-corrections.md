# ADR-0010 — Pinned metadata recovery for an unpublished release

## Status

Accepted direction — explicit human authorization to implement the proposed
recovery recorded on 2026-10-02. Implementation review, merge and publication
remain separately gated.

## Context

Stable release v0.3.0 was recorded by merged PR #160 before discovery that
PR #157's squash omitted delivery metadata. Corrections read only from the
original source commit cannot repair that immutable tree. Choosing newer main
would violate the release-source contract; rewriting history or claiming an
unpublished release is tagged would violate provenance and authority.

## Decision

Allow opt-in recovery with two immutable inputs: the unchanged release source
SHA and a separately selected correction/control SHA plus the committed
correction-file SHA-256. Both revisions must be in first-parent main history;
control must descend from source. New correction rows are bounded to the source
release range, and existing source corrections cannot be changed or removed.

The original source remains owner of version, changelog, release range,
archive build, archive verification and delivery Project configuration.
Recovery control scripts execute from the explicitly pinned reviewed revision.
Recovery metadata is retained with the prepared set, printed in the envelope,
and persisted in release notes. Its digest and notes are revalidated before
effects and after download. It adds no archive or tag and changes no source byte.

Default release behavior remains source-only. Recovery is stable-only; no
floating ref, worktree override or automatic fallback is accepted. Published
notes must retain the same pins; a recovery cannot be introduced or changed
silently after publication. Exact-envelope authorization and the protected
release environment remain required.

## Alternatives

- Stop permanently at the malformed immutable source: preserves existing
  rules but leaves this unshipped version unpreparable.
- Cancel/supersede through a new release protocol: would also require a
  truthful pending-PR cancellation model and skipped-release range accounting.
- Arbitrary later source SHA, false tagged label or history rewrite: rejected.

## Consequences

Recovery adds control-code provenance alongside artifact provenance. Preparation,
publication and verification must use the same pins; cross-phase integration
and negative drift tests are required. A metadata change affects notes and Issue
effects and therefore requires fresh human envelope authority. Issue title or
published notes drift is reported during re-verification.

Revisit if release notes need detached signatures or a broader cancellation
protocol. This ADR introduces neither product Runtime behavior nor human
acceptance of an Axiom release.

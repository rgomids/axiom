# ADR-0013 — Explicit control-code repair of a published recovery release

## Status

Proposed for human review on 2026-10-03. The maintainer explicitly selected
implementation of this repair after the v0.3.0 read-back failure. Merge accepts
the repository change; selecting the merged repair SHA, authorizing the exact
remaining-effects envelope and approving the environment remain separate gates.

## Context

v0.3.0 was published immutable at original source `b79d3bf8` with the ADR-0010
correction/control pin `db8ed3af` and its committed correction-file digest.
The prepared and published notes and archives match byte for byte. Publication
run [37125803488](https://github.com/rgomids/axiom/actions/runs/37125803488) stopped
before Issue recording and Release PR handoff because two unquoted Bash RHS
comparisons interpreted Markdown as glob patterns. This can both reject
identical notes and accept different strings. Quoting fixes the verifier, but
ADR-0010 correctly continues selecting the original immutable control commit.
Changing that pin in published notes or adopting current main silently would
break the existing provenance contract. A published version cannot be skipped
as though it had never shipped.

## Decision

Extend ADR-0010 with opt-in `--repair-revision <full SHA>` for an already
published immutable stable recovery release. Preserve original source,
correction revision, correction digest, prepared run, notes, archives and tag.
The additional SHA identifies execution code, not corrected release metadata.

The repair revision must be a distinct descendant of the original correction
revision, belong to first-parent main and have every required CI check green;
a Check Run counts only when both its name and its app match the context and
`integration_id` of the versioned ruleset.
Only a clean checkout at that exact SHA executes repair code. Original pins
are independently validated through the original pinned recovery protocol.
The published release must retain those pins, be immutable/non-draft/stable,
and target the original source. The selected code still performs the original
notes, asset, checksum, tag, latest and delivery checks; there is no bypass.

Envelope v4 binds `repair_revision` in addition to all v3 fields. It describes
only remaining Issue/Project effects and Release PR label handoff. Repair
refuses preparation, missing/draft/mutable releases or altered provenance;
it cannot create/publish/edit a release, upload/delete assets, create/move a
tag or change latest. Classification is rechecked before effects. Every retry
requires a fresh envelope and human authorization, then the protected release
environment. The workflow records the execution SHA and authorized digest;
immutable notes continue recording original correction provenance.

`status`, `verify` and `publish` accept the explicit repair SHA. `verify` can
read original pins from published notes, as before; it never discovers a repair
SHA. `start --repair-revision` may use that code only to verify the recovered
previous release before starting a new version. It does not alter the next
release's source or preparation/publication inputs. No default path adopts a
repair revision or falls back from a failing pinned verifier. An explicit repair
SHA is never ignored: when the target (or, for `start`, previous) release has no
published recovery provenance, the command refuses instead of continuing on the
normal path.

## Alternatives considered

- Quote the comparisons only: fixes future control commits but leaves the
  existing pinned verifier and future previous-release checks broken.
- Repin corrections or edit published notes: rejected; changes immutable
  release provenance, and would no longer match the prepared set.
- Always use current main: rejected; execution authority floats silently.
- Skip v0.3.0: cannot erase an already published immutable version or its
  outstanding delivery bookkeeping.
- Explicit additional reviewed execution pin: adds a bounded protocol and
  another human-reviewed field while retaining every original check and byte.

## Consequences

Repair adds one immutable execution input without changing release assets or
metadata. It needs review, merge, green CI, explicit SHA selection and new
remaining-effects authority. It retains the dedicated publication credential
and environment gate of ADR-0012. It amends that credential routing: Release
API calls use the PAT; Issue records and PR labels use the protected job's
`GITHUB_TOKEN`, passed as `AXIOM_RELEASE_REPOSITORY_TOKEN`, with Issues write
and Pull requests write. Delivery trusts `github-actions[bot]` records, so PAT
authorship would invalidate recording and idempotency. The workflow refuses
either missing credential before scripts execute. Both workflow jobs add only
`checks: read`, which reading the repair revision's Check Runs requires. Local tests prove that a broken original
verifier still fails by default, repaired verification requires opt-in,
provenance/CI drift refuses and completion makes no release or asset write.
Remote permission checks and human acceptance remain separate evidence.

## Revisit when

A general execution-provenance store or detached signed release evidence is
adopted. This amendment introduces neither automatic repair nor a general
permission to replace historical control code.

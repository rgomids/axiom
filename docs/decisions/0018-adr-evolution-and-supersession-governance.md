# ADR-0018 — ADR evolution and supersession governance

## Status

Accepted on 2026-10-05 by the human decision recorded in
[Issue #169](https://github.com/rgomids/axiom/issues/169) ("Human decision
recorded — 2026-10-05"). Upstream adoption of its implementation remains
subject to PR review. This is not release-publication authority.

This ADR formalizes the precedent already used by
[ADR-0010](0010-windows-native-filesystem-boundary.md#decision) and
[ADR-0015](0015-installer-host-eligibility-os-family-architecture.md#supersedes).
It supersedes no ADR.

## Context

Accepted ADRs are durable decision records, and Axiom forbids silently
rewriting approved decisions. The ADR template listed lifecycle statuses, but
nothing defined how a later decision replaces only one statement of an older
accepted ADR:

- a change can be correct under a newly intended architecture while
  contradicting an older ADR;
- the older ADR stays discoverable and can look fully current;
- editing the older ADR in place rewrites history;
- a new ADR without an annotation in the older one leaves no navigation from
  the obsolete statement to the current rule;
- no repository check validated these relationships.

ADR-0015 superseded only the Windows OS-version bound of ADR-0010. That
fragment was preserved struck through, annotated in place with a link to
ADR-0015, and ADR-0015 listed it in a `Supersedes` section. Issue #169 asked
for one canonical convention and recorded the human choice between an explicit
`Partially Superseded` status and `Accepted` plus structured annotations.

## Decision

### 1. Accepted ADRs are historical records

Once an ADR is accepted, its decision text, rationale, alternatives and
consequences are not deleted or rewritten to read as if a later decision had
always been the original one. Corrections of spelling, formatting or broken
links that do not change meaning remain ordinary documentation maintenance.

A material change to an accepted decision is recorded in a **new ADR** that
explains why the earlier decision no longer fully applies. The new ADR follows
the normal acceptance boundary: it stays `Proposed` until a human accepts it.
Passing checks, implementation readiness or a merge never infer acceptance.

### 2. Lifecycle

The lifecycle status of an ADR is the first word of the first non-empty line
of its `## Status` section, ignoring a leading `**` and a leading `Status:`
label. It is exactly one of:

| Status | Meaning |
|---|---|
| `Proposed` | Recorded for human decision; not yet in force. |
| `Accepted` | In force, possibly with some fragments partially superseded. |
| `Superseded` | Entirely replaced by a later accepted ADR. |
| `Rejected` | Considered and declined; never in force. |

There is no `Partially Superseded` status. Free text may follow the status
word, so existing Status sections remain valid without rewriting.

### 3. Partial supersession

When a later accepted ADR replaces only a bounded statement or section of an
earlier ADR:

- the earlier ADR **keeps the status `Accepted`**;
- the historical fragment stays visible and may be rendered with Markdown
  strikethrough;
- a supersession annotation is placed immediately after the affected
  fragment;
- decisions not covered by an annotation remain in force.

Only an `Accepted` ADR supersedes another one; a `Superseded` ADR keeps the
supersession relationships it established while it was in force. Only an
`Accepted` or `Superseded` ADR carries supersession annotations.

The annotation is a blockquote whose first line is exactly:

```markdown
> Superseded by [ADR-NNNN — <exact title of the superseding ADR>](NNNN-<slug>.md#<section>), accepted YYYY-MM-DD.
```

- The link targets a sibling ADR file by its filename. The `#<section>`
  anchor is optional and, when present, names a heading of that ADR,
  preferably the section that owns the current rule.
- The link label is the superseding ADR's H1 title without `# `.
- The date is the superseding ADR's acceptance date (see below).
- Further `>` lines may state the exact scope: what is superseded, what
  remains in force, and that the text is preserved for traceability.

The annotation is navigation and history metadata. It must not restate the
replacement decision in full; the superseding ADR owns it. Example:

```markdown
Windows support is limited to local NTFS user storage on
~~Windows 10 (1809+) and Windows 11~~ amd64.

> Superseded by [ADR-0015 — Installer host eligibility is OS-family and architecture based](0015-installer-host-eligibility-os-family-architecture.md#decision), accepted 2026-10-04.
> Only the struck OS-version bound is superseded; the rest remains in force.
```

### 4. Full supersession

When a later accepted ADR replaces an entire ADR, the earlier ADR keeps all of
its text and the first line of its `## Status` section becomes exactly:

```markdown
Superseded by [ADR-NNNN — <exact title of the superseding ADR>](NNNN-<slug>.md), accepted YYYY-MM-DD.
```

The link targets the ADR file itself, without an anchor. The original status
text stays below that line, as its own paragraph, as history.

**Acceptance date.** The acceptance statement of an ADR is the first paragraph
of its `## Status` section, or, for a `Superseded` ADR, the preserved paragraph
immediately after its `Superseded by` line; it starts with `Accepted`. Its
acceptance date is the single `Accepted: YYYY-MM-DD` value when the statement
starts with `Status: Accepted`, and otherwise the single distinct
`YYYY-MM-DD` date in the statement's first sentence. Other dates in the Status
section are never acceptance dates. An ADR whose acceptance date cannot be
determined this way cannot serve as a supersession target.

### 5. Backlink

A superseding ADR, partial or full, has a `## Supersedes` section that links
to every earlier ADR it supersedes by its file path (optionally with an anchor
to the affected section) and identifies the superseded fragment. Every ADR
link in that section must correspond to a `Superseded` status or at least one
partial-supersession annotation in the linked ADR that points back to it.
Links to non-ADR artifacts (for example Specifications) may also appear there.

### 6. Identity

An ADR is identified by its file path under `docs/decisions/`, never only by
its number: two accepted ADRs share the prefix `0010`. Its H1 title is
`# ADR-NNNN — <title>`, where `NNNN` is the filename prefix and the title is
unique. Labels of annotations, of the `Superseded by` status line and of
index entries must match that title.

### 7. Index

The [ADR index](README.md#index) lists every ADR exactly once as:

```markdown
- [ADR-NNNN — <title>](NNNN-<slug>.md) — <Status> <free text>
```

- `<Status>` (optionally bold) is the ADR's lifecycle status.
- A partially superseded ADR keeps `Accepted` and adds the clause
  `Partially superseded by [ADR-MMMM](MMMM-<slug>.md)`, listing every ADR its
  annotations point to, separated by `, ` or ` and `. This is a navigation
  note, not a lifecycle status.
- A fully superseded ADR's entry starts its status text with
  `Superseded by [ADR-MMMM](MMMM-<slug>.md)`.
- Links in these notes are labelled exactly `ADR-MMMM` and target the ADR
  file without an anchor.

### 8. Structural validation versus semantic review

The repository harness deterministically validates, offline and fail-closed,
only the structure above: lifecycle values, annotation and full-supersession
syntax, existing and non-self-referential targets, accepted targets, titles,
dates and anchors, reciprocal backlinks, unique identity and index
reconciliation.

Whether an implementation, Specification or new ADR **semantically contradicts
an accepted ADR** is not decided by tooling. It is a review responsibility:
reviewers inspect the ADRs affected by a change, and an unreconciled material
contradiction is a blocking (major or higher) finding until the
implementation complies with the accepted ADR or an ADR evolution under this
lifecycle resolves it.

## Alternatives considered

1. **A `Partially Superseded` status.** Visible at a glance, but it labels the
   whole document as obsolete when most of it remains in force, and readers
   still need fragment-level navigation. Rejected by the human decision.
2. **Edit the earlier ADR in place.** Simplest to read, but it destroys
   decision history and hides that the rule changed. Rejected.
3. **New ADR only, no annotation in the earlier one.** Preserves history but
   leaves the obsolete statement looking current, with no path to the current
   rule. Rejected.
4. **Machine-readable front matter for supersession.** Easier to parse, but it
   requires rewriting every existing ADR and duplicates information already
   expressed in prose and links. Rejected; the annotation line is the
   structured metadata.

## Consequences

### Positive

- One representation for partial and full supersession, consistent with the
  ADR-0010 / ADR-0015 precedent.
- Readers can navigate from an obsolete statement directly to the ADR that
  owns the current rule, and back.
- Broken, self-referential, one-sided or unindexed supersession relationships
  fail repository validation instead of drifting.

### Negative / trade-offs

- The ADR status alone does not reveal partial supersession; the index note
  and in-place annotations must be read.
- The annotation, `Supersedes` section and index entry must be kept in sync;
  the checker enforces it, at the cost of stricter formatting.
- Structural validation cannot detect a semantic contradiction that no one
  annotated. Review remains responsible for it.

## Revisit when

- ADRs adopt a machine-readable metadata format;
- a supersession relationship cannot be expressed as a fragment annotation
  (for example many interleaved fragments);
- ADRs move out of `docs/decisions/` or gain non-sibling references;
- deterministic semantic conflict detection becomes possible through an
  explicit contract.

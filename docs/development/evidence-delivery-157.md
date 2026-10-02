# Pinned recovery of unpublished v0.3.0

Recorded: 2026-10-02. Status: proposed for human review; no approval inferred.
Scope: restore metadata omitted from one immutable squash commit through an
explicitly authorized recovery contract. Source version, Runtime, release
branch, tags and repository settings remain unchanged.

## Sources and classification

- [PR #157](https://github.com/rgomids/axiom/pull/157) declares
  `Related-Issues: #153` and `Completes-Issues: none`.
- Squash `684b5aca88b1176d317fa86aa223b6bc654e4519` has only its subject
  and a short summary; both metadata lines are absent. Its three changed files
  are Specification 004's `spec.md`, `plan.md` and `tasks.md`.
- [Issue #153](https://github.com/rgomids/axiom/issues/153) remains open;
  contract reconciliation does not deliver its implementation or acceptance.
- [PR #161](https://github.com/rgomids/axiom/pull/161) implements I153-T01
  only, explicitly leaves I153-T02/T03 pending, and declares #153 related only.
- [Failed preparation](https://github.com/rgomids/axiom/actions/runs/37024363626)
  refuses #157's absent metadata while generating notes for `v0.3.0` at
  `b79d3bf8cbf21247ca30cae06ff000e7f89a5adf`.

Proposed correction in `.github/delivery-corrections.txt`:

```text
684b5aca88b1176d317fa86aa223b6bc654e4519 related=153 completes=none
```

The existing legacy #143/#148 records are unchanged. The correction restores
exactly the relationship declared by #157; it cannot release or close #153.
Human review/approval is still required before integration.

## Complete release range

The first-parent range after the previous manifest-changing release commit
`2182b3e1fa587a110182dd11042c0298dbfe8597` contains five commits:

| PR | Metadata | Related | Completes |
|---|---|---|---|
| #158 | Declared | #86 | #86 |
| #159 | Declared | #86 | None |
| #157 | Proposed correction | #153 | None |
| #161 | Declared | #153 | None |
| #160 | Release commit exception | None | None |

No other ordinary commit lacks metadata. Expected delivered set is exactly #86.
#153 remains related only. Changelog retains feature #158 and fixes #159/#161.

## Validation and its limits

- Original exact revision: delivery resolution fails on #157, as expected.
- `./scripts/test-delivery.sh`: PASS. Before the record was added, two updated
  inventory assertions failed; after it was added, both passed. Existing tests
  already prove related-only corrections resolve without inferring completion,
  uncommitted corrections have no effect, and invalid metadata fails closed.
- Actual-history isolated Git fixture: PASS. Its synthetic release-shaped
  revision differs from the original release tree only in
  `.github/delivery-corrections.txt`, preserves the original release parent and
  resolves all five commits with `issues=86`, `issue.86=#158` and just the one
  permitted undeclared release commit. #157 reports `metadata=corrected`.
- `scripts/release-notes.sh` on that fixture: PASS, using read-only GitHub
  lookup of #86's title. Notes contain all three visible changelog entries,
  only #86 under delivered Issues, and the synthetic fixture revision.
- Fixture with the correction placed after the original release commit:
  correctly fails `revision is not the release commit of 0.3.0`.

Fixture objects are temporary test data in a separate repository; no work
commit or ref was written to the source repository. Raw outputs and the
validation driver remain at `/tmp/axiom-delivery-157`.
This fixture demonstrates metadata behavior, not a releasable production SHA.

Additional executed checks: `./scripts/validate-repository.sh .`,
`./scripts/check-sensitive-files.sh .`,
`gitleaks detect --source . --no-git --redact`,
`bash -n scripts/test-delivery.sh` and `git diff --check`: all PASS.
Git HEAD and live main remained `b79d3bf8cbf21247ca30cae06ff000e7f89a5adf`.

## Authorized recovery implementation

On 2026-10-02 the maintainer explicitly authorized implementation of the pinned
metadata recovery proposal. ADR-0010 records the direction. That authority does
not authorize merge, publication, tags or repository-setting changes.

The implementation adds opt-in correction/control SHA and committed-file digest
inputs to status, prepare and publish. Correction rows must preserve existing
source records and target only commits in the original source release range.
Both SHAs must be in first-parent main history; control descends from source.
All correction bytes are read from committed objects, never working-tree state.

Workflows use sibling checkouts: source builds/verifies original archives;
control executes pinned recovery scripts. Prepared metadata, envelope version 3
and notes bind both correction pins. Download verification discovers published
pins and validates notes, source archives and delivered Issue records. Default
flow retains its existing source-only envelope version 2. Project configuration
remains at source; the control checkout cannot expand Project authority.

Initial default-flow tests found Bash 3.2's nounset behavior for an empty optional
argument array. Dispatch now uses a nonempty complete argument array with
conditional pin additions. This preserves default and recovery dispatch paths.

Executed recovery validation:

- `python3 scripts/test-release-corrections.py`: PASS, 12 offline Git cases.
- `./scripts/test-delivery.sh`: PASS.
- `./scripts/test-release-pipeline.sh`: PASS against original committed source;
  closed archive inventory, provenance and no-publication workflow verified.
- Repository validation and Gitleaks: PASS.
- Bash syntax and diff hygiene: PASS.
- Focused recovery integration: PASS, including published download verification,
  original artifact verifier and Project authority, pins/notes drift, RC refusal,
  envelope changes and exact preparation/publication dispatch inputs.
- Historical sync/reconciliation: PASS. Draft, prerelease, wrong source,
  duplicate and malformed published recovery provenance each refuse without
  effects; normal recovered publication repairs only #86 to Released.
- Independent final engineering/security review: no blocking findings. Automatic
  verification trusts published notes plus protected-main history; original
  explicit pins detect replacement of that record, not an independent signed
  publication-authority history.
- `./scripts/test-release-flow.sh`: PASS, 365 checks at final runtime state;
  focused recovery integration also passed all 24 checks. Fake GitHub effects
  are isolated fixtures; no production mutation occurs in this suite.

The original release SHA is still immutable and remains blocked without these
explicit pins. Recovery requires human review/merge of this correction/control
change, then a green merged correction SHA and its exact committed-file digest.
Remote preparation is pending that gate. No recovery preparation or publish
workflow was dispatched during implementation. Product Go code is unchanged.

Live read-back on 2026-10-02 confirms #160 was merged by a human as
`chore(main): release 0.3.0`, at the unchanged original source SHA. Manifest is
0.3.0 and changelog has feature #158 plus fixes #159/#161. Source Linux/macOS
and release-contract CI passed; original preparation and delivery sync failed
on the omitted metadata. Local recovery results do not replace pending CI of
the correction/control PR. Title-protection PR #162 remains open with green
checks; its merge and required-check configuration remain human follow-ups.
The GitHub release lookup for v0.3.0 returned HTTP 404 and the remote tag lookup
returned no ref. No publication exists for this candidate.

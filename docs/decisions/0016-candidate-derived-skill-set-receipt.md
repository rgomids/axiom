# ADR-0016 — The owned upgrade publishes the candidate's Codex skill-set receipt when it runs as the candidate

## Status

Proposed on 2026-10-04 for
[Issue #186](https://github.com/rgomids/axiom/issues/186). Becomes Accepted
when the human reviewer merges the PR that introduces it. It replaces the
interim S7 limitation recorded as
[T20 — Codex skill-set receipt after an upgrade](../specifications/004-mvp-v1-baseline/evidence-s7.md#t20--codex-skill-set-receipt-after-an-upgrade);
that historical Evidence is not rewritten.

## Context

`.axiom-skill-set.receipt` in the Codex skill root states the skill-set
version, binary compatibility and manifest digest of the skill files Axiom
published there. It is derived from a binary's runtime constants, which
release archive format 1 does not carry: the archive's `skills-manifest.txt`
says `1`/`1`, the runtime uses `2`/`2`.

S7/T20 therefore made the owned upgrade publish skill files but never the
receipt, ending `partial` / `refresh_required` and asking for `axiom
first-run`. It recommended keeping that for S7 and choosing between (a) a
release-format extension and (c) executing the upgraded binary "before RC".
Neither was decided, and stable releases shipped with the limitation. Issue
#186 is its user-visible effect: every published upgrade that changed skill
text (v0.1.0–v0.3.0 → v0.4.1, reproduced with the published binaries) ended
`install_status=partial`, exit 1, with `runtime codex status` failing although
every skill was `equivalent`.

The release installer, however, never runs an older binary: the POSIX facade,
the bootstrap and the Windows installer all execute the **candidate's own**
`axiom` from the verified bundle (`install-release`, or `upgrade` from the
v0.1.x facade). That process already holds the candidate's runtime constants.

## Decision

1. The owned upgrade publishes the Codex skill-set receipt derived from the
   running binary **only when the running binary proves it is the candidate
   release**: its build provenance is a clean release whose version and
   revision equal the candidate's checksum-verified release metadata, and its
   embedded skill files are exactly the candidate's skill files.
2. The receipt is a separate authorized effect (`skill_receipt`) in the
   preview and its digest, planned after every skill file, published through
   the same anchored skill-root session with an exact expected-revision
   recheck, a private stage named with the upgrade stage prefix, and
   post-commit confirmation. An interrupted stage is a recoverable leftover of
   the marked operation only.
3. A present receipt is replaced only when it is this binary's receipt or one
   an earlier Axiom revision wrote for that root. An unrecognized receipt is
   preserved and the result is `partial` / `skillReceipt=conflict`.
4. When the proof in (1) does not hold (for example an installed older binary
   given a newer archive through `axiom upgrade`), behavior is unchanged:
   `partial` / `refresh_required`, next action `axiom first-run`.
5. Build provenance never grants ownership; ownership of each skill and of the
   receipt is decided exactly as before.

## Alternatives

- **(a) Release-format v2 carrying the runtime constants.** Works for any
  running binary but changes archive format, `LoadCandidate`, the build and
  every installer layer, and does not help archives already published.
- **(c) Execute the upgraded binary after publication.** A new trust topology
  (running a freshly published executable from the installer).
- **Keep `partial`.** Leaves every supported upgrade that changes skill text
  ending in a failure exit, which is the defect in #186.

## Consequences

- The release installer converges binary, installation receipt, Codex skill
  files and Codex skill-set receipt in one authorized operation; an equivalent
  rerun is a no-op.
- Correctness depends on the candidate's skill files equaling the running
  binary's embedded skills, which the check enforces rather than assumes.
- Claude is still converged by `axiom first-run`, unchanged.
- Harder to change later: the `skill_receipt` effect kind and the
  `skillReceipt` values (`current`, `unchanged`, `refresh_required`,
  `conflict`) become part of the upgrade preview/result contract.

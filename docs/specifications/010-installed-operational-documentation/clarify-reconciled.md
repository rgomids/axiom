# Specification 010 — Reconciled Clarification

**PROPOSED — R1, 2026-10-20.** Companion to [Spec R1](spec-reconciled.md),
not a new acceptance register. Original [Q1–Q6 exact decisions](clarify.md#decision-register),
[D1/D3 acceptance](acceptance-resume-2.md) and literal [C1](acceptance-resume-3.md)
remain unchanged. [Decision H1](reconciliation.md#grouped-human-decision-h1) binds
all proposed differences; no C2 direction has been adopted.

| Question / accepted scope | R1 disposition / contract impact |
| --- | --- |
| Q1 corpus | Preserve English dedicated operational corpus + generated CLI/man reference, all minimum classes, exclusions and local links. Authoring and generation depend on content/CLI contracts, not process-image experiments |
| Q2 installation binding | Preserve private per-installation namespace, physical target, immutable generations, aliases/spaces, no global/PATH/CWD fallback. Replace exact whole-image proof with compiled release-content descriptor plus validated current installation association. This revises accepted Q2/D2 and needs ADR-0034 acceptance |
| Q3 packaging | Preserve external verified bundle, closed explicit versions and full provenance/integrity. No embed/extraction of corpus. Descriptor is small build identity, not compiled documentation. v1 and Runtime formats/receipts remain intact |
| Q4 lifecycle | Preserve binary publication boundary, generation-before-binary ordering, truthful partials, retained generations, no automatic rollback/GC, independent compatible A return and bounded recovery. Replace blanket P1 approval gate with causal internal checkpoints after baseline acceptance |
| Q5 access/man | Preserve accepted D3 grammar/schema direction, verified-only paths, expected/installed/loaded distinction, shared page coverage and session-local man selection. Qualify exact image-related field semantics to association facts; do not label a disk hash as a memory-image hash |
| Q6 / AC-08 | No change: real separate Codex/Claude, effective source denial, independent file/network observation with descendants, controls and full historical matrix. Missing evidence stays UNVERIFIED/NOT_RUN. No shortcut via transcript, synthetic Runtime or green CI |

## C1 and C2 disposition proposed

C1 already permits development budgets and requires measured complete T05/T06
before T07; it did not remove the historical G1→T05/T06 edge. R1 expressly
removes that edge **only after H1 acceptance**. Keep every accepted numeric budget
and the mandatory stop on overflow/incomplete coverage; do not silently raise
limits or truncate content. No automatic quota tuner or new quota protocol.

C2 proves the experimental inode/later-hash inference false on native macOS.
R1 proposes a different observable guarantee and records the loss of original
whole-file image identity. Its regression requires compiled A to reject B content
regardless of a synthetic matching on-disk binding. If association cannot be
proved, refusal suffices. Exact replay inputs/outcomes remain in
[historical G1](gate-g1-resume-3.md#c2--executing-file-mutability-before-binary-pin).
A future result against R1 never rewrites that FAIL. No raw Darwin ABI/cgo choice
is adopted; keeping the old guarantee instead means further bounded research.

## Migration and source qualification

v0.1.0 remains an **UNVERIFIED migration-floor objective**, as qualified in D1-e.
Do not narrow it through a new minimum version or call passive archive inspection
native migration proof. Maintain exact historical receipt variants and qualify
the existing producer/consumer matrix before claiming the floor. Unqualified
routes must refuse unsupported effects and present honest explicit local-install
or inspection guidance; this does not certify an alternative migration. If any
required historical route cannot be qualified, submit its exact failure/support
impact for grouped human disposition before delivery. Source install remains
excluded from qualified docs, as already accepted; no new source feature is added.

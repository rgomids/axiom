# Specification 010 — Reconciled delivery baseline

## Status and authority

**PROPOSED — revision R1, 2026-10-20; RECONCILE only.**
[AXM-37](https://linear.app/rgomids/issue/AXM-37), same Draft
[PR #335](https://github.com/rgomids/axiom/pull/335). This is a complete proposed
behavioral baseline, not acceptance or a continuation of Implement. It takes
precedence over earlier Specification/Clarification text **only after explicit
human acceptance of this exact revision and the linked ADR proposal**. Until
then, accepted contracts and G1 BLOCKED remain in force. Historical
[Spec](spec.md), [Clarification](clarify.md), decisions and Evidence are unchanged.
[Reconciliation](reconciliation.md) owns the delta, classifications and decision.

## Outcome and journeys

An operator or supported native agent with an installed Axiom can find and read
its operational documentation without checkout or documentary internet access.
GitHub Docs-as-Code supplies approved English text; executable help/tree owns
command syntax, flags and authority. Reading requires no generator, Go, Git,
Provider login or Runtime. Inference transport is separate from documentation.

1. Install a previously trusted local release archive on a supported host; inspect
   version/provenance and discover the local index from the selected executable.
2. Read concepts, installation/verification, Project/Runtime/Profile and Work
   Item/Workflow guides, generated command/configuration reference and recovery
   guidance. Required links close locally; network-only extras are labelled.
3. Consult generated `man axiom` and shared command pages using explicit session
   selection. Compare actual selected page with executable; select A explicitly
   if PATH chooses A and man chooses B. Without man, Markdown/help still work.
4. Upgrade using verified assets; read a complete validated generation or an
   explicit refusal. Retain independent compatible A and B, including correct
   skills/configuration/man selection; never pretend in-place downgrade or
   Project/Execution rollback occurred.
5. Skills use discovery/help and verified local text, distinguish expected,
   installed and loaded facts, and report absence/mismatch/unknown honestly.

## Minimal observable contract

| Stable requirement | Required behavior in R1 |
| --- | --- |
| OD-01–02 / corpus | Dedicated English operational text plus reference/man generated from the real public command/flag tree. Explicit inventory, full source commit, source/output mapping, transformation identity, hashes and notices. No historical Specs/ADRs/Evidence, maintainer harness, secrets, personal paths, private Provider material or whole-docs wildcard |
| OD-03–05 / association | Resolve the physical executable's installation-local namespace, independent of CWD, repeated PATH lookup, Project, global registry or checkout. Compare a descriptor compiled into the running program with the complete installed generation, binding and observed current executable. Version labels alone never suffice; same-label/different-content and crossed A/B must be detected. Paths with spaces and supported read-only aliases work; copied/moved binaries require authorized rebinding or refuse |
| OD-04,06 / discovery | `axiom --json docs inspect` and `axiom --human docs inspect`, additive docs schema v1 in the existing completion envelope. Preserve accepted states/reasons/exits and verified-only paths, except image-proof meanings expressly revised in the Plan. No network, writes, extraction, repair, credentials or Runtime startup. Help/version remain available without docs; consumers revalidate inventory/content at actual read |
| OD-07–09 / delivery | Prepare complete immutable generation and installation binding before binary publication. Confirmed binary publication activates the new set; receipt/skill/ack failures remain partial effects, never fictional rollback. Expected revisions and installation-wide coordination prevent stale publishers. Concurrent readers obtain complete verified content or refuse; uncertain publication requires bounded inspection and fresh exact recovery authority |
| OD-09 / compatible return | Keep independent verified A while B remains installed. Explicitly select compatible A and matching Runtime roots/man; prove actual consultation and local inspection. Preserve shared data and B; refuse if A cannot read current state or skill roots conflict. Ordinary in-place downgrade remains refused |
| OD-10–11 / safety | Preserve ADR-0005 confinement, traversal/type/link/ownership/permission/ACL checks, bounded inputs, deterministic fault handling and expected revisions. Verify package and per-file integrity; preserve foreign notes/residue as conflicts. No automatic repair, rollback, pruning, privilege change or shell-profile edits; retain generations |
| OD-12–13 / delivery matrix | macOS arm64, Linux amd64 and Linux arm64. Generated root/shared man coverage accounts for every public leaf and flag. No generator on installed host. Windows remains withdrawn. Legacy v1 has docs absent until qualified migration. Source installs remain explicitly unqualified for documentary delivery; help stays usable |

### Explicit identity change requiring acceptance

R1 verifies **release-content and installation association**, not identity of the
entire executable file at process launch or every mapped memory page. The compiled
descriptor identifies the exact docs/command map, full source/build target and
expected embedded skill inventory. Current on-disk binary digest is an observed
installation fact, never evidence of original memory-image bytes. A mismatch,
unsupported descriptor, ambiguous operation or unprovable installation binding
returns refusal without usable verified paths. A surviving A process must not
accept B content: unequal descriptors refuse; equal descriptors permit only the
same exact documentary content after installation/content validation.

This replaces the exact image-proof interpretation of OD-03–05/08 and D2; it is a
material relaxation, not a claim that the failed C2 oracle passed. It requires
[ADR-0034 Proposed](../../decisions/0034-release-content-documentation-association.md).
The known C2 fixture must reject B by compiled descriptor; inode equality or a
hash of the later file is insufficient. Coordinated substitution of all mutable
local metadata cannot establish authenticity; no resistance to arbitrary
same-UID process/memory manipulation is added. Do not automatically exclude the
controlled mutation: test it at declared boundaries under the new explicit oracle.

## Acceptance and failure evidence

All R1 product criteria are **NOT_RUN**. Earlier experiment outcomes retain their
own PASS/FAIL/UNVERIFIED status. Evidence records exact source/artifact hashes,
commands, host/architecture/filesystem, expected/observed outcomes and limitations.
Reuse one observation across criteria where it establishes the same fact; no
obligation to create a new harness or human gate for each criterion.

| ID | Sufficient observable evidence / failure cases |
| --- | --- |
| AC-01 | Clean install from trusted local artifact on all three native host rows, network denied and no checkout/maintainer home. Read all minimum content classes and local journey links; record selected paths/hashes. No Runtime needed |
| AC-02 | A/B explicit paths, switched PATH, arbitrary CWD/spaces and supported aliases; same-label different bytes, crossed docs/skills, relocation/copy, old A after B replacement and controlled C2 mutation. Verify correct descriptor/content or refusal; no fallback. Original whole-image byte identity is expressly no longer claimed |
| AC-03 | Equivalent reinstall, A→B upgrade, competing publishers with different receipt roots and overlapping readers. Inject incomplete write, space/quota, permissions, pre-publication, uncertain publication, confirmed commit/pre-ack and cleanup failures. Read-back proves complete content or refusal and truthful confirmed effects |
| AC-04 | Actual compatible independent A return with matching docs/skills/man while B and user data remain intact; incompatible state/roots refuse. Separately preserve downgrade refusal |
| AC-05 | Traversal, unsafe links/types/counts, changed ancestor/leaf/lock at declared boundaries, permissions/ACL, missing/extra/modified/truncated files, hash/manifest mismatch, bounded sizes and repeat install. Preservation snapshots prove no foreign/user data loss |
| AC-06 | Same approved source/descriptor produces identical documentary outputs twice. Exact real CLI/help parity, full leaf/flag/man coverage, exclusions, local links and secret scans; strict old/new shell/Go refusal before candidate execution. Full corpus measured within accepted budgets before packaging, no coverage cuts |
| AC-07 | All native host rows: session-selected `man axiom`, public coverage, page provenance, A/B PATH/man divergence and correct explicit selection, upgrade/return, spaces and missing reader. Markdown/help remain usable |
| AC-08 | Preserve Q6 in full: separate fresh real native Codex and Claude sessions, effective denial of checkout, maintainer memories/config and remote documentary sources; independent OS file/network observation including descendants, positive/negative controls, A/B and mismatch/absent/tampered matrix. Explain command/readiness from installed docs and observed CLI. Fake Runtime/transcript/self-report is insufficient |

[Q6's seven-step protocol](clarify.md#q6--reproducible-native-proof-protocol)
remains normative without reduction. Apply host-specific controls on the supported
native Runtime/host combinations; do not silently replace any historical V9 row
with a representative run. If a row/control is unavailable, record NOT_RUN or
UNVERIFIED and return one grouped decision, without full qualification. Inference/
auth transport is declared separately; opaque server retrieval and training
knowledge are not proven absent. No Runtime air-gap claim. No vendor features,
permissions or disposable environments are presumed available or authorized.

Failure diagnostics identify expected/observed components, confirmed/unknown
effects and safe next inspection. Missing/invalid/unknown docs never yield usable
verified paths or silently fall back. Foreign data stays intact; an independently
changed state invalidates preview authority. No installation success is inferred
from documentary checks.

## Boundaries

No service, registry, generic transaction framework, new parser/library, broad
refactor, publication, merge, acceptance, Linear mutation or successor unblock.
Axiom != Lingo; Project != Repository; Role != Model; Execution != Agent;
Provider != Transport; Integration != MCP; Skill != workflow source of truth;
Evidence != chat history. [Proposed Plan](plan-reconciled.md) and
[Tasks](tasks-reconciled.md) describe one later authorized implementation.

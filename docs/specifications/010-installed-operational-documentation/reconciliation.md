# AXM-37 — Integrated reconciliation R1

**PROPOSED — 2026-10-20; Human decision required H1.**
Same [Draft PR #335](https://github.com/rgomids/axiom/pull/335),
branch `docs/axm-37-offline-operational-documentation`, entry head
`5dd66917d601357126f3c53c85b01581ead3d664`. RECONCILE only: no implementation,
new Work Item/Execution/dispatch, Linear mutation, merge, release or acceptance.
This public technical synthesis uses the user's explicit publication authority
for relevant reconciliation documentation; no raw private Provider data imported.

## Reading order and authority

1. [Specification R1](spec-reconciled.md): complete proposed observable contract.
2. [Clarification R1](clarify-reconciled.md): preserved Q1–Q6 and named deltas.
3. [ADR-0034 Proposed](../../decisions/0034-release-content-documentation-association.md):
   bounded identity change, alternatives and proposed partial supersession.
4. [Plan R1](plan-reconciled.md): minimal implementation and exact textual wire delta.
5. [Tasks R1](tasks-reconciled.md): stable T01–T16, causal inputs/E01–E16/AC coverage.
6. [Evidence](evidence-reconcile.md): checks actually run and recoverable handoff.

New companion files form one reviewable baseline; original `spec.md`, `clarify.md`,
`plan.md`, `tasks.md`, acceptance artifacts, ADR-0033 and experimental Evidence
remain byte-for-byte unchanged from entry. Their headers and statuses are historical;
read their exact acceptance addenda, not an inferred latest header. R1 becomes a
replacement baseline only if H1 is explicitly accepted. Older authority does not
activate R1 or restart Implement during this turn.

## Original value versus later commitments

AXM-37's original defect was an operator consulting maintainer checkout files to
operate an installed product. Product requirements: offline local text, deterministic
installation association, executable help primacy, coherent lifecycle/integrity,
user-data preservation, native agent proof and subsequently explicit man support.
The card's historical Windows wording is constrained by the accepted withdrawal.

Later accepted Q1–Q6 chose dedicated English text, private namespace, external
bundle, immutable retained generations, independent A return, CLI discovery,
session-local man and strong native source observation. ADR-0033 joined these
cross-layer choices. Plan D1/D3 selected closed format/layout/discovery detail with
qualified quotas/migration floor; C1 removed final quota freeze from early G1,
without opening T05/T06. D2/D4 algorithms remained qualification obligations.
Experiments and fixtures never became accepted production mechanisms.

Historical positive results: 105 public leaves/549 flag registrations/15 proposed
pages; three native 24-case reader suites and fresh B; 21-transition fixture models.
These are bounded observations, not current product coverage or full native AC
qualification. Historical negative C2: native macOS append before pin, same inode,
active A consumed B fixture hashes, oracle **FAIL**. Emulated amd64 image proof
also previously failed; emulation is not native Evidence. Full migrations/man/
offline/Runtime trials remain NOT_RUN/UNVERIFIED. See
[G1 report](gate-g1-resume-3.md), [C1 literal](acceptance-resume-3.md),
[D1/D3 qualifications](acceptance-resume-2.md) and their linked exact inputs.

## Requirement disposition and old → proposed matrix

E = essential; R = necessary with reduced mechanism/sequencing; D = deferred;
X = unjustified for this slice. Risk column states residual cost/coverage, not
permission to waive an accepted requirement. All material deltas stay proposed.

| Requirement / origin | Class / why | Old → proposed | Risk, compatibility and AC impact |
| --- | --- | --- | --- |
| Offline minimal corpus, local links and exclusions / original + Q1 | E / direct user value | Preserve all content classes; dedicated approved English sources and explicit allowlist | Authoring/reconciliation cost remains; no smaller content promise. AC-01/06 |
| Real CLI/help parity + man / original clarification + Q5/D3 | E / correct operation | Same tree-driven reference, shared root/group coverage, session selection, mismatch diagnosis | Native reader differences still need qualification; 15 pages are measured proposal, not immutable command count. AC-06/07 |
| Local installation/version/provenance/integrity / original + Q2/Q3 | E / prevent silent mixed content | Same namespace, independent roots, strict hashes/full source, verified-only paths and no fallback | No authenticity from paired mutable hashes. Same-label content must remain distinguishable. AC-02/05 |
| Exact original whole-file/memory-image proof / D2 interpretation | R / association is needed; forensic image proof exceeds minimum journey | Unqualified OS image probes → compiled content descriptor + verified installation association or refusal; explicit Spec/Q2/D2/ADR partial change | Loses exact launch-file/image proof; coordinated mutable local substitution still unauthenticated. C2 old FAIL retained, new B-content-refusal oracle must pass. AC-02 semantics revised explicitly |
| C1 final quota measurement / D1-d + human C1 | R / bounded inputs essential; circular sequencing is not | Preserve every numeric development/security budget; complete T05/T06 and freeze before T07 | Overflow/missing coverage still stops packaging for human disposition; no coverage trimming. AC-05/06 unchanged |
| G1 blocks corpus and generation / Plan/Tasks | X / authorship does not depend on OS image identity | Remove T04→T05/T06; baseline approval permits content work in later Implement | Earlier content may need revision; no unsafe install enabled. Measured corpus reduces speculation before packaging |
| D2→D4 model, installer→reader, broad qualification→offline/agent edges / task ordering | R / integration needs proof; model/fixtures/independent trials do not need completed predecessor | T03 depends T01; T10 depends T04/T07; T14/T15 depend integrated T09/T12, not entire T13 | Fixture results never count as installer qualification; all branches join T16. AC coverage unchanged |
| Lock/publication/recovery / Q4/D4/ADR-0005 | E with R mechanism / stale authority and mixed sets remain risks | Reuse anchored sessions/revisions; one installation-derived writer scope, complete immutable objects before binary commit; no generic transaction engine | Same-target/different-receipt contention, unsafe/replaced lock and uncertainty must be tested; fixture's exact flock/path not automatically adopted. AC-03/05 |
| Independent return and retention / Q4 | E / accepted compatibility behavior | Preserve A/B return, current-state check, skills/man verification and no downgrade/automatic rollback | Disk accumulation remains. Return can refuse on incompatibility; refusal alone never proves successful return. AC-04 |
| Migration floor v0.1.0 / D1-e qualified | E qualification obligation / preserve accepted target honestly | Keep objective UNVERIFIED until historical receipt/artifact/native matrix proves it | High residual testing cost; no narrower floor selected. Failure requires explicit support decision. AC-03/06 |
| Source-derived docs install / Q3 accepted exclusion | D / separate delivery route, already unqualified | Keep explicit unqualified/absence diagnostics; no borrowed release docs | Source users lack qualified corpus; help remains. No new acceptance reduction |
| Q6/AC-08 native source observation / accepted Q6 | E / actual agent journey requires source Evidence | Preserve seven-step protocol, real Codex/Claude, independent OS file/network/descendant observation, controls and full matrix | Largest residual qualification cost; provider opacity and unavailable rows remain explicit UNVERIFIED. No new representative-only substitute |
| Three native host rows / existing support and ADR-0021 | E / platform commitment | Preserve macOS arm64, Linux amd64/arm64; Windows still withdrawn | No emulation/cross-build substitution; filesystem assumptions declared. AC-01–07 and applicable V9 matrix |
| Automatic GC/rollback, source expansion, global registry, service, generic protocol/framework | D for future authorized feature; X for this slice | No such production scope introduced; generation retention and explicit existing recovery remain | Disk cost/manual inspection accepted; no live-process registry or cleanup proof machinery needed now |
| Separate approval at every unit / planning choice | X / human decisions belong to material contracts and external effects | One baseline decision, later one cohesive Implement prompt, internal executable checkpoints | New material delta, quota/floor change or unavailable required controls still needs human disposition. No release/acceptance authority inferred |

## Architecture consistency review

| ADR | R1 relationship and disposition |
| --- | --- |
| ADR-0033 | Preserve corpus, external package, private generation topology, lifecycle ordering/retention/return, CLI/skills/man, security/platform/Q6. Exact executing-image interpretation conflicts; ADR-0034 proposes bounded partial supersession, old text/index status intact until human acceptance |
| ADR-0005 | No supersession. Keep exact authorized roots, supported boundary replacements, writer concurrency, complete-state readers, faults/permissions/limits, truthful commit/recovery. No arbitrary same-UID/power-loss/media guarantees added or used to erase C2 |
| ADR-0016 | No supersession. Candidate-running proof/embedded skill equality and positive ownership alone authorize candidate Codex receipt. Documentary descriptor cannot substitute; Claude first-run remains separate |
| ADR-0018 | Preserve immutable accepted decisions. New ADR stays Proposed; proposed scope under `Proposed partial supersession`, because validator forbids active `Supersedes` on Proposed. Reciprocal annotations/index changes occur only after acceptance |
| ADR-0021 | Preserve Windows withdrawal and exact prepared-artifact/native acceptance boundary. No Windows row or new unsupported platform |

Authenticity = trust in release acquisition. Package integrity = observed hashes/
inventory. Current disk identity = observed file plus installation facts. Memory/
launch image identity = not established by those facts. R1 compares a constant in
the running program to the verified installed content; it does not upgrade any
one of these claims into another. This distinction is the reason an ADR change
is required even though local same-UID security guarantees are not broadened.

## Grouped human decision H1

**Pending. No option adopted.** One decision on the exact submitted revision:

- **Recommended:** accept Spec/Clarification/Plan/Tasks R1 and ADR-0034's bounded
  partial replacement of ADR-0033 image-proof semantics. Accept Plan's exact
  unshipped discovery-field amendment and compiled G/D identity; permit causal
  content/contract work before safety integration; keep C1 budgets, v0.1.0 floor
  objective, platforms, all AC/Q6 requirements and independent human acceptance.
- **Alternative:** retain exact original image-proof guarantee and G1 baseline;
  keep C2 BLOCKED and authorize a separate bounded research proposal before any
  dependent implementation. No qualified stronger mechanism is currently proven.

Trade-off: recommended path reduces native identity research/ABI maintenance and
false sequencing, but explicitly relinquishes whole-file launch/image proof.
Most cost still lies in content, coordinated strict v2 installer evolution,
publication fault tests and unchanged full native/Q6 qualification. If security
needs original image identity rather than content association, reject recommendation.

Harder to change after shipment: descriptor meaning, discovery schema, namespace,
package migration and published links. No quota increase, lower migration floor,
platform reduction or AC-08 relaxation is bundled as an unnoticed concession.
If H1 is accepted, first record exact acceptance and ADR-0018 annotations/index
without changing old rationale. Then create the separately requested complete
Implement prompt for one cohesive T01–T16 execution with internal checkpoints.
Acceptance of this documentary baseline is not itself that implementation prompt,
a G1 PASS, product acceptance, merge/release authority or AXM-39/12 unblock.

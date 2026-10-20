# Implementation resumption 2 — D1/D3 acceptance reconciliation

**2026-10-20 — accepted directions, qualified budgets/baselines; G1 BLOCKED.**

This addendum records the exact [human decision](https://github.com/rgomids/axiom/pull/335#issuecomment-6301616799)
on [contracts revision 1 at f96d5df](https://github.com/rgomids/axiom/blob/f96d5dfb8efb4b4d6018b7a523e89532734c006f/docs/specifications/010-installed-operational-documentation/contracts-proposed.md).
That document, its vectors, previous Evidence, accepted Plan/Tasks and ADRs remain
historical inputs, unchanged. Their Proposed/NOT_RUN headers describe their own
revision, not the disposition below. The [implementation authority](https://github.com/rgomids/axiom/pull/335#issuecomment-6298186645)
still binds Tasks `088eb89ac1d2a4328a956cec7047990b566e9934`, the same Draft PR #335
and its existing implementation continuation; no new Execution is created.

| Disposition | Current authority and remaining condition |
| --- | --- |
| D1-a ACCEPTED | Independent archive/metadata/installation receipt v2 and inventory/provenance/man-map/binding versions as specified; Runtime/skill formats preserved |
| D1-b ACCEPTED | Closed encodings, strict ordering/validation and acyclic SHA-256 DAG as specified; executable codec qualification remains outstanding |
| D1-c ACCEPTED, subject to D2 | Private `.axiom-docs/axiom/`, immutable generations/bindings, no global mutable pointer; concrete mechanism needs native T02/T04 qualification and review |
| D1-d ACCEPTED WITH QUALIFICATION | Exact table is an initial development, negative-test and measurement budget. Final public quotas require measurement of final corpus and CLI/man reference. No coverage reduction; any excess needs comparative proposal and human disposition before freeze |
| D1-e ACCEPTED WITH QUALIFICATION | Preserve v1, shell/Go matrix and pre-execution refusal. v0.1.0 remains the migration-floor objective, never certified support. Each historical receipt/artifact variant needs positive Evidence and applicable native tests. Missing proof stays UNVERIFIED; narrowing needs explicit disposition |
| D3-a ACCEPTED | Additive docs schema v1 in existing envelope; canonical `axiom --json docs inspect` / `axiom --human docs inspect`; no suffix flags or global grammar change |
| D3-b ACCEPTED | Paths only in verified state; observed/expected/loaded skills remain distinct, loaded unknown without independent observer; readers revalidate identity/content |
| D3-c ACCEPTED | Root/shared man pages with complete leaf/flag coverage and local session selection; no persistent/global man configuration |

Acceptance does not complete T01/T02, qualify a resolver/lock, approve G1, accept
AC-01–AC-08, authorize production before G1, or start SDD review. No status,
blocker, merge, release or closure authority follows. The T11 suffix notation in
historical Tasks is interpreted by accepted D3-a's leading presentation grammar.

## C1 — minimum human decision requested: quota freeze dependency

**PENDING; no option below has been adopted.** [Tasks T01](tasks.md#t01--define-closed-versioned-contracts-and-compatibility-oracles)
says freeze values only with coverage Evidence; T04/G1 requires concrete reviewed
quotas and freezes P1 outputs. D1-d expressly requires final corpus/reference
measurement before final public freeze. T05/T06 author and generate those outputs,
but depend on T04/G1. Current output-count and measured-source estimates cannot
establish final bytes or source-to-output/link/reference completeness.

Recommended narrow disposition: distinguish the **development contract** from
**final public quota qualification**. Permit G1 eventually to use the accepted
initial table plus bounded E01 estimates, with final quota freeze withheld until
T05/T06 produce the full measured corpus/reference and complete coverage. An excess
or missing coverage must stop dependent packaging and return for comparative human
disposition; never trim coverage. This would require an explicit, traceable
Plan/Tasks gate addendum, not an inferred interpretation of the current approval.
It does not waive T02, T03, T04 review or any other G1 requirement, and does not
itself permit starting T05.

Alternative: keep G1's final-freeze requirement and explicitly authorize a bounded
measurement-only subset of T05/T06 before G1, solely to produce the complete
reviewable corpus/reference specimen and measurements. No installer, resolver,
production lock, public command or packaging work follows. This advances more
content work before mechanism qualification and needs a precise scope/authority
addendum. Keeping current gates unchanged instead leaves P1 blocked.

Later change costs: after public shipment, quota changes affect both pre-execution
readers and published archive compatibility. Neither option permits premature
public support claims or retrospective alteration of accepted history.

The [current checkpoint](evidence-resume-2.md) submits G1 as **BLOCKED**, with C1
and all remaining E02/E03 obligations visible. Approval of C1 alone cannot pass G1.

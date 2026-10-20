# AXM-37 — Clarification Evidence and handoff

> Historical Clarification snapshot follows. Its pending decisions and handoff
> describe rev.1, not current disposition. See the dated
> [reconciliation](#reconciliation--2026-10-20) for Q1–Q6 acceptance and the Proposed ADR.

## Scope and authority snapshot

2026-10-20; Clarification only. Existing branch
`docs/axm-37-offline-operational-documentation`, open Draft
[PR #335](https://github.com/rgomids/axiom/pull/335); local and remote initial head
`99afe2a93c21bfe3767a61133eec8c374b690abd`. Initial worktree clean.
[Specification acceptance](https://github.com/rgomids/axiom/pull/335#issuecomment-6292970482)
was read directly; exact accepted Spec blob
`ce7e58630eda9a844d13b7422aaff4b9a91c2ee3` is preserved.
The original [specify Evidence](evidence.md) remains historical; its pending
acceptance statements describe that earlier snapshot, not current authority.

Authorized public output is original repository-based technical synthesis.
No raw Linear content, local user records, credentials or chat transcript are
copied. Present instruction authorizes Clarification artifacts and their bounded
validation/delivery on the existing branch/PR, not successor execution or acceptance.

## Discovery and concurrency

- `gh pr view 335 --json headRefName,headRefOid,baseRefName,state,isDraft,updatedAt,body,statusCheckRollup`:
  open/Draft, expected branch and initial SHA. Initial PR Checks and
  delivery-metadata both SUCCESS. This says nothing about a future changed head.
- `gh api repos/rgomids/axiom/issues/comments/6292970482`: human acceptance of
  exact Spec rev.1; explicitly excludes Q1–Q6 choices and successor authority.
- `gh api repos/rgomids/axiom/pulls/335/reviews`: review `5586081096`, native
  COMMENTED on initial SHA, technical favorable opinion; not native approval.
- Linear `get_issue(id=AXM-37, includeRelations=true)` and bounded
  `list_comments(issueId=AXM-37, limit=20)`: In Review, no Blocked by, blocks
  AXM-39/AXM-12, related AXM-30/AXM-33. Clarify gate comment
  `a89e5867-e3ca-4efe-984c-32e7864cbcb2` and prompt
  `806428ba-11ff-4a19-9f22-cac7cdb0f4ea` found. The gate records a Cursor
  notification accepted for dispatch; that is a concurrency signal, not proof
  of active or completed execution. No execution-attempt result was identified
  in the returned discussion snapshot. No Linear mutation performed.
- `git worktree list --porcelain` filtered to the exact branch: this checkout
  alone owns that local branch. `git status --short` remained clean and
  `gh pr view 335 --json headRefOid,state,headRefName` still matched immediately
  before documentary edits. Remote/local snapshots do not prove absence of an
  unobserved external worker; recheck before push and preserve any drift.

Canonical Execution check used real installed read-only surfaces:
`axiom --json version` returned Axiom `0.20.0`, revision `dd2ab51f609e`, clean;
`axiom workflow list --help` requires an exact configured Project identity.
`axiom project list --help` and `axiom --json project list --include-archived`
were inspected. No verified Project-to-AXM-37 binding was established; Project
is not Repository and a Provider card does not supply that binding. Unrelated
local records were not traversed or copied. **Execution remains unknown**, not
absent. No workflow start/advance/fact, simulated ledger or status change.
These commands are maintainer discovery, not AC-08 native-runtime Evidence.

## Bounded source observations

All source references below describe initial head `99afe2a9`; they support
[proposals](clarify.md), not implementation or human acceptance.

| Source / reproducible inspection | Observation and decision relevance |
| --- | --- |
| [SDD](../../../.agents/skills/axiom-sdd/SKILL.md), [documentation governance](../../documentation.md), [security policy](../../../.agents/policies/security.md), [contribution policy](../../../.agents/policies/contribution.md) | Clarify material behavior/security/compatibility; use reversible defaults when justified; preserve source ownership, classification and human gates |
| [Installation](../../installation.md), [commands](../../commands.md), [conceptual model](../../architecture/conceptual-model.md) | Operational inputs are mixed with development/historical reference; cutting all docs is inappropriate. Existing prose says three archive skills while builder has four; discrepancy preserved, no unrelated correction |
| [Builder](../../../scripts/build-release-archives.sh), [bootstrap](../../../scripts/install.sh), [source installer](../../../scripts/install-axiom.sh) | Explicit clean/development build provenance, short revision, bundle hashes, independent destination defaults/overrides. No operational text/man assets in current builder |
| [Facade](../../../scripts/install-release.sh), [archive parser](../../../internal/install/archive.go), [upgrade](../../../internal/install/upgrade.go) | Strict schemas and bounded archive validation; current receipt fields lack doc identity. Adding assets/metadata is a compatibility decision. Binary effect is confirmed separately before receipt/skill convergence |
| [Fresh installation](../../../internal/install/fresh_posix.go), [Runtime integration](../../../internal/codexruntime/integration.go) | Binary commit can precede receipt acknowledgment; Runtime roots/receipts are distinct and installed files do not prove a native session loaded them |
| [Command tree](../../../internal/cli/commands.go), [help](../../../internal/cli/help.go), [go.mod](../../../go.mod) | Existing custom command/help contracts use Go flag; no adopted Cobra dependency. New doc discovery is not presumed available |
| [ADR-0005](../../decisions/0005-bounded-local-filesystem-threat-model.md), [ADR-0016](../../decisions/0016-candidate-derived-skill-set-receipt.md) | Bounded threats, logical commit and recovery; candidate-derived Codex receipt proof and separate Claude first-run remain unchanged |
| [ADR-0018](../../decisions/0018-adr-evolution-and-supersession-governance.md), [ADR-0021 Status](../../decisions/0021-windows-server-amd64-native-acceptance.md#status) | No silent accepted-decision rewrite; Windows withdrawal preserved without attempting unrelated ADR reconciliation |

Inspection used bounded `rg`/`sed` reads; no builder/installer/release mutation
was executed. No delegation: no verified strictly lower capability profile was
available; main session integrated all bounded fronts directly. Skills applied:
`axiom-sdd`, `linear`; Runtime skill routing was read to identify supported
read-only Execution discovery. No Runtime skill operation was dispatched.

## Validation record

Only the two new Clarification/Evidence Markdown files are intended changes.
No Spec, accepted ADR, CLI, installer, skill, Plan or Tasks is changed.
Validation outcomes are recorded after actual execution below; delivery SHA
and remote CI belong to the existing PR's delivery record after push.

| Actually executed check | Result / limit |
| --- | --- |
| `./scripts/validate-repository.sh .` | PASS; repository/harness, registry, ADR governance and sensitive-file validation. Native behavioral scenarios explicitly SKIP / UNVERIFIED |
| Ephemeral Python link/authority/scope check via stdin | PASS: 46 local links including supplied anchors; six Q1–Q6 Proposed/Pending rows; OD/AC coverage markers; accepted Spec blob unchanged; exactly two scoped new files. Structural coverage only, not semantic acceptance |
| `git diff --check` | PASS; new-file contents also reviewed via `git diff --no-index -- /dev/null <file>` and staged diff before commit |
| `gitleaks detect --source . --no-git --redact --no-banner` | PASS; no leaks found |

Staged-sensitive-file result and final changed-head CI are reported in the PR
delivery body after actual execution; initial-head CI is not reused as proof.

AC-01–AC-08: **NOT_RUN**. No clean install, native Codex/Claude session,
confinement experiment, man generation/reader trial, upgrade, rollback or
product test is claimed. Q6 is a proposed protocol, not a certified capability.

## Handoff

Q1–Q6 recommendations are **Proposed**, human dispositions **Pending** in
[the decision register](clarify.md#decision-register). First concrete decision:
Q3 external verified release assets versus bounded embed/authorized extraction.
No choice is inferred from Spec acceptance, review, Linear status or CI.
Q2/Q3/Q4/Q5 introduce architectural/public compatibility commitments requiring
explicit human disposition and later ADR acceptance. Existing ADRs are preserved.
Stop with Clarification pending; do not authorize Plan, Tasks, implementation,
Execution mutation, merge, release, AXM status/blocker changes or product acceptance.

## Reconciliation — 2026-10-20

### Authority, concurrency and bounded scope

Stage `reconcile`, initial local/remote head
`ab7bee28bd6bca8d161f098cc90f75ebfbea74a5`; worktree clean, expected branch,
one open Draft PR #335. Initial Clarification blob
`df4e62ab724a7d2792c25495bea4d47124cf2704` matched the human approval anchors.
All six specific approval comments and the
[consolidation](https://github.com/rgomids/axiom/pull/335#issuecomment-6294090913)
were read directly through `gh api`; author `rgomids`, scope and exact revision
were checked. See the [register](clarify.md#decision-register) for the six links.
No raw Provider payload, personal local records, credentials or chat history
is imported. Current explicit instruction authorizes documentary reconciliation,
Proposed ADR, validation and normal delivery on the same branch/PR.

`gh pr view`, exact-head `gh pr list`, `git ls-remote`, clean status and filtered
`git worktree list --porcelain` confirmed the same branch/head with no duplicate
PR or local checkout owning that branch. Bounded process/CWD inspection found
this executor in this checkout; other observed Codex/Claude processes had other
working directories. That snapshot cannot disprove an unobserved remote worker.
No start/restart or Runtime operation was issued. Linear issue/comment read-back
confirmed In Review, no Blocked by, AXM-39/AXM-12 still blocked by AXM-37; the
reconcile handoff dispatch is a notification receipt, not an Execution result.
Canonical Project/Execution binding remains unknown; unrelated local state was
not traversed and no ledger was reconstructed. No Linear mutation performed.

Initial-head CI read-back: PR Checks and delivery-metadata SUCCESS at `ab7bee28`.
These checks do not validate the changed reconciliation head. Remote head/branch,
PR state, comments and local scope must be rechecked immediately before push;
drift requires stopping rather than overwriting. Final delivery SHA and CI
read-back belong in the PR delivery record after actual push.

### Documentary result and architecture review

- `clarify.md` rev.2 records Q1–Q6 Accepted **in direction**, each with its exact
  approval link; alternatives and detail boundaries remain. Spec rev.1 and its
  original Evidence bytes are unchanged; the original Clarification snapshot
  above is retained and explicitly dated as historical.
- [ADR-0033](../../decisions/0033-installation-bound-operational-documentation.md)
  is Proposed, indexed once. Local index and remote main both ended at 0032
  before choosing the number. Identity/layout, package/compatibility, activation/
  return and CLI/skills/man share one executing-binary-to-generation invariant;
  one proposal avoids splitting that contract into independent authorities.
- Reviewed against ADR-0005 bounded threats/logical publication, ADR-0016
  candidate proof and separate Claude convergence, ADR-0018 human acceptance/
  evolution, and ADR-0021's Windows withdrawal. No semantic conflict requiring
  supersession was identified. Accepted ADR files are untouched. Any later
  contradiction requires a distinct human decision and ADR-0018 evolution, with
  reciprocal changes only after acceptance. Structural checks alone cannot
  establish semantic agreement.
- No concrete pathname, format version, CLI syntax/JSON schema, generator,
  commit algorithm or host observer is adopted. Original source inspection was
  checked against the current builder, strict archive/facade/receipt readers,
  upgrade/fresh-publication boundary and CLI/help contracts; no product mutation.
- Security review preserves exact authorized roots, ownership, bounded reads,
  link/replacement/concurrency checks, trusted acquisition versus local hashes,
  foreign material, truthful committed effects and read-only discovery. No new
  capability or stronger same-UID/power-loss guarantee is claimed.

Main session performed all work; no proven strictly lower subagent capability
was available. Skills: `axiom-sdd`, `axiom-document`,
`axiom-architecture-decision`. No delegation or native Runtime dispatch.

### Verified checks and limits

| Executed check | Result / limit |
| --- | --- |
| `python3 scripts/check-adr-governance.py .` | PASS, 34 ADR files including the new Proposed record; structural validation only |
| `./scripts/validate-repository.sh .` | PASS, including harness/registry/ADR/sensitive-file checks; native Runtime behavior explicitly SKIP / UNVERIFIED |
| `gitleaks detect --source . --no-git --redact --no-banner` | PASS, no leaks in the worktree scan |
| Ephemeral Python local-link/authority/history check via stdin | PASS: 64 local links/anchors, six live approval comments with exact author/revision/blob, accepted Spec blob and historical Evidence/ADR bytes preserved, ADR Proposed, no Plan/Tasks |
| `git diff --check` | PASS; tracked documentary diff reviewed, new ADR reviewed in full |

Staged security checks and final-head delivery/CI are recorded in the PR
when completed; no unexecuted check is claimed here.
No Go build/unit suite was needed for this Markdown-only change; required remote
CI remains separate. AC-01–AC-08: **NOT_RUN**, including offline install, compatible
return, man readers, native Codex/Claude confinement and OS/network controls.
Nothing in these documentary checks is functional acceptance.

### Remaining gate

Reconciliation resolves the stale Pending decision register; it does not accept
ADR-0033 or authorize Plan. **BLOCKED for declaring READY_FOR_plan** pending
independent exact-revision ADR acceptance, review/validation of reconciliation
and separate Plan authority. Q1–Q6 do not need repeat answers. Concrete contract
encodings/migration/mechanisms and native Q6 feasibility remain future design/
proof obligations. Infeasible Evidence yields UNVERIFIED and a new human decision
before weakening any AC. No Plan, Tasks, implementation, merge, release, AXM
closure, dependency unblocking or acceptance effect follows from this delivery.
